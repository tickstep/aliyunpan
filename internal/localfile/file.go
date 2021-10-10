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
package localfile

import (
	"os"
	"path/filepath"
	"strings"
)

// EqualLengthMD5 检测md5和大小是否相同
func (lfm *LocalFileMeta) EqualLengthMD5(m *LocalFileMeta) bool {
	if lfm.Length != m.Length {
		return false
	}
	if lfm.MD5 != m.MD5 {
		return false
	}
	return true
}

// EqualLengthSHA1 检测sha1和大小是否相同
func (lfm *LocalFileMeta) EqualLengthSHA1(m *LocalFileMeta) bool {
	if lfm.Length != m.Length {
		return false
	}
	if lfm.SHA1 != m.SHA1 {
		return false
	}
	return true
}

// CompleteAbsPath 补齐绝对路径
func (lfm *LocalFileMeta) CompleteAbsPath() {
	if filepath.IsAbs(lfm.Path) {
		return
	}

	absPath, err := filepath.Abs(lfm.Path)
	if err != nil {
		return
	}
	// windows
	if os.PathSeparator == '\\' {
		absPath = strings.ReplaceAll(absPath, "\\", "/")
	}
	lfm.Path = absPath
}

// GetFileSum 获取文件的大小, md5, crc32
func GetFileSum(localPath string, flag int) (lfc *LocalFileEntity, err error) {
	lfc = NewLocalFileEntity(localPath)
	defer lfc.Close()

	err = lfc.OpenPath()
	if err != nil {
		return nil, err
	}

	err = lfc.Sum(flag)
	if err != nil {
		return nil, err
	}
	return lfc, nil
}
