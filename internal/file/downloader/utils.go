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
package downloader

import (
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester"
	mathrand "math/rand"
	"mime"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"time"
)

var (
	// ContentRangeRE Content-Range 正则
	ContentRangeRE = regexp.MustCompile(`^.*? \d*?-\d*?/(\d*?)$`)

	// ranSource 随机数种子
	ranSource = mathrand.NewSource(time.Now().UnixNano())

	// ran 一个随机数实例
	ran = mathrand.New(ranSource)
)

// RandomNumber 生成指定区间随机数
func RandomNumber(min, max int) int {
	if min > max {
		min, max = max, min
	}
	return ran.Intn(max-min) + min
}

// GetFileName 获取文件名
func GetFileName(uri string, client *requester.HTTPClient) (filename string, err error) {
	if client == nil {
		client = requester.NewHTTPClient()
	}

	resp, err := client.Req("HEAD", uri, nil, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return "", err
	}

	_, params, err := mime.ParseMediaType(resp.Header.Get("Content-Disposition"))
	if err != nil {
		logger.Verbosef("DEBUG: GetFileName ParseMediaType error: %s\n", err)
		return path.Base(uri), nil
	}

	filename, err = url.QueryUnescape(params["filename"])
	if err != nil {
		return
	}

	if filename == "" {
		filename = path.Base(uri)
	}

	return
}

// ParseContentRange 解析Content-Range
func ParseContentRange(contentRange string) (contentLength int64) {
	raw := ContentRangeRE.FindStringSubmatch(contentRange)
	if len(raw) < 2 {
		return -1
	}

	c, err := strconv.ParseInt(raw[1], 10, 64)
	if err != nil {
		return -1
	}
	return c
}

func fixCacheSize(size *int) {
	if *size < 1024 {
		*size = 1024
	}
}
