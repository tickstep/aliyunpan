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

	// SyncFileDbBolt 存储同步文件状态信息的数据库
	SyncFileDbBolt struct {
		Path   string
		db     *BoltDb
		locker *sync.Mutex
	}
)

func newPanSyncDbBolt(dbFilePath string) *PanSyncDbBolt {
	return &PanSyncDbBolt{
		Path:   dbFilePath,
		locker: &sync.Mutex{},
	}
}

func (p *PanSyncDbBolt) Open() (bool, error) {
	return true, nil
}

// Add 增加一个数据项
func (p *PanSyncDbBolt) Add(item *PanFileItem) (bool, error) {
	if item == nil {
		return false, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	p.db = NewBoltDb(p.Path)
	p.db.Open()
	defer p.db.Close()

	data, err := json.Marshal(item)
	if err != nil {
		return false, err
	}
	return p.db.Add(&BoltItem{
		FilePath: item.Path,
		IsFolder: item.IsFolder(),
		Data:     string(data),
	})
}

// AddFileList 增加批量数据项
func (p *PanSyncDbBolt) AddFileList(items PanFileList) (bool, error) {
	if items == nil {
		return false, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	p.db = NewBoltDb(p.Path)
	p.db.Open()
	defer p.db.Close()

	boltItems := []*BoltItem{}
	for _, item := range items {
		data, err := json.Marshal(item)
		if err != nil {
			return false, err
		}
		boltItems = append(boltItems, &BoltItem{
			FilePath: item.Path,
			IsFolder: item.IsFolder(),
			Data:     string(data),
		})
	}
	return p.db.AddItems(boltItems)
}

// Get 获取一个数据项，数据项不存在返回错误
func (p *PanSyncDbBolt) Get(filePath string) (*PanFileItem, error) {
	filePath = FormatFilePath(filePath)
	if filePath == "" {
		return nil, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	p.db = NewBoltDb(p.Path)
	p.db.Open()
	defer p.db.Close()

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

	p.db = NewBoltDb(p.Path)
	p.db.Open()
	defer p.db.Close()

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

	p.db = NewBoltDb(p.Path)
	p.db.Open()
	defer p.db.Close()

	return p.db.Delete(filePath)
}

// Update 更新数据项，数据项不存在返回错误
func (p *PanSyncDbBolt) Update(item *PanFileItem) (bool, error) {
	if item == nil {
		return false, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	p.db = NewBoltDb(p.Path)
	p.db.Open()
	defer p.db.Close()

	data, err := json.Marshal(item)
	if err != nil {
		return false, err
	}
	return p.db.Update(item.Path, string(data))
}

// Close 关闭数据库
func (p *PanSyncDbBolt) Close() (bool, error) {
	return true, nil
}

func newLocalSyncDbBolt(dbFilePath string) *LocalSyncDbBolt {
	return &LocalSyncDbBolt{
		Path:   dbFilePath,
		locker: &sync.Mutex{},
	}
}

func (p *LocalSyncDbBolt) Open() (bool, error) {
	return true, nil
}

// Add 增加一个数据项
func (p *LocalSyncDbBolt) Add(item *LocalFileItem) (bool, error) {
	if item == nil {
		return false, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	p.db = NewBoltDb(p.Path)
	p.db.Open()
	defer p.db.Close()

	data, err := json.Marshal(item)
	if err != nil {
		return false, err
	}
	return p.db.Add(&BoltItem{
		FilePath: item.Path,
		IsFolder: item.IsFolder(),
		Data:     string(data),
	})
}

// AddFileList 增加批量数据项
func (p *LocalSyncDbBolt) AddFileList(items LocalFileList) (bool, error) {
	if items == nil {
		return false, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	p.db = NewBoltDb(p.Path)
	p.db.Open()
	defer p.db.Close()

	boltItems := []*BoltItem{}
	for _, item := range items {
		data, err := json.Marshal(item)
		if err != nil {
			return false, err
		}
		boltItems = append(boltItems, &BoltItem{
			FilePath: item.Path,
			IsFolder: item.IsFolder(),
			Data:     string(data),
		})
	}
	return p.db.AddItems(boltItems)
}

// Get 获取一个数据项，数据项不存在返回错误
func (p *LocalSyncDbBolt) Get(filePath string) (*LocalFileItem, error) {
	filePath = FormatFilePath(filePath)
	if filePath == "" {
		return nil, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	p.db = NewBoltDb(p.Path)
	p.db.Open()
	defer p.db.Close()

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

	p.db = NewBoltDb(p.Path)
	p.db.Open()
	defer p.db.Close()

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
	p.db = NewBoltDb(p.Path)
	p.db.Open()
	defer p.db.Close()
	return p.db.Delete(filePath)
}

// Update 更新数据项，数据项不存在返回错误
func (p *LocalSyncDbBolt) Update(item *LocalFileItem) (bool, error) {
	if item == nil {
		return false, fmt.Errorf("item is nil")
	}
	p.locker.Lock()
	defer p.locker.Unlock()

	p.db = NewBoltDb(p.Path)
	p.db.Open()
	defer p.db.Close()

	data, err := json.Marshal(item)
	if err != nil {
		return false, err
	}
	return p.db.Update(item.Path, string(data))
}

// Close 关闭数据库
func (p *LocalSyncDbBolt) Close() (bool, error) {
	return true, nil
}

func newSyncFileDbBolt(dbFilePath string) *SyncFileDbBolt {
	return &SyncFileDbBolt{
		Path:   dbFilePath,
		locker: &sync.Mutex{},
	}
}

// Open 打开并准备数据库
func (s *SyncFileDbBolt) Open() (bool, error) {
	return true, nil
}

// Add 存储一个数据项
func (s *SyncFileDbBolt) Add(item *SyncFileItem) (bool, error) {
	if item == nil {
		return false, fmt.Errorf("item is nil")
	}
	s.locker.Lock()
	defer s.locker.Unlock()

	s.db = NewBoltDb(s.Path)
	s.db.Open()
	defer s.db.Close()

	data, err := json.Marshal(item)
	if err != nil {
		return false, err
	}
	return s.db.Add(&BoltItem{
		FilePath: "/" + item.Id(),
		IsFolder: false,
		Data:     string(data),
	})
}

// Get 获取一个数据项
func (s *SyncFileDbBolt) Get(id string) (*SyncFileItem, error) {
	filePath := "/" + id
	if filePath == "" {
		return nil, fmt.Errorf("item is nil")
	}
	s.locker.Lock()
	defer s.locker.Unlock()

	s.db = NewBoltDb(s.Path)
	s.db.Open()
	defer s.db.Close()

	data, err := s.db.Get(filePath)
	if err == nil && data != "" {
		item := &SyncFileItem{}
		if err := json.Unmarshal([]byte(data), item); err != nil {
			return nil, err
		}
		return item, nil
	}
	return nil, err
}

// GetFileList 获取文件夹下的所有的文件列表
func (s *SyncFileDbBolt) GetFileList(Status SyncFileStatus) (SyncFileList, error) {
	filePath := "/"
	if filePath == "" {
		return nil, fmt.Errorf("item is nil")
	}
	s.locker.Lock()
	defer s.locker.Unlock()

	s.db = NewBoltDb(s.Path)
	s.db.Open()
	defer s.db.Close()

	panFileList := SyncFileList{}
	dataList, err := s.db.GetFileList(filePath)
	if err == nil && len(dataList) > 0 {
		for _, data := range dataList {
			if data == "" {
				continue
			}
			item := &SyncFileItem{}
			if err := json.Unmarshal([]byte(data), item); err != nil {
				return nil, err
			}
			if item.Status == Status {
				panFileList = append(panFileList, item)
			}
		}
		return panFileList, nil
	}
	return nil, err
}

// Delete 删除一个数据项，如果是文件夹，则会删除文件夹下面所有的文件列表
func (s *SyncFileDbBolt) Delete(id string) (bool, error) {
	filePath := "/" + id
	if filePath == "" {
		return false, fmt.Errorf("item is nil")
	}
	s.locker.Lock()
	defer s.locker.Unlock()

	s.db = NewBoltDb(s.Path)
	s.db.Open()
	defer s.db.Close()
	return s.db.Delete(filePath)
}

// Update 更新一个数据项数据
func (s *SyncFileDbBolt) Update(item *SyncFileItem) (bool, error) {
	if item == nil {
		return false, fmt.Errorf("item is nil")
	}
	s.locker.Lock()
	defer s.locker.Unlock()

	s.db = NewBoltDb(s.Path)
	s.db.Open()
	defer s.db.Close()

	data, err := json.Marshal(item)
	if err != nil {
		return false, err
	}
	return s.db.Update("/"+item.Id(), string(data))
}

// Close 关闭数据库
func (s *SyncFileDbBolt) Close() (bool, error) {
	return true, nil
}
