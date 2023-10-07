// Copyright (c) 2020 tickstep.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package config

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/library-go/expires/cachemap"
	"github.com/tickstep/library-go/logger"
	"path"
	"path/filepath"
)

type DriveInfo struct {
	DriveId   string `json:"driveId"`
	DriveName string `json:"driveName"`
	DriveTag  string `json:"driveTag"`
}
type DriveInfoList []*DriveInfo

func (d DriveInfoList) GetFileDriveId() string {
	for _, drive := range d {
		if drive.DriveTag == "File" {
			return drive.DriveId
		}
	}
	return ""
}

func (d DriveInfoList) GetAlbumDriveId() string {
	for _, drive := range d {
		if drive.DriveTag == "Album" {
			return drive.DriveId
		}
	}
	return ""
}

func (d DriveInfoList) GetResourceDriveId() string {
	for _, drive := range d {
		if drive.DriveTag == "Resource" {
			return drive.DriveId
		}
	}
	return ""
}

type PanUser struct {
	UserId      string `json:"userId"`
	Nickname    string `json:"nickname"`
	AccountName string `json:"accountName"`

	// 文件/备份盘
	Workdir           string               `json:"workdir"`
	WorkdirFileEntity aliyunpan.FileEntity `json:"workdirFileEntity"`

	// 资源库
	ResourceWorkdir           string               `json:"resourceWorkdir"`
	ResourceWorkdirFileEntity aliyunpan.FileEntity `json:"resourceWorkdirFileEntity"`

	// 相册
	AlbumWorkdir           string               `json:"albumWorkdir"`
	AlbumWorkdirFileEntity aliyunpan.FileEntity `json:"albumWorkdirFileEntity"`

	ActiveDriveId string        `json:"activeDriveId"`
	DriveList     DriveInfoList `json:"driveList"`

	RefreshToken string                  `json:"refreshToken"`
	WebToken     aliyunpan.WebLoginToken `json:"webToken"`
	TokenId      string                  `json:"tokenId"`

	panClient  *aliyunpan.PanClient
	cacheOpMap cachemap.CacheOpMap
}

type PanUserList []*PanUser

func SetupUserByCookie(webToken *aliyunpan.WebLoginToken, deviceId, deviceName string) (user *PanUser, err *apierror.ApiError) {
	tryRefreshWebToken := true

	if webToken == nil {
		return nil, apierror.NewFailedApiError("web token is empty")
	}

doLoginAct:
	appConfig := aliyunpan.AppConfig{
		AppId:     "25dzX3vbYqktVxyX",
		DeviceId:  deviceId,
		UserId:    "",
		Nonce:     0,
		PublicKey: "",
	}
	panClient := aliyunpan.NewPanClient(*webToken, aliyunpan.AppLoginToken{}, appConfig, aliyunpan.SessionConfig{
		DeviceName: deviceName,
		ModelName:  "Windows网页版",
	})
	u := &PanUser{
		WebToken:          *webToken,
		panClient:         panClient,
		Workdir:           "/",
		WorkdirFileEntity: *aliyunpan.NewFileEntityForRootDir(),
	}

	// web api token maybe expired
	userInfo, err := panClient.GetUserInfo()
	if err != nil {
		if err.Code == apierror.ApiCodeTokenExpiredCode && tryRefreshWebToken {
			tryRefreshWebToken = false
			webCookie, _ := aliyunpan.GetAccessTokenFromRefreshToken(webToken.RefreshToken)
			if webCookie != nil {
				webToken = webCookie
				goto doLoginAct
			}
		}
		return nil, err
	}
	name := "Unknown"
	if userInfo != nil {
		if userInfo.Nickname != "" {
			name = userInfo.Nickname
		}

		// update user
		u.UserId = userInfo.UserId
		u.Nickname = name
		u.AccountName = userInfo.UserName

		// default file drive
		if u.ActiveDriveId == "" {
			u.ActiveDriveId = userInfo.FileDriveId
		}

		// drive list
		u.DriveList = DriveInfoList{
			{DriveId: userInfo.FileDriveId, DriveTag: "File", DriveName: "备份盘"},
			{DriveId: userInfo.ResourceDriveId, DriveTag: "Resource", DriveName: "资源库"},
			{DriveId: userInfo.AlbumDriveId, DriveTag: "Album", DriveName: "相册"},
		}
	} else {
		// error, maybe the token has expired
		return nil, apierror.NewFailedApiError("cannot get user info, the token has expired")
	}

	// create session
	appConfig.UserId = u.UserId
	panClient.UpdateAppConfig(appConfig)
	r, e := panClient.CreateSession(nil)
	if e != nil {
		logger.Verboseln("call CreateSession error in SetupUserByCookie: " + e.Error())
	}
	if r != nil && !r.Result {
		logger.Verboseln("上传签名秘钥失败，可能是你账号登录的设备已超最大数量")
	}

	return u, nil
}

