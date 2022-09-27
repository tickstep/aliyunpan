package syncdrive

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/aliyunpan/library/requester/transfer"
	"path"
	"strings"
	"time"
)

type (
	// SyncPriorityOption 同步优先级选项
	SyncPriorityOption string

	// ScanStatus 扫描状态
	ScanStatus string

	// PanFileItem 网盘文件信息
	PanFileItem struct {
		// 网盘ID
		DriveId string `json:"driveId"`
		// 域ID
		DomainId string `json:"domainId"`
		// FileId 文件ID
		FileId string `json:"fileId"`
		// FileName 文件名
		FileName string `json:"fileName"`
		// FileSize 文件大小
		FileSize int64 `json:"fileSize"`
		// 文件类别 folder / file
		FileType string `json:"fileType"`
		// 创建时间
		CreatedAt string `json:"createdAt"`
		// 最后修改时间
		UpdatedAt string `json:"updatedAt"`
		// 后缀名，例如：dmg
		FileExtension string `json:"fileExtension"`
		// 文件上传ID
		UploadId string `json:"uploadId"`
		// 父文件夹ID
		ParentFileId string `json:"parentFileId"`
		// 内容CRC64校验值，只有文件才会有
		Crc64Hash string `json:"crc64Hash"`
		// 内容Hash值，只有文件才会有
		Sha1Hash string `json:"sha1Hash"`
		// FilePath 文件的完整路径
		Path string `json:"path"`
		// Category 文件分类，例如：image/video/doc/others
		Category string `json:"category"`
		// ScanTimeAt 扫描时间
		ScanTimeAt string `json:"scanTimeAt"`
		// ScanStatus 扫描状态
		ScanStatus ScanStatus `json:"scanStatus"`
	}
	PanFileList []*PanFileItem

	PanSyncDb interface {
		// Open 打开并准备数据库
		Open() (bool, error)
		// Add 存储一个数据项
		Add(item *PanFileItem) (bool, error)
		// AddFileList 存储批量数据项
		AddFileList(items PanFileList) (bool, error)
		// Get 获取一个数据项
		Get(filePath string) (*PanFileItem, error)
		// GetFileList 获取文件夹下的所有的文件列表
		GetFileList(filePath string) (PanFileList, error)
		// Delete 删除一个数据项，如果是文件夹，则会删除文件夹下面所有的文件列表
		Delete(filePath string) (bool, error)
		// Update 更新一个数据项数据
		Update(item *PanFileItem) (bool, error)
		// Close 关闭数据库
		Close() (bool, error)
	}

	// LocalFileItem 本地文件信息
	LocalFileItem struct {
		// FileName 文件名
		FileName string `json:"fileName"`
		// FileSize 文件大小
		FileSize int64 `json:"fileSize"`
		// 文件类别 folder / file
		FileType string `json:"fileType"`
		// 创建时间
		CreatedAt string `json:"createdAt"`
		// 最后修改时间
		UpdatedAt string `json:"updatedAt"`
		// 后缀名，例如：dmg
		FileExtension string `json:"fileExtension"`
		// 内容Hash值，只有文件才会有
		Sha1Hash string `json:"sha1Hash"`
		// FilePath 文件的完整路径
		Path string `json:"path"`
		// ScanTimeAt 扫描时间
		ScanTimeAt string `json:"scanTimeAt"`
		// ScanStatus 扫描状态
		ScanStatus ScanStatus `json:"scanStatus"`
	}
	LocalFileList []*LocalFileItem

	LocalSyncDb interface {
		// Open 打开并准备数据库
		Open() (bool, error)
		// Add 存储一个数据项
		Add(item *LocalFileItem) (bool, error)
		// AddFileList 存储批量数据项
		AddFileList(items LocalFileList) (bool, error)
		// Get 获取一个数据项
		Get(filePath string) (*LocalFileItem, error)
		// GetFileList 获取文件夹下的所有的文件列表
		GetFileList(filePath string) (LocalFileList, error)
		// Delete 删除一个数据项，如果是文件夹，则会删除文件夹下面所有的文件列表
		Delete(filePath string) (bool, error)
		// Update 更新一个数据项数据
		Update(item *LocalFileItem) (bool, error)
		// Close 关闭数据库
		Close() (bool, error)
	}

	SyncFileAction string
	SyncFileStatus string
	SyncFileItem   struct {
		Action    SyncFileAction `json:"action"`
		Status    SyncFileStatus `json:"status"`
		LocalFile *LocalFileItem `json:"localFile"`
		PanFile   *PanFileItem   `json:"panFile"`
		// LocalFolderPath 本地目录
		LocalFolderPath string `json:"localFolderPath"`
		// PanFolderPath 云盘目录
		PanFolderPath    string `json:"panFolderPath"`
		StatusUpdateTime string `json:"statusUpdateTime"`

		DriveId           string                            `json:"driveId"`
		UseInternalUrl    bool                              `json:"useInternalUrl"`
		DownloadRange     *transfer.Range                   `json:"downloadRange"`
		DownloadBlockSize int64                             `json:"downloadBlockSize"`
		UploadRange       *transfer.Range                   `json:"uploadRange"`
		UploadEntity      *aliyunpan.CreateFileUploadResult `json:"uploadEntity"`
		// UploadPartSeq 上传序号，从0开始
		UploadPartSeq   int   `json:"uploadPartSeq"`
		UploadBlockSize int64 `json:"uploadBlockSize"`
	}
	SyncFileList []*SyncFileItem

	SyncFileDb interface {
		// Open 打开并准备数据库
		Open() (bool, error)
		// Add 存储一个数据项
		Add(item *SyncFileItem) (bool, error)
		// Get 获取一个数据项
		Get(id string) (*SyncFileItem, error)
		// GetFileList 获取文件夹下的所有的文件列表
		GetFileList(Status SyncFileStatus) (SyncFileList, error)
		// Delete 删除一个数据项，如果是文件夹，则会删除文件夹下面所有的文件列表
		Delete(id string) (bool, error)
		// Update 更新一个数据项数据
		Update(item *SyncFileItem) (bool, error)
		// Close 关闭数据库
		Close() (bool, error)
	}
)

