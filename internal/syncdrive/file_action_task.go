package syncdrive

type (
	FileAction     string
	FileActionTask struct {
		Action    FileAction
		LocalFile *LocalFileItem
		PanFile   *PanFileItem
	}
)

const (
	DownloadFile    FileAction = "DownloadFile"
	UploadFile      FileAction = "UploadFile"
	DeleteLocalFile FileAction = "DeleteLocalFile"
	DeletePanFile   FileAction = "DeletePanFile"
)

func (f *FileActionTask) DoAction() error {
	return nil
}

func (f *FileActionTask) DownloadFile() error {
	return nil
}

func (f *FileActionTask) UploadFile() error {
	return nil
}

func (f *FileActionTask) DeleteLocalFile() error {
	return nil
}

func (f *FileActionTask) DeletePanFile() error {
	return nil
}
