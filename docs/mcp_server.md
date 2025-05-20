# MCP Server 集成笔记

## 架构概述

本项目采用了适配器模式将自定义的服务接口(`abstract.Service`)与第三方库 `mcp-go` 的 `MCPServer` 集成。这种设计模式允许我们在保持系统灵活性的同时，充分利用 MCP (Model Control Protocol) 的能力。

## 核心组件

### 1. MoLingServer

`MoLingServer` 是一个容器/协调器，它持有 `MCPServer` 实例和一组 `abstract.Service` 实现。它的主要职责是：

- 初始化和管理 MCP 服务器实例
- 加载并注册各种服务
- 提供服务启动功能 (通过 SSE 或 STDIO)

```go
// MoLingServer 服务器实例
type MoLingServer struct {
    ctx        context.Context     // 上下文
    server     *server.MCPServer   // MCP服务器实例
    services   []abstract.Service  // 服务列表
    logger     zerolog.Logger      // 日志记录器
    mlConfig   config.MoLingConfig // 配置
    listenAddr string              // SSE模式监听地址，如果为空，则使用STDIO模式
}
```

### 2. abstract.Service 接口

这是我们定义的服务抽象，所有具体服务都需要实现这个接口：

```go
// Service defines the interface for a service with various handlers and tools.
type Service interface {
    Ctx() context.Context
    // Resources returns a map of resources and their corresponding handler functions.
    Resources() map[mcp.Resource]server.ResourceHandlerFunc
    // ResourceTemplates returns a map of resource templates and their corresponding handler functions.
    ResourceTemplates() map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc
    // Prompts returns a map of prompts and their corresponding handler functions.
    Prompts() []PromptEntry
    // Tools returns a slice of server tools.
    Tools() []server.ServerTool
    // NotificationHandlers returns a map of notification handlers.
    NotificationHandlers() map[string]server.NotificationHandlerFunc

    // Config returns the configuration of the service as a string.
    Config() string
    // LoadConfig loads the configuration for the service from a map.
    LoadConfig(jsonData map[string]interface{}) error

    // Init initializes the service with the given context and configuration.
    Init() error

    MlConfig() *config.MoLingConfig

    // Name returns the name of the service.
    Name() comm.MoLingServerType

    // Close closes the service and releases any resources it holds.
    Close() error
}
```

### 3. MCPServer (第三方)

`MCPServer` 是 `mcp-go` 库提供的服务器实现，用于处理 Model Control Protocol 的请求：

```go
// MCPServer implements a Model Control Protocol server that can handle various types of requests
// including resources, prompts, and tools.
type MCPServer struct {
    // Separate mutexes for different resource types
    resourcesMu            sync.RWMutex
    promptsMu              sync.RWMutex
    toolsMu                sync.RWMutex
    middlewareMu           sync.RWMutex
    notificationHandlersMu sync.RWMutex
    capabilitiesMu         sync.RWMutex

    name                   string
    version                string
    instructions           string
    resources              map[string]resourceEntry
    resourceTemplates      map[string]resourceTemplateEntry
    prompts                map[string]mcp.Prompt
    promptHandlers         map[string]PromptHandlerFunc
    tools                  map[string]ServerTool
    toolHandlerMiddlewares []ToolHandlerMiddleware
    notificationHandlers   map[string]NotificationHandlerFunc
    capabilities           serverCapabilities
    paginationLimit        *int
    sessions               sync.Map
    hooks                  *Hooks
}
```

## 适配器模式实现

`MoLingServer` 通过 `loadService` 方法实现了适配器模式，将 `abstract.Service` 接口的能力映射到 `MCPServer` 实例：

```go
// loadService 加载服务
func (m *MoLingServer) loadService(srv abstract.Service) error {
    // 添加资源
    for r, rhf := range srv.Resources() {
        m.server.AddResource(r, rhf)
    }

    // 添加资源模板
    for rt, rthf := range srv.ResourceTemplates() {
        m.server.AddResourceTemplate(rt, rthf)
    }

    // 添加工具
    m.server.AddTools(srv.Tools()...)

    // 添加通知处理程序
    for n, nhf := range srv.NotificationHandlers() {
        m.server.AddNotificationHandler(n, nhf)
    }

    // 添加提示
    for _, pe := range srv.Prompts() {
        // 添加提示
        m.server.AddPrompt(pe.Prompt(), pe.Handler())
    }
    return nil
}
```

这种适配方式创建了一个清晰的映射关系：

