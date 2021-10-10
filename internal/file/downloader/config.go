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
package downloader

import (
	"github.com/tickstep/aliyunpan/library/requester/transfer"
)

const (
	//CacheSize 默认的下载缓存
	CacheSize = 8192
)

var (
	// MinParallelSize 单个线程最小的数据量
	MinParallelSize int64 = 128 * 1024 // 128kb
)

//Config 下载配置
type Config struct {
	Mode                       transfer.RangeGenMode      // 下载Range分配模式
	MaxParallel                int                        // 最大下载并发量
	CacheSize                  int                        // 下载缓冲
	BlockSize                  int64                      // 每个Range区块的大小, RangeGenMode 为 RangeGenMode2 时才有效
	MaxRate                    int64                      // 限制最大下载速度
	InstanceStateStorageFormat InstanceStateStorageFormat // 断点续传储存类型
	InstanceStatePath          string                     // 断点续传信息路径
	TryHTTP                    bool                       // 是否尝试使用 http 连接
	ShowProgress               bool                       // 是否展示下载进度条
}

//NewConfig 返回默认配置
func NewConfig() *Config {
	return &Config{
		MaxParallel: 5,
		CacheSize:   CacheSize,
	}
}

//Fix 修复配置信息, 使其合法
func (cfg *Config) Fix() {
	fixCacheSize(&cfg.CacheSize)
	if cfg.MaxParallel < 1 {
		cfg.MaxParallel = 1
	}
}

//Copy 拷贝新的配置
func (cfg *Config) Copy() *Config {
	newCfg := *cfg
	return &newCfg
}
