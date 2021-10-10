package command

import (
	"fmt"
	"testing"
)

func TestRapidUploadItem_createRapidUploadLink(t *testing.T) {
	item := &RapidUploadItem{
		FileSha1: "752FCCBFB2436A6FFCA3B287831D4FAA5654B07E",
		FileSize: 7005440,
		FilePath: "/dgsdg/rtt5/我的文件夹/file我的文件.dmg",
	}
	fmt.Println(item.createRapidUploadLink(false))
}

func TestRapidUploadItem_newRapidUploadItem(t *testing.T) {
	link := "aliyunpan://file我的文件.dmg|752FCCBFB2436A6FFCA3B287831D4FAA5654B07E|7005440|dgsdg%2Frtt5%2F%E6%88%91%E7%9A%84%E6%96%87%E4%BB%B6%E5%A4%B9"
	item,_ := newRapidUploadItem(link)
	fmt.Println(item)
}