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
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/internal/plugins"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/library-go/requester/rio/speeds"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
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
		NoRapidUpload  bool
		ShowProgress   bool
		IsOverwrite    bool // 覆盖已存在的文件，如果同名文件已存在则移到回收站里
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
	cli.BoolFlag{
		Name:  "np",
		Usage: "no progress 不展示上传进度条",
	},
	cli.BoolFlag{
		Name:  "ow",
		Usage: "overwrite, 覆盖已存在的同名文件，注意已存在的文件会被移到回收站",
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
		Usage: "block size，上传分片大小，单位KB。推荐值：1024 ~ 10240",
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
	上传指定的文件夹或者文件，上传的文件将会保存到 <目标目录>.

  示例:
    1. 将本地的 C:\Users\Administrator\Desktop\1.mp4 上传到网盘 /视频 目录
    注意区别反斜杠 "\" 和 斜杠 "/" !!!
    aliyunpan upload C:/Users/Administrator/Desktop/1.mp4 /视频

    2. 将本地的 C:\Users\Administrator\Desktop\1.mp4 和 C:\Users\Administrator\Desktop\2.mp4 上传到网盘 /视频 目录
    aliyunpan upload C:/Users/Administrator/Desktop/1.mp4 C:/Users/Administrator/Desktop/2.mp4 /视频

    3. 将本地的 C:\Users\Administrator\Desktop 整个目录上传到网盘 /视频 目录
    aliyunpan upload C:/Users/Administrator/Desktop /视频

    4. 使用相对路径
    aliyunpan upload 1.mp4 /视频

    5. 覆盖上传，已存在的同名文件会被移到回收站
    aliyunpan upload -ow 1.mp4 /视频

    6. 将本地的 C:\Users\Administrator\Video 整个目录上传到网盘 /视频 目录，但是排除所有的.jpg文件
    aliyunpan upload -exn "\.jpg$" C:/Users/Administrator/Video /视频

    7. 将本地的 C:\Users\Administrator\Video 整个目录上传到网盘 /视频 目录，但是排除所有的.jpg文件和.mp3文件，每一个排除项就是一个exn参数
    aliyunpan upload -exn "\.jpg$" -exn "\.mp3$" C:/Users/Administrator/Video /视频

    8. 将本地的 C:\Users\Administrator\Video 整个目录上传到网盘 /视频 目录，但是排除所有的 @eadir 文件夹
    aliyunpan upload -exn "^@eadir$" C:/Users/Administrator/Video /视频

  参考：
    以下是典型的排除特定文件或者文件夹的例子，注意：参数值必须是正则表达式。在正则表达式中，^表示匹配开头，$表示匹配结尾。
    1)排除@eadir文件或者文件夹：-exn "^@eadir$"
    2)排除.jpg文件：-exn "\.jpg$"
    3)排除.号开头的文件：-exn "^\."
    4)排除~号开头的文件：-exn "^~"
    5)排除 myfile.txt 文件：-exn "^myfile.txt$"
