package plugins

import (
	"crypto/md5"
	"crypto/tls"
	"fmt"
	"github.com/jordan-wright/email"
	jsoniter "github.com/json-iterator/go"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/bolt"
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester"
	"net/smtp"
	"os"
	"strings"
	"sync"
	"time"
)

type (
	PersistenceItem struct {
		Key   string `json:"key"`
		Value string `json:"value"`
		Type  string `json:"type"`
	}
)

var (
	locker              = &sync.Mutex{}
	PersistenceFilePath = ""
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

// DeletePanFile 删除云盘文件，支持文件和文件夹
func DeletePanFile(userId, driveId, panFileId string) bool {
	logger.Verboseln("[Plugin] try to delete pan file: userId=" + userId + ", driveId=" + driveId + ", panFileId=" + panFileId)
	if userId == "" || driveId == "" || panFileId == "" {
		return false
	}
	user := config.Config.UserList.GetUserByUserId(userId)
	if user == nil {
		logger.Verboseln("[Plugin] user not existed: ", userId)
		return false
	}
	fdr, err := user.PanClient().OpenapiPanClient().FileDelete(&aliyunpan.FileBatchActionParam{
		DriveId: driveId,
		FileId:  panFileId,
	})
	if err != nil || !fdr.Success {
		logger.Verboseln("[Plugin] delete pan file error: ", err)
		return false
	} else {
		logger.Verboseln("[Plugin] delete pan file success: ", panFileId)
		return true
	}
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

// GetString 获取值
func GetString(key string) string {
	locker.Lock()
	defer locker.Unlock()
	db, err := bolt.Open(PersistenceFilePath, 0755, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return ""
	}
	defer db.Close()

	tx, err := db.Begin(false)
	if err != nil {
		return ""
	}
	bkt := tx.Bucket([]byte("/"))
	if bkt == nil {
		return ""
	}
	data := bkt.Get([]byte(key))
	item := PersistenceItem{}
	if e := jsoniter.Unmarshal(data, &item); e == nil {
		return item.Value
	}
	return ""
}

// PutString 存储KV键值对
func PutString(key, value string) bool {
	locker.Lock()
	defer locker.Unlock()
	db, err := bolt.Open(PersistenceFilePath, 0755, &bolt.Options{Timeout: 5 * time.Second})
	if err != nil {
		return false
	}
	defer db.Close()

	// Start a writable transaction.
	tx, err := db.Begin(true)
	if err != nil {
		return false
	}
	defer tx.Rollback()

	rootBucket, er := tx.CreateBucketIfNotExists([]byte("/"))
	if er != nil {
		return false
	}
	data, ee := jsoniter.Marshal(&PersistenceItem{
		Key:   key,
		Value: value,
		Type:  "string",
	})
	if ee != nil {
		return false
	}
	if e := rootBucket.Put([]byte(key), data); e != nil {
		return false
	}

	// Commit the transaction and check for error.
	if err := tx.Commit(); err != nil {
		return false
	}
	return true
}

// Md5Hex md5加密
func Md5Hex(text string) string {
	hash := md5.Sum([]byte(text))
	return fmt.Sprintf("%x", hash[:])
}
