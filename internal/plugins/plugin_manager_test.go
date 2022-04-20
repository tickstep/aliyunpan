package plugins

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"testing"
)

func TestPlugin(t *testing.T) {
	pluginManager := NewPluginManager("D:\\smb\\feny\\goprojects\\dev")
	plugin, err := pluginManager.GetPlugin()
	if err != nil {
		fmt.Println(err)
	}
	ctx := &Context{
		AppName:      "aliyunpan",
		Version:      "v0.1.3",
		UserId:       "11001d48564f43b3bc5662874f04bb11",
		Nickname:     "tickstep",
		FileDriveId:  "19519121",
		AlbumDriveId: "29519122",
	}
	params := &UploadFilePrepareParams{
		LocalFilePath:      "D:\\Program Files\\aliyunpan\\Downloads\\11001d48564f43b3bc5662874f04bb11\\token.bat",
		LocalFileName:      "token.bat",
		LocalFileSize:      125330,
		LocalFileType:      "file",
		LocalFileUpdatedAt: "2022-04-14 07:05:12",
		DriveId:            "19519221",
		DriveFilePath:      "aliyunpan/Downloads/11001d48564f43b3bc5662874f04bb11/token.bat",
	}
	b, _ := jsoniter.Marshal(ctx)
	fmt.Println(string(b))
	b, _ = jsoniter.Marshal(params)
	fmt.Println(string(b))
	r, e := plugin.UploadFilePrepareCallback(ctx, params)
	if e != nil {
		fmt.Println(e)
	}
	fmt.Println(r)
}
