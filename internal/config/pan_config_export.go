// Copyright (c) 2020 tickstep.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/tickstep/aliyunpan/cmder/cmdtable"
	"github.com/tickstep/library-go/converter"
	"github.com/tickstep/library-go/requester"
)

// SetProxy 设置代理
func (c *PanConfig) SetProxy(proxy string) {
	c.Proxy = proxy
	requester.SetGlobalProxy(proxy)
}

// SetLocalAddrs 设置localAddrs
func (c *PanConfig) SetLocalAddrs(localAddrs string) {
	c.LocalAddrs = localAddrs
	requester.SetLocalTCPAddrList(strings.Split(localAddrs, ",")...)
}

// SetCacheSizeByStr 设置cache_size
func (c *PanConfig) SetCacheSizeByStr(sizeStr string) error {
	size, err := converter.ParseFileSizeStr(sizeStr)
	if err != nil {
		return err
	}
	c.CacheSize = int(size)
	return nil
}

// SetMaxDownloadRateByStr 设置 max_download_rate
func (c *PanConfig) SetMaxDownloadRateByStr(sizeStr string) error {
	size, err := converter.ParseFileSizeStr(stripPerSecond(sizeStr))
	if err != nil {
		return err
	}
	c.MaxDownloadRate = size
	return nil
}

// SetMaxUploadRateByStr 设置 max_upload_rate
func (c *PanConfig) SetMaxUploadRateByStr(sizeStr string) error {
	size, err := converter.ParseFileSizeStr(stripPerSecond(sizeStr))
	if err != nil {
		return err
	}
	c.MaxUploadRate = size
	return nil
}

// PrintTable 输出表格
func (c *PanConfig) PrintTable() {
	tb := cmdtable.NewTable(os.Stdout)
	tb.SetHeader([]string{"名称", "值", "建议值", "描述"})
	tb.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	tb.SetColumnAlignment([]int{tablewriter.ALIGN_DEFAULT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT})
	tb.AppendBulk([][]string{
		[]string{"cache_size", converter.ConvertFileSize(int64(c.CacheSize), 2), "1KB ~ 256KB", "下载缓存, 如果硬盘占用高或下载速度慢, 请尝试调大此值"},
		[]string{"max_download_parallel", strconv.Itoa(c.MaxDownloadParallel), "1 ~ 64", "最大下载并发量"},
		[]string{"max_upload_parallel", strconv.Itoa(c.MaxUploadParallel), "1 ~ 100", "最大上传并发量，即同时上传文件最大数量"},
		[]string{"max_download_load", strconv.Itoa(c.MaxDownloadLoad), "1 ~ 5", "同时进行下载文件的最大数量"},
		[]string{"max_download_rate", showMaxRate(c.MaxDownloadRate), "", "限制最大下载速度, 0代表不限制"},
		[]string{"max_upload_rate", showMaxRate(c.MaxUploadRate), "", "限制最大上传速度, 0代表不限制"},
		[]string{"transfer_url_type", strconv.Itoa(c.TransferUrlType), "1-默认，2-阿里云ECS", "上传下载URL类别。除非在阿里云ECS服务器中使用，不然请设置1"},
		[]string{"savedir", c.SaveDir, "", "下载文件的储存目录"},
		[]string{"proxy", c.Proxy, "", "设置代理, 支持 http/socks5 代理，例如：http://127.0.0.1:8888"},
		[]string{"local_addrs", c.LocalAddrs, "", "设置本地网卡地址, 多个地址用逗号隔开"},
	})
	tb.Render()
}
