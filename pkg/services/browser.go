// Copyright 2025 CFC4N <cfc4n.cs@gmail.com>. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Repository: https://github.com/gojue/moling

// Package services provides a set of services for the MoLing application.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/gojue/moling/pkg/comm"
	"github.com/gojue/moling/pkg/config"
	"github.com/gojue/moling/pkg/utils"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog"
)

const (
	BrowserDataPath                         = "browser" // Path to store browser data
	BrowserServerName comm.MoLingServerType = "Browser"
)

// BrowserServer represents the configuration for the browser service.
type BrowserServer struct {
	MLService
	config       *BrowserConfig
	name         string // The name of the service
	cancelAlloc  context.CancelFunc
	cancelChrome context.CancelFunc
}

// NewBrowserServer creates a new BrowserServer instance with the given context and configuration.
func NewBrowserServer(ctx context.Context) (Service, error) {
	bc := NewBrowserConfig()
	globalConf := ctx.Value(comm.MoLingConfigKey).(*config.MoLingConfig)
	bc.BrowserDataPath = filepath.Join(globalConf.BasePath, BrowserDataPath)
	bc.DataPath = filepath.Join(globalConf.BasePath, "data")
	logger, ok := ctx.Value(comm.MoLingLoggerKey).(zerolog.Logger)
	if !ok {
		return nil, fmt.Errorf("BrowserServer: invalid logger type: %T", ctx.Value(comm.MoLingLoggerKey))
	}
	loggerNameHook := zerolog.HookFunc(func(e *zerolog.Event, level zerolog.Level, msg string) {
		e.Str("Service", string(BrowserServerName))
	})
	bs := &BrowserServer{
		MLService: NewMLService(ctx, logger.Hook(loggerNameHook), globalConf),
		config:    bc,
	}

	err := bs.init()
	if err != nil {
		return nil, err
	}

	return bs, nil
}

