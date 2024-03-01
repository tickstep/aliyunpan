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
	"fmt"
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/cmder/cmdtable"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/file/downloader"
	"github.com/tickstep/aliyunpan/internal/functions/pandownload"
	"github.com/tickstep/aliyunpan/internal/log"
	"github.com/tickstep/aliyunpan/internal/taskframework"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/aliyunpan/library/requester/transfer"
	"github.com/tickstep/library-go/converter"
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester/rio/speeds"
	"github.com/urfave/cli"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sync/atomic"
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
		UseInternalUrl       bool     // 是否使用内置链接
		ExcludeNames         []string // 排除的文件名，包括文件夹和文件。即这些文件/文件夹不进行下载，支持正则表达式
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
	通过 aliyunpan config set -savedir <savedir>, 自定义保存的目录。支持多个文件或目录下载，支持自动跳过下载重名的文件!

	示例:
	设置保存目录, 保存到 D:\Downloads
	注意区别反斜杠 "\" 和 斜杠 "/" !!!
	aliyunpan config set -savedir D:\\Downloads
	或者
	aliyunpan config set -savedir D:/Downloads

	下载 /我的资源/1.mp4
	aliyunpan download /我的资源/1.mp4

	下载 /我的资源 目录下面所有的mp4文件，使用通配符
	aliyunpan download /我的资源/*.mp4

	下载 /我的资源 整个目录!!
	aliyunpan download /我的资源

	下载 /我的资源 整个目录，但是排除所有的jpg文件
	aliyunpan download -exn "\.jpg$" /我的资源

	下载 /我的资源/1.mp4 并保存下载的文件到本地的 d:/panfile
	aliyunpan download --saveto d:/panfile /我的资源/1.mp4

  参考：
    以下是典型的排除特定文件或者文件夹的例子，注意：参数值必须是正则表达式。在正则表达式中，^表示匹配开头，$表示匹配结尾。
    1)排除@eadir文件或者文件夹：-exn "^@eadir$"
    2)排除.jpg文件：-exn "\.jpg$"
    3)排除.号开头的文件：-exn "^\."
    4)排除~号开头的文件：-exn "^~"
    5)排除 myfile.txt 文件：-exn "^myfile.txt$"
`,
		Category: "阿里云盘",
		Before:   ReloadConfigFunc,
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
				// 使用当前工作目录
				pwd, _ := os.Getwd()
				saveTo = path.Clean(pwd)
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
				ExcludeNames:         c.StringSlice("exn"),
			}

			// 获取下载文件锁，保证下载操作单实例
			//locker := filelocker.NewFileLocker(config.GetLockerDir() + "/aliyunpan-download")
			//if e := filelocker.LockFile(locker, 0755, true, 5*time.Second); e != nil {
			//	logger.Verboseln(e)
			//	fmt.Println("本应用其他实例正在执行下载，请先停止或者等待其完成")
			//	return nil
			//}

			RunDownload(c.Args(), do)

			// 释放文件锁
			//if locker != nil {
			//	filelocker.UnlockFile(locker)
			//}
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
			cli.StringSliceFlag{
				Name:  "exn",
				Usage: "exclude name，指定排除的文件夹或者文件的名称，被排除的文件不会进行下载，只支持正则表达式。支持同时排除多个名称，每一个名称就是一个exn参数",
				Value: nil,
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
	activeUser.PanClient().WebapiPanClient().EnableCache()
	activeUser.PanClient().WebapiPanClient().ClearCache()
	defer activeUser.PanClient().WebapiPanClient().DisableCache()
	// pan token expired checker
	continueFlag := int32(0)
	atomic.StoreInt32(&continueFlag, 0)
	defer func() {
		atomic.StoreInt32(&continueFlag, 1)
	}()
	go func(flag *int32) {
		for atomic.LoadInt32(flag) == 0 {
			time.Sleep(time.Duration(1) * time.Minute)
			if RefreshTokenInNeed(activeUser, config.Config.DeviceName) {
				logger.Verboseln("update access token for download task")
			}
		}
	}(&continueFlag)

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

	paths, err := makePathAbsolute(options.DriveId, paths...)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("\n[*] 注意：由于阿里云盘接口的限制，当前不支持>100M单个文件的下载。")
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

	// 下载记录器
	fileRecorder := log.NewFileRecorder(config.GetLogDir() + "/download_file_records.csv")

	// 处理队列
	for k := range paths {
		// 使用通配符匹配
		fileList, err2 := matchPathByShellPattern(options.DriveId, paths[k])
		if err2 != nil {
			fmt.Printf("获取文件出错，请稍后重试: %s\n", paths[k])
			continue
		}
		if fileList == nil || len(fileList) == 0 {
			// 文件不存在
			fmt.Printf("文件不存在: %s\n", paths[k])
			continue
		}
		for _, f := range fileList {
			newCfg := *cfg

			// 是否排除下载
			if utils.IsExcludeFile(f.Path, &newCfg.ExcludeNames) {
				fmt.Printf("排除文件: %s\n", f.Path)
				continue
			}

			// 匹配的文件
			unit := pandownload.DownloadTaskUnit{
				Cfg:                  &newCfg, // 复制一份新的cfg
				PanClient:            panClient.WebapiPanClient(),
				VerbosePrinter:       panCommandVerbose,
				PrintFormat:          downloadPrintFormat(options.Load),
				ParentTaskExecutor:   &executor,
				DownloadStatistic:    statistic,
				IsPrintStatus:        options.IsPrintStatus,
				IsExecutedPermission: options.IsExecutedPermission,
				IsOverwrite:          options.IsOverwrite,
				NoCheck:              options.NoCheck,
				FilePanPath:          f.Path,
				DriveId:              options.DriveId,
				GlobalSpeedsStat:     globalSpeedsStat,
				FileRecorder:         fileRecorder,
			}

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
			fmt.Printf("[%s] 加入下载队列: %s\n", info.Id(), f.Path)
		}
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
