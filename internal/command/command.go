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
package command

import (
	"errors"
	"fmt"
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/cmder/cmdutil"
	"github.com/tickstep/aliyunpan/library/crypto"
	"github.com/tickstep/library-go/getip"
	"github.com/urfave/cli"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan/internal/config"
)

type (
	// 秒传数据项
	RapidUploadItem struct {
		FileSha1 string
		FileSize int64
		FilePath string // 绝对路径，包含文件名
	}
)

const (
	cryptoDescription = `
	可用的方法 <method>:
		aes-128-ctr, aes-192-ctr, aes-256-ctr,
		aes-128-cfb, aes-192-cfb, aes-256-cfb,
		aes-128-ofb, aes-192-ofb, aes-256-ofb.

	密钥 <key>:
		aes-128 对应key长度为16, aes-192 对应key长度为24, aes-256 对应key长度为32,
		如果key长度不符合, 则自动修剪key, 舍弃超出长度的部分, 长度不足的部分用'\0'填充.

	GZIP <disable-gzip>:
		在文件加密之前, 启用GZIP压缩文件; 文件解密之后启用GZIP解压缩文件, 默认启用,
		如果不启用, 则无法检测文件是否解密成功, 解密文件时会保留源文件, 避免解密失败造成文件数据丢失.`
)

var ErrBadArgs = errors.New("参数错误")
var ErrNotLogined = errors.New("未登录账号")

func GetActivePanClient() *aliyunpan.PanClient {
	return config.Config.ActiveUser().PanClient()
}

func GetActiveUser() *config.PanUser {
	return config.Config.ActiveUser()
}

func parseDriveId(c *cli.Context) string {
	driveId := config.Config.ActiveUser().ActiveDriveId
	if c.IsSet("driveId") {
		driveId = c.String("driveId")
	}
	return driveId
}

// newRapidUploadItem 通过解析秒传链接创建秒传实体
func newRapidUploadItem(rapidUploadShareLink string) (*RapidUploadItem, error) {
	if strings.IndexAny(rapidUploadShareLink, "aliyunpan://") != 0 {
		return nil, fmt.Errorf("秒传链接格式错误: %s", rapidUploadShareLink)
	}

	// 格式：aliyunpan://文件名|sha1|文件大小|<相对路径>
	rapidUploadShareLinkStr := strings.Replace(rapidUploadShareLink, "aliyunpan://", "", 1)

	item := &RapidUploadItem{}
	parts := strings.Split(rapidUploadShareLinkStr, "|")

	if len(parts) < 4 {
		return nil, fmt.Errorf("秒传链接格式错误: %s", rapidUploadShareLink)
	}

	// hash
	if len(parts[1]) == 0 {
		return nil, fmt.Errorf("文件sha1错误: %s", rapidUploadShareLink)
	}
	item.FileSha1 = strings.TrimSpace(parts[1])

	// size
	if size, e := strconv.ParseInt(parts[2], 10, 64); e == nil {
		item.FileSize = size
	} else {
		return nil, fmt.Errorf("文件大小错误: %s", rapidUploadShareLink)
	}

	// path
	relativePath, _ := url.QueryUnescape(parts[3])
	item.FilePath = path.Join(relativePath, parts[0])

	// result
	return item, nil
}

func newRapidUploadItemFromFileEntity(fileEntity *aliyunpan.FileEntity) *RapidUploadItem {
	if fileEntity == nil {
		return nil
	}
	return &RapidUploadItem{
		FileSha1: fileEntity.ContentHash,
		FileSize: fileEntity.FileSize,
		FilePath: fileEntity.Path,
	}
}

// 创建秒传链接
// 链接格式说明：aliyunpan://文件名|sha1|文件大小|<相对路径>
// "相对路径" 可以为空，为空代表存储到网盘根目录
func (r *RapidUploadItem) createRapidUploadLink(hideRelativePath bool) string {
	fullLink := &strings.Builder{}

	p := r.FilePath
	p = strings.ReplaceAll(p, "\\", "/")

	fileName := path.Base(p)
	dirPath := path.Dir(p)

	// 去掉开头/
	if strings.Index(dirPath, "/") == 0 {
		dirPath = dirPath[1:]
	}
	// 相对路径编码
	dirPath = url.QueryEscape(dirPath)

	// 隐藏相对路径
	if hideRelativePath {
		dirPath = ""
	}

	// 拼接
	fmt.Fprintf(fullLink, "aliyunpan://%s|%s|%d|%s",
		fileName, strings.ToUpper(r.FileSha1), r.FileSize, dirPath)

	return fullLink.String()
}

