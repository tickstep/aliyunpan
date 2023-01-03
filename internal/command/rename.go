// Copyright (c) 2020 tickstep.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package command

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apiutil"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/urfave/cli"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type (
	fileItem struct {
		file        *aliyunpan.FileEntity
		newFileName string
	}
	fileArray []*fileItem
)

func newFileItem(file *aliyunpan.FileEntity) *fileItem {
	return &fileItem{
		file:        file,
		newFileName: "",
	}
}

func (x fileArray) Len() int {
	return len(x)
}
func (x fileArray) Less(i, j int) bool {
	return x[i].file.FileName < x[j].file.FileName
}
func (x fileArray) Swap(i, j int) {
	x[i], x[j] = x[j], x[i]
}

func CmdRename() cli.Command {
	return cli.Command{
		Name:  "rename",
		Usage: "重命名文件",
		UsageText: `重命名文件:
	aliyunpan rename <旧文件/目录名> <新文件/目录名>`,
		Description: `
	示例:
    1. 将文件 1.mp4 重命名为 2.mp4
    aliyunpan rename 1.mp4 2.mp4

    2. 将文件 /test/1.mp4 重命名为 /test/2.mp4
    要求必须是同一个文件目录内
    aliyunpan rename /test/1.mp4 /test/2.mp4

    批量重命名，规则：rename [expression] [replacement] [file]
    其中，
    expression - 表达式，即文件名中需要去掉的部分，如果为*代表旧的文件名全部去掉
    replacement - 需要替换的新文件名，如果包含 # 号会按顺序替换成数字编号
    file - 文件匹配模式，指定需要查找的文件匹配字符串，支持通配符，例如：*.mp4 即当前目录下所有的mp4文件

    3. 批量重命名，将当前目录下所有.mp4文件中包含的字符串 "我的视频" 全部改成 "视频"
    aliyunpan rename 我的视频 视频 *.mp4

    4. 批量重命名，将当前目录下所有.mp4文件全部进行 "视频+编号.mp4" 的重命名操作，旧的名称全部去掉，即：视频001.mp4, 视频002.mp4, 视频003.mp4...
    aliyunpan rename * 视频###.mp4 *.mp4

    5. 批量重命名，将当前目录下所有.mp4文件全部进行 "视频+编号.mp4" 的重命名操作，旧的名称全部去掉，直接重命名无需人工确认操作
    aliyunpan rename -y * 视频###.mp4 *.mp4
`,
		Category: "阿里云盘",
		Before:   ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if c.NArg() != 2 && c.NArg() != 3 {
				cli.ShowCommandHelp(c, c.Command.Name)
				return nil
			}
			if config.Config.ActiveUser() == nil {
				fmt.Println("未登录账号")
				return nil
			}
			if c.NArg() == 2 {
				RunRename(parseDriveId(c), c.Args().Get(0), c.Args().Get(1))
			} else if c.NArg() == 3 {
				// 批量重命名
				RunRenameBatch(c.Bool("y"), parseDriveId(c), c.Args().Get(0), c.Args().Get(1), c.Args().Get(2))
			}
			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "driveId",
				Usage: "网盘ID",
				Value: "",
			},
			cli.BoolFlag{
				Name:  "y",
				Usage: "跳过人工确认，对批量操作有效",
			},
		},
	}
}

func RunRename(driveId string, oldName string, newName string) {
	if oldName == "" {
		fmt.Println("请指定命名文件")
		return
	}
	if newName == "" {
		fmt.Println("请指定文件新名称")
		return
	}
	activeUser := GetActiveUser()
	oldName = activeUser.PathJoin(driveId, strings.TrimSpace(oldName))
	newName = activeUser.PathJoin(driveId, strings.TrimSpace(newName))
	if path.Dir(oldName) != path.Dir(newName) {
		fmt.Println("只能命名同一个目录的文件")
		return
	}
	if !apiutil.CheckFileNameValid(path.Base(newName)) {
		fmt.Println("文件名不能包含特殊字符：" + apiutil.FileNameSpecialChars)
		return
	}

	fileId := ""
	r, err := GetActivePanClient().FileInfoByPath(driveId, activeUser.PathJoin(driveId, oldName))
	if err != nil {
		fmt.Printf("原文件不存在： %s, %s\n", oldName, err)
		return
	}
	fileId = r.FileId

	b, e := activeUser.PanClient().FileRename(driveId, fileId, path.Base(newName))
	if e != nil {
		fmt.Println(e.Err)
		return
	}
	if !b {
		fmt.Println("重命名文件失败")
		return
	}
	fmt.Printf("重命名文件成功：%s -> %s\n", path.Base(oldName), path.Base(newName))
	activeUser.DeleteOneCache(path.Dir(newName))
}

