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
package utils

import (
	"compress/gzip"
	"crypto/md5"
	"flag"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	uuid "github.com/satori/go.uuid"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/library-go/ids"
	"io"
	"io/ioutil"
	"math"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// TrimPathPrefix 去除目录的前缀
func TrimPathPrefix(path, prefixPath string) string {
	if prefixPath == "/" {
		return path
	}
	return strings.TrimPrefix(path, prefixPath)
}

// ContainsString 检测字符串是否在字符串数组里
func ContainsString(ss []string, s string) bool {
	for k := range ss {
		if ss[k] == s {
			return true
		}
	}
	return false
}

// GetURLCookieString 返回cookie字串
func GetURLCookieString(urlString string, jar *cookiejar.Jar) string {
	u, _ := url.Parse(urlString)
	cookies := jar.Cookies(u)
	cookieString := ""
	for _, v := range cookies {
		cookieString += v.String() + "; "
	}
	cookieString = strings.TrimRight(cookieString, "; ")
	return cookieString
}

// DecompressGZIP 对 io.Reader 数据, 进行 gzip 解压
func DecompressGZIP(r io.Reader) ([]byte, error) {
	gzipReader, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	gzipReader.Close()
	return ioutil.ReadAll(gzipReader)
}

// FlagProvided 检测命令行是否提供名为 name 的 flag, 支持多个name(names)
func FlagProvided(names ...string) bool {
	if len(names) == 0 {
		return false
	}
	var targetFlag *flag.Flag
	for _, name := range names {
		targetFlag = flag.Lookup(name)
		if targetFlag == nil {
			return false
		}
		if targetFlag.DefValue == targetFlag.Value.String() {
			return false
		}
	}
	return true
}

// Trigger 用于触发事件
func Trigger(f func()) {
	if f == nil {
		return
	}
	go f()
}

// TriggerOnSync 用于触发事件, 同步触发
func TriggerOnSync(f func()) {
	if f == nil {
		return
	}
	f()
}

func ParseVersionNum(versionStr string) int {
	versionStr = strings.ReplaceAll(versionStr, "-dev", "")
	versionStr = strings.ReplaceAll(versionStr, "v", "")
	versionParts := strings.Split(versionStr, ".")
	verNum := parseInt(versionParts[0])*1e4 + parseInt(versionParts[1])*1e2 + parseInt(versionParts[2])
	return verNum
}
func parseInt(numStr string) int {
	num, e := strconv.Atoi(numStr)
	if e != nil {
		return 0
	}
	return num
}

func ConvertTime(t time.Duration) string {
	seconds := int64(t.Seconds())
	return ConvertTimeSecond(seconds)
}

func ConvertTimeSecond(seconds int64) string {
	MT := int64(1 * 60)
	HT := int64(1 * 60 * 60)

	if seconds <= 0 {
		return "0秒"
	}
	if seconds < MT {
		return fmt.Sprintf("%d秒", seconds)
	}
	if seconds >= MT && seconds < HT {
		return fmt.Sprintf("%d分%d秒", seconds/MT, seconds%MT)
	}
	if seconds >= HT {
		h := seconds / HT
		tmp := seconds % HT
		return fmt.Sprintf("%d小时%d分%d秒", h, tmp/MT, tmp%MT)
	}
	return "0秒"
}

// HasSuffix 判断是否以某字符串作为结尾
func HasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// HasPrefix 判断是否以某字符串作为开始
func HasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[0:len(prefix)] == prefix
}

// GetUniqueKeyStr 获取本机唯一标识
func GetUniqueKeyStr() string {
	keyStr := ids.GetUniqueId("", 32)
	if len(keyStr) == 0 || keyStr == "" {
		// default
		keyStr = "AE8627B0296A4126A1434999C45ECAB2"
	}
	return keyStr
}

// PathExists 文件路径是否存在
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// ObjectToJsonStr 转换成json字符串
func ObjectToJsonStr(v interface{}, useIndent bool) string {
	r := ""
	if useIndent {
		if data, err := jsoniter.MarshalIndent(v, "", " "); err == nil {
			r = string(data)
		}
	} else {
		if data, err := jsoniter.MarshalIndent(v, "", ""); err == nil {
			r = string(data)
		}

	}
	return r
}

func UuidStr() string {
	u4 := uuid.NewV4()
	return u4.String()
}

// NowTimeStr 当前时间字符串，格式为：2006-01-02 15:04:05
func NowTimeStr() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func defaultZeroTime() time.Time {
	value := "1971-01-01 08:00:00"
	cz := time.FixedZone("CST", 8*3600) // 东8区
	if t, e := time.ParseInLocation("2006-01-02 15:04:05", value, cz); e == nil {
		return t
	}
	return time.Time{}
}

// ParseTimeStr 反解析时间字符串
func ParseTimeStr(value string) time.Time {
	cz := time.FixedZone("CST", 8*3600) // 东8区
	if t, e := time.ParseInLocation("2006-01-02 15:04:05", value, cz); e == nil {
		return t
	}
	return defaultZeroTime()
}

// Md5Str MD5哈希计算
func Md5Str(text string) string {
	h := md5.New()
	h.Write([]byte(text))
	re := h.Sum(nil)
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "%x", re)
	return strings.ToLower(sb.String())
}

// IsLocalAbsPath 是否是本地绝对路径
func IsLocalAbsPath(filePath string) bool {
	if runtime.GOOS == "windows" {
		// 是否是windows路径
		matched, _ := regexp.MatchString("^([a-zA-Z]:)", filePath)
		if matched {
			// windows volume label
			return true
		}
		return false
	} else {
		return path.IsAbs(filePath)
	}
}

// IsPanAbsPath 是否是云盘绝对路径
func IsPanAbsPath(filePath string) bool {
	return path.IsAbs(filePath)
}

// IsExcludeFile 是否是指定排除的文件
func IsExcludeFile(filePath string, excludeNames *[]string) bool {
	if excludeNames == nil || len(*excludeNames) == 0 {
		return false
	}

	for _, pattern := range *excludeNames {
		fileName := path.Base(strings.ReplaceAll(filePath, "\\", "/"))
		m, _ := regexp.MatchString(pattern, fileName)
		if m {
			return true
		}
	}
	return false
}

// ResizeUploadBlockSize 自动调整分片大小，方便支持极大单文件上传。返回新的分片大小
func ResizeUploadBlockSize(fileSize, defaultBlockSize int64) int64 {
	if (aliyunpan.MaxPartNum * defaultBlockSize) > fileSize {
		return defaultBlockSize
	}
	sizeOfMB := int64(math.Ceil(float64(fileSize) / float64(aliyunpan.MaxPartNum) / 1024.0))
	return sizeOfMB * 1024
}

// UnixTime2LocalFormatStr 时间戳转换为东8区时间字符串
func UnixTime2LocalFormatStr(unixTime int64) string {
	t := time.Unix(unixTime, 0)
	cz := time.FixedZone("CST", 8*3600) // 东8区
	return t.In(cz).Format("2006-01-02T15:04:05.000Z")
}
