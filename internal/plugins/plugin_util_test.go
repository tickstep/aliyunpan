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
