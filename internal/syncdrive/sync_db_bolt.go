package syncdrive

import (
	"encoding/json"
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/bolt"
	"strings"
	"sync"
	"time"
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

	// PanSyncDb 存储网盘文件信息的数据库
	PanSyncDb struct {
		Path   string
		db     *bolt.DB
		locker *sync.Mutex
	}

	// LocalFileItem 本地文件信息
	LocalFileItem struct {
	}

	// LocalSyncDb 存储本地文件信息的数据库
	LocalSyncDb struct {
		Path   string
		db     *bolt.DB
		locker *sync.Mutex
	}
)

const (
	DefaultDirKeyName string = "."
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

func NewPanSyncDb(dbFilePath string) *PanSyncDb {
	return &PanSyncDb{
		Path:   dbFilePath,
		locker: &sync.Mutex{},
	}
}

func (p *PanSyncDb) Open() (bool, error) {
	db, err := bolt.Open(p.Path, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return false, err
	}
	p.db = db
	return true, nil
}

func (p *PanSyncDb) Add(item *PanFileItem) (bool, error) {
	if item == nil {
		return false, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	// add item
	// Start a writable transaction.
	tx, err := p.db.Begin(true)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	parts := strings.Split(item.FormatFilePath(), "/")
	bkt, er := tx.CreateBucketIfNotExists([]byte("/"))
	if er != nil {
		return false, er
	}
	for _, p := range parts[:len(parts)-1] {
		if p == "" {
			continue
		}
		bkt, _ = bkt.CreateBucketIfNotExists([]byte(p))
	}
	if bkt == nil {
		return false, fmt.Errorf("create or get bucket error")
	}

	rs, err := json.Marshal(item)
	if err != nil {
		return false, err
	}
	if item.IsFolder() {
		bkt, err = bkt.CreateBucketIfNotExists([]byte(item.FormatFileName()))
		if err != nil {
			return false, err
		}
		if e := bkt.Put([]byte(DefaultDirKeyName), rs); e != nil {
			return false, e
		}
	} else {
		if e := bkt.Put([]byte(item.FormatFileName()), rs); e != nil {
			return false, e
		}
	}

	// Commit the transaction and check for error.
	if err := tx.Commit(); err != nil {
		return false, err
	}

	return true, nil
}
func (p *PanSyncDb) Get(filePath string) (*PanFileItem, error) {
	filePath = FormatFilePath(filePath)
	if filePath == "" {
		return nil, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	tx, err := p.db.Begin(false)
	if err != nil {
		return nil, err
	}

	partsOrg := strings.Split(filePath, "/")
	parts := []string{}
	for _, p := range partsOrg {
		if p == "" {
			continue
		}
		parts = append(parts, p)
	}

	bkt := tx.Bucket([]byte("/"))
	if bkt == nil {
		return nil, fmt.Errorf("item is not existed")
	}
	for _, p := range parts[:len(parts)-1] {
		bkt = bkt.Bucket([]byte(p))
		if bkt == nil {
			return nil, fmt.Errorf("item is not existed")
		}
	}
	if bkt == nil {
		return nil, fmt.Errorf("item is not existed")
	}

	dirBucket := bkt.Bucket([]byte(parts[len(parts)-1]))
	var data []byte
	if dirBucket != nil {
		// is dir
		data = dirBucket.Get([]byte(DefaultDirKeyName))
	} else {
		data = bkt.Get([]byte(parts[len(parts)-1]))
	}
	if data != nil {
		item := &PanFileItem{}
		if err := json.Unmarshal(data, item); err != nil {
			return nil, err
		}
		return item, nil
	}
	return nil, nil
}
func (p *PanSyncDb) Delete(filePath string) (bool, error) {
	return false, nil
}
func (p *PanSyncDb) Update(filePath string) (bool, error) {
	return false, nil
}
func (p *PanSyncDb) Close() (bool, error) {
	if p.db != nil {
		if e := p.db.Close(); e != nil {
			return false, e
		}
	}
	return true, nil
}
