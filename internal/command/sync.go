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
	"path"
	"strings"
	"time"
)

func CmdSync() cli.Command {
	return cli.Command{
		Name:      "sync",
		Usage:     "同步备份功能(Beta)",
		UsageText: cmder.App().Name + " sync",
		Description: `
    备份功能。支持备份本地文件到云盘，备份云盘文件到本地，双向同步备份三种模式。支持JavaScript插件对备份文件进行过滤。
    指定本地目录和对应的一个网盘目录，以备份文件。网盘目录必须和本地目录独占使用，不要用作其他用途，不然备份可能会有问题。

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
备份本地文件到文件网盘，或者备份文件网盘的文件到本地。支持命令行配置启动或者使用备份配置文件启动同步备份任务。
   
配置文件保存在：(配置目录)/sync_drive/sync_drive_config.json，样例如下：
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
    
	2. 使用命令行配置启动同步备份服务，将本地目录 D:\tickstep\Documents\设计文档 中的文件备份上传到云盘目录 /sync_drive/我的文档
	aliyunpan sync start -ldir "D:\tickstep\Documents\设计文档" -pdir "/sync_drive/我的文档" -mode "upload"

	3. 使用命令行配置启动同步备份服务，将云盘目录 /sync_drive/我的文档 中的文件备份下载到本地目录 D:\tickstep\Documents\设计文档
	aliyunpan sync start -ldir "D:\tickstep\Documents\设计文档" -pdir "/sync_drive/我的文档" -mode "download"

	4. 使用命令行配置启动同步备份服务，将云盘目录 /sync_drive/我的文档 和本地目录 D:\tickstep\Documents\设计文档 的文件进行双向同步
       同时配置同步优先选项为本地文件优先
	aliyunpan sync start -ldir "D:\tickstep\Documents\设计文档" -pdir "/sync_drive/我的文档" -mode "sync" -pri "local"

	5. 使用命令行配置启动同步备份服务，将本地目录 D:\tickstep\Documents\设计文档 中的文件备份到云盘目录 /sync_drive/我的文档
       同时配置下载并发为2，上传并发为1，下载分片大小为256KB，上传分片大小为1MB
	aliyunpan sync start -ldir "D:\tickstep\Documents\设计文档" -pdir "/sync_drive/我的文档" -mode "upload" -dp 2 -up 1 -dbs 256 -ubs 1024
    
	6. 使用配置文件启动同步备份服务，使用配置文件可以支持同时启动多个备份任务。配置文件必须存在，否则启动失败。
	aliyunpan sync start

	7. 使用配置文件启动同步备份服务，并配置下载并发为2，上传并发为1，下载分片大小为256KB，上传分片大小为1MB
	aliyunpan sync start -dp 2 -up 1 -dbs 256 -ubs 1024

`,
				Action: func(c *cli.Context) error {
					if config.Config.ActiveUser() == nil {
						fmt.Println("未登录账号")
						return nil
					}
					activeUser := GetActiveUser()

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

					var task *syncdrive.SyncTask
					localDir := c.String("ldir")
					panDir := c.String("pdir")
					mode := c.String("mode")
					// make path absolute
					if !utils.IsAbsPath(localDir) {
						pwd, _ := os.Getwd()
						localDir = path.Join(pwd, path.Clean(localDir))
					}
					panDir = activeUser.PathJoin(activeUser.ActiveDriveId, panDir)
					if localDir != "" && panDir != "" {
						if !utils.IsAbsPath(localDir) {
							fmt.Println("本地目录请指定绝对路径")
							return nil
						}
						if !path.IsAbs(panDir) {
							fmt.Println("网盘目录请指定绝对路径")
							return nil
						}
						//if b, e := utils.PathExists(localDir); e == nil {
						//	if !b {
						//		fmt.Println("本地文件夹不存在：", localDir)
						//		return nil
						//	}
						//} else {
						//	fmt.Println("本地文件夹不存在：", localDir)
						//	return nil
						//}
						task = &syncdrive.SyncTask{}
						task.LocalFolderPath = path.Clean(strings.ReplaceAll(localDir, "\\", "/"))
						task.PanFolderPath = panDir
						task.Mode = syncdrive.UploadOnly
						if mode == string(syncdrive.UploadOnly) {
							task.Mode = syncdrive.UploadOnly
						} else if mode == string(syncdrive.DownloadOnly) {
							task.Mode = syncdrive.DownloadOnly
						} else if mode == string(syncdrive.SyncTwoWay) {
							task.Mode = syncdrive.SyncTwoWay
						} else {
							task.Mode = syncdrive.UploadOnly
						}
						task.Name = path.Base(task.LocalFolderPath)
						task.Id = utils.Md5Str(task.LocalFolderPath)
					}
					opt := c.String("pri")
					var syncOpt syncdrive.SyncPriorityOption = syncdrive.SyncPriorityTimestampFirst
					if opt == "local" {
						syncOpt = syncdrive.SyncPriorityLocalFirst
					} else if opt == "pan" {
						syncOpt = syncdrive.SyncPriorityPanFirst
					} else {
						syncOpt = syncdrive.SyncPriorityTimestampFirst
					}

					RunSync(task, dp, up, downloadBlockSize, uploadBlockSize, syncOpt)
					return nil
				},
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "ldir",
						Usage: "local dir, 本地文件夹完整路径",
					},
					cli.StringFlag{
						Name:  "pdir",
						Usage: "pan dir, 云盘文件夹完整路径",
					},
					cli.StringFlag{
						Name:  "mode",
						Usage: "备份模式, 支持三种: upload(备份本地文件到云盘),download(备份云盘文件到本地),sync(双向同步备份)",
						Value: "upload",
					},
					cli.StringFlag{
						Name:  "pri",
						Usage: "优先级priority，只对双向同步备份模式有效。当网盘和本地存在同名文件，优先使用哪个，选项支持三种: time-时间优先，local-本地优先，pan-网盘优先",
						Value: "time",
					},
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

func RunSync(defaultTask *syncdrive.SyncTask, fileDownloadParallel, fileUploadParallel int, downloadBlockSize, uploadBlockSize int64,
	flag syncdrive.SyncPriorityOption) {
	useInternalUrl := config.Config.TransferUrlType == 2
	maxDownloadRate := config.Config.MaxDownloadRate
	maxUploadRate := config.Config.MaxUploadRate
	activeUser := GetActiveUser()
	panClient := activeUser.PanClient()
	panClient.DisableCache()

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

	var tasks []*syncdrive.SyncTask
	if defaultTask != nil {
		tasks = []*syncdrive.SyncTask{}
		tasks = append(tasks, defaultTask)
	}

	fmt.Println("启动同步备份进程")
	typeUrlStr := "默认链接"
	if useInternalUrl {
		typeUrlStr = "阿里ECS内部链接"
	}
	option := syncdrive.SyncOption{
		FileDownloadParallel:  fileDownloadParallel,
		FileUploadParallel:    fileUploadParallel,
		FileDownloadBlockSize: downloadBlockSize,
		FileUploadBlockSize:   uploadBlockSize,
		UseInternalUrl:        useInternalUrl,
		MaxDownloadRate:       maxDownloadRate,
		MaxUploadRate:         maxUploadRate,
		SyncPriority:          flag,
	}
	syncMgr := syncdrive.NewSyncTaskManager(activeUser, activeUser.DriveList.GetFileDriveId(), panClient, syncFolderRootPath, option)
	syncConfigFile := syncMgr.ConfigFilePath()
	if tasks != nil {
		syncConfigFile = "(使用命令行配置)"
	}
	fmt.Printf("备份配置文件：%s\n链接类型：%s\n下载并发：%d\n上传并发：%d\n下载分片大小：%s\n上传分片大小：%s\n",
		syncConfigFile, typeUrlStr, fileDownloadParallel, fileUploadParallel, converter.ConvertFileSize(downloadBlockSize, 2),
		converter.ConvertFileSize(uploadBlockSize, 2))
	if _, e := syncMgr.Start(tasks); e != nil {
		fmt.Println("启动任务失败：", e)
		return
	}

	_, ok := os.LookupEnv("ALIYUNPAN_DOCKER")
	if ok {
		// in docker container
		// 使用休眠以节省CPU资源
		for {
			time.Sleep(60 * time.Second)
		}
	} else {
		// in cmd mode
		c := ""
		fmt.Print("本命令不会退出，如需要结束同步备份进程请输入y，然后按Enter键进行停止：")
		for strings.ToLower(c) != "y" {
			fmt.Scan(&c)
		}
	}

	fmt.Println("正在停止同步备份任务，请稍等...")
	syncMgr.Stop()
}
