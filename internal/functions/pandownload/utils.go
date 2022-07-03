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
package pandownload

import (
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan/internal/localfile"
	"os"
)

// CheckFileValid 检测文件有效性
func CheckFileValid(filePath string, fileInfo *aliyunpan.FileEntity) error {
	// 检查MD5
	// 检查文件大小
	// 检查digest签名
	return nil
}

// FileExist 检查文件是否存在
//
// 只有当文件存在, 文件大小不为0或断点续传文件不存在时, 才判断为存在
func FileExist(path string) bool {
	if info, err := os.Stat(path); err == nil {
		if info.Size() == 0 {
			return false
		}
		if _, err = os.Stat(path + DownloadSuffix); err != nil {
			return true
		}
	}

	return false
}

// SymlinkFileExist 检查文件是否存在
//
// 逻辑和 FileExist 一致，增加符号链接文件的支持
func SymlinkFileExist(fullPath, rootPath string) bool {
	originSaveRootSymlinkFile := localfile.NewSymlinkFile(rootPath)
	suffixPath := localfile.GetSuffixPath(fullPath, rootPath)
	savePathSymlinkFile, savePathFileInfo, err := localfile.RetrieveRealPathFromLogicSuffixPath(originSaveRootSymlinkFile, suffixPath)
	if err != nil {
		return false
	} else {
		if savePathFileInfo.Size() == 0 {
			return false
		}
		if _, err = os.Stat(savePathSymlinkFile.RealPath + DownloadSuffix); err != nil {
			return true
		}
	}
	return false
}
