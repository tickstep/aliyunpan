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
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/cmder/cmdtable"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/file/downloader"
	"github.com/tickstep/aliyunpan/internal/functions/pandownload"
	"github.com/tickstep/aliyunpan/internal/taskframework"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/aliyunpan/library/requester/transfer"
	"github.com/tickstep/library-go/converter"
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester/rio/speeds"
	"github.com/urfave/cli"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type (
	//DownloadOptions 下载可选参数
	DownloadOptions struct {
		IsPrintStatus        bool
		IsExecutedPermission bool
		IsOverwrite          bool
		SaveTo               string
		Parallel             int
		Load                 int
		MaxRetry             int
		NoCheck              bool
		ShowProgress         bool
		DriveId              string
		UseInternalUrl       bool // 是否使用内置链接
	}

	// LocateDownloadOption 获取下载链接可选参数
	LocateDownloadOption struct {
		FromPan bool
	}
)

var (
	// MaxDownloadRangeSize 文件片段最大值
	MaxDownloadRangeSize = 55 * converter.MB

	// DownloadCacheSize 默认每个线程下载缓存大小
	DownloadCacheSize = 64 * converter.KB
)

func CmdDownload() cli.Command {
	return cli.Command{
		Name:      "download",
		Aliases:   []string{"d"},
		Usage:     "下载文件/目录",
		UsageText: cmder.App().Name + " download <文件/目录路径1> <文件/目录2> <文件/目录3> ...",
		Description: `
	下载的文件默认保存到, 程序所在目录的 download/ 目录。支持软链接文件，包括Linux/macOS(ln命令)和Windows(mklink命令)创建的符号链接文件。
	通过 aliyunpan config set -savedir <savedir>, 自定义保存的目录。
	支持多个文件或目录下载.
	自动跳过下载重名的文件!

	示例:

	设置保存目录, 保存到 D:\Downloads
	注意区别反斜杠 "\" 和 斜杠 "/" !!!
	aliyunpan config set -savedir D:\\Downloads
	或者
	aliyunpan config set -savedir D:/Downloads

	下载 /我的资源/1.mp4
	aliyunpan d /我的资源/1.mp4

	下载 /我的资源 整个目录!!
	aliyunpan d /我的资源

    下载 /我的资源/1.mp4 并保存下载的文件到本地的 d:/panfile
	aliyunpan d --saveto d:/panfile /我的资源/1.mp4
`,
		Category: "阿里云盘",
		Before:   cmder.ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				cli.ShowCommandHelp(c, c.Command.Name)
				return nil
			}

			// 处理saveTo
			var (
				saveTo string
			)
			if c.Bool("save") {
				saveTo = "."
			} else if c.String("saveto") != "" {
				saveTo = filepath.Clean(c.String("saveto"))
			}

			do := &DownloadOptions{
				IsPrintStatus:        c.Bool("status"),
				IsExecutedPermission: c.Bool("x"),
				IsOverwrite:          c.Bool("ow"),
				SaveTo:               saveTo,
				Parallel:             c.Int("p"),
				Load:                 0,
				MaxRetry:             c.Int("retry"),
				NoCheck:              c.Bool("nocheck"),
				ShowProgress:         !c.Bool("np"),
				DriveId:              parseDriveId(c),
			}

			RunDownload(c.Args(), do)
			return nil
		},
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "ow",
				Usage: "overwrite, 覆盖已存在的文件",
			},
			cli.BoolFlag{
				Name:  "status",
				Usage: "输出所有线程的工作状态",
			},
			cli.BoolFlag{
				Name:  "save",
				Usage: "将下载的文件直接保存到当前工作目录",
			},
			cli.StringFlag{
				Name:  "saveto",
				Usage: "将下载的文件直接保存到指定的目录",
			},
			cli.BoolFlag{
				Name:  "x",
				Usage: "为文件加上执行权限, (windows系统无效)",
			},
			cli.IntFlag{
				Name:  "p",
				Usage: "指定同时进行下载文件的数量（取值范围:1 ~ 20）",
			},
			cli.IntFlag{
				Name:  "retry",
				Usage: "下载失败最大重试次数",
				Value: pandownload.DefaultDownloadMaxRetry,
			},
			cli.BoolFlag{
				Name:  "nocheck",
				Usage: "下载文件完成后不校验文件",
			},
			cli.BoolFlag{
				Name:  "np",
				Usage: "no progress 不展示下载进度条",
			},
			cli.StringFlag{
				Name:  "driveId",
				Usage: "网盘ID",
				Value: "",
			},
		},
	}
}

func downloadPrintFormat(load int) string {
	if load <= 1 {
		return pandownload.DefaultPrintFormat
	}
	return "\r[%s] ↓ %s/%s %s/s in %s, left %s ..."
}

// RunDownload 执行下载网盘内文件
func RunDownload(paths []string, options *DownloadOptions) {
	activeUser := GetActiveUser()
	activeUser.PanClient().EnableCache()
	activeUser.PanClient().ClearCache()
	defer activeUser.PanClient().DisableCache()
	// pan token expired checker
	go func() {
		for {
			time.Sleep(time.Duration(1) * time.Minute)
			if RefreshTokenInNeed(activeUser) {
				logger.Verboseln("update access token for download task")
			}
		}
	}()

	if options == nil {
		options = &DownloadOptions{}
	}

	if options.MaxRetry < 0 {
		options.MaxRetry = pandownload.DefaultDownloadMaxRetry
	}

	if runtime.GOOS == "windows" {
		// windows下不加执行权限
		options.IsExecutedPermission = false
	}

	// 设置下载配置
	cfg := &downloader.Config{
		Mode:                       transfer.RangeGenMode_BlockSize,
		CacheSize:                  config.Config.CacheSize,
		BlockSize:                  MaxDownloadRangeSize,
		MaxRate:                    config.Config.MaxDownloadRate,
		InstanceStateStorageFormat: downloader.InstanceStateStorageFormatJSON,
		ShowProgress:               options.ShowProgress,
		UseInternalUrl:             config.Config.TransferUrlType == 2,
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

	paths, err := matchPathByShellPattern(options.DriveId, paths...)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("\n[0] 当前文件下载最大并发量为: %d, 下载缓存为: %s\n\n", options.Parallel, converter.ConvertFileSize(int64(cfg.CacheSize), 2))

	var (
		panClient = activeUser.PanClient()
	)
	cfg.MaxParallel = options.Parallel

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

	// 处理队列
	for k := range paths {
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
			FilePanPath:          paths[k],
			DriveId:              options.DriveId,
			GlobalSpeedsStat:     globalSpeedsStat,
		}

		// 设置储存的路径
		if options.SaveTo != "" {
			unit.OriginSaveRootPath = options.SaveTo
			unit.SavePath = filepath.Join(options.SaveTo, filepath.Base(paths[k]))
		} else {
			// 使用默认的保存路径
			unit.OriginSaveRootPath = GetActiveUser().GetSavePath("")
			unit.SavePath = GetActiveUser().GetSavePath(paths[k])
		}
		info := executor.Append(&unit, options.MaxRetry)
		fmt.Printf("[%s] 加入下载队列: %s\n", info.Id(), paths[k])
	}

	// 开始计时
	statistic.StartTimer()

	// 开始执行
	executor.Execute()

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
