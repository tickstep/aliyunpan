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
package panupload

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/library-go/requester/rio/speeds"

	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/file/uploader"
	"github.com/tickstep/aliyunpan/internal/functions"
	"github.com/tickstep/aliyunpan/internal/localfile"
	"github.com/tickstep/aliyunpan/internal/taskframework"
	"github.com/tickstep/library-go/converter"
	"github.com/tickstep/library-go/requester/rio"
)

type (
	// StepUpload 上传步骤
	StepUpload int

	// UploadTaskUnit 上传的任务单元
	UploadTaskUnit struct {
		LocalFileChecksum *localfile.LocalFileEntity // 要上传的本地文件详情
		Step              StepUpload
		SavePath          string // 保存路径
		DriveId           string // 网盘ID，例如：文件网盘，相册网盘
		FolderCreateMutex *sync.Mutex
		FolderSyncDb      SyncDb //文件备份状态数据库

		PanClient         *aliyunpan.PanClient
		UploadingDatabase *UploadingDatabase // 数据库
		Parallel          int
		NoRapidUpload     bool  // 禁用秒传，无需计算SHA1，直接上传
		BlockSize         int64 // 分片大小

		UploadStatistic *UploadStatistic

		taskInfo *taskframework.TaskInfo
		panDir   string
		panFile  string
		state    *uploader.InstanceState

		ShowProgress bool
		IsOverwrite  bool // 覆盖已存在的文件，如果同名文件已存在则移到回收站里

		// 是否使用内置链接
		UseInternalUrl bool

		// 全局速度统计
		GlobalSpeedsStat *speeds.Speeds
	}
)

const (
	// StepUploadInit 初始化步骤
	StepUploadInit StepUpload = iota
	// 上传前准备，创建上传任务
	StepUploadPrepareUpload
	// StepUploadRapidUpload 秒传步骤
	StepUploadRapidUpload
	// StepUploadUpload 正常上传步骤
	StepUploadUpload
)

const (
	StrUploadFailed = "上传文件失败"
)

func (utu *UploadTaskUnit) SetTaskInfo(taskInfo *taskframework.TaskInfo) {
	utu.taskInfo = taskInfo
}

// prepareFile 解析文件阶段
func (utu *UploadTaskUnit) prepareFile() {
	// 解析文件保存路径
	var (
		panDir, panFile = path.Split(utu.SavePath)
	)
	utu.panDir = path.Clean(panDir)
	utu.panFile = panFile

	// 检测断点续传
	utu.state = utu.UploadingDatabase.Search(&utu.LocalFileChecksum.LocalFileMeta)
	if utu.state != nil || utu.LocalFileChecksum.LocalFileMeta.UploadOpEntity != nil { // 读取到了上一次上传task请求的fileId
		utu.Step = StepUploadUpload
	}

	if utu.LocalFileChecksum.UploadOpEntity == nil {
		utu.Step = StepUploadPrepareUpload
		return
	}

	if utu.NoRapidUpload {
		utu.Step = StepUploadUpload
		return
	}

	if utu.LocalFileChecksum.Length > MaxRapidUploadSize {
		fmt.Printf("[%s] 文件超过20GB, 无法使用秒传功能, 跳过秒传...\n", utu.taskInfo.Id())
		utu.Step = StepUploadUpload
		return
	}
	// 下一步: 秒传
	utu.Step = StepUploadRapidUpload
}

// rapidUpload 执行秒传
func (utu *UploadTaskUnit) rapidUpload() (isContinue bool, result *taskframework.TaskUnitRunResult) {
	utu.Step = StepUploadRapidUpload

	// 是否可以秒传
	result = &taskframework.TaskUnitRunResult{}
	fmt.Printf("[%s] %s 检测秒传中, 请稍候...\n", utu.taskInfo.Id(), time.Now().Format("2006-01-02 15:04:06"))
	if utu.LocalFileChecksum.UploadOpEntity.RapidUpload {
		fmt.Printf("[%s] %s 秒传成功, 保存到网盘路径: %s\n", utu.taskInfo.Id(), time.Now().Format("2006-01-02 15:04:06"), utu.SavePath)
		result.Succeed = true
		return false, result
	} else {
		fmt.Printf("[%s] %s 秒传失败，开始正常上传文件\n", utu.taskInfo.Id(), time.Now().Format("2006-01-02 15:04:06"))
		result.Succeed = false
		result.ResultMessage = "文件未曾上传，无法秒传"
		return true, result
	}
}