const (
	SyncFileStatusCreate      SyncFileStatus = "create"
	SyncFileStatusUploading   SyncFileStatus = "uploading"
	SyncFileStatusDownloading SyncFileStatus = "downloading"
	SyncFileStatusFailed      SyncFileStatus = "failed"
	SyncFileStatusSuccess     SyncFileStatus = "success"
	SyncFileStatusIllegal     SyncFileStatus = "illegal"
	SyncFileStatusNotExisted  SyncFileStatus = "notExisted"

	SyncFileActionDownload          SyncFileAction = "download"
	SyncFileActionUpload            SyncFileAction = "upload"
	SyncFileActionDeleteLocal       SyncFileAction = "delete_local"
	SyncFileActionDeletePan         SyncFileAction = "delete_pan"
	SyncFileActionCreateLocalFolder SyncFileAction = "create_local_folder"
	SyncFileActionCreatePanFolder   SyncFileAction = "create_pan_folder"

	// ScanStatusNormal 正常
	ScanStatusNormal ScanStatus = "normal"
	// ScanStatusDiscard 已过期，已删除
	ScanStatusDiscard ScanStatus = "discard"

	// SyncPriorityTimestampFirst 最新时间优先
	SyncPriorityTimestampFirst = "time"
	// SyncPriorityLocalFirst 本地文件优先
	SyncPriorityLocalFirst = "local"
	// SyncPriorityPanFirst 网盘文件优先
	SyncPriorityPanFirst = "pan"
)

var (
	ErrItemNotExisted error = fmt.Errorf("item is not existed")
)

func NewPanFileItem(fe *aliyunpan.FileEntity) *PanFileItem {
	return &PanFileItem{
		DriveId:       fe.DriveId,
		DomainId:      fe.DomainId,
		FileId:        fe.FileId,
		FileName:      fe.FileName,
		FileSize:      fe.FileSize,
		FileType:      fe.FileType,
		CreatedAt:     fe.CreatedAt,
		UpdatedAt:     fe.UpdatedAt,
		FileExtension: fe.FileExtension,
		UploadId:      fe.UploadId,
		ParentFileId:  fe.ParentFileId,
		Crc64Hash:     fe.Crc64Hash,
		Sha1Hash:      fe.ContentHash,
		Path:          fe.Path,
		Category:      fe.Category,
		ScanStatus:    ScanStatusNormal,
	}
}

func (item *PanFileItem) Id() string {
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "%s%s", strings.ReplaceAll(item.Path, "\\", "/"), item.UpdatedAt)
	return utils.Md5Str(sb.String())
}

func (item *PanFileItem) FormatFileName() string {
	return item.FileName
}

func (item *PanFileItem) FormatFilePath() string {
	return FormatFilePath(item.Path)
}

func (item *PanFileItem) IsFolder() bool {
	return item.FileType == "folder"
}

func (item *PanFileItem) UpdateTimeUnix() int64 {
	return item.UpdateTime().Unix()
}

func (item *PanFileItem) UpdateTime() time.Time {
	return utils.ParseTimeStr(item.UpdatedAt)
}

func (item *PanFileItem) HashCode() string {
	return item.Path
}

func (item *PanFileItem) ScanTimeUnix() int64 {
	return item.ScanTime().Unix()
}

func (item *PanFileItem) ScanTime() time.Time {
	return utils.ParseTimeStr(item.ScanTimeAt)
}

