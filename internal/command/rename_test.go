package command

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"sort"
	"testing"
)

func TestRenameSort(t *testing.T) {
	files := fileArray{}
	files = append(files, newFileItem(&aliyunpan.FileEntity{FileName: "0我的文件01.txt"}))
	files = append(files, newFileItem(&aliyunpan.FileEntity{FileName: "4我的文件03.txt"}))
	files = append(files, newFileItem(&aliyunpan.FileEntity{FileName: "3我的文件02.txt"}))
	files = append(files, newFileItem(&aliyunpan.FileEntity{FileName: "1我的文件00.txt"}))

	sort.Sort(files)

	for _, f := range files {
		fmt.Println(f.file.FileName)
	}
}
func TestRenameNum0(t *testing.T) {
	fmt.Println(replaceNumStr("我的文件###.txt", 2))
}
func TestRenameNum1(t *testing.T) {
	fmt.Println(replaceNumStr("我的##文件###.txt", 123))
}
func TestRenameNum2(t *testing.T) {
	fmt.Println(replaceNumStr("我的文件###.txt", 1233))
}
func TestRenameNum3(t *testing.T) {
	fmt.Println(replaceNumStr("我的文件[###].txt", 1233))
}
func TestRenameNum4(t *testing.T) {
	fmt.Println(replaceNumStr("我的文件.txt", 1233))
}

func TestRenameNum5(t *testing.T) {
	fmt.Println(replaceNumStr("", 1233))
}
