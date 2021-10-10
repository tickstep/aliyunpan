// Copyright (c) 2020 tickstep.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package config

import (
	"errors"
)

var (
	//ErrNotLogin 未登录帐号错误
	ErrNotLogin = errors.New("user not login")
	//ErrConfigFilePathNotSet 未设置配置文件
	ErrConfigFilePathNotSet = errors.New("config file not set")
	//ErrConfigFileNotExist 未设置Config, 未初始化
	ErrConfigFileNotExist = errors.New("config file not exist")
	//ErrConfigFileNoPermission Config文件无权限访问
	ErrConfigFileNoPermission = errors.New("config file permission denied")
	//ErrConfigContentsParseError 解析Config数据错误
	ErrConfigContentsParseError = errors.New("config contents parse error")
)
