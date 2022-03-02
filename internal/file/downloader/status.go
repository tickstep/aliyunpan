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

type (
	//WorkerStatuser 状态
	WorkerStatuser interface {
		StatusCode() StatusCode //状态码
		StatusText() string
	}

	//StatusCode 状态码
	StatusCode int

	//WorkerStatus worker状态
	WorkerStatus struct {
		statusCode StatusCode
	}

	// DownloadStatusFunc 下载状态处理函数
	DownloadStatusFunc func(status transfer.DownloadStatuser, workersCallback func(RangeWorkerFunc))
)

const (
	//StatusCodeInit 初始化
	StatusCodeInit StatusCode = iota
	//StatusCodeSuccessed 成功
	StatusCodeSuccessed
	//StatusCodePending 等待响应
	StatusCodePending
	//StatusCodeDownloading 下载中
	StatusCodeDownloading
	//StatusCodeWaitToWrite 等待写入数据
	StatusCodeWaitToWrite
	//StatusCodeInternalError 内部错误
	StatusCodeInternalError
	//StatusCodeTooManyConnections 连接数太多
	StatusCodeTooManyConnections
	//StatusCodeNetError 网络错误
	StatusCodeNetError
	//StatusCodeFailed 下载失败
	StatusCodeFailed
	//StatusCodePaused 已暂停
	StatusCodePaused
	//StatusCodeReseted 已重设连接
	StatusCodeReseted
	//StatusCodeCanceled 已取消
	StatusCodeCanceled
	//StatusCodeDownloadUrlExpired 下载链接已过期
	StatusCodeDownloadUrlExpired
	//StatusCodeIllegalDownloadFile 文件非法，不允许下载
	StatusCodeIllegalDownloadFile
)

//GetStatusText 根据状态码获取状态信息
func GetStatusText(sc StatusCode) string {
	switch sc {
	case StatusCodeInit:
		return "初始化"
	case StatusCodeSuccessed:
		return "成功"
	case StatusCodePending:
		return "等待响应"
	case StatusCodeDownloading:
		return "下载中"
	case StatusCodeWaitToWrite:
		return "等待写入数据"
	case StatusCodeInternalError:
		return "内部错误"
	case StatusCodeTooManyConnections:
		return "连接数太多"
	case StatusCodeNetError:
		return "网络错误"
	case StatusCodeFailed:
		return "下载失败"
	case StatusCodePaused:
		return "已暂停"
	case StatusCodeReseted:
		return "已重设连接"
	case StatusCodeCanceled:
		return "已取消"
	default:
		return "未知状态码"
	}
}

//NewWorkerStatus 初始化WorkerStatus
func NewWorkerStatus() *WorkerStatus {
	return &WorkerStatus{
		statusCode: StatusCodeInit,
	}
}

//SetStatusCode 设置worker状态码
func (ws *WorkerStatus) SetStatusCode(sc StatusCode) {
	ws.statusCode = sc
}

//StatusCode 返回状态码
func (ws *WorkerStatus) StatusCode() StatusCode {
	return ws.statusCode
}

//StatusText 返回状态信息
func (ws *WorkerStatus) StatusText() string {
	return GetStatusText(ws.statusCode)
}
