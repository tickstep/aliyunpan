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
package plugins

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
		LocalFileSize      int64  `json:"localFileSize"`
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

	// UploadFileFinishParams 上传文件结束的回调函数-参数
	UploadFileFinishParams struct {
		LocalFilePath      string `json:"localFilePath"`
		LocalFileName      string `json:"localFileName"`
		LocalFileSize      int64  `json:"localFileSize"`
		LocalFileType      string `json:"localFileType"`
		LocalFileUpdatedAt string `json:"localFileUpdatedAt"`
		LocalFileSha1      string `json:"localFileSha1"`
		UploadResult       string `json:"uploadResult"`
		DriveId            string `json:"driveId"`
		DriveFilePath      string `json:"driveFilePath"`
	}

	// DownloadFilePrepareParams 下载文件前的回调函数-参数
	DownloadFilePrepareParams struct {
		DriveId            string `json:"driveId"`
		DriveFileName      string `json:"driveFileName"`
		DriveFilePath      string `json:"driveFilePath"`
		DriveFileSha1      string `json:"driveFileSha1"`
		DriveFileSize      int64  `json:"driveFileSize"`
		DriveFileType      string `json:"driveFileType"`
		DriveFileUpdatedAt string `json:"driveFileUpdatedAt"`
		LocalFilePath      string `json:"localFilePath"`
	}

	// DownloadFilePrepareResult 上传文件前的回调函数-返回结果
	DownloadFilePrepareResult struct {
		// DownloadApproved 确认该文件是否下载。yes-下载 no-不下载
		DownloadApproved string `json:"downloadApproved"`
		// LocalFilePath 保存本地的修改后的路径。注意该路径是相对路径
		LocalFilePath string `json:"localFilePath"`
	}

	// DownloadFileFinishParams 下载文件结束的回调函数-参数
	DownloadFileFinishParams struct {
		DriveId            string `json:"driveId"`
		DriveFileName      string `json:"driveFileName"`
		DriveFilePath      string `json:"driveFilePath"`
		DriveFileSha1      string `json:"driveFileSha1"`
		DriveFileSize      int64  `json:"driveFileSize"`
		DriveFileType      string `json:"driveFileType"`
		DriveFileUpdatedAt string `json:"driveFileUpdatedAt"`
		DownloadResult     string `json:"downloadResult"`
		LocalFilePath      string `json:"localFilePath"`
	}

	// SyncScanLocalFilePrepareParams 同步备份-扫描本地文件前参数
	SyncScanLocalFilePrepareParams struct {
		LocalFilePath      string `json:"localFilePath"`
		LocalFileName      string `json:"localFileName"`
		LocalFileSize      int64  `json:"localFileSize"`
		LocalFileType      string `json:"localFileType"`
		LocalFileUpdatedAt string `json:"localFileUpdatedAt"`
		DriveId            string `json:"driveId"`
	}

	// SyncScanLocalFilePrepareResult 同步备份-扫描本地文件-返回结果
	SyncScanLocalFilePrepareResult struct {
		// SyncScanLocalApproved 该文件是否确认扫描，yes-允许扫描，no-禁止扫描
		SyncScanLocalApproved string `json:"syncScanLocalApproved"`
	}

	// SyncScanPanFilePrepareParams 同步备份-扫描云盘文件前参数
	SyncScanPanFilePrepareParams struct {
		DriveId            string `json:"driveId"`
		DriveFileName      string `json:"driveFileName"`
		DriveFilePath      string `json:"driveFilePath"`
		DriveFileSha1      string `json:"driveFileSha1"`
		DriveFileSize      int64  `json:"driveFileSize"`
		DriveFileType      string `json:"driveFileType"`
		DriveFileUpdatedAt string `json:"driveFileUpdatedAt"`
	}

	// SyncScanPanFilePrepareResult 同步备份-扫描云盘文件-返回结果
	SyncScanPanFilePrepareResult struct {
		// SyncScanPanApproved 该文件是否确认扫描，yes-允许扫描，no-禁止扫描
		SyncScanPanApproved string `json:"syncScanPanApproved"`
	}

	// Plugin 插件接口
	Plugin interface {
		// Start 启动
		Start() error

		// UploadFilePrepareCallback 上传文件前的回调函数
		UploadFilePrepareCallback(context *Context, params *UploadFilePrepareParams) (*UploadFilePrepareResult, error)

		// UploadFileFinishCallback 上传文件结束的回调函数
		UploadFileFinishCallback(context *Context, params *UploadFileFinishParams) error

		// DownloadFilePrepareCallback 下载文件前的回调函数
		DownloadFilePrepareCallback(context *Context, params *DownloadFilePrepareParams) (*DownloadFilePrepareResult, error)

		// DownloadFileFinishCallback 下载文件结束的回调函数
		DownloadFileFinishCallback(context *Context, params *DownloadFileFinishParams) error

		// SyncScanLocalFilePrepareCallback 同步备份-扫描本地文件的回调函数
		SyncScanLocalFilePrepareCallback(context *Context, params *SyncScanLocalFilePrepareParams) (*SyncScanLocalFilePrepareResult, error)

		// SyncScanPanFilePrepareCallback 同步备份-扫描本地文件的回调函数
		SyncScanPanFilePrepareCallback(context *Context, params *SyncScanPanFilePrepareParams) (*SyncScanPanFilePrepareResult, error)

		// Stop 停止
		Stop() error
	}
)

var ()
