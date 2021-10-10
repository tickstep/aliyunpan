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
package uploader

import (
	"github.com/tickstep/library-go/requester/rio"
	"sync/atomic"
)

type (
	// Readed64 增加获取已读取数据量, 用于统计速度
	Readed64 interface {
		rio.ReaderLen64
		Readed() int64
	}

	readed64 struct {
		readed int64
		rio.ReaderLen64
	}
)

// NewReaded64 实现Readed64接口
func NewReaded64(rl rio.ReaderLen64) Readed64 {
	return &readed64{
		readed:      0,
		ReaderLen64: rl,
	}
}

func (r64 *readed64) Read(p []byte) (n int, err error) {
	n, err = r64.ReaderLen64.Read(p)
	atomic.AddInt64(&r64.readed, int64(n))
	return n, err
}

func (r64 *readed64) Readed() int64 {
	return atomic.LoadInt64(&r64.readed)
}
