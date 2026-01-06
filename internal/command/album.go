// Copyright (c) 2020 tickstep.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package command

import (
	"encoding/json"
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/cmder/cmdtable"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/file/downloader"
	"github.com/tickstep/aliyunpan/internal/functions/pandownload"
	"github.com/tickstep/aliyunpan/internal/global"
	"github.com/tickstep/aliyunpan/internal/taskframework"
	"github.com/tickstep/aliyunpan/internal/ui"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/aliyunpan/library/requester/transfer"
	"github.com/tickstep/library-go/converter"
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester"
	"github.com/tickstep/library-go/requester/rio/speeds"
	"github.com/urfave/cli"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func CmdAlbum() cli.Command {
	return cli.Command{
		Name:      "album",
		Aliases:   []string{"abm"},
		Usage:     "共享相册",
		UsageText: cmder.App().Name + " album",
		Category:  "阿里云盘",
		Before:    ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			cli.ShowCommandHelp(c, c.Command.Name)
			return nil
		},

		Subcommands: []cli.Command{
			{
				Name:      "list",
				Aliases:   []string{"ls"},
				Usage:     "展示共享相簿列表",
				UsageText: cmder.App().Name + " album list",
				Description: `
示例:
    展示共享相簿列表 
    aliyunpan album ls
`,
				Action: func(c *cli.Context) error {
					if config.Config.ActiveUser() == nil {
						fmt.Println("未登录账号")
						return nil
					}
					RunShareAlbumList()
					return nil
				},
				Flags: []cli.Flag{},
			},
			{
				Name:      "list-file",
				Aliases:   []string{"lf"},
				Usage:     "展示相簿中的文件",
				UsageText: cmder.App().Name + " album list-file",
				Description: `
展示相簿中文件，同名的相簿只会展示第一个符合条件的
示例:

    展示相簿中文件"我的相簿2022"
    aliyunpan album list-file "我的相簿2022"
`,
				Action: func(c *cli.Context) error {
					if config.Config.ActiveUser() == nil {
						fmt.Println("未登录账号")
						return nil
					}
					RunShareAlbumListFile(c.Args().Get(0))
					return nil
				},
				Flags: []cli.Flag{},
			},
			{
				Name:      "download-file",
				Aliases:   []string{"df"},
				Usage:     "下载相簿中的所有文件到本地",
				UsageText: cmder.App().Name + " album download-file",
				Description: `
下载相簿中的所有文件
示例:

    下载相簿 "我的相簿2022" 里面的所有文件
    aliyunpan album download-file 我的相簿2022
`,
				Action: func(c *cli.Context) error {
					if config.Config.ActiveUser() == nil {
						fmt.Println("未登录账号")
						return nil
					}
					subArgs := c.Args()
					if len(subArgs) == 0 {
						fmt.Println("请指定下载的相簿名称")
						return nil
					}

					// 处理saveTo
					var (
						saveTo string
					)
					if c.String("saveto") != "" {
						saveTo = filepath.Clean(c.String("saveto"))
					}

					do := &DownloadOptions{
						IsPrintStatus:        false,
						IsExecutedPermission: false,
						IsOverwrite:          c.Bool("ow"),
						SaveTo:               saveTo,
						Parallel:             0,
						Load:                 0,
						MaxRetry:             pandownload.DefaultDownloadMaxRetry,
						NoCheck:              false,
						ShowProgress:         !c.Bool("np"),
						DriveId:              parseDriveId(c),
						ExcludeNames:         []string{},
					}

					RunShareAlbumDownloadFile(c.Args(), do)
					return nil
				},
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "ow",
						Usage: "overwrite, 覆盖已存在的文件",
					},
					cli.StringFlag{
						Name:  "saveto",
						Usage: "将下载的文件直接保存到指定的目录",
					},
					cli.BoolFlag{
						Name:  "np",
						Usage: "no progress 不展示下载进度条",
					},
				},
			},
		},
	}
}

