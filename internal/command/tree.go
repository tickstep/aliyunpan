package command

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/library-go/converter"
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

	列出 /我的资源 内的文件和目录的树形图，使用通配符
	aliyunpan tree /我的*

	列出 /我的资源 内的文件和目录的树形图，并且显示文件对应的完整绝对路径
	aliyunpan tree -fp /我的资源

	列出 /我的资源 内的文件和目录的树形图，并且显示文件对应的文件大小
	aliyunpan tree -fs /我的资源

	列出 /我的资源 内的文件和目录的树形图，过滤大于等于 10kb 并且小于等于 10mb 的文件，同时显示文件对应的文件大小
	aliyunpan tree -fs -minSize=1kb -maxSize=10mb /我的资源
`,
		Category: "阿里云盘",
		Before:   ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if config.Config.ActiveUser() == nil {
				fmt.Println("未登录账号")
				return nil
			}
			minSize := int64(0)
			if c.IsSet("minSize") {
				if s, e := converter.ParseFileSizeStr(c.String("minSize")); e == nil {
					minSize = s
				}
			}
			maxSize := int64(0)
			if c.IsSet("maxSize") {
				if s, e := converter.ParseFileSizeStr(c.String("maxSize")); e == nil {
					maxSize = s
				}
			}
			RunTree(parseDriveId(c), c.Args().Get(0), c.Bool("fp"), c.Bool("fs"), minSize, maxSize)
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
				Usage: "full path， 树形图显示文件的完整路径",
			},
			cli.BoolFlag{
				Name:  "fs",
				Usage: "file size， 树形图显示文件的文件大小",
			},
			cli.StringFlag{
				Name:  "minSize",
				Usage: "min size， 过滤大于等于指定大小的文件，例如：100mb",
			},
			cli.StringFlag{
				Name:  "maxSize",
				Usage: "max size， 过滤小于等于指定大小的文件，例如：1gb",
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
		SizeOfFile  int64
	}

	treeConfig struct {
		showFullPath bool
		showFileSize bool
		minFileSize  int64
		maxFileSize  int64
	}
)

func getTree(driveId, pathStr string, depth int, statistic *treeStatistic, setting *treeConfig) {
	activeUser := config.Config.ActiveUser()
	pathStr = activeUser.PathJoin(driveId, pathStr)
	pathStr = path.Clean(pathStr)

	files, err := matchPathByShellPattern(driveId, pathStr)
	if err != nil {
		fmt.Println(err)
		return
	}

	var targetPathInfo *aliyunpan.FileEntity
	if len(files) == 1 {
		targetPathInfo = files[0]
	} else {
		for _, f := range files {
			if f.IsFolder() {
				targetPathInfo = f
				break
			}
		}
	}
	if targetPathInfo == nil {
		fmt.Println("路径不存在")
		return
	}

	if depth == 0 {
		fmt.Printf("%s\n", targetPathInfo.Path)
	}

	fileList := aliyunpan.FileList{}
	fileListParam := &aliyunpan.FileListParam{}
	fileListParam.ParentFileId = targetPathInfo.FileId
	fileListParam.DriveId = driveId
	fileListParam.OrderBy = aliyunpan.FileOrderByName
	fileListParam.OrderDirection = aliyunpan.FileOrderDirectionAsc
	if targetPathInfo.IsFolder() {
		fileResult, err := activeUser.PanClient().WebapiPanClient().FileListGetAll(fileListParam, 500)
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
			if setting.showFullPath {
				fmt.Printf("%v%v %v/ -> %s\n", indentPrefixStr, pathPrefix, file.FileName, targetPathInfo.Path+"/"+file.FileName)
			} else {
				fmt.Printf("%v%v %v/\n", indentPrefixStr, pathPrefix, file.FileName)
			}
			getTree(driveId, targetPathInfo.Path+"/"+file.Path, depth+1, statistic, setting)
			continue
		}

		// filter file size
		if setting.minFileSize > 0 {
			if file.FileSize < setting.minFileSize {
				continue
			}
		}
		if setting.maxFileSize > 0 {
			if file.FileSize > setting.maxFileSize {
				continue
			}
		}

		statistic.CountOfFile += 1
		statistic.SizeOfFile += file.FileSize

		if i+1 == fN {
			prefix = lastFilePrefix
		}

		// 文件大小
		fileName := &strings.Builder{}
		if setting.showFileSize {
			fmt.Fprintf(fileName, "%s (%s)", file.FileName, converter.ConvertFileSize(file.FileSize, 2))
		} else {
			fmt.Fprintf(fileName, "%s", file.FileName)
		}

		// 文件完整路径
		if setting.showFullPath {
			fmt.Printf("%v%v %v -> %s\n", indentPrefixStr, prefix, fileName.String(), targetPathInfo.Path+"/"+file.FileName)
		} else {
			fmt.Printf("%v%v %v\n", indentPrefixStr, prefix, fileName.String())
		}
	}

	return
}

// RunTree 列出树形图
func RunTree(driveId, pathStr string, showFullPath, showFileSize bool, minSize, maxSize int64) {
	activeUser := config.Config.ActiveUser()
	activeUser.PanClient().WebapiPanClient().ClearCache()
	activeUser.PanClient().WebapiPanClient().EnableCache()
	defer activeUser.PanClient().WebapiPanClient().DisableCache()
	pathStr = activeUser.PathJoin(driveId, pathStr)
	statistic := &treeStatistic{
		CountOfDir:  0,
		CountOfFile: 0,
		SizeOfFile:  0,
	}
	setting := &treeConfig{
		showFullPath: showFullPath,
		showFileSize: showFileSize,
		minFileSize:  minSize,
		maxFileSize:  maxSize,
	}
	getTree(driveId, pathStr, 0, statistic, setting)
	fmt.Printf("\n%d 个文件夹, %d 个文件, %s 总大小\n", statistic.CountOfDir, statistic.CountOfFile, converter.ConvertFileSize(statistic.SizeOfFile, 2))
}
