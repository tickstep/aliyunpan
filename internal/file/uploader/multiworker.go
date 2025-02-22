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
package uploader

import (
	"context"
	"errors"
	"github.com/oleiade/lane"
	"github.com/tickstep/aliyunpan/internal/waitgroup"
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester"
	"io"
	"strconv"
)

type (
	worker struct {
		id         int // ID从0开始计数
		partOffset int64
		splitUnit  SplitUnit
		uploadDone bool
	}

	workerList []*worker
)

func (werl *workerList) Readed() int64 {
	var readed int64
	for _, wer := range *werl {
		readed += wer.splitUnit.Readed()
	}
	return readed
}

func (muer *MultiUploader) upload() (uperr error) {
	err := muer.multiUpload.Precreate()
	if err != nil {
		return err
	}

	var (
		uploadDeque = lane.NewDeque()
	)

	// 加入队列
	// 一个worker对应一个分片
	// 这里跳过已经上传成功的分片
	for _, wer := range muer.workers {
		if !wer.uploadDone {
			uploadDeque.Append(wer)
		}
	}

	// 上传客户端
	uploadClient := requester.NewHTTPClient()
	uploadClient.SetTimeout(0)
	uploadClient.SetKeepAlive(true)

	for {
		// 阿里云盘只支持分片按顺序上传，这里必须是parallel = 1
		wg := waitgroup.NewWaitGroup(muer.config.Parallel)
		wg.AddDelta()

		uperr = nil
		e := uploadDeque.Shift()
		if e == nil { // 任务为空
			break
		}

		wer := e.(*worker)
		go func() { // 异步上传
			defer wg.Done()

			var (
				ctx, cancel = context.WithCancel(context.Background())
				doneChan    = make(chan struct{})
				uploadDone  bool
				terr        error
			)
			go func() {
				if !wer.uploadDone {
					logger.Verboseln("begin to upload part num: " + strconv.Itoa(wer.id+1))
					uploadDone, terr = muer.multiUpload.UploadFile(ctx, int(wer.id), wer.partOffset, wer.splitUnit.Range().End, wer.splitUnit, uploadClient)
				} else {
					uploadDone = true
				}
				close(doneChan)
			}()
			select { // 监听上传进程，循环阻塞
			case <-muer.canceled:
				cancel()
				return
			case <-doneChan:
				// continue
				logger.Verboseln("multiUpload worker upload file http action done")
			}
			cancel()
			if terr != nil {
				logger.Verbosef("upload file part err: %+v\n", terr)
				if me, ok := terr.(*MultiError); ok {
					if me.Terminated { // 终止
						muer.closeCanceledOnce.Do(func() { // 只关闭一次
							close(muer.canceled)
						})
						uperr = me.Err
						return
					} else if me.NeedStartOver {
						logger.Verbosef("upload start over: %d\n", wer.id)
						// 从头开始上传
						muer.closeCanceledOnce.Do(func() { // 只关闭一次
							close(muer.canceled)
						})
						uperr = me.Err
						return
					} else {
						uperr = me.Err
						return
					}
				}

				logger.Verbosef("upload worker err: %s, id: %d\n", terr, wer.id)
				wer.splitUnit.Seek(0, io.SeekStart)
				uploadDeque.Prepend(wer) // 放回上传队列首位
				return
			}
			wer.uploadDone = uploadDone

			// 通知更新
			if muer.updateInstanceStateChan != nil && len(muer.updateInstanceStateChan) < cap(muer.updateInstanceStateChan) {
				muer.updateInstanceStateChan <- struct{}{}
			}
		}()
		wg.Wait()
		if uperr != nil {
			if errors.Is(uperr, UploadPartNotSeq) {
				// 分片出现乱序，停止上传
				// 清空数据，准备重新上传
				uploadDeque = lane.NewDeque() // 清空待上传列表
			}
		}
		// 没有任务了
		if uploadDeque.Size() == 0 {
			break
		}
	}

	// 释放链路
	uploadClient.CloseIdleConnections()

	// 返回错误，通知上层客户端
	if errors.Is(uperr, UploadPartNotSeq) {
		return uperr
	}

	select {
	case <-muer.canceled:
		if uperr != nil {
			return uperr
		}
		return context.Canceled
	default:
	}

	// upload file commit
	// 检测是否全部分片上传成功
	allSuccess := true
	for _, wer := range muer.workers {
		allSuccess = allSuccess && wer.uploadDone
	}
	if allSuccess {
		e := muer.multiUpload.CommitFile()
		if e != nil {
			logger.Verboseln("upload file commit failed: " + e.Error())
			return e
		}
	} else {
		logger.Verboseln("upload file not all success: " + muer.UploadOpEntity.FileId)
	}

	return
}
