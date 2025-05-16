/*
 *
 *  Copyright 2025 CFC4N <cfc4n.cs@gmail.com>. All Rights Reserved.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 *
 *  Repository: https://github.com/gojue/moling
 *
 */

package cmd

import (
	"github.com/gojue/moling/client"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	clientCmd.PersistentFlags().BoolVar(&list, "list", false, "List the current installed MCP clients")
	clientCmd.PersistentFlags().BoolVarP(&install, "install", "i", false, "Add MoLing MCP Server configuration to the currently installed MCP clients on this computer. default is all")
	rootCmd.AddCommand(clientCmd)
}

var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Provides automated access to MoLing MCP Server for local MCP clients, Cline, Roo Code, and Claude, etc.",
	Long: `Automatically checks the MCP clients installed on the current computer, displays them, and automatically adds the MoLing MCP Server configuration to enable one-click activation, reducing the hassle of manual configuration.
Currently supports the following clients: Cline, Roo Code, Claude
    moling client -l --list   List the current installed MCP clients
    moling client -i --install Add MoLing MCP Server configuration to the currently installed MCP clients on this computer
`,
	RunE: ClientCommandFunc,
}

var (
	list    bool
	install bool
)

// ClientCommandFunc executes the "client" command.
func ClientCommandFunc(command *cobra.Command, args []string) error {
	// 1. 设置日志
	logger := setupLogger(mlConfig.BasePath)
	mlConfig.SetLogger(logger)
	logger.Debug().Msg("Starting MCP client management")

	// 2. 准备 MCP 服务器配置
	mcpConfig, err := prepareMCPServerConfig(logger)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to prepare MCP server configuration")
		return err
	}

	// 3. 创建客户端管理器
	clientManager := client.NewManager(logger, mcpConfig)

	// 4. 根据命令行参数执行对应操作
	if install {
		return installMCPConfig(clientManager, logger)
	}
	return listMCPClients(clientManager, logger)
}

// prepareMCPServerConfig 准备 MCP 服务器配置
func prepareMCPServerConfig(logger zerolog.Logger) (client.MCPServerConfig, error) {
	// 创建基本配置
	mcpConfig := client.NewMCPServerConfig(CliDescription, CliName, MCPServerName)

	// 获取可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		logger.Warn().Err(err).Msg("Unable to determine executable path, using default command name")
		return mcpConfig, nil
	}

	// 设置命令为可执行文件的完整路径
	logger.Debug().Str("exePath", exePath).Msg("Using executable path for MCP configuration")
	mcpConfig.Command = exePath

	return mcpConfig, nil
}

// installMCPConfig 安装 MCP 配置到客户端
func installMCPConfig(manager *client.Manager, logger zerolog.Logger) error {
	logger.Info().Msg("Installing MCP Server configuration into MCP clients")

	// 执行配置安装
	manager.SetupConfig()

	logger.Info().Msg("MCP Server configuration successfully installed")
	return nil
}

// listMCPClients 列出可用的 MCP 客户端
func listMCPClients(manager *client.Manager, logger zerolog.Logger) error {
	logger.Info().Msg("Listing available MCP clients")

	// 列出客户端
	manager.ListClient()

	logger.Info().Msg("MCP clients listing completed")
	return nil
}
