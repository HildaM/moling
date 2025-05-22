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

package browser

import (
	"testing"

	"github.com/gojue/moling/pkg/comm"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestBrowserServer(t *testing.T) {
	// 初始化测试环境
	logger, ctx, err := comm.InitTestEnv()
	if err != nil {
		t.Fatalf("Failed to initialize test environment: %v", err)
	}
	logger.Info().Msg("TestBrowserServer")

	// 创建BrowserServer实例
	svc, err := NewBrowserServer(ctx)
	if err != nil {
		t.Fatalf("Failed to create BrowserServer: %v", err)
	}

	// 需要转换为具体类型才能访问方法
	bs, ok := svc.(*BrowserServer)
	if !ok {
		t.Fatalf("Failed to convert Service to BrowserServer")
	}

	// 测试Params.Arguments字段是否可以正确设置和使用
	t.Run("TestArgumentsAccess", func(t *testing.T) {
		// 创建请求
		request := mcp.CallToolRequest{}

		// 初始化Params字段
		request.Params.Arguments = map[string]interface{}{
			"url": "https://www.baidu.com",
		}

		// 获取并验证参数
		args := request.GetArguments()
		url, ok := args["url"].(string)
		if !ok {
			t.Fatalf("Failed to get url argument")
		}

		if url != "https://www.baidu.com" {
			t.Errorf("Expected url to be https://www.baidu.com, got %s", url)
		}
	})

	// 下面添加其他测试案例，但跳过实际执行浏览器操作
	// 这些测试仅验证参数处理的正确性，不实际执行浏览器操作

	t.Run("TestNavigate", func(t *testing.T) {
		t.Skip("跳过实际执行浏览器操作的测试")

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]interface{}{
			"url": "https://www.baidu.com",
		}

		result, err := bs.handleNavigate(ctx, request)
		if err != nil {
			t.Fatalf("handleNavigate failed: %v", err)
		}

		if result.Content[0].(mcp.TextContent).Text != "Navigated to https://www.baidu.com" {
			t.Errorf("Unexpected result: %v", result.Content[0].(mcp.TextContent).Text)
		}
	})

	t.Run("TestScreenshot", func(t *testing.T) {
		t.Skip("跳过实际执行浏览器操作的测试")

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]interface{}{
			"name": "test_screenshot",
		}

		_, err := bs.handleScreenshot(ctx, request)
		if err != nil {
			t.Fatalf("handleScreenshot failed: %v", err)
		}
	})

	t.Run("TestClick", func(t *testing.T) {
		t.Skip("跳过实际执行浏览器操作的测试")

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]interface{}{
			"selector": "body",
		}

		_, err := bs.handleClick(ctx, request)
		if err != nil {
			t.Fatalf("handleClick failed: %v", err)
		}
	})

	t.Run("TestFill", func(t *testing.T) {
		t.Skip("跳过实际执行浏览器操作的测试")

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]interface{}{
			"selector": "input[name='q']",
			"value":    "test",
		}

		_, err := bs.handleFill(ctx, request)
		if err != nil {
			t.Fatalf("handleFill failed: %v", err)
		}
	})

	t.Run("TestSelect", func(t *testing.T) {
		t.Skip("跳过实际执行浏览器操作的测试")

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]interface{}{
			"selector": "select[name='dropdown']",
			"value":    "option1",
		}

		_, err := bs.handleSelect(ctx, request)
		if err != nil {
			t.Fatalf("handleSelect failed: %v", err)
		}
	})

	t.Run("TestHover", func(t *testing.T) {
		t.Skip("跳过实际执行浏览器操作的测试")

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]interface{}{
			"selector": "body",
		}

		_, err := bs.handleHover(ctx, request)
		if err != nil {
			t.Fatalf("handleHover failed: %v", err)
		}
	})

	t.Run("TestEvaluate", func(t *testing.T) {
		t.Skip("跳过实际执行浏览器操作的测试")

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]interface{}{
			"script": "document.title",
		}

		result, err := bs.handleEvaluate(ctx, request)
		if err != nil {
			t.Fatalf("handleEvaluate failed: %v", err)
		}

		if result == nil {
			t.Errorf("Expected non-nil result")
		}
	})

	// 测试调试相关功能
	t.Run("TestDebugEnable", func(t *testing.T) {
		t.Skip("跳过实际执行浏览器操作的测试")

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]interface{}{
			"enabled": true,
		}

		result, err := bs.handleDebugEnable(ctx, request)
		if err != nil {
			t.Fatalf("handleDebugEnable failed: %v", err)
		}

		if result == nil {
			t.Errorf("Expected non-nil result")
		}
	})

	t.Run("TestSetBreakpoint", func(t *testing.T) {
		t.Skip("跳过实际执行浏览器操作的测试")

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]interface{}{
			"url":       "https://www.baidu.com",
			"line":      float64(10),
			"column":    float64(5),
			"condition": "x > 10",
		}

		result, err := bs.handleSetBreakpoint(ctx, request)
		if err != nil {
			t.Fatalf("handleSetBreakpoint failed: %v", err)
		}

		if result == nil {
			t.Errorf("Expected non-nil result")
		}
	})

	t.Run("TestRemoveBreakpoint", func(t *testing.T) {
		t.Skip("跳过实际执行浏览器操作的测试")

		request := mcp.CallToolRequest{}
		request.Params.Arguments = map[string]interface{}{
			"breakpointId": "test-breakpoint-id",
		}

		result, err := bs.handleRemoveBreakpoint(ctx, request)
		if err != nil {
			t.Fatalf("handleRemoveBreakpoint failed: %v", err)
		}

		if result == nil {
			t.Errorf("Expected non-nil result")
		}
	})

	t.Run("TestPause", func(t *testing.T) {
		t.Skip("跳过实际执行浏览器操作的测试")

		request := mcp.CallToolRequest{}

		result, err := bs.handlePause(ctx, request)
		if err != nil {
			t.Fatalf("handlePause failed: %v", err)
		}

		if result == nil {
			t.Errorf("Expected non-nil result")
		}
	})

	t.Run("TestResume", func(t *testing.T) {
		t.Skip("跳过实际执行浏览器操作的测试")

		request := mcp.CallToolRequest{}

		result, err := bs.handleResume(ctx, request)
		if err != nil {
			t.Fatalf("handleResume failed: %v", err)
		}

		if result == nil {
			t.Errorf("Expected non-nil result")
		}
	})

	t.Run("TestGetCallstack", func(t *testing.T) {
		t.Skip("跳过实际执行浏览器操作的测试")

		request := mcp.CallToolRequest{}

		result, err := bs.handleGetCallstack(ctx, request)
		if err != nil {
			t.Fatalf("handleGetCallstack failed: %v", err)
		}

		if result == nil {
			t.Errorf("Expected non-nil result")
		}
	})
}