// Init initializes the browser server by creating a new context.
func (bs *BrowserServer) Init() error {
	//loggerNameHook := zerolog.HookFunc(func(e *zerolog.Event, level zerolog.Level, msg string) {
	//	e.Str("Service", string(bs.Name()))
	//})

	//bs.logger = bs.logger.Hook(loggerNameHook)
	err := bs.initBrowser(bs.config.BrowserDataPath)
	if err != nil {
		return fmt.Errorf("failed to initialize browser: %v", err)
	}
	err = utils.CreateDirectory(bs.config.DataPath)
	if err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	// Create a new context for the browser
	opts := append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserAgent(bs.config.UserAgent),
		chromedp.Flag("lang", bs.config.DefaultLanguage),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("disable-features", "Translate"),
		chromedp.Flag("headless", bs.config.Headless),
		chromedp.Flag("hide-scrollbars", false),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("disable-infobars", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("CommandLineFlagSecurityWarningsEnabled", false),
		chromedp.CombinedOutput(bs.Logger),
		chromedp.WindowSize(1280, 800),
		chromedp.UserDataDir(bs.config.BrowserDataPath),
		chromedp.IgnoreCertErrors,
	)
	bs.Context, bs.cancelAlloc = chromedp.NewExecAllocator(context.Background(), opts...)

	bs.Context, bs.cancelChrome = chromedp.NewContext(bs.Context,
		chromedp.WithErrorf(bs.Logger.Error().Msgf),
		chromedp.WithDebugf(bs.Logger.Debug().Msgf),
	)

	pe := PromptEntry{
		PromptVar: mcp.Prompt{
			Name:        "browser_prompt",
			Description: fmt.Sprintf("Get the relevant functions and prompts of the Browser MCP Server."),
			//Arguments:   make([]mcp.PromptArgument, 0),
		},
		HandlerFunc: bs.handlePrompt,
	}
	bs.AddPrompt(pe)
	bs.AddTool(mcp.NewTool(
		"browser_navigate",
		mcp.WithDescription("Navigate to a URL"),
		mcp.WithString("url",
			mcp.Description("URL to navigate to"),
			mcp.Required(),
		),
	), bs.handleNavigate)
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
	bs.AddTool(mcp.NewTool(
		"browser_click",
		mcp.WithDescription("Click an element on the page"),
		mcp.WithString("selector",
			mcp.Description("CSS selector for element to click"),
			mcp.Required(),
		),
	), bs.handleClick)
	bs.AddTool(mcp.NewTool(
		"browser_fill",
		mcp.WithDescription("Fill out an input field"),
		mcp.WithString("selector",
			mcp.Description("CSS selector for input field"),
			mcp.Required(),
		),
		mcp.WithString("value",
			mcp.Description("Value to fill"),
			mcp.Required(),
		),
	), bs.handleFill)
	bs.AddTool(mcp.NewTool(
		"browser_select",
		mcp.WithDescription("Select an element on the page with Select tag"),
		mcp.WithString("selector",
			mcp.Description("CSS selector for element to select"),
			mcp.Required(),
		),
		mcp.WithString("value",
			mcp.Description("Value to select"),
			mcp.Required(),
		),
	), bs.handleSelect)
	bs.AddTool(mcp.NewTool(
		"browser_hover",
		mcp.WithDescription("Hover an element on the page"),
		mcp.WithString("selector",
			mcp.Description("CSS selector for element to hover"),
			mcp.Required(),
		),
	), bs.handleHover)
	bs.AddTool(mcp.NewTool(
		"browser_evaluate",
		mcp.WithDescription("Execute JavaScript in the browser console"),
		mcp.WithString("script",
			mcp.Description("JavaScript code to execute"),
			mcp.Required(),
		),
	), bs.handleEvaluate)

	bs.AddTool(mcp.NewTool(
		"browser_debug_enable",
		mcp.WithDescription("Enable JavaScript debugging"),
		mcp.WithBoolean("enabled",
			mcp.Description("Enable or disable debugging"),
			mcp.Required(),
		),
	), bs.handleDebugEnable)

	bs.AddTool(mcp.NewTool(
		"browser_set_breakpoint",
		mcp.WithDescription("Set a JavaScript breakpoint"),
		mcp.WithString("url",
			mcp.Description("URL of the script"),
			mcp.Required(),
		),
		mcp.WithNumber("line",
			mcp.Description("Line number"),
			mcp.Required(),
		),
		mcp.WithNumber("column",
			mcp.Description("Column number (optional)"),
		),
		mcp.WithString("condition",
			mcp.Description("Breakpoint condition (optional)"),
		),
	), bs.handleSetBreakpoint)

	bs.AddTool(mcp.NewTool(
		"browser_remove_breakpoint",
		mcp.WithDescription("Remove a JavaScript breakpoint"),
		mcp.WithString("breakpointId",
			mcp.Description("Breakpoint ID to remove"),
			mcp.Required(),
		),
	), bs.handleRemoveBreakpoint)

	bs.AddTool(mcp.NewTool(
		"browser_pause",
		mcp.WithDescription("Pause JavaScript execution"),
	), bs.handlePause)

	bs.AddTool(mcp.NewTool(
		"browser_resume",
		mcp.WithDescription("Resume JavaScript execution"),
	), bs.handleResume)

	bs.AddTool(mcp.NewTool(
		"browser_get_callstack",
		mcp.WithDescription("Get current call stack when paused"),
	), bs.handleGetCallstack)
	return nil
}

// init initializes the browser server by creating the user data directory.
func (bs *BrowserServer) initBrowser(userDataDir string) error {
	_, err := os.Stat(userDataDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat user data directory: %v", err)
	}

	// Check if the directory exists, if it does, we can reuse it
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
	// Create the directory
	err = os.MkdirAll(userDataDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create user data directory: %v", err)
	}
	return nil
}

func (bs *BrowserServer) handlePrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	// 处理浏览器提示
	return &mcp.GetPromptResult{
		Description: fmt.Sprintf(""),
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: bs.config.prompt,
				},
			},
		},
	}, nil
}

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

