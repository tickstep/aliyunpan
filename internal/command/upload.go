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
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/internal/log"
	"github.com/tickstep/aliyunpan/internal/plugins"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/library-go/requester/rio/speeds"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tickstep/library-go/logger"

	"github.com/urfave/cli"

	"github.com/tickstep/aliyunpan/cmder/cmdutil"

	"github.com/oleiade/lane"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan/cmder/cmdtable"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/functions/panupload"
	"github.com/tickstep/aliyunpan/internal/localfile"
	"github.com/tickstep/aliyunpan/internal/taskframework"
	"github.com/tickstep/library-go/converter"
)

const (
	// DefaultUploadMaxAllParallel 默认所有文件并发上传数量，即可以同时并发上传多少个文件
	DefaultUploadMaxAllParallel = 1
	// DefaultUploadMaxRetry 默认上传失败最大重试次数
	DefaultUploadMaxRetry = 3
)

type (
	// UploadOptions 上传可选项
	UploadOptions struct {
		AllParallel    int // 所有文件并发上传数量，即可以同时并发上传多少个文件
		Parallel       int // 单个文件并发上传数量
		MaxRetry       int
		MaxTimeoutSec  int // http请求超时时间，单位秒
		NoRapidUpload  bool
		ShowProgress   bool
		IsOverwrite    bool // 覆盖已存在的文件，如果同名文件已存在则移到回收站里
		IsSkipSameName bool // 跳过已存在的文件，即使文件内容不一致(不检查SHA1)
		DriveId        string
		ExcludeNames   []string // 排除的文件名，包括文件夹和文件。即这些文件/文件夹不进行上传，支持正则表达式
		BlockSize      int64    // 分片大小
		UseInternalUrl bool     // 是否使用内置链接
	}
)

var UploadFlags = []cli.Flag{
	cli.IntFlag{
		Name:  "p",
		Usage: "本次操作文件上传并发数量，即可以同时并发上传多少个文件。0代表跟从配置文件设置（取值范围:1 ~ 20）",
		Value: 0,
	},
	cli.IntFlag{
		Name:  "retry",
		Usage: "上传失败最大重试次数",
		Value: DefaultUploadMaxRetry,
	},
	cli.IntFlag{
		Name:  "timeout",
		Usage: "上传请求超时时间，单位为秒。当遇到网络不好导致上传超时可以尝试调大该值，建议设置30秒以上",
	},
	cli.BoolFlag{
		Name:  "np",
		Usage: "no progress 不展示上传进度条",
	},
	cli.BoolFlag{
		Name:  "ow",
		Usage: "overwrite, 覆盖已存在的同名文件，注意已存在的文件会被移到回收站",
	},
	cli.BoolFlag{
		Name:  "skip",
		Usage: "skip same name, 跳过已存在的同名文件，即使文件内容不一致(不检查SHA1)",
	},
	cli.BoolFlag{
		Name:  "norapid",
		Usage: "不检测秒传。跳过费时的SHA1计算直接上传",
	},
	cli.StringFlag{
		Name:  "driveId",
		Usage: "网盘ID",
		Value: "",
	},
	cli.StringSliceFlag{
		Name:  "exn",
		Usage: "exclude name，指定排除的文件夹或者文件的名称，只支持正则表达式。支持同时排除多个名称，每一个名称就是一个exn参数",
		Value: nil,
	},
	cli.IntFlag{
		Name:  "bs",
		Usage: "block size，上传分片大小，单位KB。推荐值：1024 ~ 10240。当上传极大单文件时候请适当调高该值",
		Value: 10240,
	},
}