func CmdConfig() cli.Command {
	return cli.Command{
		Name:        "config",
		Usage:       "显示和修改程序配置项",
		Description: "显示和修改程序配置项",
		Category:    "配置",
		Before:      ReloadConfigFunc,
		After:       SaveConfigFunc,
		Action: func(c *cli.Context) error {
			fmt.Printf("----\n运行 %s config set 可进行设置配置\n\n当前配置:\n", cmder.App().Name)
			config.Config.PrintTable()
			return nil
		},
		Subcommands: []cli.Command{
			{
				Name:      "set",
				Usage:     "修改程序配置项",
				UsageText: cmder.App().Name + " config set [arguments...]",
				Description: `
	注意:
		可通过设置环境变量 ALIYUNPAN_CONFIG_DIR, 指定配置文件存放的目录.

		cache_size 的值支持可选设置单位, 单位不区分大小写, b 和 B 均表示字节的意思, 如 64KB, 1MB, 32kb, 65536b, 65536
		max_download_rate, max_upload_rate 的值支持可选设置单位, 单位为每秒的传输速率, 后缀'/s' 可省略, 如 2MB/s, 2MB, 2m, 2mb 均为一个意思

	例子:
		aliyunpan config set -cache_size 64KB
		aliyunpan config set -cache_size 16384 -max_download_parallel 200 -savedir D:/download`,
				Action: func(c *cli.Context) error {
					if c.NumFlags() <= 0 || c.NArg() > 0 {
						cli.ShowCommandHelp(c, c.Command.Name)
						return nil
					}
					if c.IsSet("cache_size") {
						err := config.Config.SetCacheSizeByStr(c.String("cache_size"))
						if err != nil {
							fmt.Printf("设置 cache_size 错误: %s\n", err)
							return nil
						}
					}
					if c.IsSet("max_download_parallel") {
						config.Config.MaxDownloadParallel = c.Int("max_download_parallel")
					}
					if c.IsSet("max_upload_parallel") {
						config.Config.MaxUploadParallel = c.Int("max_upload_parallel")
					}
					if c.IsSet("max_download_rate") {
						err := config.Config.SetMaxDownloadRateByStr(c.String("max_download_rate"))
						if err != nil {
							fmt.Printf("设置 max_download_rate 错误: %s\n", err)
							return nil
						}
					}
					if c.IsSet("max_upload_rate") {
						err := config.Config.SetMaxUploadRateByStr(c.String("max_upload_rate"))
						if err != nil {
							fmt.Printf("设置 max_upload_rate 错误: %s\n", err)
							return nil
						}
					}
					if c.IsSet("transfer_url_type") {
						config.Config.TransferUrlType = c.Int("transfer_url_type")
					}

					if c.IsSet("savedir") {
						config.Config.SaveDir = c.String("savedir")
					}
					if c.IsSet("proxy") {
						config.Config.SetProxy(c.String("proxy"))
					}
					if c.IsSet("local_addrs") {
						config.Config.SetLocalAddrs(c.String("local_addrs"))
					}
					if c.IsSet("file_record_config") {
						config.Config.SetFileRecorderConfig(c.String("file_record_config"))
					}

					err := config.Config.Save()
					if err != nil {
						fmt.Println(err)
						return err
					}

					config.Config.PrintTable()
					fmt.Printf("\n保存配置成功!\n\n")

					return nil
				},
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "cache_size",
						Usage: "下载缓存，单位：KB",
					},
					cli.IntFlag{
						Name:  "max_download_parallel",
						Usage: "下载文件最大并发量",
					},
					cli.IntFlag{
						Name:  "max_upload_parallel",
						Usage: "上传文件最大并发量",
					},
					cli.StringFlag{
						Name:  "max_download_rate",
						Usage: "限制最大下载速度, 0代表不限制",
					},
					cli.StringFlag{
						Name:  "max_upload_rate",
						Usage: "限制最大上传速度, 0代表不限制",
					},
					cli.IntFlag{
						Name:  "transfer_url_type",
						Usage: "上传下载URL类别，1-默认，2-阿里云ECS",
						Value: 1,
					},
					cli.StringFlag{
						Name:  "savedir",
						Usage: "下载文件的储存目录",
					},
					cli.StringFlag{
						Name:  "proxy",
						Usage: "设置代理, 支持 http/socks5 代理",
					},
					cli.StringFlag{
						Name:  "local_addrs",
						Usage: "设置本地网卡地址, 多个地址用逗号隔开",
					},
					cli.StringFlag{
						Name:  "file_record_config",
						Usage: "设置是否开启上传、下载、同步文件的结果记录功能",
					},
				},
			},
		},
	}
}