// handleScreenshot handles the screenshot action.
func (bs *BrowserServer) handleScreenshot(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := request.Params.Arguments["name"].(string)
	if !ok {
		return mcp.NewToolResultError("name must be a string"), nil
	}
	selector, _ := request.Params.Arguments["selector"].(string)
	width, _ := request.Params.Arguments["width"].(int)
	height, _ := request.Params.Arguments["height"].(int)
	if width == 0 {
		width = 1280
	}
	if height == 0 {
		height = 800
	}

	// 记录尝试截图操作
	bs.Logger.Debug().
		Str("name", name).
		Str("selector", selector).
		Int("width", width).
		Int("height", height).
		Msg("尝试截取屏幕截图")

	// 设置更长的超时时间
	timeoutDuration := time.Duration(bs.config.SelectorQueryTimeout*3) * time.Second
	runCtx, cancelFunc := context.WithTimeout(bs.Context, timeoutDuration)
	defer cancelFunc()

	var buf []byte
	var err error

	// 根据是否提供选择器决定截取全屏还是特定元素
	if selector == "" {
		// 全屏截图
		err = chromedp.Run(runCtx,
			chromedp.EmulateViewport(int64(width), int64(height)), // 设置视口大小
			chromedp.FullScreenshot(&buf, 90),                     // 90% 质量
		)
	} else {
		// 元素截图，确保使用相同的上下文
		err = chromedp.Run(runCtx,
			chromedp.WaitVisible(selector), // 等待元素可见
			chromedp.Screenshot(selector, &buf, chromedp.NodeVisible),
		)
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("截图失败: %v", err)), nil
	}

	// 使用随机数确保文件名唯一
	newName := filepath.Join(bs.config.DataPath, fmt.Sprintf("%s_%d.png", strings.TrimRight(name, ".png"), rand.Int()))
	err = os.WriteFile(newName, buf, 0644)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("保存截图失败: %v", err)), nil
	}

	bs.Logger.Debug().Str("path", newName).Msg("成功保存截图")
	return mcp.NewToolResultText(fmt.Sprintf("截图已保存至: %s", newName)), nil
}

// handleClick handles the click action on a specified element.
func (bs *BrowserServer) handleClick(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	selector, ok := request.Params.Arguments["selector"].(string)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("selector must be a string:%v", selector)), nil
	}

	// 记录尝试点击的元素选择器
	bs.Logger.Debug().Str("selector", selector).Msg("尝试点击元素")

	// 设置更长的超时时间，以确保有足够时间执行操作
	timeoutDuration := time.Duration(bs.config.SelectorQueryTimeout*3) * time.Second
	runCtx, cancelFunc := context.WithTimeout(bs.Context, timeoutDuration)
	defer cancelFunc()

	// 先尝试合并所有操作，避免分割操作可能引起的上下文问题
	err := chromedp.Run(runCtx,
		chromedp.WaitReady("body"),     // 等待页面主体加载完成
		chromedp.WaitVisible(selector), // 等待目标元素可见
		chromedp.Click(selector),       // 点击目标元素
	)

	// 如果合并操作失败，尝试使用JavaScript直接点击
	if err != nil {
		bs.Logger.Debug().Str("selector", selector).Err(err).Msg("标准点击方法失败，尝试通过JavaScript点击")

		// 使用JavaScript执行点击操作，这可以绕过一些DOM可见性和交互性问题
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

		// 使用结构化结果
		var clickResult map[string]interface{}
		err = chromedp.Run(runCtx, chromedp.Evaluate(jsClick, &clickResult))
		if err != nil {
			return mcp.NewToolResultError(fmt.Errorf("无法执行点击脚本: %v", err).Error()), nil
		}

		// 检查脚本执行结果
		success, ok := clickResult["success"].(bool)
		if !ok || !success {
			errorMsg := "未知错误"
			if errMsg, hasErr := clickResult["error"].(string); hasErr {
				errorMsg = errMsg
			}
			return mcp.NewToolResultError(fmt.Sprintf("点击失败: %s", errorMsg)), nil
		}

		bs.Logger.Debug().Str("selector", selector).Msg("通过JavaScript成功点击元素")
		return mcp.NewToolResultText(fmt.Sprintf("通过JavaScript点击了元素 %s", selector)), nil
	}

	bs.Logger.Debug().Str("selector", selector).Msg("成功点击元素")
	return mcp.NewToolResultText(fmt.Sprintf("点击了元素 %s", selector)), nil
}

