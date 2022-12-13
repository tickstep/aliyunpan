package utils

import (
	"fmt"
	"testing"
	"time"
)

func TestConvertTime(t *testing.T) {
	seconds := time.Duration(50) * time.Second
	fmt.Println(ConvertTime(seconds))

	seconds = time.Duration(150) * time.Second
	fmt.Println(ConvertTime(seconds))

	seconds = time.Duration(3600) * time.Second
	fmt.Println(ConvertTime(seconds))

	seconds = time.Duration(1246852) * time.Second
	fmt.Println(ConvertTime(seconds))
}

func TestUuidStr(t *testing.T) {
	fmt.Println(UuidStr())
}

func TestMd5Str(t *testing.T) {
	fmt.Println(Md5Str("123456"))
}

func TestParseTimeStr(t *testing.T) {
	fmt.Println(ParseTimeStr(""))
}

func TestIsAbsPath_ReturnTrue(t *testing.T) {
	fmt.Println(IsLocalAbsPath("D:\\my\\folder\\test"))
}

func TestIsAbsPath_ReturnFalse(t *testing.T) {
	fmt.Println(IsLocalAbsPath("my\\folder\\test"))
}

func TestResizeUploadBlockSize_ReturnDefaultBlockSize(t *testing.T) {
	MB := int64(1024 * 1024)                            // 1048576
	fileSize := int64(1073741824)                       // 90GB
	fmt.Println(ResizeUploadBlockSize(fileSize, 10*MB)) // 10485760 = 10240KB
}

func TestResizeUploadBlockSize_ReturnNewBlockSize(t *testing.T) {
	MB := int64(1024 * 1024)                            // 1048576
	fileSize := int64(107374182400)                     // 100GB
	fmt.Println(ResizeUploadBlockSize(fileSize, 10*MB)) // 10737664 = 10486KB
}
