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

package services

import (
	"github.com/gojue/moling/pkg/comm"
	"github.com/gojue/moling/pkg/services/abstract"
	"github.com/gojue/moling/pkg/services/browser"
	"github.com/gojue/moling/pkg/services/command"
	"github.com/gojue/moling/pkg/services/filesystem"
)

var serviceLists = make(map[comm.MoLingServerType]abstract.ServiceFactory)

// RegisterServ register service
func RegisterServ(n comm.MoLingServerType, f abstract.ServiceFactory) {
	//serviceLists = append(, f)
	serviceLists[n] = f
}

// ServiceList  get service lists
func ServiceList() map[comm.MoLingServerType]abstract.ServiceFactory {
	return serviceLists
}

func init() {
	// 浏览器操作工具
	RegisterServ(browser.BrowserServerName, browser.NewBrowserServer)
	// 命令行操作工具
	RegisterServ(command.CommandServerName, command.NewCommandServer)
	// 文件系统操作工具
	RegisterServ(filesystem.FilesystemServerName, filesystem.NewFilesystemServer)
}
