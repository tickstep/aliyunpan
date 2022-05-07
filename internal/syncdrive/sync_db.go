package syncdrive

type (
	//SyncItem interface {
	//	FileName() string
	//	FilePath() string
	//	IsDir() bool
	//}
	SyncItem struct {
		FileName string
		FilePath string
		IsDir    bool
	}

	SyncDb interface {
		Open() (bool, error)
		Add(item *SyncItem) (bool, error)
		Get(filePath string) (*SyncItem, error)
		Delete(filePath string) (bool, error)
		Update(filePath string) (bool, error)
		Close() (bool, error)
	}
)
