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

func (p *IdlePlugin) DownloadFilePrepareCallback(context *Context, params *DownloadFilePrepareParams) (*DownloadFilePrepareResult, error) {
	return nil, nil
}

func (p *IdlePlugin) Stop() error {
	return nil
}
