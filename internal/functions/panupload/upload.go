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
	"context"
	"encoding/xml"
	"fmt"
	"github.com/tickstep/library-go/logger"
	"io"
	"net/http"
	"strconv"

	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan/internal/file/uploader"
	"github.com/tickstep/library-go/requester"
	"github.com/tickstep/library-go/requester/rio"
)

type (
	PanUpload struct {
		panClient  *aliyunpan.PanClient
		targetPath string
		driveId   string

		// 网盘上传参数
		uploadOpEntity *aliyunpan.CreateFileUploadResult
		useInternalUrl bool
	}

	UploadedFileMeta struct {
		IsFolder     bool   `json:"isFolder,omitempty"` // 是否目录
		Path         string `json:"-"`                  // 本地路径，不记录到数据库
		SHA1          string `json:"sha1,omitempty"`      // 文件的 SHA1
		FileId       string `json:"id,omitempty"`       //文件、目录ID
		ParentId     string `json:"parentId,omitempty"` //父文件夹ID
		Size         int64  `json:"length,omitempty"`   // 文件大小
		ModTime      int64  `json:"modtime,omitempty"`  // 修改日期
		LastSyncTime int64  `json:"synctime,omitempty"` //最后同步时间
	}

	EmptyReaderLen64 struct {
	}
)

var (
	uploadUrlExpired = fmt.Errorf("UrlExpired")
	uploadPartNotSeq = fmt.Errorf("PartNotSequential")
	uploadTerminate = fmt.Errorf("UploadErrorTerminate")
	uploadPartAlreadyExist = fmt.Errorf("PartAlreadyExist")
)

func (e EmptyReaderLen64) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

func (e EmptyReaderLen64) Len() int64 {
	return 0
}

func NewPanUpload(panClient *aliyunpan.PanClient, targetPath, driveId string, uploadOpEntity *aliyunpan.CreateFileUploadResult, useInternalUrl bool) uploader.MultiUpload {
	return &PanUpload{
		panClient:     panClient,
		targetPath:    targetPath,
		driveId:      driveId,
		uploadOpEntity: uploadOpEntity,
		useInternalUrl: useInternalUrl,
	}
}

func (pu *PanUpload) lazyInit() {
	if pu.panClient == nil {
		pu.panClient = &aliyunpan.PanClient{}
	}
}

func (pu *PanUpload) Precreate() (err error) {
	return nil
}