| abstract.Service 方法 | MCPServer 方法 |
| -------------------- | ------------- |
| Resources() | AddResource() |
| ResourceTemplates() | AddResourceTemplate() |
| Tools() | AddTools() |
| NotificationHandlers() | AddNotificationHandler() |
| Prompts() | AddPrompt() |

## 具体实现示例

为了更好地理解这些抽象接口如何在实际代码中被使用，以下是 `Browser` 服务的实现示例，展示了如何注册各种工具及其处理函数。

### Browser 服务工具注册

BrowserServer 类继承了 MLService 基类，并在其初始化方法中注册了多个工具：

```go
// 添加 mcp 工具
// prompt
bs.AddPrompt(pe)

// 导航
bs.AddTool(mcp.NewTool(
    "browser_navigate",
    mcp.WithDescription("Navigate to a URL"),
    mcp.WithString("url",
        mcp.Description("URL to navigate to"),
        mcp.Required(),
    ),
), bs.handleNavigate)

// 截图
bs.AddTool(mcp.NewTool(
    "browser_screenshot",
    mcp.WithDescription("Take a screenshot of the current page or a specific element"),
    mcp.WithString("name",
        mcp.Description("Name for the screenshot"),
        mcp.Required(),
    ),
    mcp.WithString("selector",
        mcp.Description("CSS selector for element to screenshot"),
    ),
    mcp.WithNumber("width",
        mcp.Description("Width in pixels (default: 1700)"),
    ),
    mcp.WithNumber("height",
        mcp.Description("Height in pixels (default: 1100)"),
    ),
), bs.handleScreenshot)

// 点击
bs.AddTool(mcp.NewTool(
    "browser_click",
    mcp.WithDescription("Click an element on the page"),
    mcp.WithString("selector",
        mcp.Description("CSS selector for element to click"),
        mcp.Required(),
    ),
), bs.handleClick)

// ... 更多工具注册
```

### 工具处理函数实现

每个工具都有一个对应的处理函数，例如 `handleNavigate` 函数实现：

```go
// handleNavigate handles the navigation action.
func (bs *BrowserServer) handleNavigate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    url, ok := request.Params.Arguments["url"].(string)
    if !ok {
        return nil, fmt.Errorf("url must be a string")
    }

    err := chromedp.Run(bs.Context, chromedp.Navigate(url))
    if err != nil {
        return mcp.NewToolResultError(fmt.Sprintf("failed to navigate: %v", err)), nil
    }
    return mcp.NewToolResultText(fmt.Sprintf("Navigated to %s", url)), nil
}
```

### MLService 接口实现

BrowserServer 通过继承 MLService 类获得了 abstract.Service 接口的基础实现。MLService 基类提供了管理工具、资源和通知处理程序的方法：

```go
// MLService 提供了 Service 接口的基础实现
type MLService struct {
    Context               context.Context
    lock                  *sync.Mutex
    resources             map[mcp.Resource]server.ResourceHandlerFunc
    resourcesTemplates    map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc
    prompts               []PromptEntry
    tools                 []server.ServerTool
    notificationHandlers  map[string]server.NotificationHandlerFunc
    mlConfig              *config.MoLingConfig
}

// AddTool 添加工具
func (mls *MLService) AddTool(tool mcp.Tool, handler server.ToolHandlerFunc) {
    mls.lock.Lock()
    defer mls.lock.Unlock()
    mls.tools = append(mls.tools, server.ServerTool{Tool: tool, Handler: handler})
}

// Tools 返回工具列表
func (mls *MLService) Tools() []server.ServerTool {
    mls.lock.Lock()
    defer mls.lock.Unlock()
    return mls.tools
}
```

### 接口调用流程示例

以下是一个完整的调用流程示例，展示了用户请求如何从 MCP 服务器传递到具体服务实现：

1. 用户发送一个工具调用请求，例如导航到某个URL：
   ```json
   {
     "tool": "browser_navigate",
     "params": {
       "url": "https://example.com"
     }
   }
   ```

2. MCP 服务器接收到请求，查找名为 "browser_navigate" 的工具。

3. 服务器找到之前注册的工具处理函数 `handleNavigate`。

4. 服务器调用该处理函数，传入上下文和请求参数。

5. `handleNavigate` 函数执行导航操作并返回结果。

6. MCP 服务器将结果返回给用户。

这种流程使得系统能够以一种解耦的方式处理各种请求，同时保持代码的可维护性和可扩展性。

## 服务初始化与注册流程

