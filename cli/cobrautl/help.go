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

// Package cobrautl 提供了自定义的 Cobra 命令行帮助和使用信息格式化功能
// 通过重写默认的帮助模板，使命令行界面更加美观和实用
package cobrautl

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	// commandUsageTemplate 是自定义的命令使用模板
	commandUsageTemplate *template.Template

	// templFuncs 定义了在模板中使用的自定义函数
	templFuncs = template.FuncMap{
		// descToLines 将描述文本转换为多行字符串数组
		"descToLines": func(s string) []string {
			// 去除首尾的空白字符并按换行符分割成行
			return strings.Split(strings.Trim(s, "\n\t "), "\n")
		},
		// cmdName 生成完整的命令名称，包括所有父命令
		"cmdName": func(cmd *cobra.Command, startCmd *cobra.Command) string {
			parts := []string{cmd.Name()}
			for cmd.HasParent() && cmd.Parent().Name() != startCmd.Name() {
				cmd = cmd.Parent()
				parts = append([]string{cmd.Name()}, parts...)
			}
			return strings.Join(parts, " ")
		},
	}
)

func init() {
	// 定义自定义的命令使用说明模板
	// 模板将帮助信息分为几个清晰的部分：名称、用法、版本、命令、描述、选项等
	commandUsage := `
{{ $cmd := .Cmd }}\
{{ $cmdname := cmdName .Cmd .Cmd.Root }}\
NAME:
{{ if not .Cmd.HasParent }}\
{{printf "\t%s - %s" .Cmd.Name .Cmd.Short}}
{{else}}\
{{printf "\t%s - %s" $cmdname .Cmd.Short}}
{{end}}\

USAGE:
{{printf "\t%s" .Cmd.UseLine}}
{{ if not .Cmd.HasParent }}\

VERSION:
{{printf "\t%s" .Version}}
{{end}}\
{{if .Cmd.HasSubCommands}}\

COMMANDS:
{{range .SubCommands}}\
{{ $cmdname := cmdName . $cmd }}\
{{ if .Runnable }}\
{{printf "\t%s\t%s" $cmdname .Short}}
{{end}}\
{{end}}\
{{end}}\
{{ if .Cmd.Long }}\

DESCRIPTION:
{{range $line := descToLines .Cmd.Long}}{{printf "\t%s" $line}}
{{end}}\
{{end}}\
{{if .Cmd.HasLocalFlags}}\

OPTIONS:
{{.LocalFlags}}\
{{end}}\
{{if .Cmd.HasInheritedFlags}}\

GLOBAL OPTIONS:
{{.GlobalFlags}}\
{{end}}
`[1:]

	// 解析并编译模板，移除模板中的换行符（保持格式美观）
	commandUsageTemplate = template.Must(template.New("command_usage").Funcs(templFuncs).Parse(strings.Replace(commandUsage, "\\\n", "", -1)))
}

// molingFlagUsages 自定义标志(flags)的格式化显示
// 比默认的格式化更美观，适合MoLing项目的风格
func molingFlagUsages(flagSet *pflag.FlagSet) string {
	x := new(bytes.Buffer)

	// 遍历所有标志并格式化
	flagSet.VisitAll(func(flag *pflag.Flag) {
		// 跳过已废弃的标志
		if len(flag.Deprecated) > 0 {
			return
		}

		// 根据是否有短名称决定显示格式
		var format string
		if len(flag.Shorthand) > 0 {
			format = "  -%s, --%s" // 有短名称时显示: -s, --long
		} else {
			format = "   %s   --%s" // 无短名称时显示: --long（保持对齐）
		}

		// 处理无参数标志
		if len(flag.NoOptDefVal) > 0 {
			format = format + "["
		}

		// 字符串类型的值加引号显示
		if flag.Value.Type() == "string" {
			// 字符串值加引号
			format = format + "=%q"
		} else {
			format = format + "=%s"
		}

		// 处理无参数标志的结尾括号
		if len(flag.NoOptDefVal) > 0 {
			format = format + "]"
		}

		// 完整格式：标志 + tab + 说明
		format = format + "\t%s\n"
		shorthand := flag.Shorthand
		fmt.Fprintf(x, format, shorthand, flag.Name, flag.DefValue, flag.Usage)
	})

	return x.String()
}

// getSubCommands 递归获取命令的所有子命令
// 用于在帮助信息中显示完整的命令树
func getSubCommands(cmd *cobra.Command) []*cobra.Command {
	var subCommands []*cobra.Command
	for _, subCmd := range cmd.Commands() {
		// 添加直接子命令
		subCommands = append(subCommands, subCmd)
		// 递归添加所有子命令的子命令
		subCommands = append(subCommands, getSubCommands(subCmd)...)
	}
	return subCommands
}

// UsageFunc 是cobrautl包的核心函数，为命令提供自定义的使用说明格式
// 参数:
//   - cmd: 要格式化的命令
//   - version: 程序版本号，将显示在帮助信息中
//
// 返回:
//   - error: 格式化过程中可能出现的错误
func UsageFunc(cmd *cobra.Command, version string) error {
	// 获取所有子命令（包括子命令的子命令）
	subCommands := getSubCommands(cmd)
	// 创建一个带Tab对齐的输出写入器
	tabOut := getTabOutWithWriter(os.Stdout)

	// 使用自定义模板渲染命令使用信息
	err := commandUsageTemplate.Execute(tabOut, struct {
		Cmd         *cobra.Command   // 当前命令
		LocalFlags  string           // 本地标志格式化后的字符串
		GlobalFlags string           // 全局标志格式化后的字符串
		SubCommands []*cobra.Command // 所有子命令
		Version     string           // 版本信息
	}{
		cmd,
		molingFlagUsages(cmd.LocalFlags()),
		molingFlagUsages(cmd.InheritedFlags()),
		subCommands,
		version,
	})
	if err != nil {
		return err
	}

	// 刷新输出缓冲区
	err = tabOut.Flush()
	return err
}

// getTabOutWithWriter 创建一个新的tabwriter实例，用于格式化表格式文本
// tabwriter能确保输出的文本按Tab对齐，使帮助信息看起来更整齐
func getTabOutWithWriter(writer io.Writer) *tabwriter.Writer {
	aTabOut := new(tabwriter.Writer)
	// 初始化tabwriter: minwidth=0, tabwidth=8, padding=1, padchar='\t', flags=0
	aTabOut.Init(writer, 0, 8, 1, '\t', 0)
	return aTabOut
}
