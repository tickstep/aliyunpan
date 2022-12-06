package plugins

import (
	"crypto/tls"
	"github.com/jordan-wright/email"
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester"
	"net/smtp"
	"os"
	"strings"
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

func sendEmail(mailServer, userName, password, to, subject, body, mailType string, useSsl bool) error {
	mailServerHost := strings.Split(mailServer, ":")[0]
	auth := smtp.PlainAuth("", userName, password, mailServerHost)

	e := email.NewEmail()
	e.From = userName
	e.To = []string{to}
	e.Subject = subject
	if mailType == "html" {
		e.HTML = []byte(body)
	} else {
		e.Text = []byte(body)
	}

	if useSsl {
		return e.SendWithStartTLS(mailServer, auth, &tls.Config{ServerName: mailServerHost, InsecureSkipVerify: true})
	} else {
		return e.Send(mailServer, auth)
	}
	//
	//// 拼接消息体
	//var contentType string
	//if mailType == "html" {
	//	contentType = "Content-Type: text/" + mailType + "; charset=UTF-8"
	//} else {
	//	contentType = "Content-Type: text/plain" + "; charset=UTF-8"
	//}
	//msg := []byte("To: " + to + "\r\nFrom: " + userName + "\r\nSubject: " + subject + "\r\n" + contentType + "\r\n\r\n" + body)
	//
	//// msg 内容输出查看
	//logger.Verboseln("To: " + to + "\r\n" +
	//	"From: " + userName + "\r\n" +
	//	"Subject: " + subject + "\r\n" +
	//	"" + contentType + "\r\n\r\n" +
	//	"" + body)
	//
	//// 进行身份认证
	//hp := strings.Split(mailServer, ":")
	//auth := smtp.PlainAuth("", userName, password, hp[0])
	//
	//sendTo := strings.Split(to, ";")
	//err := smtp.SendMail(mailServer, auth, userName, sendTo, msg)
	//return err
}
