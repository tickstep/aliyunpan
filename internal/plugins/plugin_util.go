package plugins

import (
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester"
	"os"
)

// HttpGet Http的get请求
func HttpGet(header map[string]string, url string) string {
	client := requester.NewHTTPClient()
	body, err := client.Fetch("GET", url, nil, header)
	if err != nil {
		logger.Verboseln("js HttpJsonGet error ", err)
		return ""
	}
	return string(body)
}

// HttpPost Http的post请求
func HttpPost(header map[string]string, url string, data interface{}) string {
	client := requester.NewHTTPClient()
	body, err := client.Fetch("POST", url, data, header)
	if err != nil {
		logger.Verboseln("js HttpJsonPost error ", err)
		return ""
	}
	return string(body)
}

// DeleteLocalFile 删除本地文件，不支持文件夹
func DeleteLocalFile(localFilePath string) bool {
	err := os.Remove(localFilePath)
	if err != nil {
		// 删除失败
		return false
	} else {
		// 删除成功
		return true
	}
	return false
}
