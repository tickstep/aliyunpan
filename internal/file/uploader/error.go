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

import "fmt"

var (
	UploadUrlExpired                  = fmt.Errorf("UrlExpired")
	UploadPartNotSeq                  = fmt.Errorf("PartNotSequential")
	UploadNoSuchUpload                = fmt.Errorf("NoSuchUpload")
	UploadTerminate                   = fmt.Errorf("UploadErrorTerminate")
	UploadPartAlreadyExist            = fmt.Errorf("PartAlreadyExist")
	UploadHttpError                   = fmt.Errorf("HttpError")
	UploadLocalFileAlreadyClosedError = fmt.Errorf("LocalFileAlreadyClosedError")
)

type (
	// MultiError 多线程上传的错误
	MultiError struct {
		Err error
		// IsRetry 是否重试,
		Terminated    bool
		NeedStartOver bool // 是否从头开始上传
	}
)

func (me *MultiError) Error() string {
	if me.Err != nil {
		return me.Err.Error()
	}
	return ""
}
