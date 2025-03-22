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
package pandownload

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan/cmder/cmdtable"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/file/downloader"
	"github.com/tickstep/aliyunpan/internal/functions"
	"github.com/tickstep/aliyunpan/internal/global"
	"github.com/tickstep/aliyunpan/internal/localfile"
	"github.com/tickstep/aliyunpan/internal/log"
	"github.com/tickstep/aliyunpan/internal/plugins"
	"github.com/tickstep/aliyunpan/internal/taskframework"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/aliyunpan/library/requester/transfer"
	"github.com/tickstep/library-go/converter"
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester/rio/speeds"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type (
	// DownloadTaskUnit 下载的任务单元
	DownloadTaskUnit struct {
		// DownloadActionId 下载动作ID，唯一标识一次下载动作的ID。这个ID每次下载任务启动的时候会自动生成，同一次下载任务下载的文件，这个ID都是同一个值。
		DownloadActionId string
		taskInfo         *taskframework.TaskInfo // 任务信息

		Cfg                *downloader.Config
		PanClient          *config.PanClient
		SubPanClientList   []*config.PanClient // 辅助下载子账号列表
		ParentTaskExecutor *taskframework.TaskExecutor

		DownloadStatistic *DownloadStatistic // 下载统计
		GlobalSpeedsStat  *speeds.Speeds     // 全局速度统计

		// 可选项
		VerbosePrinter       *logger.CmdVerbose
		PrintFormat          string
		IsPrintStatus        bool // 是否输出各个下载线程的详细信息
		IsExecutedPermission bool // 下载成功后是否加上执行权限
		IsOverwrite          bool // 是否覆盖已存在的文件
		NoCheck              bool // 不校验文件

		FilePanSource      global.FileSourceType // 要下载的网盘文件来源
		FilePanPath        string                // 要下载的网盘文件路径
		SavePath           string                // 文件保存在本地的路径
		OriginSaveRootPath string                // 文件保存在本地的根目录路径
		DriveId            string                // 网盘ID

		fileInfo *aliyunpan.FileEntity // 文件或目录详情

		// 下载文件记录器
		FileRecorder *log.FileRecorder
	}
)

const (
	// DefaultPrintFormat 默认的下载进度输出格式
	DefaultPrintFormat = "\r[%s] ↓ %s/%s %s/s in %s, left %s ............"
	//DownloadSuffix 文件下载后缀
	DownloadSuffix = ".aliyunpan-downloading"
	//StrDownloadInitError 初始化下载发生错误
	StrDownloadInitError = "初始化下载发生错误"
	// StrDownloadFailed 下载文件失败
	StrDownloadFailed = "下载文件失败"
	// StrDownloadGetDlinkFailed 获取下载链接失败
	StrDownloadGetDlinkFailed = "获取下载链接失败"
	// StrDownloadChecksumFailed 检测文件有效性失败
	StrDownloadChecksumFailed = "检测文件有效性失败"
	// DefaultDownloadMaxRetry 默认下载失败最大重试次数
	DefaultDownloadMaxRetry = 3
)

// SetFileInfo 设置文件信息
func (dtu *DownloadTaskUnit) SetFileInfo(source global.FileSourceType, f *aliyunpan.FileEntity) {
	dtu.FilePanSource = source
	dtu.fileInfo = f
}

func (dtu *DownloadTaskUnit) SetTaskInfo(info *taskframework.TaskInfo) {
	dtu.taskInfo = info
}

func (dtu *DownloadTaskUnit) verboseInfof(format string, a ...interface{}) {
	if dtu.VerbosePrinter != nil {
		dtu.VerbosePrinter.Infof(format, a...)
	}
}