func (pu *PanUser) PanClient() *aliyunpan.PanClient {
	return pu.panClient
}

// PathJoin 合并工作目录和相对路径p, 若p为绝对路径则忽略
func (pu *PanUser) PathJoin(driveId, p string) string {
	if path.IsAbs(p) {
		return p
	}
	wd := "/"
	di := pu.GetDriveById(driveId)
	if di != nil {
		if di.IsFileDrive() {
			wd = pu.Workdir
		} else if di.IsResourceDrive() {
			wd = pu.ResourceWorkdir
		} else if di.IsAlbumDrive() {
			wd = pu.AlbumWorkdir
		}
	}
	return path.Join(wd, p)
}

func (pu *PanUser) FreshWorkdirInfo() {
	if pu.IsFileDriveActive() {
		fe, err := pu.PanClient().FileInfoById(pu.ActiveDriveId, pu.WorkdirFileEntity.FileId)
		if err != nil {
			logger.Verboseln("刷新工作目录信息失败")
			return
		}
		pu.WorkdirFileEntity = *fe
	} else if pu.IsAlbumDriveActive() {
		fe, err := pu.PanClient().FileInfoById(pu.ActiveDriveId, pu.AlbumWorkdirFileEntity.FileId)
		if err != nil {
			logger.Verboseln("刷新工作目录信息失败")
			return
		}
		pu.AlbumWorkdirFileEntity = *fe
	}

}

// GetSavePath 根据提供的网盘文件路径 panpath, 返回本地储存路径,
// 返回绝对路径, 获取绝对路径出错时才返回相对路径...
func (pu *PanUser) GetSavePath(filePanPath string) string {
	dirStr := filepath.Join(Config.SaveDir, fmt.Sprintf("%s", pu.UserId), filePanPath)
	dir, err := filepath.Abs(dirStr)
	if err != nil {
		dir = filepath.Clean(dirStr)
	}
	return dir
}

func (pu *PanUser) GetDriveByTag(tag string) *DriveInfo {
	for _, item := range pu.DriveList {
		if item.DriveTag == tag {
			return item
		}
	}
	return nil
}

func (pu *PanUser) GetDriveById(id string) *DriveInfo {
	for _, item := range pu.DriveList {
		if item.DriveId == id {
			return item
		}
	}
	return nil
}

func (pu *PanUser) GetActiveDriveInfo() *DriveInfo {
	for _, item := range pu.DriveList {
		if item.DriveId == pu.ActiveDriveId {
			return item
		}
	}
	return nil
}

func (pu *PanUser) IsFileDriveActive() bool {
	d := pu.GetActiveDriveInfo()
	return d != nil && d.IsFileDrive()
}

func (pu *PanUser) IsAlbumDriveActive() bool {
	d := pu.GetActiveDriveInfo()
	return d != nil && d.IsAlbumDrive()
}

func (pu *PanUser) IsResourceDriveActive() bool {
	d := pu.GetActiveDriveInfo()
	return d != nil && d.IsResourceDrive()
}

func (di *DriveInfo) IsFileDrive() bool {
	return di.DriveTag == "File"
}

func (di *DriveInfo) IsAlbumDrive() bool {
	return di.DriveTag == "Album"
}

func (di *DriveInfo) IsResourceDrive() bool {
	return di.DriveTag == "Resource"
}
