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
)

var serviceLists = make(map[comm.MoLingServerType]ServiceFactory)

// RegisterServ register service
func RegisterServ(n comm.MoLingServerType, f ServiceFactory) {
	//serviceLists = append(, f)
	serviceLists[n] = f
}

// ServiceList  get service lists
func ServiceList() map[comm.MoLingServerType]ServiceFactory {
	return serviceLists
}
