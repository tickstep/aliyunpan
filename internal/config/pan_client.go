package config

import (
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan_open"
)

type (
	// PanClient 云盘客户端
	PanClient struct {
		// 网页WEB接口客户端
		webapiPanClient *aliyunpan.PanClient
		// 阿里openapi接口客户端
		openapiPanClient *aliyunpan_open.OpenPanClient
	}
)

func NewPanClient(webClient *aliyunpan.PanClient, openClient *aliyunpan_open.OpenPanClient) *PanClient {
	return &PanClient{
		webapiPanClient:  webClient,
		openapiPanClient: openClient,
	}
}

func (p *PanClient) WebapiPanClient() *aliyunpan.PanClient {
	return p.webapiPanClient
}

func (p *PanClient) OpenapiPanClient() *aliyunpan_open.OpenPanClient {
	return p.openapiPanClient
}
