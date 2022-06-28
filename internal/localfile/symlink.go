package localfile

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// SymlinkFile 软链接文件，Linux的ln，Windows的mklink命令创建的文件链接。对于非软链接文件而言，真实的路径和逻辑路径是一样的。
type SymlinkFile struct {
	// LogicPath 逻辑路径
	LogicPath string `json:"logicPath"`
	// RealPath 真正的文件路径，即文件的本体
	RealPath string `json:"realPath"`
}

func (s *SymlinkFile) String() string {
	sb := &strings.Builder{}
	fmt.Fprintf(sb, "{\"logicPath\":%s, \"realPath\": %s}", s.LogicPath, s.RealPath)
	return sb.String()
}

func NewSymlinkFile(filePath string) SymlinkFile {
	p := path.Clean(strings.ReplaceAll(filePath, "\\", "/"))
	return SymlinkFile{
		LogicPath: p,
		RealPath:  p,
	}
}

type MyWalkFunc func(path SymlinkFile, info fs.FileInfo, err error) error

// RetrieveRealPath 递归调用找到软链接文件的真实文件对应的路径信息
func RetrieveRealPath(file SymlinkFile) (SymlinkFile, os.FileInfo, error) {
	info, err := os.Lstat(file.RealPath)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			// 软链接文件
			if f, e := os.Readlink(file.RealPath); e == nil {
				file.RealPath = strings.ReplaceAll(f, "\\", "/")
				return RetrieveRealPath(file)
			}
		}
	}
	return file, info, err
}

// WalkAllFile 遍历本地文件，支持软链接文件(Linux & Windows)
func WalkAllFile(file SymlinkFile, walkFn MyWalkFunc) error {
	file.LogicPath = path.Clean(strings.ReplaceAll(file.LogicPath, "\\", "/"))
	file.RealPath = path.Clean(strings.ReplaceAll(file.RealPath, "\\", "/"))

	file, info, err := RetrieveRealPath(file)
	if err != nil {
		err = walkFn(file, nil, err)
	} else {
		err = walkAllFile(file, info, walkFn)
	}
	return err
}

func walkAllFile(file SymlinkFile, info os.FileInfo, walkFn MyWalkFunc) error {
	if !info.IsDir() {
		return walkFn(file, info, nil)
	}

	files, err1 := ioutil.ReadDir(file.RealPath)
	if err1 != nil {
		return walkFn(file, nil, err1)
	}
	for _, fi := range files {
		subFile := SymlinkFile{
			LogicPath: path.Join(path.Clean(file.LogicPath), "/", fi.Name()),
			RealPath:  path.Join(path.Clean(file.RealPath), "/", fi.Name()),
		}
		subFile, fi, err1 = RetrieveRealPath(subFile)
		err := walkFn(subFile, fi, err1)
		if err != nil && err != filepath.SkipDir {
			return err
		}
		if fi.IsDir() {
			if err == filepath.SkipDir {
				continue
			}
			err = walkAllFile(subFile, fi, walkFn)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
