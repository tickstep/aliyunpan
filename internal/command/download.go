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
	"github.com/tickstep/aliyunpan/internal/global"
	"github.com/tickstep/aliyunpan/internal/log"
	"github.com/tickstep/aliyunpan/internal/taskframework"
	"github.com/tickstep/aliyunpan/internal/ui"
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
	"sort"
)

type (
	//DownloadOptions 下载可选参数
	DownloadOptions struct {
		// DownloadActionId 下载动作ID，唯一标识一次下载动作的ID。这个ID每次下载任务启动的时候会自动生成，同一次下载任务下载的文件，这个ID都是同一个值。
		DownloadActionId     string
		IsPrintStatus        bool
		IsExecutedPermission bool
		IsOverwrite          bool
		SaveTo               string
		Parallel             int // 文件下载最大线程数
		SliceParallel        int // 单个文件分片下载最大线程数
		Load                 int
		MaxRetry             int
		NoCheck              bool
		ShowProgress         bool
		DriveId              string
		ExcludeNames         []string // 排除的文件名，包括文件夹和文件。即这些文件/文件夹不进行下载，支持正则表达式
		IsMultiUserDownload  bool     // 是否启用多用户联合下载
		IsUseUIDashboard     bool     // 是否使用UI下载面板显示下载进度
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
	
	使用多用户联合下载 /我的资源/1.mp4 文件。必须保证所有登录的用户在相同的网盘（备份盘/资源盘）下，相同的路径下，有相同的文件
	aliyunpan download /我的资源/1.mp4 -md
	
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
				DownloadActionId:     utils.UuidStr(),
				IsPrintStatus:        c.Bool("status"),
				IsExecutedPermission: c.Bool("x"),
				IsOverwrite:          c.Bool("ow"),
				SaveTo:               saveTo,
				Parallel:             c.Int("p"),
				SliceParallel:        3,
				Load:                 0,
				MaxRetry:             c.Int("retry"),
				NoCheck:              c.Bool("nocheck"),
				ShowProgress:         !c.Bool("np"),
				DriveId:              parseDriveId(c),
				ExcludeNames:         c.StringSlice("exn"),
				IsMultiUserDownload:  c.Bool("md"),
				IsUseUIDashboard:     c.Bool("ui"),
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
				Usage: "parallel,指定同时进行下载文件的数量（取值范围:1 ~ 3）",
				Value: 1,
			},
			//cli.IntFlag{
			//	Name:  "sp",
			//	Usage: "slice parallel,指定单个文件下载的最大线程(分片)数（取值范围:1 ~ 3）",
			//	Value: 1,
			//},
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
			cli.BoolFlag{
				Name:  "md",
				Usage: "(BETA) Multi-User Download，使用多用户联合下载，可以对单一文件叠加所有登录用户的下载速度",
			},
			cli.BoolFlag{
				Name:  "ui",
				Usage: "(BETA) 使用UI面板显示下载详情和进度，更加直观和友好",
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
	activeUser.PanClient().OpenapiPanClient().EnableCache()
	activeUser.PanClient().OpenapiPanClient().ClearCache()
	defer activeUser.PanClient().OpenapiPanClient().DisableCache()

	//// pan token expired checker
	//continueFlag := int32(0)
	//atomic.StoreInt32(&continueFlag, 0)
	//defer func() {
	//	atomic.StoreInt32(&continueFlag, 1)
	//}()
	//go func(flag *int32) {
	//	for atomic.LoadInt32(flag) == 0 {
	//		time.Sleep(time.Duration(1) * time.Minute)
	//		if RefreshWebTokenInNeed(activeUser, config.Config.DeviceName) {
	//			logger.Verboseln("update access token for download task")
	//		}
	//	}
	//}(&continueFlag)

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

	paths, err := makePathAbsolute(options.DriveId, paths...)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 多用户下载的辅助账号列表
	var subPanClientList []*config.PanClient
	if options.IsMultiUserDownload { // 多用户下载
		c := config.Config
		for _, u := range config.Config.UserList {
			if u.UserId == activeUser.UserId {
				// 当前登录用户，作为主用户，跳过
				continue
			}
			// 初始化客户端
			user, err := config.SetupUserByCookie(u.OpenapiToken, u.WebapiToken,
				u.TicketId, u.UserId,
				c.DeviceId, c.DeviceName,
				c.ClientId, c.ClientSecret)
			if err != nil {
				logger.Verboseln("setup user error")
				continue
			}
			if subPanClientList == nil {
				subPanClientList = []*config.PanClient{}
			}
			subPanClientList = append(subPanClientList, user.PanClient())
		}

		if subPanClientList == nil || len(subPanClientList) == 0 {
			fmt.Printf("\n当前登录用户只有一个，无法启用多用户联合下载\n")
			subPanClientList = nil
		}
	}
	if subPanClientList != nil || len(subPanClientList) > 0 {
		// 已启用多用户下载
		userCount := len(subPanClientList) + 1
		fmt.Printf("\n*** 已启用多用户联合下载，用户数: %d ***\n", userCount)
		// 多用户下载，并发数必须为1，以获得最大下载速度
		options.Parallel = 1
		// 阿里OpenAPI规定：文件分片下载的并发数为3，即某用户使用 App 时，可以同时下载 1 个文件的 3 个分片，或者同时下载 3 个文件的各 1 个分片。
		options.SliceParallel = userCount * 3
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

	// 下载统计UI面板
	var dashboard *ui.DownloadDashboard = nil
	if options.IsUseUIDashboard && options.ShowProgress && !options.IsPrintStatus && ui.IsTerminal(os.Stdout) {
		dashboard = ui.NewDownloadDashboard(cfg.MaxParallel, globalSpeedsStat, &ui.DownloadDashboardOptions{
			Title:       "下载统计UI面板",
			ActiveSlots: 3,  // 下载文件进度展示，最大同时展示3个
			MaxHistory:  50, // 下载日志显示，最多同时显示50条，这个会按照窗口大小进行自适应显示
		})
	}
	logf := func(format string, a ...interface{}) {
		if dashboard != nil {
			dashboard.Logf(format, a...)
			return
		}
		fmt.Printf(format, a...)
	}

	// 下载记录器
	fileRecorder := log.NewFileRecorder(config.GetLogDir() + "/download_file_records.csv")

	// 输出当前下载配置信息
	logf("\n[0] 当前文件下载最大并发量为: %d, 单文件下载分片线程数为: %d, 下载缓存为: %s\n", options.Parallel, options.SliceParallel, converter.ConvertFileSize(int64(cfg.CacheSize), 2))

	// 处理队列
	for k := range paths {
		// 使用通配符匹配
		fileList, err2 := matchPathByShellPattern(options.DriveId, paths[k])
		if err2 != nil {
			logf("获取文件出错，请稍后重试: %s\n", paths[k])
			continue
		}
		if fileList == nil || len(fileList) == 0 {
			// 文件不存在
			logf("文件不存在: %s\n", paths[k])
			continue
		}
		// 排序，按名称排序，从小到大
		sort.Slice(fileList, func(i, j int) bool {
			return fileList[i].FileName < fileList[j].FileName
		})
		// 逐一下载
		for _, f := range fileList {
			newCfg := *cfg

			// 是否排除下载
			if utils.IsExcludeFile(f.Path, &newCfg.ExcludeNames) {
				logf("排除文件: %s\n", f.Path)
				continue
			}

			// 匹配的文件
			unit := pandownload.DownloadTaskUnit{
				DownloadActionId:     options.DownloadActionId,
				Cfg:                  &newCfg, // 复制一份新的cfg
				PanClient:            panClient,
				SubPanClientList:     subPanClientList,
				VerbosePrinter:       panCommandVerbose,
				PrintFormat:          downloadPrintFormat(options.Load),
				ParentTaskExecutor:   &executor,
				DownloadStatistic:    statistic,
				IsPrintStatus:        options.IsPrintStatus,
				IsExecutedPermission: options.IsExecutedPermission,
				IsOverwrite:          options.IsOverwrite,
				NoCheck:              options.NoCheck,
				FilePanSource:        global.FileSource,
				FilePanPath:          f.Path,
				DriveId:              options.DriveId,
				GlobalSpeedsStat:     globalSpeedsStat,
				FileRecorder:         fileRecorder,
				UI:                   dashboard,
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
			if dashboard != nil {
				dashboard.RegisterTask(info.Id(), f.Path, f.FileSize, f.IsFile())
			}
			logf("[%s] 加入下载队列: %s\n", info.Id(), f.Path)
		}
	}

	// 开始计时
	statistic.StartTimer()

	// 启动UI面板显示
	if dashboard != nil {
		dashboard.Start()
	}

	// 开始执行
	executor.Execute()

	// 关闭UI面板
	if dashboard != nil {
		dashboard.Close()
	}

	// 完成下载，输出统计结果
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