// handleFill handles the fill action on a specified input field.
func (bs *BrowserServer) handleFill(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	selector, ok := request.Params.Arguments["selector"].(string)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("failed to fill selector:%v", request.Params.Arguments["selector"])), nil
	}

	value, ok := request.Params.Arguments["value"].(string)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("failed to fill input field: %v, selector:%v", request.Params.Arguments["value"], selector)), nil
	}

	// 记录尝试填写的输入字段
	bs.Logger.Debug().Str("selector", selector).Str("value", value).Msg("尝试填写输入字段")

	// 设置更长的超时时间
	timeoutDuration := time.Duration(bs.config.SelectorQueryTimeout*3) * time.Second
	runCtx, cancelFunc := context.WithTimeout(bs.Context, timeoutDuration)
	defer cancelFunc()

	// 合并操作：等待元素可见并填写内容
	err := chromedp.Run(runCtx,
		chromedp.WaitVisible(selector),     // 等待输入字段可见
		chromedp.Clear(selector),           // 清除现有内容
		chromedp.SendKeys(selector, value), // 输入新内容
	)

	// 如果标准方法失败，尝试使用JavaScript设置值
	if err != nil {
		bs.Logger.Debug().Str("selector", selector).Err(err).Msg("标准填写方法失败，尝试通过JavaScript设置值")

		// 使用JavaScript设置输入字段的值，使用JSON安全处理的字符串
		jsFill := fmt.Sprintf(`
			(function() {
				try {
					const el = document.querySelector(%s);
					if (!el) return { success: false, error: "元素不存在" };
					
					// 设置值，使用安全处理过的字符串
					el.value = %s;
					
					// 触发输入事件，确保表单验证和事件监听器被触发
					const event = new Event('input', { bubbles: true });
					el.dispatchEvent(event);
					
					return { success: true };
				} catch(e) {
					return { success: false, error: e.message };
				}
			})()
		`, safeJSONString(selector), safeJSONString(value))

		// 使用更复杂的结果对象来接收信息
		var fillResult map[string]interface{}
		err = chromedp.Run(runCtx, chromedp.Evaluate(jsFill, &fillResult))
		if err != nil {
			return mcp.NewToolResultError(fmt.Errorf("无法执行填写脚本: %v", err).Error()), nil
		}

		// 检查脚本执行结果
		success, ok := fillResult["success"].(bool)
		if !ok || !success {
			errorMsg := "未知错误"
			if errMsg, hasErr := fillResult["error"].(string); hasErr {
				errorMsg = errMsg
			}
			return mcp.NewToolResultError(fmt.Sprintf("填写失败: %s", errorMsg)), nil
		}

		bs.Logger.Debug().Str("selector", selector).Msg("通过JavaScript成功填写输入字段")
		return mcp.NewToolResultText(fmt.Sprintf("通过JavaScript填写了输入字段 %s，值为 %s", selector, value)), nil
	}

	bs.Logger.Debug().Str("selector", selector).Msg("成功填写输入字段")
	return mcp.NewToolResultText(fmt.Sprintf("填写了输入字段 %s，值为 %s", selector, value)), nil
}

// 安全处理JSON编码的辅助函数
func safeJSONString(s string) string {
	bytes, err := json.Marshal(s)
	if err != nil {
		return `"` + strings.Replace(s, `"`, `\"`, -1) + `"`
	}
	return string(bytes)
}

