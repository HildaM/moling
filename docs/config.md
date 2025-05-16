# MoLing 配置系统分析

## 概述

MoLing 是一个基于 computer-use 和 browser-use 的 MCP (MoLing Computer Protocol) 服务器，为用户提供本地部署、无依赖的办公 AI 助手功能。本文档将从源码角度深入分析 MoLing 的配置系统，包括配置文件格式、加载过程、模块配置等方面。

## 配置文件基本信息

### 配置文件位置与格式

MoLing 的主配置文件采用 JSON 格式，默认位于用户目录下的 `.moling/config/config.json`：

- macOS/Linux: `/Users/<username>/.moling/config/config.json`
- Windows: 类似路径（尚未完全测试）

如果配置文件不存在，可以通过 `moling config --init` 命令自动创建。

### 目录结构

MoLing 在用户主目录下创建 `.moling` 文件夹，包含以下子目录：

```
.moling/
├── logs/      # 日志文件
├── config/    # 配置文件
├── browser/   # 浏览器缓存
├── data/      # 数据文件
└── cache/     # 缓存文件
```

## 配置文件加载流程

1. 应用启动时，通过 `main.go` → `cli.Start()` → `cmd.Execute()` 进入主命令流程
2. 在 `rootCmd.PersistentPreRunE` 中，执行 `mlsCommandPreFunc` 初始化基本环境
3. 当执行 `moling config` 命令时，通过 `ConfigCommandFunc` 处理配置相关操作
4. 基本配置通过命令行参数（如 `--base_path`, `--debug`, `--listen_addr` 等）进行覆盖

### 配置命令实现

配置命令由 `cli/cmd/config.go` 中的 `ConfigCommandFunc` 函数实现，主要功能是：

```go
// ConfigCommandFunc 执行 "config" 命令
func ConfigCommandFunc(command *cobra.Command, args []string) error {
    // 初始化日志
    // 检查当前配置文件
    // 获取所有服务的配置
    // 格式化并输出/保存配置
}
```

主要步骤包括：
1. 初始化日志记录器
2. 检查现有配置文件
3. 遍历所有服务，获取它们的配置信息
4. 将所有配置组合成一个完整的 JSON 文件
5. 如果配置文件不存在或使用了 `--init` 参数，将配置写入文件

## 配置文件结构

完整的配置文件结构如下：

```json
{
  "MoLingConfig": {
    "base_path": "/Users/username/.moling",
    "config_file": "config/config.json",
    "debug": false,
    "listen_addr": "",
    "module": "all",
    "version": "darwin-arm64-20250330084836-0077553"
  },
  "Browser": {
    "headless": false,
    "timeout": 30,
    "proxy": "",
    "user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36",
    "default_language": "en-US",
    "url_timeout": 10,
    "selector_query_timeout": 10,
    "data_path": "/Users/username/.moling/data",
    "browser_data_path": "/Users/username/.moling/browser",
    "prompt_file": ""
  },
  "Command": {
    "allowed_command": "ls,cat,echo,pwd,head,tail,grep,find,stat,df,du,free,top,ps,uptime,who,w,last,uname,hostname,ifconfig,netstat,ping,traceroute,route,ip,ss,lsof,vmstat,iostat,mpstat,sar,uptime,cut,sort,uniq,wc,awk,sed,diff,cmp,comm,file,basename,dirname,chmod,chown,curl,nslookup,dig,host,ssh,scp,sftp,ftp,wget,tar,gzip,scutil,networksetup, git,cd",
    "prompt_file": ""
  },
  "FileSystem": {
    "allowed_dir": "/tmp/.moling/data/",
    "cache_path": "/tmp/.moling/data",
    "prompt_file": ""
  }
}
```

## 配置系统核心组件

### MoLingConfig

`services.MoLingConfig` 是全局配置结构体，定义在 `services/config.go` 中：

```go
type MoLingConfig struct {
    ConfigFile  string          // 配置文件路径
    BasePath    string          // 应用基础路径
    Version     string          // 版本信息
    ListenAddr  string          // SSE 模式监听地址
    Debug       bool            // 调试模式
    Module      string          // 加载的模块
    Username    string          // 运行用户名
    HomeDir     string          // 用户主目录
    SystemInfo  string          // 系统信息
    Description string          // MCP 服务描述
    Command     string          // 命令
    Args        string          // 参数
    BaseUrl     string          // 基础 URL
    ServerName  string          // 服务器名称
    logger      zerolog.Logger  // 日志记录器
}
```

### 服务接口 (Service)

所有服务都实现了 `Service` 接口，定义在 `services/service.go` 中：

```go
type Service interface {
    Ctx() context.Context
    Resources() map[mcp.Resource]server.ResourceHandlerFunc
    ResourceTemplates() map[mcp.ResourceTemplate]server.ResourceTemplateHandlerFunc
    Prompts() []PromptEntry
    Tools() []server.ServerTool
    NotificationHandlers() map[string]server.NotificationHandlerFunc
    Config() string
    LoadConfig(jsonData map[string]interface{}) error
    Init() error
    MlConfig() *MoLingConfig
    Name() MoLingServerType
    Close() error
}
```