### 1. 创建 MoLingServer 实例

```go
// NewMoLingServer 创建MoLingServer实例
func NewMoLingServer(ctx context.Context, srvs []abstract.Service, mlConfig config.MoLingConfig) (*MoLingServer, error) {
    mcpServer := server.NewMCPServer(
        mlConfig.ServerName,
        mlConfig.Version,
        server.WithResourceCapabilities(true, true),
        server.WithLogging(),
        server.WithPromptCapabilities(true),
    )
    // Set the context for the server
    ms := &MoLingServer{
        ctx:        ctx,
        server:     mcpServer,
        services:   srvs,
        listenAddr: mlConfig.ListenAddr,
        logger:     ctx.Value(comm.MoLingLoggerKey).(zerolog.Logger),
        mlConfig:   mlConfig,
    }
    err := ms.init()
    return ms, err
}
```

### 2. 初始化并加载所有服务

```go
// init 初始化MoLingServer实例
func (m *MoLingServer) init() error {
    var err error
    for _, srv := range m.services {
        m.logger.Debug().Str("serviceName", string(srv.Name())).Msg("Loading service")
        err = m.loadService(srv)
        if err != nil {
            m.logger.Info().Err(err).Str("serviceName", string(srv.Name())).Msg("Failed to load service")
        }
    }
    return err
}
```

### 3. 启动服务

`MoLingServer` 支持两种服务模式：基于 SSE 的 HTTP 服务和基于 STDIO 的命令行服务。

```go
// Serve 启动服务
func (s *MoLingServer) Serve() error {
    mLogger := log.New(s.logger, s.mlConfig.ServerName, 0)

    // 监听地址不为空，启动sse服务
    if s.listenAddr != "" {
        // 设置监听地址
        ltnAddr := fmt.Sprintf("http://%s", strings.TrimPrefix(s.listenAddr, "http://"))
        // 设置控制台输出
        consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
        // 设置多级写入器
        multi := zerolog.MultiLevelWriter(consoleWriter, s.logger)
        // 设置日志记录器
        s.logger = zerolog.New(multi).With().Timestamp().Logger()
        // 设置日志记录器
        s.logger.Info().Str("listenAddr", s.listenAddr).Str("BaseURL", ltnAddr).Msg("Starting SSE server")
        // 设置日志记录器
        s.logger.Warn().Msgf("The SSE server URL must be: %s. Please do not make mistakes, even if it is another IP or domain name on the same computer, it cannot be mixed.", ltnAddr)
        return server.NewSSEServer(s.server, server.WithBaseURL(ltnAddr)).Start(s.listenAddr)
    }

    // 监听地址为空，启动stdio服务
    s.logger.Info().Msg("Starting STDIO server")
    return server.ServeStdio(s.server, server.WithErrorLogger(mLogger))
}
```

## 设计优势

这种适配器模式的架构设计有几个主要优势：

1. **解耦** - `abstract.Service` 接口与 MCP 实现完全分离，允许独立开发和测试服务。

2. **可扩展性** - 新服务只需实现 `abstract.Service` 接口，无需了解 MCP 的内部细节。

3. **组合模式** - 多个独立服务可以组合为一个统一的 MCP 服务入口。

4. **第三方库隔离** - 如果未来需要更换或升级 MCP 库，只需修改适配层，无需更改服务实现。

5. **统一接口** - 所有服务通过相同的抽象接口提供功能，保证了一致性。

## 使用示例

要创建一个新的服务并集成到 MCP 服务器，需要：

1. 实现 `abstract.Service` 接口
2. 将服务实例添加到 `MoLingServer` 的服务列表中
3. 启动 `MoLingServer`

伪代码示例：

```go
// 1. 创建实现 abstract.Service 的自定义服务
myService := NewMyCustomService(ctx)

// 2. 创建 MoLingServer 实例，包含该服务
services := []abstract.Service{myService}
mlServer, err := server.NewMoLingServer(ctx, services, config)
if err != nil {
    log.Fatal(err)
}

// 3. 启动服务
err = mlServer.Serve()
if err != nil {
    log.Fatal(err)
}
```

## 总结

MCP 服务器集成采用了适配器设计模式，将自定义服务接口与第三方 MCP 库对接。这种架构提供了良好的解耦性、可扩展性和一致性，使系统能够灵活地添加新服务并与第三方库集成。

`MoLingServer` 作为适配器，协调各个服务并将它们的能力统一暴露给 MCP 服务器，实现了一个优雅的架构设计。 