// RunRenameBatch 批量重命名文件
func RunRenameBatch(skipConfirm bool, driveId string, expression, replacement, filePattern string) {
	if len(expression) == 0 {
		fmt.Println("旧文件名不能为空")
		return
	}
	if !apiutil.CheckFileNameValid(replacement) {
		fmt.Println("新文件名不能包含特殊字符：" + apiutil.FileNameSpecialChars)
		return
	}

	if len(filePattern) == 0 {
		fmt.Println("文件匹配模式不能为空")
		return
	}
	if strings.ContainsAny(filePattern, "/") {
		fmt.Println("文件匹配模式不能包含路径分隔符")
		return
	}
	if strings.ContainsAny(filePattern, "\\") {
		fmt.Println("文件匹配模式不能包含路径分隔符")
		return
	}

	activeUser := GetActiveUser()
	absolutePath := path.Clean(activeUser.PathJoin(driveId, filePattern))
	fileList, err1 := matchPathByShellPattern(driveId, absolutePath)
	if err1 != nil {
		fmt.Println("查询文件出错：" + err1.Error())
		return
	}
	if fileList == nil || len(fileList) == 0 {
		fmt.Println("没有找到符合的文件")
		return
	}

	// 文件列表按照名字排序
	files := fileArray{}
	for _, f := range fileList {
		files = append(files, newFileItem(f))
	}
	if len(files) > 1 {
		sort.Sort(files)
	}

	// 依次处理
	var index = 1
	for _, file := range files {
		replacementStr := replaceNumStr(replacement, index)
		index += 1
		if expression == "*" {
			// 旧的名称全部替换
			file.newFileName = replacementStr
		} else {
			// 替换指定的名称片段
			file.newFileName = strings.ReplaceAll(file.file.FileName, expression, replacementStr)
		}
	}

	// 确认
	if !skipConfirm {
		fmt.Printf("以下文件将进行对应的重命名\n\n")
		idx := 1
		for _, file := range files {
			fmt.Printf("%d) %s -> %s\n", idx, file.file.FileName, file.newFileName)
			idx += 1
		}
		fmt.Printf("\n是否进行批量重命名，该操作不可逆(y/n): ")
		confirm := ""
		_, err := fmt.Scanln(&confirm)
		if err != nil || (confirm != "y" && confirm != "Y") {
			fmt.Println("用户取消了操作")
			return
		}
	}

	// 重命名
	for _, file := range files {
		b, e := activeUser.PanClient().FileRename(driveId, file.file.FileId, file.newFileName)
		if e != nil {
			fmt.Println(e.Err)
			return
		}
		if !b {
			fmt.Println("重命名文件失败")
			return
		}
		fmt.Printf("重命名文件成功：%s -> %s\n", file.file.FileName, file.newFileName)
		activeUser.DeleteOneCache(path.Dir(file.file.Path))
	}
}

// replaceNumStr 将#替换成数字编号
func replaceNumStr(name string, num int) string {
	pattern, _ := regexp.Compile("[#]+")
	r := pattern.FindAllStringSubmatchIndex(name, -1)
	if len(r) == 0 {
		return name
	}
	newName := name
	c := r[0]
	return replaceNumStr(newName[:c[0]]+generateNumStr(num, c[1]-c[0])+newName[c[1]:], num)
}
func generateNumStr(num, count int) string {
	s := strconv.Itoa(num)
	return generateZeroStr(count-len(s)) + s
}
func generateZeroStr(count int) string {
	if count <= 0 {
		return ""
	}
	s := ""
	for i := 0; i < count; i++ {
		s += "0"
	}
	return s
}