func (pu *PanUpload) UploadFile(ctx context.Context, partseq int, partOffset int64, partEnd int64, r rio.ReaderLen64) (uploadDone bool, uperr error) {
	pu.lazyInit()

	// check url expired or not
	uploadUrl := pu.uploadOpEntity.PartInfoList[partseq].UploadURL
	if pu.useInternalUrl {
		uploadUrl = pu.uploadOpEntity.PartInfoList[partseq].InternalUploadURL
	}
	if isUrlExpired(uploadUrl) {
		// get renew upload url
		infoList := make([]aliyunpan.FileUploadPartInfoParam, len(pu.uploadOpEntity.PartInfoList))
		for _,item := range pu.uploadOpEntity.PartInfoList {
			infoList = append(infoList, aliyunpan.FileUploadPartInfoParam{
				PartNumber: item.PartNumber,
			})
		}
		refreshUploadParam := &aliyunpan.GetUploadUrlParam{
			DriveId:      pu.uploadOpEntity.DriveId,
			FileId:       pu.uploadOpEntity.FileId,
			PartInfoList: infoList,
			UploadId:     pu.uploadOpEntity.UploadId,
		}
		newUploadInfo, err := pu.panClient.GetUploadUrl(refreshUploadParam)
		if err != nil {
			return false, &uploader.MultiError{
				Err: uploadUrlExpired,
				Terminated: false,
			}
		}
		pu.uploadOpEntity.PartInfoList = newUploadInfo.PartInfoList
	}

	var respErr *uploader.MultiError
	uploadFunc := func(httpMethod, fullUrl string, headers map[string]string) (*http.Response, error) {
		var resp *http.Response
		var respError error = nil
		respErr = nil

		// do http upload request
		client := requester.NewHTTPClient()
		client.SetTimeout(0)
		resp, _ = client.Req(httpMethod, fullUrl, r, headers)

		if resp != nil {
			if blen, e := strconv.Atoi(resp.Header.Get("content-length")); e == nil {
				if blen  > 0 {
					buf := make([]byte, blen)
					resp.Body.Read(buf)
					logger.Verbosef("分片上传出错: 分片%d => %s\n", partseq, string(buf))

					errResp := &apierror.ErrorXmlResp{}
					if err := xml.Unmarshal(buf, errResp); err == nil {
						if errResp.Code != "" {
							if "PartNotSequential" == errResp.Code {
								respError = uploadPartNotSeq
								respErr = &uploader.MultiError{
									Err: uploadPartNotSeq,
									Terminated: false,
									NeedStartOver: true,
								}
								return resp, respError
							} else if "AccessDenied" == errResp.Code && "Request has expired." == errResp.Message {
								respError = uploadUrlExpired
								respErr = &uploader.MultiError{
									Err: uploadUrlExpired,
									Terminated: false,
								}
								return resp, respError
							} else if "PartAlreadyExist" == errResp.Code {
								respError = uploadPartAlreadyExist
								respErr = &uploader.MultiError{
									Err: uploadPartAlreadyExist,
									Terminated: false,
								}
								return resp, respError
							}
						}
					}
				}
			} else {
				logger.Verbosef("分片上传出错: %d分片 => 原因未知\n", partseq)
			}

			// 不可恢复的错误
			switch resp.StatusCode {
			case 400, 401, 403, 413, 600:
				respError = uploadTerminate
				respErr = &uploader.MultiError{
					Terminated: true,
				}
			}
		}
		return resp, respError
	}

	// 上传一个分片数据
	uploadUrl = pu.uploadOpEntity.PartInfoList[partseq].UploadURL
	if pu.useInternalUrl {
		uploadUrl = pu.uploadOpEntity.PartInfoList[partseq].InternalUploadURL
	}
	apiError := pu.panClient.UploadFileData(uploadUrl, uploadFunc)

	if respErr != nil {
		if respErr.Err == uploadUrlExpired {
			// URL过期，获取新的URL
			guur, er := pu.panClient.GetUploadUrl(&aliyunpan.GetUploadUrlParam{
				DriveId: pu.driveId,
				FileId: pu.uploadOpEntity.FileId,
				UploadId: pu.uploadOpEntity.UploadId,
				PartInfoList: []aliyunpan.FileUploadPartInfoParam{{PartNumber:(partseq+1)}}, // 阿里云盘partNum从1开始计数，partSeq从0开始
			})
			if er != nil {
				return false, &uploader.MultiError{
					Terminated: false,
				}
			}

			// 获取新的上传URL重试一次
			pu.uploadOpEntity.PartInfoList[partseq] = guur.PartInfoList[0]
			uploadUrl := pu.uploadOpEntity.PartInfoList[partseq].UploadURL
			if pu.useInternalUrl {
				uploadUrl = pu.uploadOpEntity.PartInfoList[partseq].InternalUploadURL
			}
			apiError = pu.panClient.UploadFileData(uploadUrl, uploadFunc)
		} else if respErr.Err == uploadPartAlreadyExist {
			// already upload
			// success
			return true, nil
		} else if respErr.Err == uploadPartNotSeq {
			// 上传分片乱序了，需要重新从0分片开始上传
			// 先直接返回，后续再优化
			return false, respErr
		} else {
			return false, respErr
		}
	}

	if apiError != nil {
		return false, apiError
	}

	return true, nil
}

func (pu *PanUpload) CommitFile() (cerr error) {
	pu.lazyInit()
	var er *apierror.ApiError

	_, er = pu.panClient.CompleteUploadFile(&aliyunpan.CompleteUploadFileParam{
		DriveId: pu.driveId,
		FileId: pu.uploadOpEntity.FileId,
		UploadId: pu.uploadOpEntity.UploadId,
	})
	if er != nil {
		return er
	}
	return nil
}
