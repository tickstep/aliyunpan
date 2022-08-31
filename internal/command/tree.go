package command

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/urfave/cli"
	"path"
	"strings"
)

func CmdTree() cli.Command {
	return cli.Command{
		Name:      "tree",
		Usage:     "列出目录的树形图",
		UsageText: cmder.App().Name + " tree <目录>",
		Description: `
	列出指定目录内的文件和目录, 并以树形图的方式呈现

	示例:

	列出 当前工作目录 内的文件和目录的树形图
	aliyunpan tree

	列出 /我的资源 内的文件和目录的树形图
	aliyunpan tree /我的资源

	列出 /我的资源 内的文件和目录的树形图，并且显示文件对应的完整绝对路径
	aliyunpan tree -fp /我的资源
`,
		Category: "阿里云盘",
		Before:   cmder.ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if config.Config.ActiveUser() == nil {
				fmt.Println("未登录账号")
				return nil
			}
			RunTree(parseDriveId(c), c.Args().Get(0), c.Bool("fp"))
			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "driveId",
				Usage: "网盘ID",
				Value: "",
			},
			cli.BoolFlag{
				Name:  "fp",
				Usage: "full path， 树形图是否显示文件完整路径",
			},
		},
	}
}

const (
	indentPrefix   = "│   "
	pathPrefix     = "├──"
	lastFilePrefix = "└──"
)

type (
	treeStatistic struct {
		CountOfDir  int64
		CountOfFile int64
	}
)

func getTree(driveId, pathStr string, depth int, statistic *treeStatistic, showFullPath bool) {
	activeUser := config.Config.ActiveUser()
	pathStr = activeUser.PathJoin(driveId, pathStr)
	pathStr = path.Clean(pathStr)

	targetPathInfo, err1 := activeUser.PanClient().FileInfoByPath(driveId, pathStr)
	if err1 != nil {
		fmt.Println(err1)
		return
	}
	fileList := aliyunpan.FileList{}
	fileListParam := &aliyunpan.FileListParam{}
	fileListParam.ParentFileId = targetPathInfo.FileId
	fileListParam.DriveId = driveId
	fileListParam.OrderBy = aliyunpan.FileOrderByName
	fileListParam.OrderDirection = aliyunpan.FileOrderDirectionAsc
	if targetPathInfo.IsFolder() {
		fileResult, err := activeUser.PanClient().FileListGetAll(fileListParam, 0)
		if err != nil {
			fmt.Println(err)
			return
		}
		fileList = append(fileList, fileResult...)
	} else {
		fileList = append(fileList, targetPathInfo)
	}

	var (
		prefix          = pathPrefix
		fN              = len(fileList)
		indentPrefixStr = strings.Repeat(indentPrefix, depth)
	)
	for i, file := range fileList {
		if file.IsFolder() {
			statistic.CountOfDir += 1
			if showFullPath {
				fmt.Printf("%v%v %v/ -> %s\n", indentPrefixStr, pathPrefix, file.FileName, pathStr+"/"+file.FileName)
			} else {
				fmt.Printf("%v%v %v/\n", indentPrefixStr, pathPrefix, file.FileName)
			}
			getTree(driveId, pathStr+"/"+file.Path, depth+1, statistic, showFullPath)
			continue
		}
		statistic.CountOfFile += 1

		if i+1 == fN {
			prefix = lastFilePrefix
		}

		if showFullPath {
			fmt.Printf("%v%v %v -> %s\n", indentPrefixStr, prefix, file.FileName, pathStr+"/"+file.FileName)
		} else {
			fmt.Printf("%v%v %v\n", indentPrefixStr, prefix, file.FileName)
		}
	}

	return
}

// RunTree 列出树形图
func RunTree(driveId, pathStr string, showFullPath bool) {
	activeUser := config.Config.ActiveUser()
	activeUser.PanClient().ClearCache()
	activeUser.PanClient().EnableCache()
	defer activeUser.PanClient().DisableCache()
	pathStr = activeUser.PathJoin(driveId, pathStr)
	statistic := &treeStatistic{
		CountOfDir:  0,
		CountOfFile: 0,
	}
	fmt.Printf("%s\n", pathStr)
	getTree(driveId, pathStr, 0, statistic, showFullPath)
	fmt.Printf("\n%d 个文件夹, %d 个文件\n", statistic.CountOfDir, statistic.CountOfFile)
}
