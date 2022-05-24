package syncdrive

type (
	FileActionTaskExecutor struct {
		localFileDb LocalSyncDb
		panFileDb   PanSyncDb
	}

	FileActionTaskManager struct {
		FileActionTaskList []*FileActionTask
		task               *SyncTask
	}
)

func NewFileActionTaskManager(task *SyncTask) *FileActionTaskManager {
	return &FileActionTaskManager{
		task:               task,
		FileActionTaskList: []*FileActionTask{},
	}
}

func (f *FileActionTaskManager) StartSync() error {
	return nil
}

func (f *FileActionTaskManager) StopSync() error {
	return nil
}