func (bs *BrowserServer) handleSelect(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	selector, ok := request.Params.Arguments["selector"].(string)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("failed to select selector:%v", request.Params.Arguments["selector"])), nil
	}
	value, ok := request.Params.Arguments["value"].(string)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("failed to select value:%v", request.Params.Arguments["value"])), nil
	}

	// 记录尝试选择的下拉菜单和值
	bs.Logger.Debug().Str("selector", selector).Str("value", value).Msg("尝试设置下拉菜单选项")

	// 设置更长的超时时间
	timeoutDuration := time.Duration(bs.config.SelectorQueryTimeout*3) * time.Second
	runCtx, cancelFunc := context.WithTimeout(bs.Context, timeoutDuration)
	defer cancelFunc()

	// 合并操作：等待元素可见并设置值
	err := chromedp.Run(runCtx,
		chromedp.WaitVisible(selector),     // 等待选择器可见
		chromedp.SetValue(selector, value), // 设置选择器的值
	)

	// 如果标准方法失败，尝试使用JavaScript设置选项
	if err != nil {
		bs.Logger.Debug().Str("selector", selector).Err(err).Msg("标准选择方法失败，尝试通过JavaScript设置选项")

		// 使用JavaScript设置选择器的值
		jsSelect := fmt.Sprintf(`
			(function() {
				try {
					const selectEl = document.querySelector(%s);
					if (!selectEl) return { success: false, error: "选择器元素不存在" };
					
					// 直接设置值
					selectEl.value = %s;
					
					// 触发change事件，确保其他JavaScript代码能够响应此变化
					const event = new Event('change', { bubbles: true });
					selectEl.dispatchEvent(event);
					
					// 检查是否设置成功
					if (selectEl.value !== %s) {
						return { success: false, error: "无法设置选择器值，可能没有匹配的选项" };
					}
					
					return { success: true };
				} catch(e) {
					return { success: false, error: e.message };
				}
			})()
		`, safeJSONString(selector), safeJSONString(value), safeJSONString(value))

		// 使用结构化结果
		var selectResult map[string]interface{}
		err = chromedp.Run(runCtx, chromedp.Evaluate(jsSelect, &selectResult))
		if err != nil {
			return mcp.NewToolResultError(fmt.Errorf("无法执行选择脚本: %v", err).Error()), nil
		}

		// 检查脚本执行结果
		success, ok := selectResult["success"].(bool)
		if !ok || !success {
			errorMsg := "未知错误"
			if errMsg, hasErr := selectResult["error"].(string); hasErr {
				errorMsg = errMsg
			}
			return mcp.NewToolResultError(fmt.Sprintf("选择失败: %s", errorMsg)), nil
		}

		bs.Logger.Debug().Str("selector", selector).Msg("通过JavaScript成功设置选择器")
		return mcp.NewToolResultText(fmt.Sprintf("通过JavaScript在选择器 %s 中选择了值 %s", selector, value)), nil
	}

	bs.Logger.Debug().Str("selector", selector).Str("value", value).Msg("成功设置选择器")
	return mcp.NewToolResultText(fmt.Sprintf("在选择器 %s 中选择了值 %s", selector, value)), nil
}

// handleHover handles the hover action on a specified element.
func (bs *BrowserServer) handleHover(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	selector, ok := request.Params.Arguments["selector"].(string)
	if !ok {
		return mcp.NewToolResultError(fmt.Sprintf("selector must be a string:%v", selector)), nil
	}

	// 记录尝试悬停的元素
	bs.Logger.Debug().Str("selector", selector).Msg("尝试悬停在元素上")

	// 设置更长的超时时间
	timeoutDuration := time.Duration(bs.config.SelectorQueryTimeout*3) * time.Second
	runCtx, cancelFunc := context.WithTimeout(bs.Context, timeoutDuration)
	defer cancelFunc()

	// 合并操作：等待元素可见并悬停
	var res bool
	err := chromedp.Run(runCtx,
		chromedp.WaitVisible(selector), // 等待元素可见
		chromedp.Evaluate(`
			(function() {
				const el = document.querySelector(`+safeJSONString(selector)+`);
				if (!el) return false;
				el.dispatchEvent(new Event('mouseover', {bubbles: true}));
				el.dispatchEvent(new Event('mouseenter', {bubbles: false}));
				return true;
			})()
		`, &res),
	)

	// 如果标准方法失败，尝试使用另一种JavaScript方法
	if err != nil {
		bs.Logger.Debug().Str("selector", selector).Err(err).Msg("标准悬停方法失败，尝试另一种JavaScript方法")

		// 另一种实现悬停的方式
		jsHover := fmt.Sprintf(`
			(function() {
				try {
					const el = document.querySelector(%s);
					if (!el) return { success: false, error: "元素不存在" };
					
					// 尝试模拟完整的鼠标悬停事件序列
					['mouseenter', 'mouseover', 'mousemove'].forEach(type => {
						const event = new MouseEvent(type, {
							view: window,
							bubbles: true,
							cancelable: true
						});
						el.dispatchEvent(event);
					});
					return { success: true };
				} catch(e) {
					return { success: false, error: e.message };
				}
			})()
		`, safeJSONString(selector))

		// 使用结构化结果
		var hoverResult map[string]interface{}
		err = chromedp.Run(runCtx, chromedp.Evaluate(jsHover, &hoverResult))
		if err != nil {
			return mcp.NewToolResultError(fmt.Errorf("无法执行悬停脚本: %v", err).Error()), nil
		}

		// 检查脚本执行结果
		success, ok := hoverResult["success"].(bool)
		if !ok || !success {
			errorMsg := "未知错误"
			if errMsg, hasErr := hoverResult["error"].(string); hasErr {
				errorMsg = errMsg
			}
			return mcp.NewToolResultError(fmt.Sprintf("悬停失败: %s", errorMsg)), nil
		}

		bs.Logger.Debug().Str("selector", selector).Msg("通过JavaScript成功悬停在元素上")
		return mcp.NewToolResultText(fmt.Sprintf("通过JavaScript悬停在了元素 %s 上", selector)), nil
	}

	bs.Logger.Debug().Str("selector", selector).Bool("result", res).Msg("成功悬停在元素上")
	return mcp.NewToolResultText(fmt.Sprintf("悬停在了元素 %s 上，结果:%t", selector, res)), nil
}