// download 执行下载文件（非目录）
func (dtu *DownloadTaskUnit) download() (err error) {
	var (
		writer downloader.Writer
		file   *os.File
	)

	// 创建下载的目录
	// 获取SavePath所在的目录
	dir := filepath.Dir(dtu.SavePath)
	//fileInfo, err := os.Stat(dir)
	//if err != nil {
	//	// 目录不存在, 创建
	//	err = os.MkdirAll(dir, 0777)
	//	if err != nil {
	//		return err
	//	}
	//} else if !fileInfo.IsDir() {
	//	// SavePath所在的目录不是目录
	//	return fmt.Errorf("%s, path %s: not a directory", StrDownloadInitError, dir)
	//}
	// 支持本地符号链接文件，整体逻辑和上面注释代码一致
	savePathSymlinkFile := localfile.SymlinkFile{
		LogicPath: dtu.SavePath,
		RealPath:  "",
	}
	originSaveRootSymlinkFile := localfile.NewSymlinkFile(dtu.OriginSaveRootPath)
	suffixPath := localfile.GetSuffixPath(dir, dtu.OriginSaveRootPath)
	saveDirPathSymlinkFile, saveDirPathFileInfo, err := localfile.RetrieveRealPathFromLogicSuffixPath(originSaveRootSymlinkFile, suffixPath)
	if err != nil && !os.IsExist(err) {
		// 本地保存目录不存在，需要创建对应的保存目录
		realSaveDirPath := saveDirPathSymlinkFile.RealPath
		suffixPath = localfile.GetSuffixPath(filepath.Dir(dtu.SavePath), saveDirPathSymlinkFile.LogicPath) // 获取后缀不存在的路径
		if suffixPath != "" {
			realSaveDirPath = filepath.Join(realSaveDirPath, suffixPath) // 拼接
		}
		// 创建保存目录
		err = os.MkdirAll(realSaveDirPath, 0777)
		if err != nil {
			return err
		}

		// 拼接完整的保存文件路径
		savePathSymlinkFile.RealPath = filepath.Join(realSaveDirPath, filepath.Base(localfile.CleanPath(dtu.SavePath)))
	} else if !saveDirPathFileInfo.IsDir() {
		// SavePath所在的目录不是目录
		return fmt.Errorf("%s, path %s: not a directory", StrDownloadInitError, saveDirPathSymlinkFile.RealPath)
	} else {
		// 保存目录已存在，直接拼接成文件完整保存路径
		savePathSymlinkFile.RealPath = filepath.Join(saveDirPathSymlinkFile.RealPath, filepath.Base(localfile.CleanPath(dtu.SavePath)))
	}
	savePathSymlinkFile, _, _ = localfile.RetrieveRealPath(savePathSymlinkFile)

	// 下载配置文件存储路径
	dtu.Cfg.InstanceStatePath = savePathSymlinkFile.RealPath + DownloadSuffix

	// 打开文件
	writer, file, err = downloader.NewDownloaderWriterByFilename(savePathSymlinkFile.RealPath, os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("%s, %s", StrDownloadInitError, err)
	}
	defer file.Close()

	der := downloader.NewDownloader(writer, dtu.Cfg, dtu.PanClient, dtu.SubPanClientList, dtu.GlobalSpeedsStat)
	der.SetFileInfo(dtu.FilePanSource, dtu.fileInfo)
	der.SetDriveId(dtu.DriveId)
	der.SetStatusCodeBodyCheckFunc(func(respBody io.Reader) error {
		// 解析错误
		return apierror.NewFailedApiError("")
	})

	// 检查输出格式
	if dtu.PrintFormat == "" {
		dtu.PrintFormat = DefaultPrintFormat
	}

	// 这里用共享变量的方式
	isComplete := false
	der.OnDownloadStatusEvent(func(status transfer.DownloadStatuser, workersCallback func(downloader.RangeWorkerFunc)) {
		// 这里可能会下载结束了, 还会输出内容
		builder := &strings.Builder{}
		if dtu.IsPrintStatus {
			// 输出所有的worker状态
			var (
				tb = cmdtable.NewTable(builder)
			)
			tb.SetHeader([]string{"#", "status", "range", "left", "speeds", "error"})
			workersCallback(func(key int, worker *downloader.Worker) bool {
				wrange := worker.GetRange()
				tb.Append([]string{fmt.Sprint(worker.ID()), worker.GetStatus().StatusText(), wrange.ShowDetails(), strconv.FormatInt(wrange.Len(), 10), converter.ConvertFileSize(worker.GetSpeedsPerSecond(), 2) + "/s", fmt.Sprint(worker.Err())})
				return true
			})

			// 先空两行
			builder.WriteString("\n\n")
			tb.Render()
		}

		// 如果下载速度为0, 剩余下载时间未知, 则用 - 代替
		var leftStr string
		left := status.TimeLeft()
		if left < 0 {
			leftStr = "-"
		} else {
			leftStr = left.String()
		}

		if dtu.Cfg.ShowProgress {
			downloadedPercentage := fmt.Sprintf("%.2f%%", float64(status.Downloaded())/float64(status.TotalSize())*100)
			fmt.Fprintf(builder, "\r[%s] ↓ %s/%s(%s) %s/s(%s/s) in %s, left %s ............", dtu.taskInfo.Id(),
				converter.ConvertFileSize(status.Downloaded(), 2),
				converter.ConvertFileSize(status.TotalSize(), 2),
				downloadedPercentage,
				converter.ConvertFileSize(status.SpeedsPerSecond(), 2),
				converter.ConvertFileSize(dtu.GlobalSpeedsStat.GetSpeeds(), 2),
				status.TimeElapsed()/1e7*1e7, leftStr,
			)
		}

		if !isComplete {
			// 如果未完成下载, 就输出
			fmt.Print(builder.String())
		}
	})

	der.OnExecute(func() {
		fmt.Printf("[%s] 下载开始\n", dtu.taskInfo.Id())
	})

	err = der.Execute()
	if err != nil {
		// check zero size file
		if err == downloader.ErrNoWokers && dtu.fileInfo.FileSize == 0 {
			// success for 0 size file
			dtu.verboseInfof("download success for zero size file")
		} else if err == downloader.ErrFileDownloadForbidden {
			// 文件被禁止下载
			isComplete = false
			// 删除本地文件
			removeErr := os.Remove(dtu.SavePath)
			if removeErr != nil {
				dtu.verboseInfof("[%s] remove file error: %s\n", dtu.taskInfo.Id(), removeErr)
			}
			fmt.Printf("[%s] 下载失败，文件不合法或者被禁止下载: %s\n", dtu.taskInfo.Id(), dtu.SavePath)
			return err
		} else {
			// 下载发生错误
			// 下载失败, 删去空文件
			if info, infoErr := file.Stat(); infoErr == nil {
				if info.Size() == 0 {
					// 空文件, 应该删除
					dtu.verboseInfof("[%s] remove empty file: %s\n", dtu.taskInfo.Id(), dtu.SavePath)
					removeErr := os.Remove(dtu.SavePath)
					if removeErr != nil {
						dtu.verboseInfof("[%s] remove file error: %s\n", dtu.taskInfo.Id(), removeErr)
					}
				}
			}
			return err
		}
	} else {
		isComplete = true
	}

	// 下载成功
	if dtu.IsExecutedPermission {
		err = file.Chmod(0766)
		if err != nil {
			fmt.Printf("[%s] 警告, 加执行权限错误: %s\n", dtu.taskInfo.Id(), err)
		}
	}
	fmt.Printf("\n[%s] 下载完成, 保存位置: %s\n", dtu.taskInfo.Id(), dtu.SavePath)

	return nil
}

