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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gojue/moling/services"
	"github.com/spf13/cobra"
	"os"
	"path/filepath"
)

func init() {
	rootCmd.AddCommand(configCmd)
}

// configCmd 显示当前服务列表的配置
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show the configuration of the current service list",
	Long: `Show the configuration of the current service list. You can refer to the configuration file to modify the configuration.
`,
	RunE: ConfigCommandFunc,
}

// ConfigCommandFunc executes the "config" command.
func ConfigCommandFunc(command *cobra.Command, args []string) error {
	// 1. 设置日志
	logger := setupLogger(mlConfig.BasePath)
	mlConfig.SetLogger(logger)

	// 2. 创建上下文
	ctx := createContext(logger)

	// 3. 加载现有配置文件(如果存在)
	configFilePath := filepath.Join(mlConfig.BasePath, mlConfig.ConfigFile)
	existingConfig, hasConfig, err := loadExistingConfig(configFilePath)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to load config")
		return err
	}

	// 4. 构建完整配置(合并全局配置与各服务配置)
	configData, err := buildConfigData(ctx, existingConfig)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to build config")
		return err
	}

	// 5. 格式化配置数据为美观的JSON
	formattedJson, err := formatConfigJson(configData)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to format config")
		return err
	}

	// 6. 如果配置文件不存在，则创建
	if err := saveConfigIfNeeded(formattedJson, configFilePath, hasConfig); err != nil {
		logger.Error().Err(err).Msg("Failed to save config")
		return err
	}

	// 7. 输出配置信息
	logger.Info().Str("config", configFilePath).Msg("Current loaded configuration file path")
	logger.Info().Msg("You can modify the configuration file to change the settings.")
	logger.Info().Msgf("Configuration details: \n%s", formattedJson)

	return nil
}

// loadExistingConfig 加载现有配置文件(如果存在)
func loadExistingConfig(configFilePath string) (map[string]interface{}, bool, error) {
	// 尝试读取配置文件
	configData, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, false, nil
	}

	// 解析配置文件内容为JSON
	var configJson map[string]interface{}
	if err := json.Unmarshal(configData, &configJson); err != nil {
		return nil, true, fmt.Errorf("invalid JSON in config file: %v", err)
	}
	return configJson, true, nil
}

// buildConfigData 构建完整配置，包括全局配置和各服务配置
func buildConfigData(ctx context.Context, existingConfig map[string]interface{}) (string, error) {
	// 创建配置缓冲区
	bf := bytes.Buffer{}
	bf.WriteString("\n{\n")

	// 添加全局配置
	if err := addGlobalConfig(&bf); err != nil {
		return "", err
	}

	// 添加各服务配置
	if err := addServiceConfigs(ctx, &bf, existingConfig); err != nil {
		return "", err
	}

	bf.WriteString("}\n")
	return bf.String(), nil
}

// addGlobalConfig 添加全局配置到缓冲区
func addGlobalConfig(bf *bytes.Buffer) error {
	mlConfigJson, err := json.Marshal(mlConfig)
	if err != nil {
		return fmt.Errorf("Error marshaling GlobalConfig: %v", err)
	}
	bf.WriteString("\t\"MoLingConfig\":\n")
	bf.WriteString(fmt.Sprintf("\t%s,\n", mlConfigJson))
	return nil
}

// addServiceConfigs 添加各服务配置到缓冲区
func addServiceConfigs(ctx context.Context, bf *bytes.Buffer, existingConfig map[string]interface{}) error {
	first := true
	for srvName, nsv := range services.ServiceList() {
		// 初始化服务
		srv, err := initSingleService(ctx, srvName, nsv, existingConfig)
		if err != nil {
			return err
		}

		// 添加服务配置到缓冲区
		if !first {
			bf.WriteString(",\n")
		}
		bf.WriteString(fmt.Sprintf("\t\"%s\":\n", srv.Name()))
		bf.WriteString(fmt.Sprintf("\t%s\n", srv.Config()))
		first = false
	}
	return nil
}

// formatConfigJson 格式化配置数据为美观的JSON
func formatConfigJson(configData string) ([]byte, error) {
	var jsonObj interface{}
	if err := json.Unmarshal([]byte(configData), &jsonObj); err != nil {
		return nil, fmt.Errorf("Error unmarshaling JSON: %v, payload:%s", err, configData)
	}

	formattedJson, err := json.MarshalIndent(jsonObj, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("Error marshaling JSON: %v", err)
	}
	return formattedJson, nil
}

// saveConfigIfNeeded 如果配置文件不存在，则创建
// 首次运行自动创建配置：当用户首次运行 moling config 命令时，会自动创建一个包含默认配置的配置文件
// 避免覆盖用户自定义配置：如果配置文件已存在，会完全跳过写入操作，保护用户的自定义设置
func saveConfigIfNeeded(formattedJson []byte, configFilePath string, hasConfig bool) error {
	if !hasConfig {
		if err := os.WriteFile(configFilePath, formattedJson, 0644); err != nil {
			return fmt.Errorf("Error writing configuration file: %v", err)
		}
	}
	return nil
}
