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
package uploader

import (
	"context"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/library-go/converter"
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester"
	"github.com/tickstep/library-go/requester/rio"
	"github.com/tickstep/library-go/requester/rio/speeds"
	"sync"
	"time"
)

type (
	// MultiUpload 支持多线程的上传, 可用于断点续传
	MultiUpload interface {
		Precreate() (perr error)
		UploadFile(ctx context.Context, partseq int, partOffset int64, partEnd int64, readerlen64 rio.ReaderLen64, uploadClient *requester.HTTPClient) (uploadDone bool, terr error)
		CommitFile() (cerr error)
	}

	// MultiUploader 多线程上传
	MultiUploader struct {
		onExecuteEvent      requester.Event        //开始上传事件
		onSuccessEvent      requester.Event        //成功上传事件
		onFinishEvent       requester.Event        //结束上传事件
		onCancelEvent       requester.Event        //取消上传事件
		onErrorEvent        requester.EventOnError //上传出错事件
		onUploadStatusEvent UploadStatusFunc       //上传状态事件

		instanceState *InstanceState

		multiUpload      MultiUpload       // 上传体接口
		file             rio.ReaderAtLen64 // 上传
		config           *MultiUploaderConfig
		workers          workerList
		speedsStat       *speeds.Speeds
		rateLimit        *speeds.RateLimit
		globalSpeedsStat *speeds.Speeds // 全局速度统计

		executeTime             time.Time
		finished                chan struct{}
		canceled                chan struct{}
		closeCanceledOnce       sync.Once
		updateInstanceStateChan chan struct{}

		// 网盘上传参数
		UploadOpEntity *aliyunpan.CreateFileUploadResult `json:"uploadOpEntity"`
	}

	// MultiUploaderConfig 多线程上传配置
	MultiUploaderConfig struct {
		Parallel  int   // 上传并发量
		BlockSize int64 // 上传分块
		MaxRate   int64 // 限制最大上传速度
	}
)

// NewMultiUploader 初始化上传
func NewMultiUploader(multiUpload MultiUpload, file rio.ReaderAtLen64, config *MultiUploaderConfig, uploadOpEntity *aliyunpan.CreateFileUploadResult, globalSpeedsStat *speeds.Speeds) *MultiUploader {
	return &MultiUploader{
		multiUpload:      multiUpload,
		file:             file,
		config:           config,
		UploadOpEntity:   uploadOpEntity,
		globalSpeedsStat: globalSpeedsStat,
	}
}

// SetInstanceState 设置InstanceState, 断点续传信息
func (muer *MultiUploader) SetInstanceState(is *InstanceState) {
	muer.instanceState = is
}

func (muer *MultiUploader) lazyInit() {
	if muer.finished == nil {
		muer.finished = make(chan struct{}, 1)
	}
	if muer.canceled == nil {
		muer.canceled = make(chan struct{})
	}
	if muer.updateInstanceStateChan == nil {
		muer.updateInstanceStateChan = make(chan struct{}, 1)
	}
	if muer.config == nil {
		muer.config = &MultiUploaderConfig{}
	}
	if muer.config.Parallel <= 0 {
		muer.config.Parallel = 4
	}
	if muer.config.BlockSize <= 0 {
		muer.config.BlockSize = 1 * converter.GB
	}
	if muer.speedsStat == nil {
		muer.speedsStat = &speeds.Speeds{}
	}
}

func (muer *MultiUploader) check() {
	if muer.file == nil {
		panic("file is nil")
	}
	if muer.multiUpload == nil {
		panic("multiUpload is nil")
	}
	if muer.UploadOpEntity == nil {
		panic("upload parameter is nil")
	}
}

// Execute 执行上传
func (muer *MultiUploader) Execute() error {
	muer.check()
	muer.lazyInit()

	// 初始化限速
	if muer.config.MaxRate > 0 {
		muer.rateLimit = speeds.NewRateLimit(muer.config.MaxRate)
		defer muer.rateLimit.Stop()
	}

	// 分配任务
	if muer.instanceState != nil {
		muer.workers = muer.getWorkerListByInstanceState(muer.instanceState)
		logger.Verboseln("upload task CREATED from instance state\n")
	} else {
		muer.workers = muer.getWorkerListByInstanceState(&InstanceState{
			BlockList: SplitBlock(muer.file.Len(), muer.config.BlockSize),
		})

		logger.Verboseln("upload task CREATED: block size: %d, num: %d\n", muer.config.BlockSize, len(muer.workers))
	}

	// 开始上传
	muer.executeTime = time.Now()
	utils.Trigger(muer.onExecuteEvent)

	// 通知更新
	if muer.updateInstanceStateChan != nil {
		muer.updateInstanceStateChan <- struct{}{}
	}
	muer.uploadStatusEvent()

	err := muer.upload()

	// 完成
	muer.finished <- struct{}{}
	if err != nil {
		if err == context.Canceled {
			if muer.onCancelEvent != nil {
				muer.onCancelEvent()
			}
		} else if muer.onErrorEvent != nil {
			muer.onErrorEvent(err)
		}
	} else {
		utils.TriggerOnSync(muer.onSuccessEvent)
	}
	utils.TriggerOnSync(muer.onFinishEvent)
	return err
}

// InstanceState 返回断点续传信息
func (muer *MultiUploader) InstanceState() *InstanceState {
	blockStates := make([]*BlockState, 0, len(muer.workers))
	for _, wer := range muer.workers {
		blockStates = append(blockStates, &BlockState{
			ID:         wer.id,
			Range:      wer.splitUnit.Range(),
			UploadDone: wer.uploadDone,
		})
	}
	return &InstanceState{
		BlockList: blockStates,
	}
}

// Cancel 取消上传
func (muer *MultiUploader) Cancel() {
	close(muer.canceled)
}

//OnExecute 设置开始上传事件
func (muer *MultiUploader) OnExecute(onExecuteEvent requester.Event) {
	muer.onExecuteEvent = onExecuteEvent
}

//OnSuccess 设置成功上传事件
func (muer *MultiUploader) OnSuccess(onSuccessEvent requester.Event) {
	muer.onSuccessEvent = onSuccessEvent
}

//OnFinish 设置结束上传事件
func (muer *MultiUploader) OnFinish(onFinishEvent requester.Event) {
	muer.onFinishEvent = onFinishEvent
}

//OnCancel 设置取消上传事件
func (muer *MultiUploader) OnCancel(onCancelEvent requester.Event) {
	muer.onCancelEvent = onCancelEvent
}

//OnError 设置上传发生错误事件
func (muer *MultiUploader) OnError(onErrorEvent requester.EventOnError) {
	muer.onErrorEvent = onErrorEvent
}

//OnUploadStatusEvent 设置上传状态事件
func (muer *MultiUploader) OnUploadStatusEvent(f UploadStatusFunc) {
	muer.onUploadStatusEvent = f
}