`,
		Category: "阿里云盘",
		Before:   cmder.ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if c.NArg() < 2 {
				cli.ShowCommandHelp(c, c.Command.Name)
				return nil
			}

			subArgs := c.Args()
			RunUpload(subArgs[:c.NArg()-1], subArgs[c.NArg()-1], &UploadOptions{
				AllParallel:   c.Int("p"), // 多文件上传的时候，允许同时并行上传的文件数量
				Parallel:      1,          // 一个文件同时多少个线程并发上传的数量。阿里云盘只支持单线程按顺序进行文件part数据上传，所以只能是1
				MaxRetry:      c.Int("retry"),
				NoRapidUpload: c.Bool("norapid"),
				ShowProgress:  !c.Bool("np"),
				IsOverwrite:   c.Bool("ow"),
				DriveId:       parseDriveId(c),
				ExcludeNames:  c.StringSlice("exn"),
				BlockSize:     int64(c.Int("bs") * 1024),
			})
			return nil
		},
		Flags: UploadFlags,
	}
}

func CmdRapidUpload() cli.Command {
	return cli.Command{
		Name:      "rapidupload",
		Aliases:   []string{"ru"},
		Usage:     "手动秒传文件",
		UsageText: cmder.App().Name + " rapidupload \"aliyunpan://file.dmg|752FCCBFB2436A6FFCA3B287831D4FAA5654B07E|7005440|pan_folder\"",
		Description: `
	使用此功能秒传文件, 前提是知道文件的大小, sha1, 且网盘中存在一模一样的文件.
	上传的文件将会保存到网盘的目标目录。文件的秒传链接可以通过share或者export命令获取。

	链接格式说明：aliyunpan://文件名|sha1|文件大小|<相对路径>
    "相对路径" 可以为空，为空代表存储到网盘根目录

	示例:
	1. 如果秒传成功, 则保存到网盘路径 /pan_folder/file.dmg
	aliyunpan rapidupload "aliyunpan://file.dmg|752FCCBFB2436A6FFCA3B287831D4FAA5654B07E|7005440|pan_folder"

	2. 如果秒传成功, 则保存到网盘路径 /file.dmg
	aliyunpan rapidupload "aliyunpan://file.dmg|752FCCBFB2436A6FFCA3B287831D4FAA5654B07E|7005440|"

	3. 同时秒传多个文件，如果秒传成功, 则保存到网盘路径 /pan_folder/file.dmg, /pan_folder/file1.dmg
	aliyunpan rapidupload "aliyunpan://file.dmg|752FCCBFB2436A6FFCA3B287831D4FAA5654B07E|7005440|pan_folder" "aliyunpan://file1.dmg|752FCCBFB2436A6FFCA3B287831D4FAA5654B07E|7005440|pan_folder"
