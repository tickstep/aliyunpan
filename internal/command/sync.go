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
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/syncdrive"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/library-go/converter"
	"github.com/tickstep/library-go/logger"
	"github.com/urfave/cli"
	"os"
	"strings"
	"time"
)

func CmdSync() cli.Command {
	return cli.Command{
		Name:      "sync",
		Usage:     "同步备份功能(Beta)",
		UsageText: cmder.App().Name + " sync",
		Description: `
	备份功能。指定本地目录和对应的一个网盘目录，以备份文件。网盘目录必须和本地目录独占使用，不要用作其他他用途，不然备份可能会有问题。
	备份功能支持以下三种模式：
	1. upload 
       备份本地文件，即上传本地文件到网盘，始终保持本地文件有一个完整的备份在网盘
	2. download 
       备份云盘文件，即下载网盘文件到本地，始终保持网盘的文件有一个完整的备份在本地
	3. sync（慎用！！！双向备份过程会删除文件）
       双向备份，保持网盘文件和本地文件严格一致

	请输入以下命令查看如何配置和启动：
    aliyunpan sync start -h
`,
		Category: "阿里云盘",
		Before:   cmder.ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			cli.ShowCommandHelp(c, c.Command.Name)
			return nil
		},
		Subcommands: []cli.Command{
			{
				Name:      "start",
				Usage:     "启动sync同步备份任务",
				UsageText: cmder.App().Name + " sync start [arguments...]",
				Description: `
使用备份配置文件启动sync同步备份任务。备份配置文件必须存在，不然启动失败。
同步备份任务的配置文件保存在：(配置目录)/sync_drive/sync_drive_config.json，样例如下：
{
 "configVer": "1.0",
 "syncTaskList": [
  {
   "name": "设计文档备份",
   "localFolderPath": "D:/tickstep/Documents/设计文档",
   "panFolderPath": "/sync_drive/我的文档",
   "mode": "upload"
  }
 ]
}
相关字段说明如下：
name - 任务名称
localFolderPath - 本地目录
panFolderPath - 网盘目录
mode - 模式，支持三种: upload(备份本地文件到云盘),download(备份云盘文件到本地),sync(双向同步备份)

	例子:
	1. 查看帮助
	aliyunpan sync start -h

	2. 使用默认配置启动同步备份服务
	aliyunpan sync start

	3. 启动sync服务，并配置下载并发为2，上传并发为1，下载分片大小为256KB，上传分片大小为1MB
	aliyunpan sync start -dp 2 -up 1 -dbs 256 -ubs 1024

`,
				Action: func(c *cli.Context) error {
					if config.Config.ActiveUser() == nil {
						fmt.Println("未登录账号")
						return nil
					}
					dp := c.Int("dp")
					if dp == 0 {
						dp = config.Config.MaxDownloadParallel
					}
					if dp == 0 {
						dp = 2
					}

					up := c.Int("up")
					if up == 0 {
						up = config.Config.MaxUploadParallel
					}
					if up == 0 {
						up = 2
					}

					downloadBlockSize := int64(c.Int("dbs") * 1024)
					if downloadBlockSize == 0 {
						downloadBlockSize = int64(config.Config.CacheSize)
					}
					if downloadBlockSize == 0 {
						downloadBlockSize = int64(256 * 1024)
					}

					uploadBlockSize := int64(c.Int("ubs") * 1024)
					if uploadBlockSize == 0 {
						uploadBlockSize = aliyunpan.DefaultChunkSize
					}

					RunSync(dp, up, downloadBlockSize, uploadBlockSize)
					return nil
				},
				Flags: []cli.Flag{
					cli.IntFlag{
						Name:  "dp",
						Usage: "download parallel, 下载并发数量，即可以同时并发下载多少个文件。0代表跟从配置文件设置（取值范围:1 ~ 10）",
						Value: 0,
					},
					cli.IntFlag{
						Name:  "up",
						Usage: "upload parallel, 上传并发数量，即可以同时并发上传多少个文件。0代表跟从配置文件设置（取值范围:1 ~ 10）",
						Value: 0,
					},
					cli.IntFlag{
						Name:  "dbs",
						Usage: "download block size，下载分片大小，单位KB。推荐值：1024 ~ 10240",
						Value: 1024,
					},
					cli.IntFlag{
						Name:  "ubs",
						Usage: "upload block size，上传分片大小，单位KB。推荐值：1024 ~ 10240",
						Value: 10240,
					},
				},
			},
		},
	}
}

func RunSync(fileDownloadParallel, fileUploadParallel int, downloadBlockSize, uploadBlockSize int64) {
	useInternalUrl := config.Config.TransferUrlType == 2
	maxDownloadRate := config.Config.MaxDownloadRate
	maxUploadRate := config.Config.MaxUploadRate
	activeUser := GetActiveUser()
	panClient := activeUser.PanClient()

	// pan token expired checker
	go func() {
		for {
			time.Sleep(time.Duration(1) * time.Minute)
			if RefreshTokenInNeed(activeUser) {
				logger.Verboseln("update access token for sync task")
				panClient.UpdateToken(activeUser.WebToken)
			}
		}
	}()

	syncFolderRootPath := config.GetSyncDriveDir()
	if b, e := utils.PathExists(syncFolderRootPath); e == nil {
		if !b {
			os.MkdirAll(syncFolderRootPath, 0755)
		}
	}

	fmt.Println("启动同步备份进程")
	typeUrlStr := "默认链接"
	if useInternalUrl {
		typeUrlStr = "阿里ECS内部链接"
	}
	syncMgr := syncdrive.NewSyncTaskManager(activeUser, activeUser.DriveList.GetFileDriveId(), panClient, syncFolderRootPath,
		fileDownloadParallel, fileUploadParallel, downloadBlockSize, uploadBlockSize, useInternalUrl,
		maxDownloadRate, maxUploadRate)
	fmt.Printf("备份配置文件：%s\n链接类型：%s\n下载并发：%d\n上传并发：%d\n下载分片大小：%s\n上传分片大小：%s\n",
		syncMgr.ConfigFilePath(), typeUrlStr, fileDownloadParallel, fileUploadParallel, converter.ConvertFileSize(downloadBlockSize, 2),
		converter.ConvertFileSize(uploadBlockSize, 2))
	if _, e := syncMgr.Start(); e != nil {
		fmt.Println("启动任务失败：", e)
		return
	}
	c := ""
	fmt.Print("本命令不会退出，如需要结束同步备份进程请输入y，然后按Enter键进行停止：")
	for strings.ToLower(c) != "y" {
		fmt.Scan(&c)
	}
	fmt.Println("正在停止同步备份任务，请稍等...")
	syncMgr.Stop()
}
