package syncdrive

import (
	"encoding/json"
	"fmt"
	"sync"
)

type (

	// PanSyncDbBolt 存储网盘文件信息的数据库
	PanSyncDbBolt struct {
		Path   string
		db     *BoltDb
		locker *sync.Mutex
	}

	// LocalSyncDbBolt 存储本地文件信息的数据库
	LocalSyncDbBolt struct {
		Path   string
		db     *BoltDb
		locker *sync.Mutex
	}
)

const (
	DefaultDirKeyName string = "."
)

func newPanSyncDbBolt(dbFilePath string) *PanSyncDbBolt {
	return &PanSyncDbBolt{
		Path:   dbFilePath,
		locker: &sync.Mutex{},
	}
}

func (p *PanSyncDbBolt) Open() (bool, error) {
	p.db = NewBoltDb(p.Path)
	return p.db.Open()
}

// Add 增加一个数据项
func (p *PanSyncDbBolt) Add(item *PanFileItem) (bool, error) {
	if item == nil {
		return false, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	data, err := json.Marshal(item)
	if err != nil {
		return false, err
	}
	return p.db.Add(item.Path, item.IsFolder(), string(data))
}

// Get 获取一个数据项，数据项不存在返回错误
func (p *PanSyncDbBolt) Get(filePath string) (*PanFileItem, error) {
	filePath = FormatFilePath(filePath)
	if filePath == "" {
		return nil, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	data, err := p.db.Get(filePath)
	if err == nil && data != "" {
		item := &PanFileItem{}
		if err := json.Unmarshal([]byte(data), item); err != nil {
			return nil, err
		}
		return item, nil
	}
	return nil, err
}

func (p *PanSyncDbBolt) GetFileList(filePath string) (PanFileList, error) {
	filePath = FormatFilePath(filePath)
	if filePath == "" {
		return nil, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	panFileList := PanFileList{}
	dataList, err := p.db.GetFileList(filePath)
	if err == nil && len(dataList) > 0 {
		for _, data := range dataList {
			if data == "" {
				continue
			}
			item := &PanFileItem{}
			if err := json.Unmarshal([]byte(data), item); err != nil {
				return nil, err
			}
			panFileList = append(panFileList, item)
		}
		return panFileList, nil
	}
	return nil, err
}

// Delete 删除一个数据项
func (p *PanSyncDbBolt) Delete(filePath string) (bool, error) {
	filePath = FormatFilePath(filePath)
	if filePath == "" {
		return false, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	return p.db.Delete(filePath)
}

// Update 更新数据项，数据项不存在返回错误
func (p *PanSyncDbBolt) Update(item *PanFileItem) (bool, error) {
	if item == nil {
		return false, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	data, err := json.Marshal(item)
	if err != nil {
		return false, err
	}
	return p.db.Update(item.Path, item.IsFolder(), string(data))
}

// Close 关闭数据库
func (p *PanSyncDbBolt) Close() (bool, error) {
	p.locker.Lock()
	defer p.locker.Unlock()
	return p.db.Close()
}

func newLocalSyncDbBolt(dbFilePath string) *LocalSyncDbBolt {
	return &LocalSyncDbBolt{
		Path:   dbFilePath,
		locker: &sync.Mutex{},
	}
}

func (p *LocalSyncDbBolt) Open() (bool, error) {
	p.db = NewBoltDb(p.Path)
	return p.db.Open()
}

// Add 增加一个数据项
func (p *LocalSyncDbBolt) Add(item *LocalFileItem) (bool, error) {
	if item == nil {
		return false, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	data, err := json.Marshal(item)
	if err != nil {
		return false, err
	}
	return p.db.Add(item.Path, item.IsFolder(), string(data))
}

// Get 获取一个数据项，数据项不存在返回错误
func (p *LocalSyncDbBolt) Get(filePath string) (*LocalFileItem, error) {
	filePath = FormatFilePath(filePath)
	if filePath == "" {
		return nil, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	data, err := p.db.Get(filePath)
	if err == nil && data != "" {
		item := &LocalFileItem{}
		if err := json.Unmarshal([]byte(data), item); err != nil {
			return nil, err
		}
		return item, nil
	}
	return nil, err
}

func (p *LocalSyncDbBolt) GetFileList(filePath string) (LocalFileList, error) {
	filePath = FormatFilePath(filePath)
	if filePath == "" {
		return nil, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	LocalFileList := LocalFileList{}
	dataList, err := p.db.GetFileList(filePath)
	if err == nil && len(dataList) > 0 {
		for _, data := range dataList {
			if data == "" {
				continue
			}
			item := &LocalFileItem{}
			if err := json.Unmarshal([]byte(data), item); err != nil {
				return nil, err
			}
			LocalFileList = append(LocalFileList, item)
		}
		return LocalFileList, nil
	}
	return nil, err
}

// Delete 删除一个数据项
func (p *LocalSyncDbBolt) Delete(filePath string) (bool, error) {
	filePath = FormatFilePath(filePath)
	if filePath == "" {
		return false, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	return p.db.Delete(filePath)
}

// Update 更新数据项，数据项不存在返回错误
func (p *LocalSyncDbBolt) Update(item *LocalFileItem) (bool, error) {
	if item == nil {
		return false, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	data, err := json.Marshal(item)
	if err != nil {
		return false, err
	}
	return p.db.Update(item.Path, item.IsFolder(), string(data))
}

// Close 关闭数据库
func (p *LocalSyncDbBolt) Close() (bool, error) {
	p.locker.Lock()
	defer p.locker.Unlock()
	return p.db.Close()
}
