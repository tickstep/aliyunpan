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
package downloader

import (
	"context"
	"errors"
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/library/requester/transfer"
	"github.com/tickstep/library-go/cachepool"
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester"
	"github.com/tickstep/library-go/requester/rio/speeds"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

type (
	//Worker 工作单元
	Worker struct {
		totalSize        int64 // 整个文件的大小, worker请求range时会获取尝试获取该值, 如果不匹配, 则返回错误
		wrange           *transfer.Range
		speedsStat       *speeds.Speeds
		globalSpeedsStat *speeds.Speeds // 全局速度统计
		id               int            // work id
		fileId           string         // 文件ID
		driveId          string
		url              string // 下载地址
		acceptRanges     string
		panClient        *config.PanClient
		client           *requester.HTTPClient
		writerAt         io.WriterAt
		writeMu          *sync.Mutex
		execMu           sync.Mutex

		pauseChan              chan struct{}
		workerCancelFunc       context.CancelFunc
		resetFunc              context.CancelFunc
		readRespBodyCancelFunc func()
		err                    error // 错误信息
		status                 WorkerStatus
		downloadStatus         *transfer.DownloadStatus // 总的下载状态
	}

	// WorkerList worker列表
	WorkerList []*Worker
)

// Duplicate 构造新的列表
func (wl WorkerList) Duplicate() WorkerList {
	n := make(WorkerList, len(wl))
	copy(n, wl)
	return n
}

// NewWorker 初始化Worker
func NewWorker(id int, driveId string, fileId, durl string, writerAt io.WriterAt, globalSpeedsStat *speeds.Speeds) *Worker {
	return &Worker{
		id:               id,
		url:              durl,
		writerAt:         writerAt,
		fileId:           fileId,
		driveId:          driveId,
		globalSpeedsStat: globalSpeedsStat,
	}
}

// ID 返回worker ID
func (wer *Worker) ID() int {
	return wer.id
}

func (wer *Worker) lazyInit() {
	if wer.client == nil {
		wer.client = requester.NewHTTPClient()
	}
	if wer.pauseChan == nil {
		wer.pauseChan = make(chan struct{})
	}
	if wer.wrange == nil {
		wer.wrange = &transfer.Range{}
	}
	if wer.wrange.LoadBegin() == 0 && wer.wrange.LoadEnd() == 0 {
		// 取消多线程下载
		wer.acceptRanges = ""
		wer.wrange.StoreEnd(-2)
	}
	if wer.speedsStat == nil {
		wer.speedsStat = &speeds.Speeds{}
	}
}

// SetTotalSize 设置整个文件的大小, worker请求range时会获取尝试获取该值, 如果不匹配, 则返回错误
func (wer *Worker) SetTotalSize(size int64) {
	wer.totalSize = size
}

// SetClient 设置http客户端
func (wer *Worker) SetClient(c *requester.HTTPClient) {
	wer.client = c
}

func (wer *Worker) SetPanClient(p *config.PanClient) {
	wer.panClient = p
}

// SetAcceptRange 设置AcceptRange
func (wer *Worker) SetAcceptRange(acceptRanges string) {
	wer.acceptRanges = acceptRanges
}

// SetRange 设置请求范围
func (wer *Worker) SetRange(r *transfer.Range) {
	if wer.wrange == nil {
		wer.wrange = r
		return
	}
	wer.wrange.StoreBegin(r.LoadBegin())
	wer.wrange.StoreEnd(r.LoadEnd())
}

// SetWriteMutex 设置数据写锁
func (wer *Worker) SetWriteMutex(mu *sync.Mutex) {
	wer.writeMu = mu
}

// SetDownloadStatus 增加其他需要统计的数据
func (wer *Worker) SetDownloadStatus(downloadStatus *transfer.DownloadStatus) {
	wer.downloadStatus = downloadStatus
}

// SetUrl 更新新的下载路径
func (wer *Worker) SetUrl(newUrl string) {
	wer.url = newUrl
}

// GetStatus 返回下载状态
func (wer *Worker) GetStatus() WorkerStatuser {
	// 空接口与空指针不等价
	return &wer.status
}

// GetRange 返回worker范围
func (wer *Worker) GetRange() *transfer.Range {
	return wer.wrange
}

// GetSpeedsPerSecond 获取每秒的速度
func (wer *Worker) GetSpeedsPerSecond() int64 {
	return wer.speedsStat.GetSpeeds()
}

// Pause 暂停下载
func (wer *Worker) Pause() {
	wer.lazyInit()
	if wer.acceptRanges == "" {
		logger.Verbosef("WARNING: worker unsupport pause")
		return
	}

	if wer.status.statusCode == StatusCodePaused {
		return
	}
	wer.pauseChan <- struct{}{}
	wer.status.statusCode = StatusCodePaused
}

// Resume 恢复下载
func (wer *Worker) Resume() {
	if wer.status.statusCode != StatusCodePaused {
		return
	}
	go wer.Execute()
}

// Cancel 取消下载
func (wer *Worker) Cancel() error {
	if wer.workerCancelFunc == nil {
		return errors.New("cancelFunc not set")
	}
	wer.workerCancelFunc()
	if wer.readRespBodyCancelFunc != nil {
		wer.readRespBodyCancelFunc()
	}
	return nil
}

// Reset 重设连接
func (wer *Worker) Reset() {
	logger.Verbosef("Reset worker: %d\n", wer.ID())
	if wer.resetFunc == nil {
		logger.Verbosef("DEBUG: worker: resetFunc not set")
		return
	}
	wer.resetFunc()
	if wer.readRespBodyCancelFunc != nil {
		wer.readRespBodyCancelFunc()
	}
	wer.ClearStatus()
	go wer.Execute()
}

// RefreshDownloadUrl 重新刷新下载链接
func (wer *Worker) RefreshDownloadUrl() {
	var apierr *apierror.ApiError
	logger.Verbosef("get new download url for worker: %d\n", wer.ID())
	durl, apierr := wer.panClient.OpenapiPanClient().GetFileDownloadUrl(&aliyunpan.GetFileDownloadUrlParam{DriveId: wer.driveId, FileId: wer.fileId})
	if apierr != nil {
		logger.Verbosef("get new download url for worker: %d, error: %+v\n", wer.ID(), apierr)
		wer.status.statusCode = StatusCodeTooManyConnections
		return
	}
	wer.url = durl.Url
	logger.Verbosef("get new download url for worker: %d, new url: %s\n", wer.ID(), wer.url)
}

// Canceled 是否已经取消
func (wer *Worker) Canceled() bool {
	return wer.status.statusCode == StatusCodeCanceled
}

// Completed 是否已经完成
func (wer *Worker) Completed() bool {
	switch wer.status.statusCode {
	case StatusCodeSuccessed, StatusCodeCanceled:
		return true
	default:
		return false
	}
}

// Failed 是否失败
func (wer *Worker) Failed() bool {
	switch wer.status.statusCode {
	case StatusCodeFailed, StatusCodeInternalError, StatusCodeTooManyConnections, StatusCodeNetError:
		return true
	default:
		return false
	}
}

// ClearStatus 清空状态
func (wer *Worker) ClearStatus() {
	wer.status.statusCode = StatusCodeInit
}

// Err 返回worker错误
func (wer *Worker) Err() error {
	return wer.err
}

// Execute 执行任务
func (wer *Worker) Execute() {
	logger.Verbosef("Execute worker: %d\n", wer.ID())
	wer.lazyInit()

	wer.execMu.Lock()
	defer wer.execMu.Unlock()

	wer.status.statusCode = StatusCodeInit
	single := wer.acceptRanges == ""

	// 如果已暂停, 退出
	if wer.status.statusCode == StatusCodePaused {
		return
	}

	if !single {
		// 已完成
		if rlen := wer.wrange.Len(); rlen <= 0 {
			if rlen < 0 {
				logger.Verbosef("DEBUG: RangeLen is negative at begin: %v, %d\n", wer.wrange, wer.wrange.Len())
			}
			wer.status.statusCode = StatusCodeSuccessed
			return
		}
	}

	// zero size file
	if wer.totalSize == 0 {
		wer.status.statusCode = StatusCodeSuccessed
		return
	}

	workerCancelCtx, workerCancelFunc := context.WithCancel(context.Background())
	wer.workerCancelFunc = workerCancelFunc
	resetCtx, resetFunc := context.WithCancel(context.Background())
	wer.resetFunc = resetFunc

	wer.status.statusCode = StatusCodePending

	// check url expired or not
	if IsUrlExpired(wer.url) {
		logger.Verbosef("download url expired, renew url and reset worker: %d\n", wer.ID())
		wer.status.statusCode = StatusCodeDownloadUrlExpired
		wer.err = errors.New("403")
		return
	}

	// do download data
	var resp *http.Response
	apierr := wer.panClient.OpenapiPanClient().DownloadFileData(wer.url, aliyunpan.FileDownloadRange{
		Offset: wer.wrange.Begin,
		End:    wer.wrange.End - 1,
	}, func(httpMethod, fullUrl string, headers map[string]string) (*http.Response, error) {
		resp, wer.err = wer.client.Req(httpMethod, fullUrl, nil, headers)
		if wer.err != nil {
			return nil, wer.err
		}
		return resp, wer.err
	})

	if resp != nil {
		defer func() {
			resp.Body.Close()
		}()
		wer.readRespBodyCancelFunc = func() {
			resp.Body.Close()
		}
	}
	if wer.err != nil || apierr != nil {
		wer.status.statusCode = StatusCodeNetError
		return
	}

	// 判断响应状态
	switch resp.StatusCode {
	case 200, 206:
		// do nothing, continue
		wer.status.statusCode = StatusCodeDownloading
		break
	case 416: //Requested Range Not Satisfiable
		fallthrough
	case 403: // 链接过期也会返回403
		if respBody, e := ioutil.ReadAll(resp.Body); e == nil {
			respBodyStr := string(respBody)
			if strings.Contains(respBodyStr, "Request has expired") { // 链接已过期
				logger.Verboseln("download url return 403 error and expired url response")
				wer.status.statusCode = StatusCodeDownloadUrlExpired
				wer.err = errors.New(resp.Status)
			} else if strings.Contains(respBodyStr, "ExceedMaxConcurrency") { // 遇到限流报错
				// 普通应用：文件分片下载的并发数为 3，即某用户使用 App 时，可以同时下载 1 个文件的 3 个分片，或者同时下载 3 个文件的各 1 个分片。
				// 超过并发，再次调用接口，报错 http status：403。示例报错信息如下：
				// <?xml version="1.0" encoding="UTF-8"?>
				//<Error>
				//  <Code>RequestDeniedByCallback</Code>
				//  <Message>Callback deny this request reason: ExceedMaxConcurrency</Message>
				//  <RequestId>66B5720887CECD32333EB8C1</RequestId>
				//  <HostId>cn-beijing-data.aliyundrive.net</HostId>
				//  <EC>0007-00000209</EC>
				//  <RecommendDoc>https://api.aliyun.com/troubleshoot?q=0007-00000209</RecommendDoc>
				//</Error>
				logger.Verboseln("download url return 403 error and exceed max concurrency response")
				wer.status.statusCode = StatusCodeDownloadUrlExceedMaxConcurrency
				wer.err = errors.New(resp.Status)

				// 遇到限流，本线程延迟后，再重试
				time.Sleep(10 * time.Second)
			}
		}
		return
	case 406: // Not Acceptable
		wer.status.statusCode = StatusCodeNetError
		wer.err = errors.New(resp.Status)
		return
	case 404:
		logger.Verboseln("request download url 404 error")
		fallthrough
	case 429, 509: // Too Many Requests
		wer.status.SetStatusCode(StatusCodeTooManyConnections)
		wer.err = errors.New(resp.Status)
		return
	default:
		wer.status.statusCode = StatusCodeNetError
		wer.err = fmt.Errorf("unexpected http status code, %d, %s", resp.StatusCode, resp.Status)
		return
	}

	var (
		contentLength = resp.ContentLength
		rangeLength   = wer.wrange.Len()
	)

	if !single {
		// 检查请求长度
		if contentLength != rangeLength {
			wer.status.statusCode = StatusCodeNetError
			wer.err = fmt.Errorf("Content-Length is unexpected: %d, need %d", contentLength, rangeLength)
			return
		}
		// 检查总大小
		if wer.totalSize > 0 {
			total := ParseContentRange(resp.Header.Get("Content-Range"))
			if total > 0 {
				if total != wer.totalSize {
					wer.status.statusCode = StatusCodeInternalError // 这里设置为内部错误, 强制停止下载
					wer.err = fmt.Errorf("Content-Range total length is unexpected: %d, need %d", total, wer.totalSize)
					return
				}
			}
		}
	}

	var (
		buf       = cachepool.SyncPool.Get().([]byte)
		n, nn     int
		n64, nn64 int64
	)
	defer cachepool.SyncPool.Put(buf)

	for {
		select {
		case <-workerCancelCtx.Done(): //取消
			wer.status.statusCode = StatusCodeCanceled
			return
		case <-resetCtx.Done(): //重设连接
			wer.status.statusCode = StatusCodeReseted
			return
		case <-wer.pauseChan: //暂停
			return
		default:
			wer.status.statusCode = StatusCodeDownloading

			// 初始化数据
			var readErr error
			n = 0

			// 读取数据
			for n < len(buf) && readErr == nil && (single || wer.wrange.Len() > 0) {
				nn, readErr = resp.Body.Read(buf[n:])
				nn64 = int64(nn)

				// 更新速度统计
				if wer.downloadStatus != nil {
					wer.downloadStatus.AddSpeedsDownloaded(nn64) // 限速在这里阻塞
				}
				wer.speedsStat.Add(nn64)
				if wer.globalSpeedsStat != nil {
					wer.globalSpeedsStat.Add(nn64)
				}
				n += nn
			}

			if n > 0 && readErr == io.EOF {
				readErr = io.ErrUnexpectedEOF
			}

			n64 = int64(n)

			// 非单线程模式下
			if !single {
				rangeLength = wer.wrange.Len()

				// 已完成
				if rangeLength <= 0 {
					wer.status.statusCode = StatusCodeCanceled
					wer.err = errors.New("worker already complete")
					return
				}

				if n64 > rangeLength {
					// 数据大小不正常
					n64 = rangeLength
					n = int(rangeLength)
					readErr = io.EOF
				}
			}

			// 写入数据
			if wer.writerAt != nil {
				wer.status.statusCode = StatusCodeWaitToWrite
				if wer.writeMu != nil {
					wer.writeMu.Lock() // 加锁, 减轻硬盘的压力
				}
				_, wer.err = wer.writerAt.WriteAt(buf[:n], wer.wrange.Begin) // 写入数据
				if wer.err != nil {
					if wer.writeMu != nil {
						wer.writeMu.Unlock() //解锁
					}
					wer.status.statusCode = StatusCodeInternalError
					return
				}

				if wer.writeMu != nil {
					wer.writeMu.Unlock() //解锁
				}
				wer.status.statusCode = StatusCodeDownloading
			}

			// 更新下载统计数据
			wer.wrange.AddBegin(n64)
			if wer.downloadStatus != nil {
				wer.downloadStatus.AddDownloaded(n64)
				if single {
					wer.downloadStatus.AddTotalSize(n64)
				}
			}

			if readErr != nil {
				rlen := wer.wrange.Len()
				switch {
				case single && readErr == io.ErrUnexpectedEOF:
					// 单线程判断下载成功
					fallthrough
				case readErr == io.EOF:
					fallthrough
				case rlen <= 0:
					// 下载完成
					// 小于0可能是因为 worker 被 duplicate
					wer.status.statusCode = StatusCodeSuccessed
					if rlen < 0 {
						logger.Verbosef("DEBUG: RangeLen is negative at end: %v, %d\n", wer.wrange, wer.wrange.Len())
					}
					return
				default:
					// 其他错误, 返回
					wer.status.statusCode = StatusCodeFailed
					wer.err = readErr
					return
				}
			}
		}
	}
}
