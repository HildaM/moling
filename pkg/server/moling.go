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

package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gojue/moling/pkg/comm"
	"github.com/gojue/moling/pkg/config"
	"github.com/gojue/moling/pkg/services/abstract"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rs/zerolog"
)

// MoLingServer 服务器实例
type MoLingServer struct {
	ctx        context.Context     // 上下文
	server     *server.MCPServer   // MCP服务器实例
	services   []abstract.Service  // 服务列表
	logger     zerolog.Logger      // 日志记录器
	mlConfig   config.MoLingConfig // 配置
	listenAddr string              // SSE模式监听地址，如果为空，则使用STDIO模式
}

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