func CmdTool() cli.Command {
	return cli.Command{
		Name:  "tool",
		Usage: "工具箱",
		Action: func(c *cli.Context) error {
			cli.ShowCommandHelp(c, c.Command.Name)
			return nil
		},
		Subcommands: []cli.Command{
			{
				Name:  "getip",
				Usage: "获取IP地址",
				Action: func(c *cli.Context) error {
					fmt.Printf("内网IP地址: \n")
					for _, address := range cmdutil.ListAddresses() {
						fmt.Printf("%s\n", address)
					}
					fmt.Printf("\n")

					ipAddr, err := getip.IPInfoFromTechainBaiduByClient(config.Config.HTTPClient(""))
					if err != nil {
						fmt.Printf("获取公网IP错误: %s\n", err)
						return nil
					}

					fmt.Printf("公网IP地址: %s\n", ipAddr)
					return nil
				},
			},
			{
				Name:        "enc",
				Usage:       "加密文件",
				UsageText:   cmder.App().Name + " enc -method=<method> -key=<key> [files...]",
				Description: cryptoDescription,
				Action: func(c *cli.Context) error {
					if c.NArg() <= 0 {
						cli.ShowCommandHelp(c, c.Command.Name)
						return nil
					}

					for _, filePath := range c.Args() {
						encryptedFilePath, err := crypto.EncryptFile(c.String("method"), []byte(c.String("key")), filePath, !c.Bool("disable-gzip"))
						if err != nil {
							fmt.Printf("%s\n", err)
							continue
						}

						fmt.Printf("加密成功, %s -> %s\n", filePath, encryptedFilePath)
					}

					return nil
				},
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "method",
						Usage: "加密方法",
						Value: "aes-128-ctr",
					},
					cli.StringFlag{
						Name:  "key",
						Usage: "加密密钥",
						Value: cmder.App().Name,
					},
					cli.BoolFlag{
						Name:  "disable-gzip",
						Usage: "不启用GZIP",
					},
				},
			},
			{
				Name:        "dec",
				Usage:       "解密文件",
				UsageText:   cmder.App().Name + " dec -method=<method> -key=<key> [files...]",
				Description: cryptoDescription,
				Action: func(c *cli.Context) error {
					if c.NArg() <= 0 {
						cli.ShowCommandHelp(c, c.Command.Name)
						return nil
					}

					for _, filePath := range c.Args() {
						decryptedFilePath, err := crypto.DecryptFile(c.String("method"), []byte(c.String("key")), filePath, !c.Bool("disable-gzip"))
						if err != nil {
							fmt.Printf("%s\n", err)
							continue
						}

						fmt.Printf("解密成功, %s -> %s\n", filePath, decryptedFilePath)
					}

					return nil
				},
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "method",
						Usage: "加密方法",
						Value: "aes-128-ctr",
					},
					cli.StringFlag{
						Name:  "key",
						Usage: "加密密钥",
						Value: cmder.App().Name,
					},
					cli.BoolFlag{
						Name:  "disable-gzip",
						Usage: "不启用GZIP",
					},
				},
			},
		},
	}
}
