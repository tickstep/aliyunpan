package config

import (
	"github.com/tickstep/aliyunpan-api/aliyunpan_open"
	"github.com/tickstep/aliyunpan-api/aliyunpan_web"
)

type (
	// PanClient 云盘客户端
	PanClient struct {
		// 网页WEB接口客户端
		webapiPanClient *aliyunpan_web.PanClient
		// 阿里openapi接口客户端
		openapiPanClient *aliyunpan_open.OpenPanClient
	}
)

func NewPanClient(webClient *aliyunpan_web.PanClient, openClient *aliyunpan_open.OpenPanClient) *PanClient {
	return &PanClient{
		webapiPanClient:  webClient,
		openapiPanClient: openClient,
	}
}

func (p *PanClient) WebapiPanClient() *aliyunpan_web.PanClient {
	return p.webapiPanClient
}

func (p *PanClient) OpenapiPanClient() *aliyunpan_open.OpenPanClient {
	return p.openapiPanClient
}

func (p *PanClient) UpdateClient(openClient *aliyunpan_open.OpenPanClient, webClient *aliyunpan_web.PanClient) {
	p.webapiPanClient = webClient
	p.openapiPanClient = openClient
}