// handleError 下载错误处理器
func (dtu *DownloadTaskUnit) handleError(result *taskframework.TaskUnitRunResult) {
	switch value := result.Err.(type) {
	case *apierror.ApiError:
		switch value.ErrCode() {
		case apierror.ApiCodeFileNotFoundCode:
		case apierror.ApiCodeForbiddenFileInTheRecycleBin:
			result.NeedRetry = false
			break
		default:
			result.NeedRetry = true
		}
	case *os.PathError:
		// 系统级别的错误, 可能是权限问题
		result.NeedRetry = false
	default:
		if result.Err == downloader.ErrFileDownloadForbidden {
			result.NeedRetry = false
		} else {
			// 其他错误, 尝试重试
			result.NeedRetry = true
		}
	}
	time.Sleep(1 * time.Second)
}

// checkFileValid 检测文件有效性
func (dtu *DownloadTaskUnit) checkFileValid(result *taskframework.TaskUnitRunResult) (ok bool) {
	if dtu.NoCheck {
		// 不检测文件有效性
		return
	}

	if dtu.fileInfo.FileSize >= 128*converter.MB {
		// 大文件, 输出一句提示消息
		fmt.Printf("[%s] 开始检验文件有效性, 请稍候...\n", dtu.taskInfo.Id())
	}

	// 就在这里处理校验出错
	err := CheckFileValid(dtu.SavePath, dtu.fileInfo)
	if err != nil {
		result.ResultMessage = StrDownloadChecksumFailed
		result.Err = err
		switch err {
		case ErrDownloadNotSupportChecksum:
			// 文件不支持校验
			result.ResultMessage = "检验文件有效性"
			result.Err = err
			fmt.Printf("[%s] 检验文件有效性: %s\n", dtu.taskInfo.Id(), err)
			return true
		case ErrDownloadFileBanned:
			// 违规文件
			result.NeedRetry = false
			return
		case ErrDownloadChecksumFailed:
			// 校验失败, 需要重新下载
			result.NeedRetry = true
			// 设置允许覆盖
			dtu.IsOverwrite = true
			return
		default:
			result.NeedRetry = false
			return
		}
	}

	fmt.Printf("[%s] 检验文件有效性成功: %s\n", dtu.taskInfo.Id(), dtu.SavePath)
	return true
}

