# ChromeDP 学习笔记

## 简介

[chromedp](https://github.com/chromedp/chromedp) 是一个用于驱动浏览器的Go库，它不依赖于外部二进制文件（如Selenium或PhantomJS），而是直接使用Chrome DevTools Protocol (CDP) 来控制Chrome/Chromium浏览器。这使得它成为Go中进行Web自动化、爬虫和测试的强大工具。

本文档记录了chromedp库的初始化和基本使用方法，主要基于项目中的browser.go实现。

## 初始化流程

ChromeDP的初始化通常分为以下几个步骤：

1. 创建ExecAllocator（浏览器实例）
2. 创建上下文（Context）
3. 配置选项
4. 资源管理

### 创建ExecAllocator

ExecAllocator负责启动和管理Chrome/Chromium实例。以下是从browser.go中提取的初始化代码：

```go
// 创建浏览器上下文
opts := append(
    chromedp.DefaultExecAllocatorOptions[:],                         // 默认浏览器配置
    chromedp.UserAgent(bs.config.UserAgent),                         // 用户代理
    chromedp.Flag("lang", bs.config.DefaultLanguage),                // 语言
    chromedp.Flag("disable-blink-features", "AutomationControlled"), // 禁用自动化控制
    chromedp.Flag("enable-automation", false),                       // 禁用自动化
    chromedp.Flag("disable-features", "Translate"),                  // 禁用翻译
    chromedp.Flag("headless", bs.config.Headless),                   // 是否无头模式
    chromedp.Flag("hide-scrollbars", false),                         // 是否隐藏滚动条
    chromedp.Flag("mute-audio", true),                               // 是否静音
    chromedp.Flag("disable-infobars", true),                         // 禁用信息栏
    chromedp.Flag("disable-extensions", true),                       // 禁用扩展
    chromedp.Flag("CommandLineFlagSecurityWarningsEnabled", false),  // 禁用安全警告
    chromedp.CombinedOutput(bs.Logger),                              // 输出日志
    chromedp.WindowSize(1280, 800),                                  // 窗口大小
    chromedp.UserDataDir(bs.config.BrowserDataPath),                 // 用户数据目录
    chromedp.IgnoreCertErrors,                                       // 忽略证书错误
)
bs.Context, bs.cancelAlloc = chromedp.NewExecAllocator(context.Background(), opts...)
```

### 创建上下文（Context）

上下文用于实际与浏览器交互，在创建了ExecAllocator之后，需要创建一个chromedp上下文：

```go
bs.Context, bs.cancelChrome = chromedp.NewContext(bs.Context,
    chromedp.WithErrorf(bs.Logger.Error().Msgf),
    chromedp.WithDebugf(bs.Logger.Debug().Msgf),
)
```

### 配置选项详解

chromedp提供了许多配置选项，以下是browser.go中使用的主要选项解释：

1. **基础配置**
   - `chromedp.DefaultExecAllocatorOptions[:]`：使用默认的配置选项作为基础

2. **浏览器身份**
   - `chromedp.UserAgent()`：设置浏览器的User-Agent
   - `chromedp.Flag("lang", ...)`：设置浏览器语言

3. **自动化控制**
   - `chromedp.Flag("disable-blink-features", "AutomationControlled")`：隐藏自动化特征，避免被网站检测
   - `chromedp.Flag("enable-automation", false)`：禁用自动化标志，同样是为了避免检测

4. **界面选项**
   - `chromedp.Flag("headless", ...)`：是否使用无头模式（不显示浏览器界面）
   - `chromedp.Flag("hide-scrollbars", false)`：是否隐藏滚动条
   - `chromedp.WindowSize(1280, 800)`：设置浏览器窗口大小
   - `chromedp.Flag("disable-infobars", true)`：禁用信息栏
   - `chromedp.Flag("mute-audio", true)`：静音

5. **性能和安全**
   - `chromedp.Flag("disable-extensions", true)`：禁用扩展，提高性能
   - `chromedp.IgnoreCertErrors`：忽略证书错误
   - `chromedp.UserDataDir()`：指定用户数据目录，可用于保存会话

6. **日志和调试**
   - `chromedp.CombinedOutput(bs.Logger)`：捕获浏览器输出
   - `chromedp.WithErrorf()`：设置错误日志处理函数
   - `chromedp.WithDebugf()`：设置调试日志处理函数

### 资源管理

chromedp会创建临时目录和启动浏览器进程，需要正确管理这些资源，特别是在程序结束时进行清理：

```go
// 在BrowserServer结构体中保存取消函数
cancelAlloc        context.CancelFunc // 资源清理方法
cancelChrome       context.CancelFunc // 浏览器清理方法

// 在Close方法中清理资源
func (bs *BrowserServer) Close() error {
    bs.Logger.Debug().Msg("Closing browser server")
    bs.cancelAlloc()
    bs.cancelChrome()
    // Cancel the context to stop the browser
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()
    return chromedp.Cancel(ctx)
}
```

### 浏览器锁处理

在初始化过程中，还需要处理可能存在的浏览器锁，以防止多个进程同时使用同一个用户数据目录：

```go
func (bs *BrowserServer) initBrowser(userDataDir string) error {
    _, err := os.Stat(userDataDir)
    if err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("failed to stat user data directory: %v", err)
    }

    // 检查目录是否存在，如果存在，可以重用它
    if err == nil {
        //  判断浏览器运行锁
        singletonLock := filepath.Join(userDataDir, "SingletonLock")
        _, err = os.Stat(singletonLock)
        if err == nil {
            bs.Logger.Debug().Msg("Browser is already running, removing SingletonLock")
            err = os.RemoveAll(singletonLock)
            if err != nil {
                bs.Logger.Error().Str("Lock", singletonLock).Msgf("Browser can't work due to failed removal of SingletonLock: %v", err)
            }
        }
        return nil
    }
    // 创建目录
    err = os.MkdirAll(userDataDir, 0755)
    if err != nil {
        return fmt.Errorf("failed to create user data directory: %v", err)
    }
    return nil
}
```

## 基本操作示例

chromedp使用链式操作来执行浏览器动作，这些动作通过`chromedp.Run()`方法执行。以下是一些基本操作示例，摘自browser.go：

### 导航（Navigation）

```go
// 导航到URL
err := chromedp.Run(bs.Context, chromedp.Navigate(url))
if err != nil {
    return mcp.NewToolResultError(fmt.Sprintf("failed to navigate: %v", err)), nil
}
```

### 截图（Screenshot）

```go
// 全屏截图
err = chromedp.Run(runCtx,
    chromedp.EmulateViewport(int64(width), int64(height)), // 设置视口大小
    chromedp.FullScreenshot(&buf, 90),                     // 90% 质量
)

// 元素截图
err = chromedp.Run(runCtx,
    chromedp.WaitVisible(selector), // 等待元素可见
    chromedp.Screenshot(selector, &buf, chromedp.NodeVisible),
)
```

### 点击元素（Click）

```go
// 点击元素
err := chromedp.Run(runCtx,
    chromedp.WaitReady("body"),     // 等待页面主体加载完成
    chromedp.WaitVisible(selector), // 等待目标元素可见
    chromedp.Click(selector),       // 点击目标元素
)
```

### 填写表单（Form Filling）

```go
// 填写表单字段
err := chromedp.Run(runCtx,
    chromedp.WaitVisible(selector),     // 等待输入字段可见
    chromedp.Clear(selector),           // 清除现有内容
    chromedp.SendKeys(selector, value), // 输入新内容
)
```

### 执行JavaScript（JavaScript Execution）

```go
// 执行JavaScript
var result interface{}
err := chromedp.Run(runCtx, chromedp.Evaluate(script, &result))
```

### 等待元素（Waiting for Elements）

```go
// 等待元素可见
chromedp.WaitVisible(selector)

// 等待元素准备就绪
chromedp.WaitReady(selector)
```

## 超时和上下文管理

在进行可能耗时的操作时，应该设置超时，以避免程序无限期等待：

```go
// 设置更长的超时时间
timeoutDuration := time.Duration(bs.config.SelectorQueryTimeout*3) * time.Second
runCtx, cancelFunc := context.WithTimeout(bs.Context, timeoutDuration)
defer cancelFunc()

// 使用带超时的上下文执行操作
err := chromedp.Run(runCtx, /* actions */)
```

## 错误处理和恢复策略

browser.go中实现了许多错误处理和恢复策略，例如在标准操作失败时使用JavaScript替代方案：

```go
// 如果标准方法失败，尝试使用JavaScript直接点击
if err != nil {
    bs.Logger.Debug().Str("selector", selector).Err(err).Msg("标准点击方法失败，尝试通过JavaScript点击")

    // 使用JavaScript执行点击操作
    jsClick := fmt.Sprintf(`
        (function() {
            try {
                const el = document.querySelector(%s);
                if (!el) return { success: false, error: "元素不存在" };
                
                // 尝试点击元素
                el.click();
                
                return { success: true };
            } catch(e) {
                return { success: false, error: e.message };
            }
        })()
    `, safeJSONString(selector))

    var clickResult map[string]interface{}
    err = chromedp.Run(runCtx, chromedp.Evaluate(jsClick, &clickResult))
    // 处理结果...
}
```

## 最佳实践

从browser.go中可以总结出以下chromedp使用的最佳实践：

1. **适当的资源管理**：使用取消函数确保浏览器资源被正确释放
2. **超时控制**：为操作设置合理的超时时间
3. **渐进增强**：当标准API失败时使用JavaScript作为后备方案
4. **错误处理**：捕获并记录所有错误，提供详细的错误信息
5. **日志记录**：记录所有重要操作和状态变化，便于调试
6. **防检测**：配置浏览器选项以避免被网站检测为自动化工具
7. **单例锁管理**：在使用用户数据目录时，处理可能的锁冲突

## 总结

chromedp是一个功能强大的浏览器自动化库，通过Chrome DevTools Protocol直接控制Chrome/Chromium浏览器。它的初始化过程需要设置适当的选项和管理资源，但提供了丰富的API用于模拟用户操作、截取网页内容和执行JavaScript。

browser.go中的实现展示了如何在实际项目中使用chromedp，包括初始化、执行操作、错误处理和资源管理，为使用chromedp开发类似功能提供了很好的参考。 