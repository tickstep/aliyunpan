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
	"io"
	"os"
)

type (
	// Fder 获取fd接口
	Fder interface {
		Fd() uintptr
	}

	// Writer 下载器数据输出接口
	Writer interface {
		io.WriterAt
	}
)

// NewDownloaderWriterByFilename 创建下载器数据输出接口, 类似于os.OpenFile
func NewDownloaderWriterByFilename(name string, flag int, perm os.FileMode) (writer Writer, file *os.File, err error) {
	file, err = os.OpenFile(name, flag, perm)
	if err != nil {
		return
	}

	writer = file
	return
}