// upload 上传文件
func (utu *UploadTaskUnit) upload() (result *taskframework.TaskUnitRunResult) {
	utu.Step = StepUploadUpload

	// 创建分片上传器
	// 阿里云盘默认就是分片上传，每一个分片对应一个part_info
	// 但是不支持分片同时上传，必须单线程，并且按照顺序从1开始一个一个上传
	muer := uploader.NewMultiUploader(
		NewPanUpload(utu.PanClient, utu.SavePath, utu.DriveId, utu.LocalFileChecksum.UploadOpEntity, utu.UseInternalUrl),
		rio.NewFileReaderAtLen64(utu.LocalFileChecksum.GetFile()), &uploader.MultiUploaderConfig{
			Parallel:  utu.Parallel,
			BlockSize: utu.BlockSize,
			MaxRate:   config.Config.MaxUploadRate,
		}, utu.LocalFileChecksum.UploadOpEntity, utu.GlobalSpeedsStat)

	// 设置断点续传
	if utu.state != nil {
		muer.SetInstanceState(utu.state)
	}

	muer.OnUploadStatusEvent(func(status uploader.Status, updateChan <-chan struct{}) {
		select {
		case <-updateChan:
			utu.UploadingDatabase.UpdateUploading(&utu.LocalFileChecksum.LocalFileMeta, muer.InstanceState())
			utu.UploadingDatabase.Save()
		default:
		}

		if utu.ShowProgress {
			fmt.Printf("\r[%s] ↑ %s/%s %s/s(%s/s) in %s ............", utu.taskInfo.Id(),
				converter.ConvertFileSize(status.Uploaded(), 2),
				converter.ConvertFileSize(status.TotalSize(), 2),
				converter.ConvertFileSize(status.SpeedsPerSecond(), 2),
				converter.ConvertFileSize(utu.GlobalSpeedsStat.GetSpeeds(), 2),
				status.TimeElapsed(),
			)
		}
	})

	// result
	result = &taskframework.TaskUnitRunResult{}
	muer.OnSuccess(func() {
		fmt.Printf("\n")
		fmt.Printf("[%s] %s 上传文件成功, 保存到网盘路径: %s\n", utu.taskInfo.Id(), time.Now().Format("2006-01-02 15:04:06"), utu.SavePath)
		// 统计
		utu.UploadStatistic.AddTotalSize(utu.LocalFileChecksum.Length)
		utu.UploadingDatabase.Delete(&utu.LocalFileChecksum.LocalFileMeta) // 删除
		utu.UploadingDatabase.Save()
		result.Succeed = true
	})
	muer.OnError(func(err error) {
		apiError, ok := err.(*apierror.ApiError)
		if !ok {
			// 未知错误类型 (非预期的)
			// 不重试
			result.ResultMessage = "上传文件错误"
			result.Err = err
			return
		}

		// 默认需要重试
		result.NeedRetry = true

		switch apiError.ErrCode() {
		default:
			result.ResultMessage = StrUploadFailed
			result.NeedRetry = false
			result.Err = apiError
		}
		return
	})
	muer.Execute()

	return
}

func (utu *UploadTaskUnit) OnRetry(lastRunResult *taskframework.TaskUnitRunResult) {
	// 输出错误信息
	if lastRunResult.Err == nil {
		// result中不包含Err, 忽略输出
		fmt.Printf("[%s] %s, 重试 %d/%d\n", utu.taskInfo.Id(), lastRunResult.ResultMessage, utu.taskInfo.Retry(), utu.taskInfo.MaxRetry())
		return
	}
	fmt.Printf("[%s] %s, %s, 重试 %d/%d\n", utu.taskInfo.Id(), lastRunResult.ResultMessage, lastRunResult.Err, utu.taskInfo.Retry(), utu.taskInfo.MaxRetry())
}

