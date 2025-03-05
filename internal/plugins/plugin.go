// Copyright (c) 2020 tickstep.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
		DriveFileId        string `json:"driveFileId"`
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

	// SyncFileFinishParams 同步备份-一个文件同步结束-回调参数
	SyncFileFinishParams struct {
		Action        string `json:"action"`
		ActionResult  string `json:"actionResult"`
		DriveId       string `json:"driveId"`
		DriveFileId   string `json:"driveFileId"`
		FileName      string `json:"fileName"`
		FilePath      string `json:"filePath"`
		FileSha1      string `json:"fileSha1"`
		FileSize      int64  `json:"fileSize"`
		FileType      string `json:"fileType"`
		FileUpdatedAt string `json:"fileUpdatedAt"`
	}

	// SyncAllFileFinishParams 同步备份-全部文件同步结束-回调参数
	SyncAllFileFinishParams struct {
		// Name 任务名称
		Name string `json:"name"`
		// Id 任务ID
		Id string `json:"id"`
		// UserId 账号ID
		UserId string `json:"userId"`
		// DriveName 网盘名称，backup-备份盘，resource-资源盘
		DriveName string `json:"driveName"`
		// DriveId 网盘ID，目前支持文件网盘
		DriveId string `json:"driveId"`
		// LocalFolderPath 本地目录
		LocalFolderPath string `json:"localFolderPath"`
		// PanFolderPath 云盘目录
		PanFolderPath string `json:"panFolderPath"`
		// Mode 备份模式
		Mode string `json:"mode"`
		// Policy 备份策略
		Policy string `json:"policy"`
	}

	// UserTokenRefreshFinishParams 用户Token刷新完成后回调函数
	UserTokenRefreshFinishParams struct {
		Result    string `json:"result"`
		Message   string `json:"message"`
		OldToken  string `json:"oldToken"`
		NewToken  string `json:"newToken"`
		UpdatedAt string `json:"updatedAt"`
	}

	// RemoveFilePrepareParams 删除文件前的回调函数-参数
	RemoveFilePrepareParams struct {
		Count int                      `json:"count"`
		Items []*RemoveFilePrepareItem `json:"items"`
	}
	RemoveFilePrepareItem struct {
		// DriveId 网盘ID
		DriveId string `json:"driveId"`
		// DriveFileId 网盘文件的ID
		DriveFileId string `json:"driveFileId"`
		// DriveFileName 网盘文件名
		DriveFileName string `json:"driveFileName"`
		// DriveFilePath 网盘文件路径
		DriveFilePath string `json:"driveFilePath"`
		// DriveFileSize 网盘文件大小
		DriveFileSize int64 `json:"driveFileSize"`
		// DriveFileType 网盘文件类型，file-文件，folder-文件夹
		DriveFileType string `json:"driveFileType"`
		// DriveFileUpdatedAt 网盘文件修改时间，格式：2025-03-03 10:39:14
		DriveFileUpdatedAt string `json:"driveFileUpdatedAt"`
		// DriveFileCreatedAt 网盘文件创建时间，格式：2025-03-03 10:39:14
		DriveFileCreatedAt string `json:"driveFileCreatedAt"`
	}
	// RemoveFilePrepareResult 删除文件前的回调函数-返回结果
	RemoveFilePrepareResult struct {
		Result []*RemoveFilePrepareResultItem `json:"result"`
	}
	RemoveFilePrepareResultItem struct {
		// DriveId 网盘ID
		DriveId string `json:"driveId"`
		// DriveFileId 网盘文件的ID
		DriveFileId string `json:"driveFileId"`
		// RemoveApproved 确认该文件是否删除。yes-删除 no-不删除
		RemoveApproved string `json:"removeApproved"`
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

		// SyncScanPanFilePrepareCallback 同步备份-扫描云盘文件的回调函数
		SyncScanPanFilePrepareCallback(context *Context, params *SyncScanPanFilePrepareParams) (*SyncScanPanFilePrepareResult, error)

		// SyncFileFinishCallback 同步备份-同步一个文件完成时的回调函数
		SyncFileFinishCallback(context *Context, params *SyncFileFinishParams) error

		// SyncAllFileFinishCallback 同步备份-同步全部文件完成时的回调函数
		SyncAllFileFinishCallback(context *Context, params *SyncAllFileFinishParams) error

		// UserTokenRefreshFinishCallback 用户Token刷新完成后回调函数
		UserTokenRefreshFinishCallback(context *Context, params *UserTokenRefreshFinishParams) error

		// RemoveFilePrepareCallback 删除文件前的回调函数
		RemoveFilePrepareCallback(context *Context, params *RemoveFilePrepareParams) (*RemoveFilePrepareResult, error)

		// Stop 停止
		Stop() error
	}
)
