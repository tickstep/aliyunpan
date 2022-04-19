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
package plugin

type (
	// Context 插件回调函数上下文信息
	Context struct {
		AppName      string `json:"appName"`
		Version      string `json:"version"`
		UserId       string `json:"userId"`
		Nickname     string `json:"nickname"`
		FileDriveId  string `json:"fileDriveId"`
		AlbumDriveId string `json:"albumDriveId"`
	}

	// UploadFilePrepareParams 上传文件前的回调函数-参数
	UploadFilePrepareParams struct {
		LocalFilePath      string `json:"localFilePath"`
		LocalFileName      string `json:"localFileName"`
		LocalFileSize      int    `json:"localFileSize"`
		LocalFileType      string `json:"localFileType"`
		LocalFileUpdatedAt string `json:"localFileUpdatedAt"`
		DriveId            string `json:"driveId"`
		DriveFilePath      string `json:"driveFilePath"`
	}

	// UploadFilePrepareResult 上传文件前的回调函数-返回结果
	UploadFilePrepareResult struct {
		// UploadApproved 确认该文件是否上传。yes-上传 no-不上传
		UploadApproved string `json:"uploadApproved"`
		// DriveFilePath 保存网盘的修改后的路径。注意该路径是相对路径
		DriveFilePath string `json:"driveFilePath"`
	}

	// Plugin 插件接口
	Plugin interface {
		// Start 启动
		Start() error

		// UploadFilePrepareCallback 上传文件前的回调函数
		UploadFilePrepareCallback(context *Context, params *UploadFilePrepareParams) (*UploadFilePrepareResult, error)

		// Stop 停止
		Stop() error
	}
)

var ()