func (utu *UploadTaskUnit) OnSuccess(lastRunResult *taskframework.TaskUnitRunResult) {
	//文件上传成功
	if utu.FolderSyncDb == nil || lastRunResult == ResultLocalFileNotUpdated { //不需要更新数据库
		return
	}
	ufm := &UploadedFileMeta{
		IsFolder: false,
		SHA1:     utu.LocalFileChecksum.SHA1,
		ModTime:  utu.LocalFileChecksum.ModTime,
		Size:     utu.LocalFileChecksum.Length,
	}

	if utu.LocalFileChecksum.UploadOpEntity != nil {
		ufm.FileId = utu.LocalFileChecksum.UploadOpEntity.FileId
		ufm.ParentId = utu.LocalFileChecksum.UploadOpEntity.ParentFileId
	} else {
		efi, _ := utu.PanClient.FileInfoByPath(utu.DriveId, utu.SavePath)
		if efi != nil {
			ufm.FileId = efi.FileId
			ufm.ParentId = efi.ParentFileId
		}
	}
	utu.FolderSyncDb.Put(utu.SavePath, ufm)
}

func (utu *UploadTaskUnit) OnFailed(lastRunResult *taskframework.TaskUnitRunResult) {
	// 失败
}

var ResultLocalFileNotUpdated = &taskframework.TaskUnitRunResult{ResultCode: 1, Succeed: true, ResultMessage: "本地文件未更新，无需上传！"}
var ResultUpdateLocalDatabase = &taskframework.TaskUnitRunResult{ResultCode: 2, Succeed: true, ResultMessage: "本地文件和云端文件MD5一致，无需上传！"}

func (utu *UploadTaskUnit) OnComplete(lastRunResult *taskframework.TaskUnitRunResult) {

}

func (utu *UploadTaskUnit) RetryWait() time.Duration {
	return functions.RetryWait(utu.taskInfo.Retry())
}