func RunShareAlbumList() {
	activeUser := GetActiveUser()
	records, err := activeUser.PanClient().OpenapiPanClient().ShareAlbumListGetAll()
	if err != nil {
		fmt.Printf("获取相簿列表失败: %s\n", err)
		return
	}

	tb := cmdtable.NewTable(os.Stdout)
	tb.SetHeader([]string{"#", "ALBUM_ID", "名称", "更新日期", "创建日期"})
	tb.SetColumnAlignment([]int{tablewriter.ALIGN_DEFAULT, tablewriter.ALIGN_DEFAULT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_DEFAULT, tablewriter.ALIGN_DEFAULT})
	for k, record := range records {
		tb.Append([]string{strconv.Itoa(k + 1), record.AlbumId, record.Name, record.UpdatedAtStr(), record.CreatedAtStr()})
	}
	tb.Render()
}

func getShareAlbumFromName(activeUser *config.PanUser, name string) *aliyunpan.AlbumEntity {
	records, err := activeUser.PanClient().OpenapiPanClient().ShareAlbumListGetAll()
	if err != nil {
		fmt.Printf("获取相簿列表失败: %s\n", err)
		return nil
	}

	for _, record := range records {
		if name == record.Name {
			return record
		}
	}
	return nil
}

func RunShareAlbumListFile(name string) {
	if len(name) == 0 {
		fmt.Printf("相簿名称不能为空\n")
		return
	}

	activeUser := GetActiveUser()
	record := getShareAlbumFromName(activeUser, name)
	if record == nil {
		return
	}

	fileList, er := activeUser.PanClient().OpenapiPanClient().ShareAlbumListFileGetAll(&aliyunpan.ShareAlbumListFileParam{
		AlbumId: record.AlbumId,
		Limit:   100,
	})
	if er != nil {
		fmt.Printf("获取相簿文件列表失败：%s\n", er)
		return
	}
	renderTable(opLs, false, "", fileList)
}

