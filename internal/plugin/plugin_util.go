package plugin

import (
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester"
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
