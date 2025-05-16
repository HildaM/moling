# Cobra 命令行工具快速入门指南

Cobra 是一个用于创建强大的现代 CLI (命令行界面) 应用程序的 Go 库。本指南将帮助你快速上手开发基于 Cobra 的命令行工具，参考了 Cobra 官方推荐实践和 MoLing 项目的实际应用。

## 目录

1. [安装 Cobra](#安装-cobra)
2. [项目结构](#项目结构)
3. [基本概念](#基本概念)
4. [快速开始](#快速开始)
5. [命令模板](#命令模板)
6. [高级功能](#高级功能)
7. [最佳实践](#最佳实践)

## 安装 Cobra

使用 Go 模块安装 Cobra 库：

```bash
go get -u github.com/spf13/cobra@latest
```

可选：安装 Cobra CLI 工具（用于生成文件）：

```bash
go install github.com/spf13/cobra-cli@latest
```

## 项目结构

一个典型的 Cobra 应用程序结构如下：

```
myapp/
├── cmd/
│   ├── root.go     # 根命令
│   ├── command1.go # 子命令1
│   └── command2.go # 子命令2
├── main.go         # 程序入口
└── go.mod          # Go 模块文件
```

## 基本概念

Cobra 基于三个基本概念：

1. **Commands**（命令）：表示要执行的操作
2. **Args**（参数）：命令的参数
3. **Flags**（标志）：修改命令行为的选项

## 快速开始

### 1. 创建 main.go

```go
package main

import (
	"yourapp/cmd"
)

func main() {
	cmd.Execute()
}
```

### 2. 创建根命令 (cmd/root.go)

```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "yourapp",
	Short: "YourApp is a CLI application",
	Long: `A longer description that spans multiple lines and provides
more detailed information about your CLI application.`,
	Run: func(cmd *cobra.Command, args []string) {
		// 如果没有指定子命令，可以执行默认逻辑或显示帮助信息
		cmd.Help()
	},
}

// Execute 添加所有子命令到根命令并设置标志
// 这由 main.main() 调用，只需要执行一次
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// 在这里，你可以定义根命令的标志和配置设置
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "开启详细输出")

	// 启用命令自动补全
	rootCmd.CompletionOptions.DisableDefaultCmd = false
}
```

### 3. 添加子命令 (cmd/command1.go)

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var command1Cmd = &cobra.Command{
	Use:   "command1",
	Short: "命令1的简短描述",
	Long:  `命令1的详细描述...`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("command1 被调用")
		// 获取标志值
		flagValue, _ := cmd.Flags().GetString("flag1")
		if flagValue != "" {
			fmt.Printf("flag1 的值是: %s\n", flagValue)
		}
	},
}

func init() {
	// 将命令添加到根命令
	rootCmd.AddCommand(command1Cmd)
	
	// 添加本地标志
	command1Cmd.Flags().StringP("flag1", "f", "", "命令1的标志")
}
```

## 命令模板

以下是一个完整的命令模板，你可以用它来创建新的命令：

```go
package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

// 定义命令相关的变量
var (
	// 用于存储标志值的变量
	flagVariable string
	boolVariable bool
)

// 创建命令对象
var commandName = &cobra.Command{
	// 命令的基本信息
	Use:     "commandName [args]",         // 使用方式
	Aliases: []string{"alias1", "alias2"}, // 命令别名
	Short:   "命令的简短描述",                 // 简短描述
	Long: `命令的详细描述，可以包含多行文本，
用于解释命令的用途和使用方法。`,             // 详细描述
	Example: `  yourapp commandName --flag=value
  yourapp commandName -f value arg1`,     // 使用示例
	
	// 验证参数
	Args: cobra.MinimumNArgs(1),           // 至少需要一个参数
	
	// 命令执行函数
	RunE: func(cmd *cobra.Command, args []string) error {
		// 命令的实际逻辑
		fmt.Printf("命令被执行，参数: %v\n", args)
		fmt.Printf("标志值: %s, 布尔标志: %v\n", flagVariable, boolVariable)
		
		// 返回 nil 表示成功，或返回 error 表示失败
		return nil
	},
	
	// 其他选项
	SilenceUsage:  true,                   // 出错时不显示用法
	SilenceErrors: false,                  // 不屏蔽错误输出
}

// init 函数会在包被导入时自动执行
func init() {
	// 将命令添加到父命令
	rootCmd.AddCommand(commandName)
	
	// 持久标志 (对当前命令及其子命令有效)
	commandName.PersistentFlags().StringVarP(&flagVariable, "flag", "f", "默认值", "标志的说明")
	
	// 本地标志 (仅对当前命令有效)
	commandName.Flags().BoolVarP(&boolVariable, "bool", "b", false, "布尔标志的说明")
	
	// 必需的标志
	commandName.MarkFlagRequired("flag")
}
```

## 高级功能

### 预运行和后运行钩子

每个命令可以定义以下钩子：

```go
var command = &cobra.Command{
	// ...其他设置...
	
	// 在 Run 之前执行
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// 父命令和子命令都会执行
	},
	PreRun: func(cmd *cobra.Command, args []string) {
		// 仅当前命令执行
	},
	
	// 主要执行逻辑
	Run: func(cmd *cobra.Command, args []string) {
		// 主要逻辑
	},
	
	// 在 Run 之后执行
	PostRun: func(cmd *cobra.Command, args []string) {
		// 仅当前命令执行
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		// 父命令和子命令都会执行
	},
}
```

### 自定义帮助和使用信息

```go
rootCmd.SetHelpTemplate(`自定义帮助模板: {{.Name}}`)
rootCmd.SetUsageTemplate(`自定义用法模板: {{.UseLine}}`)
rootCmd.SetHelpCommand(customHelpCmd)
rootCmd.SetHelpFunc(customHelpFunc)
rootCmd.SetUsageFunc(customUsageFunc)
```

## 最佳实践

以下是从 MoLing 项目中提取的一些最佳实践：

### 1. 清晰的命令结构

MoLing 项目使用一个根命令和多个子命令：

```go
// 根命令
var rootCmd = &cobra.Command{
	Use:        "moling",
	Short:      "MoLing 是一个本地部署的 AI 助手",
	SuggestFor: []string{"molin", "moli", "mling"},
	// ...
}

// 子命令示例
var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "提供自动访问 MoLing MCP 服务器的功能",
	// ...
}

func init() {
	rootCmd.AddCommand(clientCmd)
}
```

### 2. 使用变量存储标志值

```go
var (
	list    bool
	install bool
)

func init() {
	clientCmd.PersistentFlags().BoolVar(&list, "list", false, "列出当前安装的 MCP 客户端")
	clientCmd.PersistentFlags().BoolVarP(&install, "install", "i", false, "将 MoLing MCP 服务器配置添加到当前计算机上安装的 MCP 客户端")
}
```

### 3. 分离命令执行逻辑

将命令执行逻辑提取到单独的函数中，而不是内联在 `Run` 中：

```go
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "配置 MoLing MCP 服务器",
	RunE:  ConfigCommandFunc,
}

func ConfigCommandFunc(command *cobra.Command, args []string) error {
	// 命令执行逻辑...
	return nil
}
```

### 4. 自定义帮助和使用信息

MoLing 使用自定义函数来格式化帮助和使用信息：

```go
func usageFunc(c *cobra.Command) error {
	return cobrautl.UsageFunc(c, GitVersion)
}

func Execute() {
	rootCmd.SetUsageFunc(usageFunc)
	rootCmd.SetHelpTemplate(`{{.UsageString}}`)
	// ...
}
```

### 5. 全局预运行钩子

使用 `PersistentPreRunE` 在执行任何命令前进行全局设置：

```go
var rootCmd = &cobra.Command{
	// ...
	PersistentPreRunE: mlsCommandPreFunc,
}

func mlsCommandPreFunc(cmd *cobra.Command, args []string) error {
	// 全局初始化代码...
	return nil
}
```

## 实际案例：简单的文件工具

以下是一个实际的案例，展示如何创建一个简单的文件工具：

```go
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	recursive bool
	output    string
)

var rootCmd = &cobra.Command{
	Use:   "filetool",
	Short: "一个简单的文件处理工具",
	Long:  `filetool 是一个用于文件操作的 CLI 工具，提供文件查找、统计等功能。`,
}

var findCmd = &cobra.Command{
	Use:   "find [目录]",
	Short: "查找文件",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]
		
		walkFunc := func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			
			if !info.IsDir() {
				fmt.Println(path)
			} else if !recursive && path != dir {
				return filepath.SkipDir
			}
			
			return nil
		}
		
		return filepath.Walk(dir, walkFunc)
	},
}

func init() {
	rootCmd.AddCommand(findCmd)
	
	findCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "递归查找子目录")
	findCmd.Flags().StringVarP(&output, "output", "o", "", "输出结果到文件")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
```

## 结论

使用 Cobra 开发命令行工具既快速又强大。通过遵循本指南中的模式和最佳实践，你可以创建出专业、用户友好的 CLI 应用程序。Cobra 提供了构建复杂命令结构所需的所有工具，同时保持代码的可读性和可维护性。

参考资料：
- [Cobra 官方文档](https://github.com/spf13/cobra/blob/master/README.md)
- [MoLing 项目](https://github.com/gojue/moling) 