每个服务负责提供其专属的配置结构并实现相应的配置方法。

## 主要服务配置

### 1. Browser 服务配置

浏览器服务使用 `BrowserConfig` 结构体：

```go
type BrowserConfig struct {
    PromptFile           string // 提示文件路径
    prompt               string // 提示内容
    Headless             bool   // 无头模式
    Timeout              int    // 超时时间
    Proxy                string // 代理设置
    UserAgent            string // 用户代理
    DefaultLanguage      string // 默认语言
    URLTimeout           int    // URL 加载超时
    SelectorQueryTimeout int    // 选择器查询超时
    DataPath             string // 数据路径
    BrowserDataPath      string // 浏览器数据路径
}
```

默认值由 `NewBrowserConfig()` 函数提供，包括默认UA、超时等设置。

### 2. Command 服务配置

命令服务使用 `CommandConfig` 结构体：

```go
type CommandConfig struct {
    PromptFile      string   // 提示文件路径
    prompt          string   // 提示内容
    AllowedCommand  string   // 允许的命令列表（逗号分隔）
    allowedCommands []string // 内部使用的命令列表
}
```

通过 `allowedCmdDefault` 提供默认命令列表，包括常见的系统命令如 `ls`, `cat`, `echo` 等。

### 3. FileSystem 服务配置

文件系统服务使用 `FileSystemConfig` 结构体：

```go
type FileSystemConfig struct {
    PromptFile  string   // 提示文件路径
    prompt      string   // 提示内容 
    AllowedDir  string   // 允许访问的目录（逗号分隔）
    allowedDirs []string // 内部使用的目录列表
    CachePath   string   // 缓存路径
}
```

默认情况下，允许访问系统临时目录。

## 配置加载与合并机制

MoLing 使用 `mergeJSONToStruct` 函数来将 JSON 配置合并到结构体中，确保配置变更能正确应用：

```go
// LoadConfig 加载服务配置
func (srv *Service) LoadConfig(jsonData map[string]interface{}) error {
    err := mergeJSONToStruct(srv.config, jsonData)
    if err != nil {
        return err
    }
    return srv.config.Check()
}
```

配置加载流程：
1. 读取配置文件（如果存在）
2. 解析为 JSON 映射
3. 对于每个服务，从映射中提取相应的配置部分
4. 调用服务的 `LoadConfig` 方法加载配置
5. 调用服务的 `Init` 方法初始化

## 服务注册机制

MoLing 使用服务注册机制管理所有服务：

```go
// 服务列表
var serviceLists = make(map[MoLingServerType]func(ctx context.Context) (Service, error))

// 注册服务
func RegisterServ(n MoLingServerType, f func(ctx context.Context) (Service, error)) {
    serviceLists[n] = f
}

// 获取服务列表
func ServiceList() map[MoLingServerType]func(ctx context.Context) (Service, error) {
    return serviceLists
}
```

每个服务通过 `init()` 函数注册自己：

```go
func init() {
    RegisterServ(BrowserServerName, NewBrowserServer)
    RegisterServ(CommandServerName, NewCommandServer)
    RegisterServ(FilesystemServerName, NewFilesystemServer)
}
```

## 配置使用场景

1. **命令行工具**：配置文件主要通过 `moling config` 命令管理
2. **MCP 客户端集成**：MoLing 可与 Claude, Cline 等 MCP 客户端集成，通过 `moling client --install` 命令自动配置

## 配置上下文传递

MoLing 使用 Go 的 `context` 机制在服务之间传递配置信息：

```go
// 创建上下文
ctx := context.WithValue(context.Background(), services.MoLingConfigKey, mlConfig)
ctx = context.WithValue(ctx, services.MoLingLoggerKey, logger)
```

服务可以从上下文中获取配置和日志记录器：

```go
globalConf := ctx.Value(MoLingConfigKey).(*MoLingConfig)
logger := ctx.Value(MoLingLoggerKey).(zerolog.Logger)
```

## 最佳实践与建议

1. **配置文件修改**：使用文本编辑器直接编辑 `~/.moling/config/config.json`
2. **服务限制**：
   - 命令服务：通过 `allowed_command` 限制可执行的命令
   - 文件系统服务：通过 `allowed_dir` 限制可访问的目录
3. **自定义提示**：每个服务都支持通过 `prompt_file` 自定义提示文本
4. **模块选择**：通过 `module` 参数选择性加载模块，如 `--module=Browser,FileSystem`

## 总结

MoLing 的配置系统采用了模块化设计，支持全局配置和服务特定配置。通过 JSON 格式的配置文件和简单的命令行界面，用户可以方便地管理和修改应用行为。服务注册机制使得系统易于扩展，而上下文传递确保配置信息能够在各个组件之间有效共享。 