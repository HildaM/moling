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

package config

import (
	"fmt"

	"github.com/rs/zerolog"
)

// Config is an interface that defines a method for checking configuration validity.
type Config interface {
	// Check validates the configuration and returns an error if the configuration is invalid.
	Check() error
}

// MoLingConfig is a struct that holds the configuration for the MoLing server.
type MoLingConfig struct {
	ConfigFile string `json:"config_file"` // The path to the configuration file.
	BasePath   string `json:"base_path"`   // The base path for the server, used for storing files. automatically created if not exists. eg: /Users/user1/.moling
	//AllowDir   []string `json:"allow_dir"`   // The directories that are allowed to be accessed by the server.
	Version    string `json:"version"`     // The version of the MoLing server.
	ListenAddr string `json:"listen_addr"` // The address to listen on for SSE mode.
	Debug      bool   `json:"debug"`       // Debug mode, if true, the server will run in debug mode.
	Module     string `json:"module"`      // The module to load, default: all
	Username   string // The username of the user running the server.
	HomeDir    string // The home directory of the user running the server. macOS: /Users/user1, Linux: /home/user1
	SystemInfo string // The system information of the user running the server. macOS: Darwin 15.3.3, Linux: Ubuntu 20.04.1 LTS

	// for MCP Server Config
	Description string // Description of the MCP Server, default: CliDescription
	Command     string //	Command to start the MCP Server, STDIO mode only,  default: CliName
	Args        string // Arguments to pass to the command, STDIO mode only, default: empty
	BaseUrl     string // BaseUrl , SSE mode only.
	ServerName  string // ServerName MCP ServerName, add to the MCP Client config
	logger      zerolog.Logger
}

func (cfg *MoLingConfig) Check() error {
	panic("not implemented yet") // TODO: Implement Check
}

func (cfg *MoLingConfig) Logger() zerolog.Logger {
	return cfg.logger
}

func (cfg *MoLingConfig) SetLogger(logger zerolog.Logger) {
	cfg.logger = logger
}

func (cfg *MoLingConfig) String() string {
	return fmt.Sprintf("ConfigFile: %s, BasePath: %s, Version: %s, ListenAddr: %s, Debug: %t, Module: %s, Username: %s, HomeDir: %s, SystemInfo: %s", cfg.ConfigFile, cfg.BasePath, cfg.Version, cfg.ListenAddr, cfg.Debug, cfg.Module, cfg.Username, cfg.HomeDir, cfg.SystemInfo)
}
