package plugins

import (
	"fmt"
	"testing"
)

func TestDeleteLocalFile(t *testing.T) {
	fmt.Println(DeleteLocalFile("/Volumes/Downloads/dev/upload/2"))
}

func TestSendEmail(t *testing.T) {
	fmt.Println(sendEmail("smtp.qq.com:465", "111xxx@qq.com", "xxxxxx", "12545xxx@qq.com", "title", "hello", "text", true))
}

func TestPutString(t *testing.T) {
	PersistenceFilePath = "/Volumes/Downloads/kv.bolt"
	PutString("test1", "ok1234-new")
}

func TestGetString(t *testing.T) {
	PersistenceFilePath = "/Volumes/Downloads/kv.bolt"
	v := GetString("test1")
	fmt.Println(v)
}