func CmdUpload() cli.Command {
	return cli.Command{
		Name:      "upload",
		Aliases:   []string{"u"},
		Usage:     "上传文件/目录",
		UsageText: cmder.App().Name + " upload <本地文件/目录的路径1> <文件/目录2> <文件/目录3> ... <目标目录>",
		Description: `
	上传指定的文件夹或者文件，上传的文件将会保存到 <目标目录>。支持软链接文件，包括Linux/macOS(ln命令)和Windows(mklink命令)创建的符号链接文件。

  示例:
    1. 将本地的 C:\Users\Administrator\Desktop\1.mp4 上传到网盘 /视频 目录
    注意区别反斜杠 "\" 和 斜杠 "/" !!!
    aliyunpan upload C:/Users/Administrator/Desktop/1.mp4 /视频

    2. 将本地的 C:\Users\Administrator\Desktop\1.mp4 和 C:\Users\Administrator\Desktop\2.mp4 上传到网盘 /视频 目录
    aliyunpan upload C:/Users/Administrator/Desktop/1.mp4 C:/Users/Administrator/Desktop/2.mp4 /视频

    3. 将本地的 C:\Users\Administrator\Desktop 整个目录上传到网盘 /视频 目录
    aliyunpan upload C:/Users/Administrator/Desktop /视频

    4. 将本地 200GB 极大文件 C:\Users\Administrator\Desktop\1.mp4 上传到网盘 /视频 目录，需要调高上传分片大小
    aliyunpan upload -bs 30720 C:/Users/Administrator/Desktop/1.mp4 /视频

    5. 使用相对路径
    aliyunpan upload 1.mp4 /视频

    6. 覆盖上传，已存在的同名文件会被移到回收站
    aliyunpan upload -ow 1.mp4 /视频

    7. 将本地的 C:\Users\Administrator\Video 整个目录上传到网盘 /视频 目录，但是排除所有的.jpg文件
    aliyunpan upload -exn "\.jpg$" C:/Users/Administrator/Video /视频

    8. 将本地的 C:\Users\Administrator\Video 整个目录上传到网盘 /视频 目录，但是排除所有的.jpg文件和.mp3文件，每一个排除项就是一个exn参数
    aliyunpan upload -exn "\.jpg$" -exn "\.mp3$" C:/Users/Administrator/Video /视频

    9. 将本地的 C:\Users\Administrator\Video 整个目录上传到网盘 /视频 目录，但是排除所有的 @eadir 文件夹
    aliyunpan upload -exn "^@eadir$" C:/Users/Administrator/Video /视频

    10. 跳过已存在的同名文件，即使文件内容不一致(不检查SHA1)
    aliyunpan upload -skip 1.mp4 /视频

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
			if c.NArg() < 2 {
				cli.ShowCommandHelp(c, c.Command.Name)
				return nil
			}

			subArgs := c.Args()

			timeout := 0
			if c.IsSet("timeout") {
				timeout = c.Int("timeout")
				if timeout < 0 {
					timeout = 0
				}
			}

			// 获取上传文件锁，保证上传操作单实例
			//locker := filelocker.NewFileLocker(config.GetLockerDir() + "/aliyunpan-upload")
			//if e := filelocker.LockFile(locker, 0755, true, 5*time.Second); e != nil {
			//	logger.Verboseln(e)
			//	fmt.Println("本应用其他实例正在执行上传，请先停止或者等待其完成")
			//	return nil
			//}

			RunUpload(subArgs[:c.NArg()-1], subArgs[c.NArg()-1], &UploadOptions{
				AllParallel:    c.Int("p"), // 多文件上传的时候，允许同时并行上传的文件数量
				Parallel:       1,          // 一个文件同时多少个线程并发上传的数量。阿里云盘只支持单线程按顺序进行文件part数据上传，所以只能是1
				MaxRetry:       c.Int("retry"),
				MaxTimeoutSec:  timeout,
				NoRapidUpload:  c.Bool("norapid"),
				ShowProgress:   !c.Bool("np"),
				IsOverwrite:    c.Bool("ow"),
				IsSkipSameName: c.Bool("skip"),
				DriveId:        parseDriveId(c),
				ExcludeNames:   c.StringSlice("exn"),
				BlockSize:      int64(c.Int("bs") * 1024),
			})

			// 释放文件锁
			//if locker != nil {
			//	filelocker.UnlockFile(locker)
			//}
			return nil
		},
		Flags: UploadFlags,
	}
}

// RunUpload 执行文件上传
func RunUpload(localPaths []string, savePath string, opt *UploadOptions) {
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
	//			logger.Verboseln("update access token for upload task")
	//		}
	//	}
	//}(&continueFlag)

	if opt == nil {
		opt = &UploadOptions{}
	}

	// 检测opt
	if opt.AllParallel <= 0 {
		opt.AllParallel = config.Config.MaxUploadParallel
		if opt.AllParallel == 0 {
			opt.AllParallel = config.DefaultFileUploadParallelNum
		}
	}
	if opt.AllParallel > config.MaxFileUploadParallelNum {
		fmt.Printf("警告: 当前上传文件并发数过大，可能会被阿里风控导致上传失败，建议调小。\n")
	}

	if opt.Parallel <= 0 {
		opt.Parallel = 1
	}
	if opt.MaxRetry < 0 {
		opt.MaxRetry = DefaultUploadMaxRetry
	}
	opt.UseInternalUrl = config.Config.TransferUrlType == 2

	// 超时时间
	if opt.MaxTimeoutSec > 0 {
		activeUser.PanClient().OpenapiPanClient().SetTimeout(time.Duration(opt.MaxTimeoutSec) * time.Second)
	}

	fmt.Printf("\n[0] 当前文件上传最大并发量为: %d, 上传分片大小为: %s\n", opt.AllParallel, converter.ConvertFileSize(opt.BlockSize, 2))

	savePath = activeUser.PathJoin(opt.DriveId, savePath)
	_, err1 := activeUser.PanClient().OpenapiPanClient().FileInfoByPath(opt.DriveId, savePath)
	if err1 != nil {
		fmt.Printf("警告: 上传文件, 获取云盘路径 %s 错误, %s\n", savePath, err1)
	}

	switch len(localPaths) {
	case 0:
		fmt.Printf("本地路径为空\n")
		return
	}

	// 打开上传状态数据库
	uploadDatabase, err := panupload.NewUploadingDatabase()
	if err != nil {
		fmt.Printf("打开上传未完成数据库错误: %s\n", err)
		return
	}
	defer uploadDatabase.Close()

	var (
		// 使用 task framework
		executor = &taskframework.TaskExecutor{
			IsFailedDeque: true, // 失败统计
		}
		// 统计
		statistic = &panupload.UploadStatistic{}

		folderCreateMutex = &sync.Mutex{}

		pluginManger = plugins.NewPluginManager(config.GetPluginDir())
	)
	executor.SetParallel(opt.AllParallel)
	statistic.StartTimer() // 开始计时

	// 全局速度统计
	globalSpeedsStat := &speeds.Speeds{}

	// 获取当前插件
	plugin, _ := pluginManger.GetPlugin()

	// 上传记录器
	fileRecorder := log.NewFileRecorder(config.GetLogDir() + "/upload_file_records.csv")

	// 遍历指定的文件并创建上传任务
	for _, curPath := range localPaths {
		var walkFunc localfile.MyWalkFunc
		curPath = filepath.Clean(curPath)
		localPathDir := filepath.Dir(curPath)

		// 是否排除上传
		if utils.IsExcludeFile(curPath, &opt.ExcludeNames) {
			fmt.Printf("排除文件: %s\n", curPath)
			continue
		}

		// 避免去除文件名开头的"."
		if localPathDir == "." {
			localPathDir = ""
		}

		walkFunc = func(file localfile.SymlinkFile, fi os.FileInfo, err error) error {
			if err != nil {
				// skip this error file and continue recurse
				logger.Verboseln("upload process file: ", file, " error: ", err)
				return nil
			}
			if os.PathSeparator == '\\' {
				file.LogicPath = cmdutil.ConvertToWindowsPathSeparator(file.LogicPath)
				file.RealPath = cmdutil.ConvertToWindowsPathSeparator(file.RealPath)
			}

			// 是否排除上传
			if utils.IsExcludeFile(file.LogicPath, &opt.ExcludeNames) {
				fmt.Printf("排除文件: %s\n", file.LogicPath)
				return filepath.SkipDir
			}

			subSavePath := strings.TrimPrefix(file.LogicPath, localPathDir)

			// 针对 windows 的目录处理
			if os.PathSeparator == '\\' {
				subSavePath = cmdutil.ConvertToUnixPathSeparator(subSavePath)
			}
			subSavePath = path.Clean(savePath + aliyunpan.PathSeparator + subSavePath)

			// 插件回调
			ft := "file"
			if fi.IsDir() {
				ft = "folder"
			}
			pluginParam := &plugins.UploadFilePrepareParams{
				LocalFilePath:      file.LogicPath,
				LocalFileName:      fi.Name(),
				LocalFileSize:      fi.Size(),
				LocalFileType:      ft,
				LocalFileUpdatedAt: fi.ModTime().Format("2006-01-02 15:04:05"),
				DriveId:            activeUser.ActiveDriveId,
				DriveFilePath:      strings.TrimPrefix(strings.TrimPrefix(subSavePath, savePath), "/"),
			}
			if uploadFilePrepareResult, er := plugin.UploadFilePrepareCallback(plugins.GetContext(activeUser), pluginParam); er == nil && uploadFilePrepareResult != nil {
				if strings.Compare("yes", uploadFilePrepareResult.UploadApproved) != 0 {
					// skip upload this file
					fmt.Printf("插件禁止该文件上传: %s\n", file.LogicPath)
					return filepath.SkipDir
				}
				if uploadFilePrepareResult.DriveFilePath != "" {
					targetSavePanRelativePath := strings.TrimPrefix(uploadFilePrepareResult.DriveFilePath, "/")
					subSavePath = path.Clean(savePath + aliyunpan.PathSeparator + targetSavePanRelativePath)
					fmt.Printf("插件修改文件网盘保存路径为: %s\n", subSavePath)
				}
			}

			// 创建对应的文件上传任务
			// 上传里面的文件会创建对应的缺失文件夹
			if !fi.IsDir() {
				taskinfo := executor.Append(&panupload.UploadTaskUnit{
					LocalFileChecksum: localfile.NewLocalSymlinkFileEntity(file),
					SavePath:          subSavePath,
					DriveId:           opt.DriveId,
					PanClient:         activeUser.PanClient(),
					UploadingDatabase: uploadDatabase,
					FolderCreateMutex: folderCreateMutex,
					Parallel:          opt.Parallel,
					NoRapidUpload:     opt.NoRapidUpload,
					BlockSize:         opt.BlockSize,
					UploadStatistic:   statistic,
					ShowProgress:      opt.ShowProgress,
					IsOverwrite:       opt.IsOverwrite,
					IsSkipSameName:    opt.IsSkipSameName,
					UseInternalUrl:    opt.UseInternalUrl,
					GlobalSpeedsStat:  globalSpeedsStat,
					FileRecorder:      fileRecorder,
				}, opt.MaxRetry)
				fmt.Printf("[%s] 加入上传队列: %s\n", taskinfo.Id(), file.LogicPath)
			} else {
				// 创建文件夹
				// 这样空文件夹也可以正确上传
				saveFilePath := subSavePath
				if saveFilePath != "/" {
					folderCreateMutex.Lock()
					fmt.Printf("正在检测和创建云盘文件夹: %s\n", saveFilePath)
					_, apierr1 := activeUser.PanClient().OpenapiPanClient().FileInfoByPath(opt.DriveId, saveFilePath)
					time.Sleep(1 * time.Second)
					if apierr1 != nil && apierr1.Code == apierror.ApiCodeFileNotFoundCode {
						logger.Verbosef("%s 创建云盘文件夹: %s\n", utils.NowTimeStr(), saveFilePath)
						rs, apierr := activeUser.PanClient().OpenapiPanClient().MkdirByFullPath(opt.DriveId, saveFilePath)
						if apierr != nil || rs.FileId == "" {
							fmt.Printf("创建云盘文件夹失败: %s\n", saveFilePath)
						}
					}
					folderCreateMutex.Unlock()
				}
			}
			return nil
		}

		file := localfile.NewSymlinkFile(curPath)
		if err = localfile.WalkAllFile(file, walkFunc); err != nil {
			if err != filepath.SkipDir {
				fmt.Printf("警告: 遍历错误: %s\n", err)
			}
		}
	}

	// 执行上传任务
	var failedList []*lane.Deque
	executor.Execute()
	failed := executor.FailedDeque()
	if failed.Size() > 0 {
		failedList = append(failedList, failed)
	}

	fmt.Printf("\n")
	fmt.Printf("上传结束, 时间: %s, 数据总量: %s\n", utils.ConvertTime(statistic.Elapsed()), converter.ConvertFileSize(statistic.TotalSize(), 2))

	// 输出上传失败的文件列表
	for _, failed := range failedList {
		if failed.Size() != 0 {
			fmt.Printf("以下文件上传失败: \n")
			tb := cmdtable.NewTable(os.Stdout)
			for e := failed.Shift(); e != nil; e = failed.Shift() {
				item := e.(*taskframework.TaskInfoItem)
				tb.Append([]string{item.Info.Id(), item.Unit.(*panupload.UploadTaskUnit).LocalFileChecksum.Path.LogicPath})
			}
			tb.Render()
		}
	}
	activeUser.DeleteCache(GetAllPathFolderByPath(savePath))
}