func RunShareAlbumDownloadFile(albumNames []string, options *DownloadOptions) {
	if len(albumNames) == 0 {
		fmt.Printf("请指定相簿名称\n")
		return
	}

	activeUser := GetActiveUser()
	if options == nil {
		options = &DownloadOptions{}
	}

	if options.MaxRetry < 0 {
		options.MaxRetry = pandownload.DefaultDownloadMaxRetry
	}
	options.IsExecutedPermission = false

	// 设置下载配置
	cfg := &downloader.Config{
		Mode:                       transfer.RangeGenMode_BlockSize,
		CacheSize:                  config.Config.CacheSize,
		BlockSize:                  MaxDownloadRangeSize,
		MaxRate:                    config.Config.MaxDownloadRate,
		InstanceStateStorageFormat: downloader.InstanceStateStorageFormatJSON,
		ShowProgress:               options.ShowProgress,
		ExcludeNames:               options.ExcludeNames,
	}
	if cfg.CacheSize == 0 {
		cfg.CacheSize = int(DownloadCacheSize)
	}

	// 设置下载最大并发量
	if options.Parallel < 1 {
		options.Parallel = config.Config.MaxDownloadParallel
		if options.Parallel == 0 {
			options.Parallel = config.DefaultFileDownloadParallelNum
		}
	}
	if options.Parallel > config.MaxFileDownloadParallelNum {
		options.Parallel = config.MaxFileDownloadParallelNum
	}
	// 设置单个文件下载分片线程数
	if options.SliceParallel < 1 {
		options.SliceParallel = 1
	}

	// 保存文件的本地根文件夹
	originSaveRootPath := ""
	if options.SaveTo != "" {
		originSaveRootPath = options.SaveTo
	} else {
		// 使用默认的保存路径
		originSaveRootPath = GetActiveUser().GetSavePath("")
	}
	fi, err1 := os.Stat(originSaveRootPath)
	if err1 != nil && !os.IsExist(err1) {
		os.MkdirAll(originSaveRootPath, 0777) // 首先在本地创建目录
	} else {
		if !fi.IsDir() {
			fmt.Println("本地保存路径不是文件夹，请删除或者创建对应的文件夹：", originSaveRootPath)
			return
		}
	}

	var (
		panClient = activeUser.PanClient()
	)
	cfg.MaxParallel = options.Parallel
	cfg.SliceParallel = options.SliceParallel

	var (
		executor = taskframework.TaskExecutor{
			IsFailedDeque: true, // 统计失败的列表
		}
		statistic = &pandownload.DownloadStatistic{}
	)
	// 配置执行器任务并发数，即同时下载文件并发数
	executor.SetParallel(cfg.MaxParallel)

	// 全局速度统计
	globalSpeedsStat := &speeds.Speeds{}

	var dashboard *ui.DownloadDashboard
	if options.ShowProgress && !options.IsPrintStatus && ui.IsTerminal(os.Stdout) {
		dashboard = ui.NewDownloadDashboard(cfg.MaxParallel, globalSpeedsStat, &ui.DownloadDashboardOptions{
			Title: "AliyunPan CLI - 下载中心",
		})
	}
	logf := func(format string, a ...interface{}) {
		if dashboard != nil {
			dashboard.Logf(format, a...)
			return
		}
		fmt.Printf(format, a...)
	}
	if dashboard != nil {
		dashboard.Logf("[0] 当前文件下载最大并发量为: %d, 下载缓存为: %s", options.Parallel, converter.ConvertFileSize(int64(cfg.CacheSize), 2))
	} else {
		fmt.Printf("\n[0] 当前文件下载最大并发量为: %d, 下载缓存为: %s\n\n", options.Parallel, converter.ConvertFileSize(int64(cfg.CacheSize), 2))
	}

	// 处理队列
	allShareAlbumList, err := activeUser.PanClient().OpenapiPanClient().ShareAlbumListGetAll()
	if err != nil {
		logf("获取相簿列表失败: %s\n", err)
		return
	}
	for k := range albumNames {
		var record *aliyunpan.AlbumEntity
		for _, album := range allShareAlbumList {
			if album.Name == albumNames[k] {
				record = album
				break
			}
		}
		if record == nil {
			continue
		}
		// 获取相簿下的所有文件
		fileList, er := activeUser.PanClient().OpenapiPanClient().ShareAlbumListFileGetAll(&aliyunpan.ShareAlbumListFileParam{
			AlbumId: record.AlbumId,
			Limit:   100,
		})
		if er != nil {
			logf("获取相簿文件出错，请稍后重试: %s\n", albumNames[k])
			continue
		}
		if fileList == nil || len(fileList) == 0 {
			logf("相簿里面没有文件: %s\n", albumNames[k])
			continue
		}

		// 相薄文件是没有子文件夹的，这个fileList就是该相册里面的全部文件了，所以不需要再遍历子文件夹了
		idx := 0
		for {
			if idx >= len(fileList) {
				break
			}
			f := fileList.Item(idx)
			idx += 1
			// 处理实况照片
			if f.IsAlbumLivePhotoFile() {
				// 如果是实况照片，则需要下载图片+视频两个文件
				// 获取下载链接
				durl, apierr := activeUser.PanClient().OpenapiPanClient().ShareAlbumGetFileDownloadUrl(&aliyunpan.ShareAlbumGetFileUrlParam{
					AlbumId: f.AlbumId,
					DriveId: f.DriveId,
					FileId:  f.FileId,
				})
				if apierr != nil {
					logger.Verbosef("ERROR: get album file download url error: %s\n", f.FileId)
					logf("下载照片失败: %s\n", f.FileName)
					continue
				}
				if durl.StreamsUrl != nil { // 实况图片(照片+视频)下载链接
					// 照片文件
					photoFile := cloneFileEntity(f)
					photoFileSize := int64(0)
					if durl.StreamsUrl.Heic != "" {
						beforeStr, _ := strings.CutSuffix(f.FileName, ".livp")
						photoFile.FileName = beforeStr + ".HEIC"
						photoFile.FileExtension = "heic"
						photoFileSize = getHttpDownloadFileSize(durl.StreamsUrl.Heic)
					} else if durl.StreamsUrl.Jpeg != "" {
						beforeStr, _ := strings.CutSuffix(f.FileName, ".livp")
						photoFile.FileName = beforeStr + ".JPG"
						photoFile.FileExtension = "jpg"
						photoFileSize = getHttpDownloadFileSize(durl.StreamsUrl.Jpeg)
					}
					photoFile.FileSize = photoFileSize
					fileList = append(fileList, photoFile)

					// 视频文件
					videoFile := cloneFileEntity(f)
					videoFile.FileSize = f.FileSize - photoFileSize
					if durl.StreamsUrl.Mov != "" {
						beforeStr, _ := strings.CutSuffix(f.FileName, ".livp")
						videoFile.FileName = beforeStr + ".MOV"
						videoFile.FileExtension = "mov"
					}
					fileList = append(fileList, videoFile)
				}
				continue
			}
			// 补全虚拟网盘路径，规则：/共享相册/<相簿名称>/文件名称
			f.Path = "/共享相册/" + albumNames[k] + "/" + f.FileName

			// 生成下载项
			newCfg := *cfg
			unit := pandownload.DownloadTaskUnit{
				Cfg:                  &newCfg, // 复制一份新的cfg
				PanClient:            panClient,
				VerbosePrinter:       panCommandVerbose,
				PrintFormat:          downloadPrintFormat(options.Load),
				ParentTaskExecutor:   &executor,
				DownloadStatistic:    statistic,
				IsPrintStatus:        options.IsPrintStatus,
				IsExecutedPermission: options.IsExecutedPermission,
				IsOverwrite:          options.IsOverwrite,
				NoCheck:              options.NoCheck,
				FilePanSource:        global.AlbumSource,
				FilePanPath:          f.Path,
				DriveId:              f.DriveId, // 一个相簿的文件会来自多个网盘（资源库/备份盘）
				GlobalSpeedsStat:     globalSpeedsStat,
				FileRecorder:         nil,
				UI:                   dashboard,
			}

			// 设置相簿文件信息
			unit.SetFileInfo(global.AlbumSource, f)

			// 设置储存的路径
			if options.SaveTo != "" {
				unit.OriginSaveRootPath = options.SaveTo
				unit.SavePath = filepath.Join(options.SaveTo, f.Path)
			} else {
				// 使用默认的保存路径
				unit.OriginSaveRootPath = GetActiveUser().GetSavePath("")
				unit.SavePath = GetActiveUser().GetSavePath(f.Path)
			}
			info := executor.Append(&unit, options.MaxRetry)
			if dashboard != nil {
				dashboard.RegisterTask(info.Id(), f.Path, f.FileSize, f.IsFile())
			}
			logf("[%s] 加入下载队列: %s\n", info.Id(), f.Path)
		}
	}

	// 开始计时
	statistic.StartTimer()

	// 开始执行
	if dashboard != nil {
		dashboard.Start()
	}
	executor.Execute()
	if dashboard != nil {
		dashboard.Close()
	}

	fmt.Printf("\n下载结束, 时间: %s, 数据总量: %s\n", utils.ConvertTime(statistic.Elapsed()), converter.ConvertFileSize(statistic.TotalSize(), 2))

	// 输出失败的文件列表
	failedList := executor.FailedDeque()
	if failedList.Size() != 0 {
		fmt.Printf("以下文件下载失败: \n")
		tb := cmdtable.NewTable(os.Stdout)
		for e := failedList.Shift(); e != nil; e = failedList.Shift() {
			item := e.(*taskframework.TaskInfoItem)
			tb.Append([]string{item.Info.Id(), item.Unit.(*pandownload.DownloadTaskUnit).FilePanPath})
		}
		tb.Render()
	}
}

func cloneFileEntity(entity *aliyunpan.FileEntity) *aliyunpan.FileEntity {
	data, err := json.Marshal(entity) // 序列化原始对象
	if err != nil {
		panic(err)
	}
	newEntity := &aliyunpan.FileEntity{}
	err = json.Unmarshal(data, newEntity) // 反序列化到新的对象中
	if err != nil {
		panic(err)
	}
	return newEntity
}

func getHttpDownloadFileSize(fileUrl string) int64 {
	client := requester.NewHTTPClient()
	client.SetKeepAlive(true)
	client.SetTimeout(10 * time.Minute)
	// header
	headers := map[string]string{
		"user-agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		"referer":    "https://www.aliyundrive.com/",
	}
	resp, err := client.Req("GET", fileUrl, nil, headers)
	if err != nil {
		return -1
	}
	fileSize := resp.ContentLength
	return fileSize
}
