package localfile

import (
	"fmt"
	"github.com/tickstep/aliyunpan/internal/utils"
	"os"
	"path/filepath"
	"testing"
)

func TestMyWalkFile(t *testing.T) {
	count := 0
	walkFunc := func(file SymlinkFile, fi os.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return err
		}
		count += 1
		fmt.Println("file: ", utils.ObjectToJsonStr(file, false))
		//fmt.Println("file: ", file)
		return nil
	}

	//curPath := "D:\\smb\\feny\\goprojects\\dev\\lks"
	curPath := "/Volumes/Downloads/dev/lks"
	file := NewSymlinkFile(curPath)
	if err := WalkAllFile(file, walkFunc); err != nil {
		if err != filepath.SkipDir {
			fmt.Printf("警告: 遍历错误: %s\n", err)
		}
	}
	fmt.Println("count: ", count)
}

func TestRetrieveRealPath(t *testing.T) {
	curPath := "/Volumes/Downloads/dev/lks/test"
	file := NewSymlinkFile(curPath)
	sf, _, e := RetrieveRealPath(file)
	if e != nil {
		fmt.Println(e)
	}
	fmt.Println(sf)
}

func TestRetrieveRealPathFromLogicPath(t *testing.T) {
	curPath := "/Volumes/Downloads/dev/lks/test/未命名文件夹cmd/sync_drive_config.json"
	sf, _, e := RetrieveRealPathFromLogicPath(curPath)
	if e != nil {
		fmt.Println(e)
	}
	fmt.Println(sf)
}

func TestRetrieveRealPathFromLogicSuffixPath(t *testing.T) {
	rootPath := NewSymlinkFile("/Volumes/Downloads/dev/测试同步盘/new_lks")
	rootPath, _, _ = RetrieveRealPath(rootPath)
	suffixPath := "test/未命名文件夹cmd/sync_drive_config.json"
	sf, _, e := RetrieveRealPathFromLogicSuffixPath(rootPath, suffixPath)
	if e != nil {
		fmt.Println(e)
	}
	fmt.Println(sf)
}
