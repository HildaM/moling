package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"context"

	"github.com/gojue/moling/pkg/comm"
	"github.com/gojue/moling/pkg/config"
	"github.com/gojue/moling/pkg/services"
	"github.com/gojue/moling/pkg/services/abstract"
	"github.com/gojue/moling/pkg/utils"
	"github.com/rs/zerolog"
)

const (
	CliName            = "moling"
	CliNameZh          = "魔灵"
	MCPServerName      = "MoLing MCP Server"
	CliDescription     = "MoLing is a computer-use and browser-use based MCP server. It is a locally deployed, dependency-free office AI assistant."
	CliDescriptionZh   = "MoLing（魔灵）是一款基于computer-use和浏browser-use的 MCP 服务器，它是一个本地部署、无依赖的办公 AI 助手。"
	CliHomepage        = "https://gojue.cc/moling"
	CliAuthor          = "CFC4N <cfc4ncs@gmail.com>"
	CliGithubRepo      = "https://github.com/gojue/moling"
	CliDescriptionLong = `
MoLing is a computer-based MCP Server that implements system interaction through operating system APIs, enabling file system operations such as reading, writing, merging, statistics, and aggregation, as well as the ability to execute system commands. It is a dependency-free local office automation assistant.

Requiring no installation of any dependencies, MoLing can be run directly and is compatible with multiple operating systems, including Windows, Linux, and macOS. This eliminates the hassle of dealing with environment conflicts involving Node.js, Python, and other development environments.

Usage:
  moling
  moling -l 127.0.0.1:6789
  moling -h
  moling client -i
  moling config 
`
	CliDescriptionLongZh = `MoLing（魔灵）是一个computer-use的MCP Server，基于操作系统API实现了系统交互，可以实现文件系统的读写、合并、统计、聚合等操作，也可以执行系统命令操作。是一个无需任何依赖的本地办公自动化助手。
没有任何安装依赖，直接运行，兼容Windows、Linux、macOS等操作系统。再也不用苦恼NodeJS、Python等环境冲突等问题。

Usage:
  moling
  moling -l 127.0.0.1:29118
  moling -h
  moling client -i
  moling config 
`
)

const (
	MLConfigName = "config.json"     // config file name of MoLing Server
	MLRootPath   = ".moling"         // config file name of MoLing Server
	MLPidName    = "moling.pid"      // pid file name
	LogFileName  = "moling.log"      //	log file name
	MaxLogSize   = 1024 * 1024 * 512 // 512MB
)

var (
	GitVersion = "unknown_arm64_v0.0.0_2025-03-22 20:08"
	mlConfig   = &config.MoLingConfig{
		Version:    GitVersion,
		ConfigFile: filepath.Join("config", MLConfigName),
		BasePath:   filepath.Join(os.TempDir(), MLRootPath), // will set in mlsCommandPreFunc
	}

	// mlDirectories is a list of directories to be created in the base path
	mlDirectories = []string{
		"logs",    // log file
		"config",  // config file
		"browser", // browser cache
		"data",    // data
		"cache",
	}
)

// initLogger init logger
func initLogger(mlDataPath string) zerolog.Logger {
	// 设置全局日志级别
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if mlConfig.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// 初始化 RotateWriter
	logFile := filepath.Join(mlDataPath, "logs", LogFileName)
	rw, err := utils.NewRotateWriter(logFile, MaxLogSize) // 512MB 阈值
	if err != nil {
		panic(fmt.Sprintf("failed to open log file %s: %v", logFile, err))
	}

	// 创建子日志，附带时间戳
	logger := zerolog.New(rw).With().Timestamp().Logger()
	logger.Info().Uint32("MaxLogSize", MaxLogSize).Msgf("Log files are automatically rotated when they exceed the size threshold, and saved to %s.1 and %s.2 respectively", LogFileName, LogFileName)
	return logger
}

// setupLogger 初始化日志记录器，支持控制台和文件双重输出
func setupLogger(basePath string) zerolog.Logger {
	fileLogger := initLogger(basePath)
	consoleWriter := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339} // 控制台输出
	multi := zerolog.MultiLevelWriter(consoleWriter, fileLogger)                     // 双重输出
	return zerolog.New(multi).With().Timestamp().Logger()
}

// createContext 创建包含全局配置和日志的上下文
func createContext(logger zerolog.Logger) context.Context {
	ctx := context.WithValue(context.Background(), comm.MoLingConfigKey, mlConfig)
	return context.WithValue(ctx, comm.MoLingLoggerKey, logger)
}

// initSingleService 初始化单个服务
func initSingleService(ctx context.Context, serviceType comm.MoLingServerType, serviceFactory abstract.ServiceFactory, configJson map[string]interface{}) (abstract.Service, error) {
	// 创建服务实例
	service, err := serviceFactory(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create service %s: %v", serviceType, err)
	}

	// 如果存在配置，尝试加载到服务中
	if configJson != nil {
		// 从配置中提取对应服务的设置
		if rawSettings, exists := configJson[string(serviceType)]; exists {
			if serviceSettings, ok := rawSettings.(map[string]interface{}); ok {
				// 将提取的配置加载到服务
				if err := service.LoadConfig(serviceSettings); err != nil {
					return nil, fmt.Errorf("failed to load config for service %s: %v", service.Name(), err)
				}
			}
		}
	}

	// 初始化服务
	if err := service.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize service %s: %v", service.Name(), err)
	}
	return service, nil
}

// initServices 批量初始化服务
func initServices(ctx context.Context, configJson map[string]interface{}, logger zerolog.Logger) ([]abstract.Service, map[string]func() error, error) {
	var moduleList []string
	if mlConfig.Module != "Browser" {
		moduleList = strings.Split(mlConfig.Module, ",")
	}

	var servicesList []abstract.Service
	closers := make(map[string]func() error)

	for serviceName, serviceFactory := range services.ServiceList() {
		// 检查模块是否需要加载
		if len(moduleList) > 0 {
			// 如果模块列表不为空，则检查模块是否在列表中
			if !utils.StringInSlice(string(serviceName), moduleList) {
				logger.
					Debug().
					Str("moduleName", string(serviceName)).
					Msgf("initServices debug, module %s not in %v, skip", string(serviceName), moduleList)
				continue
			}
			logger.Debug().Str("moduleName", string(serviceName)).Msgf("initServices debug, starting %s service", serviceName)
		}

		// 使用通用的初始化函数
		service, err := initSingleService(ctx, serviceName, serviceFactory, configJson)
		if err != nil {
			logger.Error().Err(err).Msgf("failed to initialize service %s", serviceName)
			return nil, nil, err
		}

		servicesList = append(servicesList, service)
		closers[string(service.Name())] = service.Close
	}
	return servicesList, closers, nil
}
