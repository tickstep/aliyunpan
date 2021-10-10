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

import "errors"

var (
	// ErrDownloadNotSupportChecksum 文件不支持校验
	ErrDownloadNotSupportChecksum = errors.New("该文件不支持校验")
	// ErrDownloadChecksumFailed 文件校验失败
	ErrDownloadChecksumFailed = errors.New("该文件校验失败, 文件md5值与服务器记录的不匹配")
	// ErrDownloadFileBanned 违规文件
	ErrDownloadFileBanned = errors.New("该文件可能是违规文件, 不支持校验")
	// ErrDlinkNotFound 未取得下载链接
	ErrDlinkNotFound = errors.New("未取得下载链接")
	// ErrShareInfoNotFound 未在已分享列表中找到分享信息
	ErrShareInfoNotFound = errors.New("未在已分享列表中找到分享信息")
)
