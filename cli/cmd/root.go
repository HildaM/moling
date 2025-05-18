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

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/gojue/moling/cli/cobrautl"
	"github.com/gojue/moling/pkg/server"
	"github.com/gojue/moling/pkg/services"
	"github.com/gojue/moling/pkg/utils"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

func init() {
	// set default config file path
	currentUser, err := user.Current()
	if err == nil {
		mlConfig.BasePath = filepath.Join(currentUser.HomeDir, MLRootPath)
	}

	cobra.EnablePrefixMatching = true
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.PersistentFlags().StringVar(&mlConfig.BasePath, "base_path", mlConfig.BasePath, "MoLing Base Data Path, automatically set by the system, cannot be changed, display only.")
	rootCmd.PersistentFlags().BoolVarP(&mlConfig.Debug, "debug", "d", false, "Debug mode, default is false.")
	rootCmd.PersistentFlags().StringVarP(&mlConfig.ListenAddr, "listen_addr", "l", "", "listen address for SSE mode. default:'', not listen, used STDIO mode.")
	rootCmd.PersistentFlags().StringVarP(&mlConfig.Module, "module", "m", "all", "module to load, default: all; others: Browser,FileSystem,Command, etc. Multiple modules are separated by commas")
	rootCmd.SilenceUsage = true
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:        CliName,
	Short:      CliDescription,
	SuggestFor: []string{"molin", "moli", "mling"},

	Long: CliDescriptionLong,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	RunE:              mlsCommandFunc,
	PersistentPreRunE: mlsCommandPreFunc,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// 设置使用说明
	rootCmd.SetUsageFunc(func(c *cobra.Command) error {
		return cobrautl.UsageFunc(c, GitVersion)
	})
	// 设置帮助模板
	rootCmd.SetHelpTemplate(`{{.UsageString}}`)
	// 禁用默认命令
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	// 设置版本
	rootCmd.Version = GitVersion
	// 设置版本模板
	rootCmd.SetVersionTemplate(`{{with .Name}}{{printf "%s " .}}{{end}}{{printf "version:\t%s" .Version}}
`)

	// 执行命令
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// mlsCommandFunc 服务核心启动函数
func mlsCommandFunc(command *cobra.Command, args []string) error {
	// 初始化日志
	logger := initLogger(mlConfig.BasePath)
	mlConfig.SetLogger(logger)

	// 检查运行实例和配置文件
	pidFilePath := filepath.Join(mlConfig.BasePath, MLPidName)
	if err := checkRunningInstance(pidFilePath, logger); err != nil {
		return err
	}

	// 加载配置文件
	configFilePath := filepath.Join(mlConfig.BasePath, mlConfig.ConfigFile)
	configJson, err := loadConfigFile(configFilePath, logger)
	if err != nil {
		return err
	}

	// 创建并启动服务
	ctx := createContext(logger)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// 加载并初始化服务
	servicesList, closers, err := initServices(ctx, configJson, logger)
	if err != nil {
		cancel()
		return err
	}

	// 启动MCP服务器
	_, err = startMoLingServer(ctx, servicesList, logger)
	if err != nil {
		cancel()
		return err
	}

	// 等待信号并执行优雅关闭
	return waitForShutdownSignal(cancel, closers, pidFilePath, logger)
}

// checkRunningInstance 检查是否有已运行的实例
func checkRunningInstance(pidFilePath string, logger zerolog.Logger) error {
	logger.Info().Str("pid", pidFilePath).Msg("Starting MoLing MCP Server...")
	if err := utils.CreatePIDFile(pidFilePath); err != nil {
		return err
	}
	return nil
}

// loadConfigFile 加载配置文件
func loadConfigFile(configFilePath string, logger zerolog.Logger) (map[string]interface{}, error) {
	logger.Info().Str("ServerName", MCPServerName).Str("version", GitVersion).Msg("start")

	var configJson map[string]interface{}
	configContent, err := os.ReadFile(configFilePath)
	if err == nil {
		if err = json.Unmarshal(configContent, &configJson); err != nil {
			return nil, fmt.Errorf("Error unmarshaling JSON: %v, config file:%s", err, configFilePath)
		}
	}

	logger.Info().Str("config_file", configFilePath).Msg("load config file")
	return configJson, nil
}

// startMoLingServer 启动MoLing服务器
func startMoLingServer(ctx context.Context, servicesList []services.Service, logger zerolog.Logger) (*server.MoLingServer, error) {
	server, err := server.NewMoLingServer(ctx, servicesList, *mlConfig)
	if err != nil {
		logger.Error().Err(err).Msg("failed to create server")
		return nil, err
	}

	go func() {
		if err := server.Serve(); err != nil {
			logger.Error().Err(err).Msg("failed to start server")
		}
	}()
	return server, nil
}

// waitForShutdownSignal 等待关闭信号并优雅关闭服务
func waitForShutdownSignal(cancelFunc context.CancelFunc, closers map[string]func() error, pidFilePath string, logger zerolog.Logger) error {
	// 创建信号通道
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 监控父进程退出
	go monitorParentProcess(sigChan, logger)

	// 等待信号
	_ = <-sigChan
	logger.Info().Msg("Received signal, shutting down...")

	// 优雅关闭所有服务
	shutdownServices(closers, cancelFunc, logger)

	// 清理PID文件
	if err := utils.RemovePIDFile(pidFilePath); err != nil {
		logger.Error().Err(err).Msgf("failed to remove pid file %s", pidFilePath)
		return err
	}

	logger.Info().Msgf("removed pid file %s", pidFilePath)
	logger.Info().Msg(" Bye!")
	return nil
}

// monitorParentProcess 监控父进程是否退出
func monitorParentProcess(sigChan chan<- os.Signal, logger zerolog.Logger) {
	ppid := os.Getppid()
	for {
		time.Sleep(1 * time.Second)
		newPpid := os.Getppid()
		if newPpid == 1 {
			logger.Info().Msgf("parent process changed, origin PPid:%d, New PPid:%d", ppid, newPpid)
			logger.Warn().Msg("parent process exited")
			sigChan <- syscall.SIGTERM
			break
		}
	}
}

// shutdownServices 优雅关闭所有服务
func shutdownServices(closers map[string]func() error, cancelFunc context.CancelFunc, logger zerolog.Logger) {
	var wg sync.WaitGroup
	done := make(chan struct{})

	// 并行关闭所有服务
	go func() {
		for serviceName, closer := range closers {
			wg.Add(1)
			go func(name string, closeFn func() error) {
				defer wg.Done()
				if err := closeFn(); err != nil {
					logger.Error().Err(err).Msgf("failed to close service %s", name)
				} else {
					logger.Info().Msgf("service %s closed", name)
				}
			}(serviceName, closer)
		}

		// 等待所有服务关闭
		wg.Wait()
		close(done)
	}()

	// 等待服务关闭或超时
	select {
	case <-time.After(5 * time.Second):
		cancelFunc()
		logger.Info().Msg("timeout, all services closed forcefully")
	case <-done:
		cancelFunc()
		logger.Info().Msg("all services closed gracefully")
	}
}
