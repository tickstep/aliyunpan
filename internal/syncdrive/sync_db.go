package syncdrive

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
)

type (
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
	}
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

func NewPanSyncDb(dbFilePath string) PanSyncDb {
	return interface{}(newPanSyncDbBolt(dbFilePath)).(PanSyncDb)
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

func NewLocalSyncDb(dbFilePath string) LocalSyncDb {
	return interface{}(newLocalSyncDbBolt(dbFilePath)).(LocalSyncDb)
}
