package plugins

type (
	IdlePlugin struct {
		Name string
	}
)

func NewIdlePlugin() *IdlePlugin {
	return &IdlePlugin{
		Name: "IdlePlugin",
	}
}

func (p *IdlePlugin) Start() error {
	return nil
}

func (p *IdlePlugin) UploadFilePrepareCallback(context *Context, params *UploadFilePrepareParams) (*UploadFilePrepareResult, error) {
	return nil, nil
}

func (p *IdlePlugin) UploadFileFinishCallback(context *Context, params *UploadFileFinishParams) error {
	return nil
}

func (p *IdlePlugin) DownloadFilePrepareCallback(context *Context, params *DownloadFilePrepareParams) (*DownloadFilePrepareResult, error) {
	return nil, nil
}

func (p *IdlePlugin) DownloadFileFinishCallback(context *Context, params *DownloadFileFinishParams) error {
	return nil
}

func (p *IdlePlugin) SyncScanLocalFilePrepareCallback(context *Context, params *SyncScanLocalFilePrepareParams) (*SyncScanLocalFilePrepareResult, error) {
	return nil, nil
}

func (p *IdlePlugin) SyncScanPanFilePrepareCallback(context *Context, params *SyncScanPanFilePrepareParams) (*SyncScanPanFilePrepareResult, error) {
	return nil, nil
}

func (p *IdlePlugin) Stop() error {
	return nil
}