`,
		Category: "阿里云盘",
		Before:   cmder.ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if c.NArg() <= 0 {
				cli.ShowCommandHelp(c, c.Command.Name)
				return nil
			}
			subArgs := c.Args()
			RunRapidUpload(parseDriveId(c), c.Bool("ow"), subArgs, c.String("path"))
			return nil
		},
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "ow",
				Usage: "overwrite, 覆盖已存在的文件。已存在的文件会并移到回收站",
			},
			cli.StringFlag{
				Name:  "path",
				Usage: "存储到网盘目录，绝对路径，例如：/myfolder",
				Value: "",
			},
			cli.StringFlag{
				Name:  "driveId",
				Usage: "网盘ID",
				Value: "",
			},
		},
	}
}

// RunUpload 执行文件上传
func RunUpload(localPaths []string, savePath string, opt *UploadOptions) {
	activeUser := GetActiveUser()
	// pan token expired checker
	go func() {
		for {
			time.Sleep(time.Duration(1) * time.Minute)
			if RefreshTokenInNeed(activeUser) {
				logger.Verboseln("update access token for upload task")
			}
		}
	}()

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
		opt.AllParallel = config.MaxFileUploadParallelNum
	}

	if opt.Parallel <= 0 {
		opt.Parallel = 1
	}
	if opt.MaxRetry < 0 {
		opt.MaxRetry = DefaultUploadMaxRetry
	}
	opt.UseInternalUrl = config.Config.TransferUrlType == 2

	fmt.Printf("\n[0] 当前文件上传最大并发量为: %d, 上传分片大小为: %s\n", opt.AllParallel, converter.ConvertFileSize(opt.BlockSize, 2))

	savePath = activeUser.PathJoin(opt.DriveId, savePath)
	_, err1 := activeUser.PanClient().FileInfoByPath(opt.DriveId, savePath)
	if err1 != nil {
		fmt.Printf("警告: 上传文件, 获取云盘路径 %s 错误, %s\n", savePath, err1)
	}

	switch len(localPaths) {
	case 0:
		fmt.Printf("本地路径为空\n")
		return
	}

	// 打开上传状态
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

		pluginManger = plugins.NewPluginManager(config.GetConfigDir())
	)
	executor.SetParallel(opt.AllParallel)
	statistic.StartTimer() // 开始计时

	// 全局速度统计
	globalSpeedsStat := &speeds.Speeds{}

	// 获取当前插件
	plugin, _ := pluginManger.GetPlugin()

	// 遍历指定的文件并创建上传任务
	for _, curPath := range localPaths {
		var walkFunc filepath.WalkFunc
		var db panupload.SyncDb
		curPath = filepath.Clean(curPath)
		localPathDir := filepath.Dir(curPath)

		// 是否排除上传
		if isExcludeFile(curPath, opt) {
			fmt.Printf("排除文件: %s\n", curPath)
			continue
		}

		// 避免去除文件名开头的"."
		if localPathDir == "." {
			localPathDir = ""
		}

		if fi, err := os.Stat(curPath); err == nil && fi.IsDir() {
			//使用绝对路径避免异常
			dbpath, err := filepath.Abs(curPath)
			if err != nil {
				dbpath = curPath
			}
			dbpath += string(os.PathSeparator) + BackupMetaDirName
			if di, err := os.Stat(dbpath); err == nil && di.IsDir() {
				db, err = panupload.OpenSyncDb(dbpath+string(os.PathSeparator)+"db", BackupMetaBucketName)
				if db != nil {
					defer func(syncDb panupload.SyncDb) {
						db.Close()
					}(db)
				} else {
					fmt.Println(curPath, "同步数据库打开失败,跳过该目录的备份", err)
					continue
				}
			}
		}

		walkFunc = func(file string, fi os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if os.PathSeparator == '\\' {
				file = cmdutil.ConvertToWindowsPathSeparator(file)
			}

			// 是否排除上传
			if isExcludeFile(file, opt) {
				fmt.Printf("排除文件: %s\n", file)
				return filepath.SkipDir
			}

			if fi.Mode()&os.ModeSymlink != 0 { // 读取 symbol link
				err = WalkAllFile(file+string(os.PathSeparator), walkFunc)
				return err
			}

			subSavePath := strings.TrimPrefix(file, localPathDir)

			// 针对 windows 的目录处理
			if os.PathSeparator == '\\' {
				subSavePath = cmdutil.ConvertToUnixPathSeparator(subSavePath)
			}

			subSavePath = path.Clean(savePath + aliyunpan.PathSeparator + subSavePath)
			var ufm *panupload.UploadedFileMeta

			if db != nil {
				if ufm = db.Get(subSavePath); ufm.Size == fi.Size() && ufm.ModTime == fi.ModTime().Unix() {
					logger.Verbosef("文件未修改跳过:%s\n", file)
					return nil
				}
			}

			if fi.IsDir() { // 备份目录处理
				if strings.HasPrefix(fi.Name(), BackupMetaDirName) {
					return filepath.SkipDir
				}
				//不存在同步数据库时跳过
				if db == nil || ufm.FileId != "" {
					return nil
				}
				panClient := activeUser.PanClient()
				fmt.Println(subSavePath, "云盘文件夹预创建")
				//首先尝试直接创建文件夹
				if ufm = db.Get(path.Dir(subSavePath)); ufm.IsFolder == true && ufm.FileId != "" {
					rs, err := panClient.Mkdir(opt.DriveId, ufm.FileId, fi.Name())
					if err == nil && rs != nil && rs.FileId != "" {
						db.Put(subSavePath, &panupload.UploadedFileMeta{FileId: rs.FileId, IsFolder: true, ModTime: fi.ModTime().Unix(), ParentId: rs.ParentFileId})
						return nil
					}
				}
				rs, err := panClient.MkdirRecursive(opt.DriveId, "", "", 0, strings.Split(path.Clean(subSavePath), "/"))
				if err == nil && rs != nil && rs.FileId != "" {
					db.Put(subSavePath, &panupload.UploadedFileMeta{FileId: rs.FileId, IsFolder: true, ModTime: fi.ModTime().Unix(), ParentId: rs.ParentFileId})
					return nil
				}
				fmt.Println(subSavePath, "创建云盘文件夹失败", err)
				return filepath.SkipDir
			}

			// 插件回调
			if !fi.IsDir() { // 针对文件上传前进行回调
				pluginParam := &plugins.UploadFilePrepareParams{
					LocalFilePath:      file,
					LocalFileName:      fi.Name(),
					LocalFileSize:      fi.Size(),
					LocalFileType:      "file",
					LocalFileUpdatedAt: fi.ModTime().Format("2006-01-02 15:04:05"),
					DriveId:            activeUser.ActiveDriveId,
					DriveFilePath:      strings.TrimPrefix(strings.TrimPrefix(subSavePath, savePath), "/"),
				}
				if uploadFilePrepareResult, er := plugin.UploadFilePrepareCallback(plugins.GetContext(activeUser), pluginParam); er == nil && uploadFilePrepareResult != nil {
					if strings.Compare("yes", uploadFilePrepareResult.UploadApproved) != 0 {
						// skip upload this file
						fmt.Printf("插件禁止该文件上传: %s\n", file)
						return filepath.SkipDir
					}
					if uploadFilePrepareResult.DriveFilePath != "" {
						targetSavePanRelativePath := strings.TrimPrefix(uploadFilePrepareResult.DriveFilePath, "/")
						subSavePath = path.Clean(savePath + aliyunpan.PathSeparator + targetSavePanRelativePath)
						fmt.Printf("插件修改文件网盘保存路径为: %s\n", subSavePath)
					}
				}
			}

			taskinfo := executor.Append(&panupload.UploadTaskUnit{
				LocalFileChecksum: localfile.NewLocalFileEntity(file),
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
				FolderSyncDb:      db,
				UseInternalUrl:    opt.UseInternalUrl,
				GlobalSpeedsStat:  globalSpeedsStat,
			}, opt.MaxRetry)

			fmt.Printf("[%s] 加入上传队列: %s\n", taskinfo.Id(), file)
			return nil
		}
		if err := WalkAllFile(curPath, walkFunc); err != nil {
			fmt.Printf("警告: 遍历错误: %s\n", err)
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
				tb.Append([]string{item.Info.Id(), item.Unit.(*panupload.UploadTaskUnit).LocalFileChecksum.Path})
			}
			tb.Render()
		}
	}
	activeUser.DeleteCache(GetAllPathFolderByPath(savePath))
}

// 是否是排除上传的文件
func isExcludeFile(filePath string, opt *UploadOptions) bool {
	if opt == nil || len(opt.ExcludeNames) == 0 {
		return false
	}

	for _, pattern := range opt.ExcludeNames {
		fileName := path.Base(filePath)

		m, _ := regexp.MatchString(pattern, fileName)
		if m {
			return true
		}
	}
	return false
}

func WalkAllFile(dirPath string, walkFn filepath.WalkFunc) error {
	info, err := os.Lstat(dirPath)
	if err != nil {
		err = walkFn(dirPath, nil, err)
	} else {
		err = walkAllFile(dirPath, info, walkFn)
	}
	return err
}

func walkAllFile(dirPath string, info os.FileInfo, walkFn filepath.WalkFunc) error {
	if !info.IsDir() {
		return walkFn(dirPath, info, nil)
	}

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return walkFn(dirPath, nil, err)
	}
	for _, fi := range files {
		subFilePath := dirPath + "/" + fi.Name()
		err := walkFn(subFilePath, fi, err)
		if err != nil && err != filepath.SkipDir {
			return err
		}
		if fi.IsDir() {
			if err == filepath.SkipDir {
				continue
			}
			err := walkAllFile(subFilePath, fi, walkFn)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// RunRapidUpload 秒传
func RunRapidUpload(driveId string, isOverwrite bool, fileMetaList []string, savePanPath string) {
	activeUser := GetActiveUser()
	savePanPath = activeUser.PathJoin(driveId, savePanPath)

	if len(fileMetaList) == 0 {
		fmt.Println("秒传链接为空")
		return
	}

	items := []*RapidUploadItem{}
	// parse file meta strings
	for _, fileMeta := range fileMetaList {
		item, e := newRapidUploadItem(fileMeta)
		if e != nil {
			fmt.Println(e)
			continue
		}
		if item == nil {
			fmt.Println("秒传链接格式错误: ", fileMeta)
			continue
		}

		// pan path
		item.FilePath = path.Join(savePanPath, item.FilePath)

		// append
		items = append(items, item)
	}

	// upload one by one
	for _, item := range items {
		fmt.Println("准备秒传:", item.FilePath)
		if ee := doRapidUpload(driveId, isOverwrite, item); ee != nil {
			fmt.Println(ee)
		} else {
			fmt.Printf("秒传成功, 保存到网盘路径:%s\n", item.FilePath)
		}
	}
}

func doRapidUpload(driveId string, isOverwrite bool, item *RapidUploadItem) error {
	activeUser := GetActiveUser()
	panClient := activeUser.PanClient()

	var apierr *apierror.ApiError
	var rs *aliyunpan.MkdirResult
	var appCreateUploadFileParam *aliyunpan.CreateFileUploadParam
	var saveFilePath string

	panDir, panFileName := path.Split(item.FilePath)
	saveFilePath = item.FilePath
	if panDir != "/" {
		rs, apierr = panClient.MkdirRecursive(driveId, "", "", 0, strings.Split(path.Clean(panDir), "/"))
		if apierr != nil || rs.FileId == "" {
			return fmt.Errorf("创建云盘文件夹失败")
		}
	} else {
		rs = &aliyunpan.MkdirResult{}
		rs.FileId = aliyunpan.DefaultRootParentFileId
	}
	time.Sleep(time.Duration(2) * time.Second)

	if isOverwrite {
		// 标记覆盖旧同名文件
		// 检查同名文件是否存在
		efi, apierr := panClient.FileInfoByPath(driveId, saveFilePath)
		if apierr != nil && apierr.Code != apierror.ApiCodeFileNotFoundCode {
			return fmt.Errorf("检测同名文件失败，请稍后重试")
		}
		if efi != nil && efi.FileId != "" {
			// existed, delete it
			fileDeleteResult, err1 := panClient.FileDelete([]*aliyunpan.FileBatchActionParam{{DriveId: efi.DriveId, FileId: efi.FileId}})
			if err1 != nil || len(fileDeleteResult) == 0 {
				return fmt.Errorf("无法删除文件，请稍后重试")
			}
			time.Sleep(time.Duration(500) * time.Millisecond)
			if fileDeleteResult[0].Success {
				logger.Verboseln("检测到同名文件，已移动到回收站: ", saveFilePath)
			} else {
				return fmt.Errorf("无法删除文件，请稍后重试")
			}
		}
	}

	appCreateUploadFileParam = &aliyunpan.CreateFileUploadParam{
		DriveId:      driveId,
		Name:         panFileName,
		Size:         item.FileSize,
		ContentHash:  item.FileSha1,
		ParentFileId: rs.FileId,
	}
	uploadOpEntity, apierr := panClient.CreateUploadFile(appCreateUploadFileParam)
	if apierr != nil {
		return fmt.Errorf("创建秒传任务失败：" + apierr.Error())
	}

	if uploadOpEntity.RapidUpload {
		logger.Verboseln("秒传成功, 保存到网盘路径: ", path.Join(panDir, uploadOpEntity.FileName))
	} else {
		return fmt.Errorf("失败，文件未曾上传，无法秒传")
	}
	return nil
}