func (dtu *DownloadTaskUnit) OnRetry(lastRunResult *taskframework.TaskUnitRunResult) {
	// 输出错误信息
	if lastRunResult.Err == nil {
		// result中不包含Err, 忽略输出
		fmt.Printf("[%s] %s, 重试 %d/%d\n", dtu.taskInfo.Id(), lastRunResult.ResultMessage, dtu.taskInfo.Retry(), dtu.taskInfo.MaxRetry())
		return
	}
	fmt.Printf("[%s] %s, %s, 重试 %d/%d\n", dtu.taskInfo.Id(), lastRunResult.ResultMessage, lastRunResult.Err, dtu.taskInfo.Retry(), dtu.taskInfo.MaxRetry())
}

func (dtu *DownloadTaskUnit) OnSuccess(lastRunResult *taskframework.TaskUnitRunResult) {
	// 执行插件
	dtu.pluginCallback("success")

	// 下载文件数据记录
	if config.Config.FileRecordConfig == "1" {
		if dtu.fileInfo.IsFile() {
			if dtu.FileRecorder != nil {
				dtu.FileRecorder.Append(&log.FileRecordItem{
					Status:   "成功",
					TimeStr:  utils.NowTimeStr(),
					FileSize: dtu.fileInfo.FileSize,
					FilePath: dtu.fileInfo.Path,
				})
			}
		}
	}
}

func (dtu *DownloadTaskUnit) OnFailed(lastRunResult *taskframework.TaskUnitRunResult) {
	// 失败
	dtu.pluginCallback("fail")

	// 失败
	if lastRunResult.Err == nil {
		// result中不包含Err, 忽略输出
		fmt.Printf("[%s] %s\n", dtu.taskInfo.Id(), lastRunResult.ResultMessage)
		return
	}
	fmt.Printf("[%s] %s, %s\n", dtu.taskInfo.Id(), lastRunResult.ResultMessage, lastRunResult.Err)
}

func (dtu *DownloadTaskUnit) pluginCallback(result string) {
	if dtu.fileInfo == nil {
		return
	}
	pluginManger := plugins.NewPluginManager(config.GetPluginDir())
	plugin, _ := pluginManger.GetPlugin()
	pluginParam := &plugins.DownloadFileFinishParams{
		DownloadActionId:   dtu.DownloadActionId,
		DriveId:            dtu.fileInfo.DriveId,
		DriveFileId:        dtu.fileInfo.FileId,
		DriveFilePath:      dtu.fileInfo.Path,
		DriveFileName:      dtu.fileInfo.FileName,
		DriveFileSize:      dtu.fileInfo.FileSize,
		DriveFileType:      "file",
		DriveFileSha1:      dtu.fileInfo.ContentHash,
		DriveFileUpdatedAt: dtu.fileInfo.UpdatedAt,
		DownloadResult:     result,
		LocalFilePath:      dtu.SavePath,
	}
	if er := plugin.DownloadFileFinishCallback(plugins.GetContext(config.Config.ActiveUser()), pluginParam); er != nil {
		logger.Verboseln("插件DownloadFileFinishCallback调用失败： {}", er)
	} else {
		logger.Verboseln("插件DownloadFileFinishCallback调用成功")
	}
}

func (dtu *DownloadTaskUnit) OnComplete(lastRunResult *taskframework.TaskUnitRunResult) {
}

func (dtu *DownloadTaskUnit) OnCancel(lastRunResult *taskframework.TaskUnitRunResult) {

}

func (dtu *DownloadTaskUnit) RetryWait() time.Duration {
	return functions.RetryWait(dtu.taskInfo.Retry())
}