func (bs *BrowserServer) handleEvaluate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	script, ok := request.Params.Arguments["script"].(string)
	if !ok {
		return mcp.NewToolResultError("script must be a string"), nil
	}

	// 记录尝试执行的脚本
	bs.Logger.Debug().Str("script", script).Msg("尝试执行JavaScript脚本")

	// 设置更长的超时时间
	timeoutDuration := time.Duration(bs.config.SelectorQueryTimeout*2) * time.Second
	runCtx, cancelFunc := context.WithTimeout(bs.Context, timeoutDuration)
	defer cancelFunc()

	// 始终检查脚本并包装，确保可以处理return语句和DOM操作
	hasReturnStatement := strings.Contains(script, "return ") && !strings.HasPrefix(script, "(function")
	hasDOMSelector := strings.Contains(script, "querySelector") || strings.Contains(script, "getElementById") ||
		strings.Contains(script, "getElementsBy")

	// 一般情况下直接将脚本包装在自执行函数中
	if hasReturnStatement || hasDOMSelector {
		bs.Logger.Debug().
			Bool("hasReturn", hasReturnStatement).
			Bool("hasDOMSelector", hasDOMSelector).
			Msg("检测到需要包装的脚本")

		// 如果包含DOM选择器，尝试提取并检查元素
		if hasDOMSelector {
			bs.Logger.Debug().Msg("检测到DOM操作，添加安全检查")

			// 提取可能的选择器，这是试探性的，不总是能精确匹配所有情况
			selectorRegex := regexp.MustCompile(`querySelector\(['"]([^'"]+)['"]\)`)
			matches := selectorRegex.FindStringSubmatch(script)

			if len(matches) > 1 {
				selector := matches[1]
				bs.Logger.Debug().Str("selector", selector).Msg("检测到选择器")

				// 先检查元素是否存在
				var exists bool
				checkScript := fmt.Sprintf(`document.querySelector(%s) !== null`, safeJSONString(selector))
				err := chromedp.Run(runCtx, chromedp.Evaluate(checkScript, &exists))

				if err != nil {
					bs.Logger.Warn().Err(err).Str("selector", selector).Msg("检查元素存在性时出错，继续执行")
				} else if !exists {
					// 如果元素不存在，获取页面中所有同类型元素的信息
					var suggestions []interface{}
					suggestionsScript := ""

					// 根据选择器类型给出建议
					if strings.Contains(selector, "textarea") {
						suggestionsScript = `Array.from(document.querySelectorAll('textarea')).map(el => ({ tag: 'textarea', id: el.id, name: el.name, class: el.className }))`
					} else if strings.Contains(selector, "input") {
						suggestionsScript = `Array.from(document.querySelectorAll('input')).map(el => ({ tag: 'input', type: el.type, id: el.id, name: el.name, class: el.className }))`
					} else {
						// 通用选择器，获取该标签的所有实例
						tagMatch := regexp.MustCompile(`(\w+)(\[|\.|\#|$)`).FindStringSubmatch(selector)
						if len(tagMatch) > 1 {
							tag := tagMatch[1]
							suggestionsScript = fmt.Sprintf(`Array.from(document.querySelectorAll('%s')).map(el => ({ tag: '%s', id: el.id, name: el.getAttribute('name'), class: el.className }))`, tag, tag)
						} else {
							suggestionsScript = `Array.from(document.querySelectorAll('*')).filter(el => el.id || el.name).map(el => ({ tag: el.tagName.toLowerCase(), id: el.id, name: el.getAttribute('name'), class: el.className }))`
						}
					}

					// 获取页面上的相似元素
					if suggestionsScript != "" {
						err = chromedp.Run(runCtx, chromedp.Evaluate(suggestionsScript, &suggestions))
						if err == nil && len(suggestions) > 0 {
							suggestionStr, _ := json.Marshal(suggestions)
							bs.Logger.Warn().
								Str("selector", selector).
								Str("suggestions", string(suggestionStr)).
								Msg("元素不存在，但找到了相似元素")
							// 这里我们只记录警告，不再直接返回错误，让脚本继续执行
							// 因为有些脚本会处理元素不存在的情况
						}
					}
				}
			}
		}

		// 无论是否包含DOM选择器，都包装脚本以处理return语句和错误捕获
		wrappedScript := fmt.Sprintf(`
			(function() { 
				try {
					%s 
				} catch(e) {
					return { 
						success: false, 
						error: e.message,
						stack: e.stack,
						type: e.name
					};
				}
			})()
		`, script)

		script = wrappedScript
	}

	// 执行脚本
	var result interface{}
	err := chromedp.Run(runCtx, chromedp.Evaluate(script, &result))

	// 如果执行失败，尝试修复
	if err != nil {
		if strings.Contains(err.Error(), "Illegal return statement") {
			bs.Logger.Debug().Msg("检测到非法的return语句，尝试更强健的包装方式")

			// 使用替代方式处理return语句
			alternativeScript := fmt.Sprintf(`
				(function() {
					let __result;
					try {
						__result = (function() {
							%s
						})();
						return { success: true, result: __result };
					} catch(e) {
						return { success: false, error: e.message };
					}
				})()
			`, strings.ReplaceAll(script, "return ", "__result = "))

			err = chromedp.Run(runCtx, chromedp.Evaluate(alternativeScript, &result))
			if err != nil {
				// 最后一个尝试
				lastResortScript := fmt.Sprintf(`
					(function() {
						try {
							const fn = new Function('return (function() { %s })()')
							const result = fn();
							return { success: true, result: result };
						} catch(e) {
							return { success: false, error: e.message };
						}
					})()
				`, strings.ReplaceAll(script, "return ", "return "))

				err = chromedp.Run(runCtx, chromedp.Evaluate(lastResortScript, &result))
				if err != nil {
					return mcp.NewToolResultError(fmt.Errorf("尝试所有方法后仍无法执行脚本: %v", err).Error()), nil
				}
			}
		} else {
			return mcp.NewToolResultError(fmt.Errorf("执行脚本失败: %v", err).Error()), nil
		}
	}

	// 检查返回的结果是否包含错误信息
	if resultMap, ok := result.(map[string]interface{}); ok {
		// 先检查是否有错误
		if success, exists := resultMap["success"].(bool); exists && !success {
			if errorMsg, hasError := resultMap["error"].(string); hasError {
				return mcp.NewToolResultError(fmt.Sprintf("脚本执行遇到错误: %s", errorMsg)), nil
			}
		}

		// 如果有result字段，则返回它，这是我们包装后的结果
		if actualResult, hasResult := resultMap["result"]; hasResult && resultMap["success"] == true {
			result = actualResult
		}
	}

	bs.Logger.Debug().Interface("result", result).Msg("脚本执行成功")
	return mcp.NewToolResultText(fmt.Sprintf("脚本执行成功，结果: %v", result)), nil
}

func (bs *BrowserServer) Close() error {
	bs.Logger.Debug().Msg("Closing browser server")
	bs.cancelAlloc()
	bs.cancelChrome()
	// Cancel the context to stop the browser
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return chromedp.Cancel(ctx)
}

// Config returns the configuration of the service as a string.
func (bs *BrowserServer) Config() string {
	cfg, err := json.Marshal(bs.config)
	if err != nil {
		bs.Logger.Err(err).Msg("failed to marshal config")
		return "{}"
	}
	return string(cfg)
}

func (bs *BrowserServer) Name() comm.MoLingServerType {
	return BrowserServerName
}

// LoadConfig loads the configuration from a JSON object.
func (bs *BrowserServer) LoadConfig(jsonData map[string]interface{}) error {
	err := utils.MergeJSONToStruct(bs.config, jsonData)
	if err != nil {
		return err
	}
	return bs.config.Check()
}

func init() {
	RegisterServ(BrowserServerName, NewBrowserServer)
}