func NewPanSyncDb(dbFilePath string) PanSyncDb {
	return interface{}(newPanSyncDbBolt(dbFilePath)).(PanSyncDb)
}

func (item *LocalFileItem) Id() string {
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "%s%s", strings.ReplaceAll(item.Path, "\\", "/"), item.UpdatedAt)
	return utils.Md5Str(sb.String())
}

func (item *LocalFileItem) FormatFileName() string {
	return item.FileName
}

func (item *LocalFileItem) FormatFilePath() string {
	return FormatFilePath(item.Path)
}

func (item *LocalFileItem) IsFolder() bool {
	return item.FileType == "folder"
}

func (item *LocalFileItem) IsFile() bool {
	return item.FileType == "file"
}

func (item *LocalFileItem) UpdateTimeUnix() int64 {
	return item.UpdateTime().Unix()
}

func (item *LocalFileItem) UpdateTime() time.Time {
	return utils.ParseTimeStr(item.UpdatedAt)
}

func (item *LocalFileItem) ScanTimeUnix() int64 {
	return item.ScanTime().Unix()
}

func (item *LocalFileItem) ScanTime() time.Time {
	return utils.ParseTimeStr(item.ScanTimeAt)
}

func (item *LocalFileItem) HashCode() string {
	return item.Path
}

func NewLocalSyncDb(dbFilePath string) LocalSyncDb {
	return interface{}(newLocalSyncDbBolt(dbFilePath)).(LocalSyncDb)
}

func (l LocalFileList) FindFileByPath(filePath string) *LocalFileItem {
	for _, item := range l {
		if filePath == item.Path {
			return item
		}
	}
	return nil
}

func (p PanFileList) FindFileByPath(filePath string) *PanFileItem {
	for _, item := range p {
		if strings.ReplaceAll(filePath, "\\", "/") == item.Path {
			return item
		}
	}
	return nil
}

func (item *SyncFileItem) Id() string {
	sb := &strings.Builder{}
	if item.Action == SyncFileActionDownload || item.Action == SyncFileActionDeleteLocal || item.Action == SyncFileActionCreateLocalFolder {
		fmt.Fprintf(sb, "%s%s", string(item.Action), item.PanFile.Id())
	} else if item.Action == SyncFileActionUpload || item.Action == SyncFileActionDeletePan || item.Action == SyncFileActionCreatePanFolder {
		fmt.Fprintf(sb, "%s%s", string(item.Action), item.LocalFile.Id())
	}
	return utils.Md5Str(sb.String())
}

func (item *SyncFileItem) StatusUpdateTimeUnix() int64 {
	if ts, er := time.Parse("2006-01-02 15:04:05", item.StatusUpdateTime); er != nil {
		return ts.Unix()
	}
	return 0
}

// getPanFullPath 获取网盘文件的路径
func (item *SyncFileItem) getPanFileFullPath() string {
	if item.PanFile != nil {
		return item.PanFile.Path
	}
	localPath := item.LocalFile.Path
	localPath = strings.ReplaceAll(localPath, "\\", "/")
	localRootPath := strings.ReplaceAll(item.LocalFolderPath, "\\", "/")

	relativePath := strings.TrimPrefix(localPath, localRootPath)
	return path.Join(path.Clean(item.PanFolderPath), relativePath)
}

// getLocalFullPath 获取本地文件的路径
func (item *SyncFileItem) getLocalFileFullPath() string {
	if item.LocalFile != nil {
		return item.LocalFile.Path
	}
	panPath := item.PanFile.Path
	panPath = strings.ReplaceAll(panPath, "\\", "/")
	panRootPath := strings.ReplaceAll(item.PanFolderPath, "\\", "/")

	relativePath := strings.TrimPrefix(panPath, panRootPath)
	return path.Join(path.Clean(item.LocalFolderPath), relativePath)
}

// getLocalFileDownloadingFullPath 获取本地文件下载时的路径
func (item *SyncFileItem) getLocalFileDownloadingFullPath() string {
	return item.getLocalFileFullPath() + DownloadingFileSuffix
}

func (item *SyncFileItem) String() string {
	sb := &strings.Builder{}
	fp := ""
	if item.Action == SyncFileActionDownload {
		fp = item.PanFile.Path
	} else if item.Action == SyncFileActionUpload {
		fp = item.LocalFile.Path
	}
	fmt.Fprintf(sb, "ID:%s\nAction:%s\nStatus:%s\nPath:%s\n",
		item.Id(), item.Action, item.Status, fp)
	return sb.String()
}

func (item *SyncFileItem) HashCode() string {
	return item.Id()
}

func NewSyncFileDb(dbFilePath string) SyncFileDb {
	return interface{}(newSyncFileDbBolt(dbFilePath)).(SyncFileDb)
}