func (dtu *DownloadTaskUnit) Run() (result *taskframework.TaskUnitRunResult) {
	result = &taskframework.TaskUnitRunResult{}
	// 获取文件信息
	var apierr *apierror.ApiError
	if dtu.fileInfo == nil || dtu.taskInfo.Retry() > 0 {
		// 没有获取文件信息
		// 如果是动态添加的下载任务, 是会写入文件信息的
		// 如果该任务重试过, 则应该再获取一次文件信息
		dtu.fileInfo, apierr = dtu.PanClient.OpenapiPanClient().FileInfoByPath(dtu.DriveId, dtu.FilePanPath)
		if apierr != nil {
			// 如果不是未登录或文件不存在, 则不重试
			result.ResultMessage = "获取下载路径信息错误"
			result.Err = apierr
			dtu.handleError(result)
			return
		}
		time.Sleep(1 * time.Second)
	}

	// 输出文件信息
	fmt.Print("\n")
	fmt.Printf("[%s] ----\n%s\n", dtu.taskInfo.Id(), dtu.fileInfo.String())

	// 调用插件
	ft := "file"
	if dtu.fileInfo.IsFolder() {
		ft = "folder"
	}
	pluginManger := plugins.NewPluginManager(config.GetPluginDir())
	plugin, _ := pluginManger.GetPlugin()
	localFilePath := strings.TrimPrefix(dtu.SavePath, dtu.OriginSaveRootPath)
	localFilePath = strings.TrimPrefix(strings.TrimPrefix(localFilePath, "\\"), "/")
	pluginParam := &plugins.DownloadFilePrepareParams{
		DownloadActionId:   dtu.DownloadActionId,
		DriveId:            dtu.fileInfo.DriveId,
		DriveFilePath:      dtu.fileInfo.Path,
		DriveFileName:      dtu.fileInfo.FileName,
		DriveFileSize:      dtu.fileInfo.FileSize,
		DriveFileType:      ft,
		DriveFileSha1:      dtu.fileInfo.ContentHash,
		DriveFileUpdatedAt: dtu.fileInfo.UpdatedAt,
		LocalFilePath:      localFilePath,
	}
	if downloadFilePrepareResult, er := plugin.DownloadFilePrepareCallback(plugins.GetContext(config.Config.ActiveUser()), pluginParam); er == nil && downloadFilePrepareResult != nil {
		if strings.Compare("yes", downloadFilePrepareResult.DownloadApproved) != 0 {
			// skip download this file
			fmt.Printf("插件取消了该文件下载: %s\n", dtu.fileInfo.Path)
			result.Succeed = false
			result.Cancel = true
			return
		}
		if downloadFilePrepareResult.LocalFilePath != "" {
			targetSaveRelativePath := strings.TrimPrefix(downloadFilePrepareResult.LocalFilePath, "/")
			targetSaveRelativePath = strings.TrimPrefix(targetSaveRelativePath, "\\")
			dtu.SavePath = path.Clean(dtu.OriginSaveRootPath + string(os.PathSeparator) + targetSaveRelativePath)
			fmt.Printf("插件修改文件下载保存路径为: %s\n", dtu.SavePath)
		}
	}

	// 如果是一个目录, 将子文件和子目录加入队列
	if dtu.fileInfo.IsFolder() {
		//_, err := os.Stat(dtu.SavePath)
		//if err != nil && !os.IsExist(err) {
		//	os.MkdirAll(dtu.SavePath, 0777) // 首先在本地创建目录, 保证空目录也能被保存
		//}
		// 支持本地符号逻辑文件，整体逻辑等效上面的注释代码
		originSaveRootSymlinkFile := localfile.NewSymlinkFile(dtu.OriginSaveRootPath)
		suffixPath := localfile.GetSuffixPath(dtu.SavePath, dtu.OriginSaveRootPath)
		savePathSymlinkFile, _, err := localfile.RetrieveRealPathFromLogicSuffixPath(originSaveRootSymlinkFile, suffixPath)
		if err != nil && !os.IsExist(err) {
			realSavePath := savePathSymlinkFile.RealPath
			suffixPath = localfile.GetSuffixPath(dtu.SavePath, savePathSymlinkFile.LogicPath) // 获取后缀不存在的路径
			if suffixPath != "" {
				realSavePath = filepath.Join(realSavePath, suffixPath)
			}
			err1 := os.MkdirAll(realSavePath, 0777) // 首先在本地创建目录, 保证空目录也能被保存
			if err1 != nil {
				result.ResultMessage = "创建目录错误"
				result.Succeed = false
				result.Err = err1
				result.NeedRetry = false
				return
			}
		}

		// 获取该目录下的文件列表
		fileList, apierr := dtu.PanClient.OpenapiPanClient().FileListGetAll(&aliyunpan.FileListParam{
			DriveId:      dtu.DriveId,
			ParentFileId: dtu.fileInfo.FileId,
		}, 1000)
		if apierr != nil {
			// retry one more time
			time.Sleep(3 * time.Second)

			fileList, apierr = dtu.PanClient.OpenapiPanClient().FileListGetAll(&aliyunpan.FileListParam{
				DriveId:      dtu.DriveId,
				ParentFileId: dtu.fileInfo.FileId,
			}, 1000)
			if apierr != nil {
				logger.Verbosef("[%s] get download file list for %s error: %s\n",
					dtu.taskInfo.Id(), dtu.FilePanPath, apierr)

				// 下次重试
				result.ResultMessage = "获取目录信息错误"
				result.Succeed = false
				result.Err = apierr
				result.NeedRetry = true
				return
			}
		}
		if fileList == nil {
			result.ResultMessage = "获取目录信息错误"
			result.Err = err
			result.NeedRetry = true
			return
		}
		time.Sleep(1 * time.Second)

		// 创建对应的任务进行下载
		for k := range fileList {
			fileList[k].Path = path.Join(dtu.FilePanPath, fileList[k].FileName)

			// 是否排除下载
			if utils.IsExcludeFile(fileList[k].Path, &dtu.Cfg.ExcludeNames) {
				fmt.Printf("排除文件: %s\n", fileList[k].Path)
				continue
			}

			if fileList[k].IsFolder() {
				logger.Verbosef("[%s] create sub folder download task: %s\n",
					dtu.taskInfo.Id(), fileList[k].Path)
			}

			// 添加子任务
			subUnit := *dtu
			newCfg := *dtu.Cfg
			subUnit.Cfg = &newCfg
			subUnit.fileInfo = fileList[k] // 保存文件信息
			subUnit.FilePanSource = dtu.FilePanSource
			subUnit.FilePanPath = fileList[k].Path
			subUnit.SavePath = filepath.Join(dtu.OriginSaveRootPath, fileList[k].Path) // 保存位置

			// 加入父队列，按照队列调度进行下载
			info := dtu.ParentTaskExecutor.Append(&subUnit, dtu.taskInfo.MaxRetry())
			fmt.Printf("[%s] 加入下载队列: %s\n", info.Id(), fileList[k].Path)
		}

		// 本下载任务执行成功
		result.Succeed = true
		return
	}

	fmt.Printf("[%s] 准备下载: %s\n", dtu.taskInfo.Id(), dtu.FilePanPath)

	//if !dtu.IsOverwrite && FileExist(dtu.SavePath) {
	//	fmt.Printf("[%s] 文件已经存在: %s, 跳过...\n", dtu.taskInfo.Id(), dtu.SavePath)
	//	result.Succeed = true // 执行成功
	//	return
	//}
	// 支持符号文件，逻辑和注释代码一致
	if !dtu.IsOverwrite && SymlinkFileExist(dtu.SavePath, dtu.OriginSaveRootPath) {
		fmt.Printf("[%s] 文件已经存在: %s, 跳过...\n", dtu.taskInfo.Id(), dtu.SavePath)
		result.Succeed = true // 执行成功
		return
	}

	fmt.Printf("[%s] 将会下载到路径: %s\n", dtu.taskInfo.Id(), dtu.SavePath)

	var ok bool
	er := dtu.download()

	if er != nil {
		// 以上执行不成功, 返回
		result.ResultMessage = StrDownloadFailed
		result.Err = er
		dtu.handleError(result)
		return result
	}

	// 检测文件有效性
	ok = dtu.checkFileValid(result)
	if !ok {
		// 校验不成功, 返回结果
		return result
	}

	//// 文件下载成功，更改文件修改时间和云盘的同步
	//if err := os.Chtimes(dtu.SavePath, utils.ParseTimeStr(dtu.fileInfo.CreatedAt), utils.ParseTimeStr(dtu.fileInfo.CreatedAt)); err != nil {
	//	logger.Verbosef(err.Error())
	//}

	// 统计下载
	dtu.DownloadStatistic.AddTotalSize(dtu.fileInfo.FileSize)
	// 下载成功
	result.Succeed = true
	return
}
