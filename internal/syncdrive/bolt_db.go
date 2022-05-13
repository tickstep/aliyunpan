package syncdrive

import (
	"fmt"
	"github.com/tickstep/bolt"
	"path"
	"strings"
	"sync"
	"time"
)

type (
	// BoltDb 存储本地文件信息的数据库
	BoltDb struct {
		Path   string
		db     *bolt.DB
		locker *sync.Mutex
	}
)

func NewBoltDb(dbFilePath string) *BoltDb {
	return &BoltDb{
		Path:   dbFilePath,
		locker: &sync.Mutex{},
	}
}

func (p *BoltDb) Open() (bool, error) {
	db, err := bolt.Open(p.Path, 0600, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return false, err
	}
	p.db = db
	return true, nil
}

// Add 增加一个数据项
func (p *BoltDb) Add(filePath string, isFolder bool, data string) (bool, error) {
	filePath = FormatFilePath(filePath)
	p.locker.Lock()
	defer p.locker.Unlock()

	// add item
	// Start a writable transaction.
	tx, err := p.db.Begin(true)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	parts := strings.Split(filePath, "/")
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

	fileName := path.Base(filePath)
	if isFolder {
		bkt, err = bkt.CreateBucketIfNotExists([]byte(fileName))
		if err != nil {
			return false, err
		}
		if e := bkt.Put([]byte(DefaultDirKeyName), []byte(data)); e != nil {
			return false, e
		}
	} else {
		if e := bkt.Put([]byte(fileName), []byte(data)); e != nil {
			return false, e
		}
	}

	// Commit the transaction and check for error.
	if err := tx.Commit(); err != nil {
		return false, err
	}

	return true, nil
}

// Get 获取一个数据项，数据项不存在返回错误
func (p *BoltDb) Get(filePath string) (string, error) {
	filePath = FormatFilePath(filePath)
	if filePath == "" {
		return "", fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	tx, err := p.db.Begin(false)
	if err != nil {
		return "", err
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
		return "", ErrItemNotExisted
	}
	for _, p := range parts[:len(parts)-1] {
		bkt = bkt.Bucket([]byte(p))
		if bkt == nil {
			return "", ErrItemNotExisted
		}
	}
	if bkt == nil {
		return "", ErrItemNotExisted
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
		return string(data), nil
	}
	return "", ErrItemNotExisted
}

func (p *BoltDb) GetFileList(filePath string) ([]string, error) {
	dataList := []string{}
	filePath = FormatFilePath(filePath)
	if filePath == "" {
		return dataList, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	tx, err := p.db.Begin(false)
	if err != nil {
		return dataList, err
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
		return dataList, ErrItemNotExisted
	}
	for _, p := range parts[:len(parts)-1] {
		bkt = bkt.Bucket([]byte(p))
		if bkt == nil {
			return dataList, ErrItemNotExisted
		}
	}
	if bkt == nil {
		return dataList, ErrItemNotExisted
	}

	dirBucket := bkt.Bucket([]byte(parts[len(parts)-1]))
	if dirBucket == nil {
		return dataList, ErrItemNotExisted
	}

	c := dirBucket.Cursor()
	for k, v := c.First(); k != nil; k, v = c.Next() {
		if string(k) == DefaultDirKeyName {
			continue
		}
		dataList = append(dataList, string(v))
	}
	return dataList, nil
}

// Delete 删除一个数据项
func (p *BoltDb) Delete(filePath string) (bool, error) {
	filePath = FormatFilePath(filePath)
	if filePath == "" {
		return false, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	tx, err := p.db.Begin(true)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	partsOrg := strings.Split(filePath, "/")
	parts := []string{}
	for _, p := range partsOrg {
		if p == "" {
			continue
		}
		parts = append(parts, p)
	}

	// get parent node
	bkt := tx.Bucket([]byte("/"))
	if bkt == nil {
		return false, ErrItemNotExisted
	}
	for _, p := range parts[:len(parts)-1] {
		bkt = bkt.Bucket([]byte(p))
		if bkt == nil {
			return false, ErrItemNotExisted
		}
	}
	if bkt == nil {
		return false, ErrItemNotExisted
	}

	targetName := []byte(parts[len(parts)-1])
	dirBucket := bkt.Bucket(targetName)
	var er error
	if dirBucket != nil {
		// is dir, delete bucket
		er = bkt.DeleteBucket(targetName)
	} else {
		// is file, delete item
		er = bkt.Delete(targetName)
	}
	// Commit the transaction and check for error.
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return er == nil, nil
}

// Update 更新数据项，数据项不存在返回错误
func (p *BoltDb) Update(filePath string, isFolder bool, data string) (bool, error) {
	filePath = FormatFilePath(filePath)
	p.locker.Lock()
	defer p.locker.Unlock()

	// update item
	// Start a writable transaction.
	tx, err := p.db.Begin(true)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	// get parent bucket of update item
	parts := strings.Split(filePath, "/")
	bkt := tx.Bucket([]byte("/"))
	if bkt == nil {
		return false, ErrItemNotExisted
	}
	for _, p := range parts[:len(parts)-1] {
		if p == "" {
			continue
		}
		bkt = bkt.Bucket([]byte(p))
		if bkt == nil {
			return false, ErrItemNotExisted
		}
	}
	if bkt == nil {
		return false, ErrItemNotExisted
	}

	// update content
	fileName := path.Base(filePath)
	if isFolder {
		bkt = bkt.Bucket([]byte(fileName))
		if bkt == nil {
			return false, ErrItemNotExisted
		}
		if e := bkt.Put([]byte(DefaultDirKeyName), []byte(data)); e != nil {
			return false, e
		}
	} else {
		// is file
		if e := bkt.Put([]byte(fileName), []byte(data)); e != nil {
			return false, e
		}
	}

	// Commit the transaction and check for error.
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// Close 关闭数据库
func (p *BoltDb) Close() (bool, error) {
	p.locker.Lock()
	defer p.locker.Unlock()
	if p.db != nil {
		if e := p.db.Close(); e != nil {
			return false, e
		}
	}
	return true, nil
}
