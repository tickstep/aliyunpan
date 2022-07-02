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

// SymlinkFile 软链接文件，Linux/macOS的ln，Windows的mklink命令创建的文件链接。对于非软链接文件而言，真实的路径和逻辑路径是一样的。
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
	if p == "." {
		p = ""
	}
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

// RetrieveRealPathFromLogicPath 遍历路径获取逻辑路径真正的文件路径。如果逻辑路径不完全存在，则返回已经存在的那部分路径
//
// logicFilePath - 目标逻辑路径
// 由于目标逻辑路径期间有可能会经过多次符号逻辑文件，同时有部分逻辑路径可能是不存在的，所以需要按照逻辑文件起始开始进行遍历，直至完成逻辑文件路径的所有遍历，
// 或者直到不存在的逻辑文件部分，然后返回。
//
// 例如逻辑文件路径：/Volumes/Downloads/dev/sync_drive_config.json。
//
// 如果/Volumes/Downloads存在而后面部分不存在，则最终结果返回/Volumes/Downloads对应的文件信息，同时返回error
func RetrieveRealPathFromLogicPath(logicFilePath string) (SymlinkFile, os.FileInfo, error) {
	logicFilePath = strings.ReplaceAll(logicFilePath, "\\", "/")
	logicFilePath = path.Clean(logicFilePath)
	if logicFilePath == "." {
		logicFilePath = ""
	}
	if logicFilePath == "/" {
		return RetrieveRealPath(NewSymlinkFile(logicFilePath))
	}
	pathParts := strings.Split(logicFilePath, "/")
	exitedSymlinkFile := NewSymlinkFile("")
	var exitedFileInfo os.FileInfo

	sf := NewSymlinkFile("")
	var fi os.FileInfo
	var err error
	for _, p := range pathParts {
		if p == "" {
			continue
		}
		if strings.Contains(p, ":") {
			// windows volume label, e.g: C:/ D:/
			sf.LogicPath += p
			sf.RealPath += p
			exitedSymlinkFile = sf
			continue
		}
		sf.LogicPath += "/" + p
		sf.RealPath += "/" + p
		sf, fi, err = RetrieveRealPath(sf)
		if err != nil {
			// may be permission deny or not existed
			return exitedSymlinkFile, exitedFileInfo, err
		}
		exitedSymlinkFile = sf
		exitedFileInfo = fi
	}
	return exitedSymlinkFile, exitedFileInfo, nil
}

// WalkAllFile 遍历本地文件，支持软链接（符号逻辑）文件(Linux & Windows & macOS)
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
		if fi == nil {
			continue
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