func (utu *UploadTaskUnit) Run() (result *taskframework.TaskUnitRunResult) {
	err := utu.LocalFileChecksum.OpenPath()
	if err != nil {
		fmt.Printf("[%s] 文件不可读, 错误信息: %s, 跳过...\n", utu.taskInfo.Id(), err)
		return
	}
	defer utu.LocalFileChecksum.Close() // 关闭文件

	timeStart := time.Now()
	result = &taskframework.TaskUnitRunResult{}

	fmt.Printf("[%s] %s 准备上传: %s => %s\n", utu.taskInfo.Id(), time.Now().Format("2006-01-02 15:04:06"), utu.LocalFileChecksum.Path, utu.SavePath)

	defer func() {
		var msg string
		if result.Err != nil {
			msg = "失败！" + result.ResultMessage + "," + result.Err.Error()
		} else if result.Succeed {
			msg = "成功！" + result.ResultMessage
		} else {
			msg = result.ResultMessage
		}
		fmt.Printf("[%s] %s 文件上传结果： %s 耗时 %s\n", utu.taskInfo.Id(), time.Now().Format("2006-01-02 15:04:06"), msg, utils.ConvertTime(time.Now().Sub(timeStart)))
	}()
	// 准备文件
	utu.prepareFile()
	fmt.Printf("[%s] %s 准备结束, 准备耗时 %s\n", utu.taskInfo.Id(), time.Now().Format("2006-01-02 15:04:06"), utils.ConvertTime(time.Now().Sub(timeStart)))

	var apierr *apierror.ApiError
	var rs *aliyunpan.MkdirResult
	var appCreateUploadFileParam *aliyunpan.CreateFileUploadParam
	var sha1Str string
	var contentHashName string
	var checkNameMode string
	var saveFilePath string
	var testFileMeta = &UploadedFileMeta{}
	var uploadOpEntity *aliyunpan.CreateFileUploadResult
	var proofCode = ""
	var localFileInfo os.FileInfo
	var localFile *os.File
	timeStart2 := time.Now()
	timeStart3 := time.Now()
	timeStart4 := time.Now()

	switch utu.Step {
	case StepUploadPrepareUpload:
		goto StepUploadPrepareUpload
	case StepUploadRapidUpload:
		goto stepUploadRapidUpload
	case StepUploadUpload:
		goto stepUploadUpload
	}

StepUploadPrepareUpload:
	// 创建上传任务
	if utu.FolderSyncDb != nil {
		//启用了备份功能，强制使用覆盖同名文件功能
		utu.IsOverwrite = true
		testFileMeta = utu.FolderSyncDb.Get(utu.SavePath)
	}

	// 创建云盘文件夹
	timeStart2 = time.Now()
	// utu.FolderCreateMutex.Lock()
	saveFilePath = path.Dir(utu.SavePath)
	if saveFilePath != "/" {
		fmt.Printf("[%s] %s 正在检测和创建云盘文件夹: %s\n", utu.taskInfo.Id(), time.Now().Format("2006-01-02 15:04:06"), saveFilePath)
		//同步功能先尝试从数据库获取
		if utu.FolderSyncDb != nil {
			timeStart3 = time.Now()
			utu.FolderCreateMutex.Lock()
			if test := utu.FolderSyncDb.Get(saveFilePath); test.FileId != "" && test.IsFolder {
				rs = &aliyunpan.MkdirResult{FileId: test.FileId}
			}
			utu.FolderCreateMutex.Unlock()
			fmt.Printf("[%s] %s 检测和创建云盘文件夹完毕[from db], 耗时 %s\n", utu.taskInfo.Id(), time.Now().Format("2006-01-02 15:04:06"), utils.ConvertTime(time.Now().Sub(timeStart3)))
		}
		if rs == nil {
			timeStart4 = time.Now()
			fmt.Printf("[%s] %s 创建云盘文件夹: %s\n", utu.taskInfo.Id(), time.Now().Format("2006-01-02 15:04:06"), saveFilePath)
			utu.FolderCreateMutex.Lock()
			// rs, apierr = utu.PanClient.MkdirRecursive(utu.DriveId, "", "", 0, strings.Split(path.Clean(saveFilePath), "/"))
			// 可以直接创建的，不用循环创建
			rs, apierr = utu.PanClient.Mkdir(utu.DriveId, "root", saveFilePath)
			utu.FolderCreateMutex.Unlock()
			if apierr != nil || rs.FileId == "" {
				result.Err = apierr
				result.ResultMessage = "创建云盘文件夹失败"
				return
			}
			fmt.Printf("[%s] %s 创建云盘文件夹, 耗时 %s\n", utu.taskInfo.Id(), time.Now().Format("2006-01-02 15:04:06"), utils.ConvertTime(time.Now().Sub(timeStart4)))
		}
	} else {
		rs = &aliyunpan.MkdirResult{}
		rs.FileId = ""
	}
	// time.Sleep(time.Duration(2) * time.Second)
	// utu.FolderCreateMutex.Unlock()
	fmt.Printf("[%s] %s 检测和创建云盘文件夹完毕, 耗时 %s\n", utu.taskInfo.Id(), time.Now().Format("2006-01-02 15:04:06"), utils.ConvertTime(time.Now().Sub(timeStart2)))

	sha1Str = ""
	proofCode = ""
	contentHashName = "sha1"
	checkNameMode = "auto_rename"
	if !utu.NoRapidUpload {
		// 计算文件SHA1
		fmt.Printf("[%s] %s 正在计算文件SHA1: %s\n", utu.taskInfo.Id(), time.Now().Format("2006-01-02 15:04:06"), utu.LocalFileChecksum.Path)
		utu.LocalFileChecksum.Sum(localfile.CHECKSUM_SHA1)
		if testFileMeta.SHA1 == utu.LocalFileChecksum.SHA1 {
			return ResultUpdateLocalDatabase
		}
		sha1Str = utu.LocalFileChecksum.SHA1
		if utu.LocalFileChecksum.Length == 0 {
			sha1Str = aliyunpan.DefaultZeroSizeFileContentHash
		}

		// proof code
		localFile, _ = os.Open(utu.LocalFileChecksum.Path)
		localFileInfo, _ = localFile.Stat()
		proofCode = aliyunpan.CalcProofCode(utu.PanClient.GetAccessToken(), rio.NewFileReaderAtLen64(localFile), localFileInfo.Size())
		localFile.Close()
	} else {
		fmt.Printf("[%s] %s 已经禁用秒传检测，直接上传\n", utu.taskInfo.Id(), time.Now().Format("2006-01-02 15:04:06"))
		contentHashName = "none"
		checkNameMode = "auto_rename"
	}

	if utu.IsOverwrite {
		// 标记覆盖旧同名文件
		// 检查同名文件是否存在
		efi, apierr := utu.PanClient.FileInfoByPath(utu.DriveId, utu.SavePath)
		if apierr != nil && apierr.Code != apierror.ApiCodeFileNotFoundCode {
			result.Err = apierr
			result.ResultMessage = "检测同名文件失败"
			return
		}
		if efi != nil && efi.FileId != "" {
			if strings.ToUpper(efi.ContentHash) == strings.ToUpper(sha1Str) {
				result.Succeed = true
				result.Extra = efi
				fmt.Printf("[%s] %s 检测到同名文件，文件内容完全一致，无需重复上传: %s\n", utu.taskInfo.Id(), time.Now().Format("2006-01-02 15:04:06"), utu.SavePath)
				return
			}
			// existed, delete it
			var fileDeleteResult []*aliyunpan.FileBatchActionResult
			var err *apierror.ApiError
			fileDeleteResult, err = utu.PanClient.FileDelete([]*aliyunpan.FileBatchActionParam{{DriveId: efi.DriveId, FileId: efi.FileId}})
			if err != nil || len(fileDeleteResult) == 0 {
				result.Err = err
				result.ResultMessage = "无法删除文件，请稍后重试"
				return
			}
			time.Sleep(time.Duration(500) * time.Millisecond)
			fmt.Printf("[%s] %s 检测到同名文件，已移动到回收站: %s\n", utu.taskInfo.Id(), time.Now().Format("2006-01-02 15:04:06"), utu.SavePath)
		}
	}

	// 创建上传任务
	appCreateUploadFileParam = &aliyunpan.CreateFileUploadParam{
		DriveId:         utu.DriveId,
		Name:            filepath.Base(utu.LocalFileChecksum.Path),
		Size:            utu.LocalFileChecksum.Length,
		ContentHash:     sha1Str,
		ContentHashName: contentHashName,
		CheckNameMode:   checkNameMode,
		ParentFileId:    rs.FileId,
		BlockSize:       utu.BlockSize,
		ProofCode:       proofCode,
		ProofVersion:    "v1",
	}

	uploadOpEntity, apierr = utu.PanClient.CreateUploadFile(appCreateUploadFileParam)
	if apierr != nil {
		result.Err = apierr
		result.ResultMessage = "创建上传任务失败：" + apierr.Error()
		return
	}

	utu.LocalFileChecksum.UploadOpEntity = uploadOpEntity
	utu.LocalFileChecksum.ParentFolderId = rs.FileId

stepUploadRapidUpload:
	// 秒传
	if !utu.NoRapidUpload {
		isContinue, rapidUploadResult := utu.rapidUpload()
		if !isContinue {
			// 秒传成功, 返回秒传的结果
			return rapidUploadResult
		}
	}

stepUploadUpload:
	// 正常上传流程
	uploadResult := utu.upload()

	return uploadResult
}
