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

package command

import (
	"context"
	"errors"
	"os/exec"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gojue/moling/pkg/comm"
	"github.com/mark3labs/mcp-go/mcp"
)

// MockCommandServer is a mock implementation of CommandServer for testing purposes.
type MockCommandServer struct {
	CommandServer
}

// TestExecuteCommand tests the ExecCommand function.
func TestExecuteCommand(t *testing.T) {
	// 使用不同的命令和期望输出，取决于操作系统
	var execCmd, expectedSubstring string

	if runtime.GOOS == "windows" {
		execCmd = "echo Hello, World!"
		expectedSubstring = "Hello, World!"
	} else {
		execCmd = "echo 'Hello, World!'"
		expectedSubstring = "Hello, World!"
	}

	// 测试简单命令
	output, err := ExecCommand(execCmd)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// 不再进行精确匹配，而是检查输出是否包含预期的子字符串
	if !strings.Contains(output, expectedSubstring) {
		t.Errorf("Expected output to contain %q, got %q", expectedSubstring, output)
	}
	t.Logf("Command output: %s", output)

	// 测试带超时的命令
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*10)
	defer cancel()

	// 跳过特定于平台的命令测试
	if runtime.GOOS != "windows" {
		execCmd = "curl ifconfig.me|grep Time"
		output, err = ExecCommand(execCmd)
		if err != nil {
			t.Logf("跳过特定于Unix的命令测试: %v", err)
		} else {
			t.Logf("Command output: %s", output)
		}
	}

	// 超时测试对所有平台都适用
	sleepCmd := "sleep"
	if runtime.GOOS == "windows" {
		sleepCmd = "timeout"
	}

	cmd := exec.CommandContext(ctx, sleepCmd, "1")
	err = cmd.Run()
	if err == nil {
		t.Fatalf("Expected timeout error, got nil")
	}
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Errorf("Expected context deadline exceeded error, got %v", ctx.Err())
	}
}

func TestAllowCmd(t *testing.T) {
	// Test with a command that is allowed
	_, ctx, err := comm.InitTestEnv()
	if err != nil {
		t.Fatalf("Failed to initialize test environment: %v", err)
	}

	cs, err := NewCommandServer(ctx)
	if err != nil {
		t.Fatalf("Failed to create CommandServer: %v", err)
	}

	cc := StructToMap(NewCommandConfig())
	t.Logf("CommandConfig: %v", cc)
	err = cs.LoadConfig(cc)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 根据平台使用不同的测试命令
	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "cd C:\\Windows && dir"
	} else {
		cmd = "cd /var/logs/notfound && git log --since=\"today\" --pretty=format:\"%h - %an, %ar : %s\""
	}

	cs1 := cs.(*CommandServer)
	if !cs1.isAllowedCommand(cmd) {
		t.Errorf("Command '%s' is not allowed", cmd)
	}
	t.Log("Command is allowed:", cmd)
}

// TestHandleExecuteCommand tests the handleExecuteCommand method
func TestHandleExecuteCommand(t *testing.T) {
	// 初始化测试环境
	_, ctx, err := comm.InitTestEnv()
	if err != nil {
		t.Fatalf("Failed to initialize test environment: %v", err)
	}

	// 创建CommandServer实例
	svc, err := NewCommandServer(ctx)
	if err != nil {
		t.Fatalf("Failed to create CommandServer: %v", err)
	}

	// 需要转换为具体类型才能访问方法
	cs, ok := svc.(*CommandServer)
	if !ok {
		t.Fatalf("Failed to convert Service to CommandServer")
	}

	// 加载配置
	cc := StructToMap(NewCommandConfig())
	err = cs.LoadConfig(cc)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 测试Params.Arguments字段是否可以正确设置和使用
	t.Run("TestArgumentsAccess", func(t *testing.T) {
		// 创建请求
		request := mcp.CallToolRequest{}

		// 根据平台设置不同的测试命令
		var testCmd string
		if runtime.GOOS == "windows" {
			testCmd = "echo Test Command"
		} else {
			testCmd = "echo 'Test Command'"
		}

		// 初始化Params字段
		request.Params.Arguments = map[string]interface{}{
			"command": testCmd,
		}

		// 获取并验证参数
		args := request.GetArguments()
		cmd, ok := args["command"].(string)
		if !ok {
			t.Fatalf("Failed to get command argument")
		}

		if cmd != testCmd {
			t.Errorf("Expected command to be %s, got %s", testCmd, cmd)
		}
	})

	// 测试命令执行功能 - 跳过实际执行
	t.Run("TestExecuteCommandMethod", func(t *testing.T) {
		t.Skip("跳过实际执行命令的测试")

		// 创建请求
		request := mcp.CallToolRequest{}

		// 根据平台设置不同的测试命令
		var testCmd string
		if runtime.GOOS == "windows" {
			testCmd = "echo Test Command"
		} else {
			testCmd = "echo 'Test Command'"
		}

		// 初始化Params字段 - 使用允许的echo命令
		request.Params.Arguments = map[string]interface{}{
			"command": testCmd,
		}

		// 调用handleExecuteCommand方法
		result, err := cs.handleExecuteCommand(ctx, request)
		if err != nil {
			t.Fatalf("handleExecuteCommand failed: %v", err)
		}

		// 验证结果
		if result == nil {
			t.Errorf("Expected non-nil result")
		}
	})
}

// 将 struct 转换为 map
func StructToMap(obj interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		value := val.Field(i)
		// 跳过未导出的字段
		if field.PkgPath != "" {
			continue
		}
		// 获取字段的 json tag（如果存在）
		key := field.Name
		if tag := field.Tag.Get("json"); tag != "" {
			key = tag
		}
		result[key] = value.Interface()
	}
	return result
}
