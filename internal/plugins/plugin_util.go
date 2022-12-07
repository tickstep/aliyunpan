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

// SendTextMail 发送文本邮件
func SendTextMail(mailServer, userName, password, to, subject, body string) bool {
	if e := sendEmail(mailServer, userName, password, to, subject, body, "text", true); e != nil {
		logger.Verboseln("js SendTextMail error ", e)
		return false
	}
	return true
}

// SendHtmlMail 发送HTML富文本邮件
func SendHtmlMail(mailServer, userName, password, to, subject, body string) bool {
	if e := sendEmail(mailServer, userName, password, to, subject, body, "html", true); e != nil {
		logger.Verboseln("js SendHtmlMail error ", e)
		return false
	}
	return true
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
}
