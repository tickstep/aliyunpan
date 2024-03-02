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
	"github.com/tickstep/aliyunpan-api/aliyunpan_open"
	"github.com/tickstep/aliyunpan-api/aliyunpan_open/openapi"
	"github.com/tickstep/aliyunpan/internal/functions/panlogin"
	"github.com/tickstep/library-go/expires/cachemap"
	"github.com/tickstep/library-go/logger"
	"path"
	"path/filepath"
	"time"
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

// PanClientToken 授权Token
type PanClientToken struct {
	// AccessToken AccessToken
	AccessToken string `json:"accessToken"`
	// Expired 过期时间戳，单位秒
	Expired int64 `json:"expired"`
}

// GetExpiredTimeCstStr 获取东八区时间字符串
func (t *PanClientToken) GetExpiredTimeCstStr() string {
	cz := time.FixedZone("CST", 8*3600) // 东8区
	tm := time.Unix(t.Expired, 0).In(cz)
	return tm.Format("2006-01-02 15:04:05")
}

type PanUser struct {
	// 用户信息
	UserId      string `json:"userId"`
	Nickname    string `json:"nickname"`
	AccountName string `json:"accountName"`

	// 文件（备份盘）
	Workdir           string               `json:"workdir"`
	WorkdirFileEntity aliyunpan.FileEntity `json:"workdirFileEntity"`

	// 资源库
	ResourceWorkdir           string               `json:"resourceWorkdir"`
	ResourceWorkdirFileEntity aliyunpan.FileEntity `json:"resourceWorkdirFileEntity"`

	// 相册
	AlbumWorkdir           string               `json:"albumWorkdir"`
	AlbumWorkdirFileEntity aliyunpan.FileEntity `json:"albumWorkdirFileEntity"`

	// 用户网盘信息
	ActiveDriveId string        `json:"activeDriveId"`
	DriveList     DriveInfoList `json:"driveList"`

	// 授权Token信息
	TicketId     string          `json:"ticketId"`
	WebapiToken  *PanClientToken `json:"webapiToken"`
	OpenapiToken *PanClientToken `json:"openapiToken"`

	// API客户端
	//panClient  *aliyunpan.PanClient `json:"-"`
	panClient  *PanClient          `json:"-"`
	cacheOpMap cachemap.CacheOpMap `json:"-"`
}

type PanUserList []*PanUser

func SetupUserByCookie(openapiToken, webapiToken *PanClientToken, ticketId, userId, deviceId, deviceName, clientId, clientSecret string) (user *PanUser, err *apierror.ApiError) {
	tryRefreshWebToken := true
	tryRefreshOpenToken := true
	loginHelper := panlogin.NewLoginHelper(DefaultTokenServiceWebHost)

	if openapiToken == nil {
		return nil, apierror.NewFailedApiError("openapi token is empty")
	}
	if webapiToken == nil {
		return nil, apierror.NewFailedApiError("webapi token is empty")
	}

doOpenLoginAct:
	// setup openapi client
	openPanClient := aliyunpan_open.NewOpenPanClient(openapi.ApiConfig{
		TicketId:     ticketId,
		UserId:       userId,
		ClientId:     clientId,
		ClientSecret: clientSecret,
	}, openapi.ApiToken{
		AccessToken: openapiToken.AccessToken,
		ExpiredAt:   openapiToken.Expired,
	}, nil)

	// open api token maybe expired
	// check & refresh new one
	openUserInfo, err := openPanClient.GetUserInfo()
	if err != nil {
		if err.Code == apierror.ApiCodeTokenExpiredCode && tryRefreshOpenToken {
			tryRefreshOpenToken = false
			wt, e := loginHelper.GetOpenapiNewToken(ticketId, userId)
			if e != nil {
				logger.Verboseln("get openapi token from server error: ", e)
				return nil, apierror.NewFailedApiError("get new openapi token error, try login again")
			}
			if wt != nil {
				openapiToken = &PanClientToken{
					AccessToken: wt.AccessToken,
					Expired:     wt.Expired,
				}
			}
			time.Sleep(time.Duration(1) * time.Second)
			goto doOpenLoginAct
		}
		return nil, err
	}

doWebLoginAct:
	// setup webapi client
	var webUserInfo *aliyunpan.UserInfo
	var err2 *apierror.ApiError
	var webPanClient *aliyunpan.PanClient
	if webapiToken != nil && webapiToken.AccessToken != "" {
		appConfig := aliyunpan.AppConfig{
			AppId:     "25dzX3vbYqktVxyX",
			DeviceId:  deviceId,
			UserId:    userId,
			Nonce:     0,
			PublicKey: "",
		}
		webPanClient = aliyunpan.NewPanClient(aliyunpan.WebLoginToken{
			AccessTokenType: "Bearer",
			AccessToken:     webapiToken.AccessToken,
			RefreshToken:    "",
			ExpiresIn:       7200,
			ExpireTime:      webapiToken.GetExpiredTimeCstStr(),
		}, aliyunpan.AppLoginToken{}, appConfig, aliyunpan.SessionConfig{
			DeviceName: deviceName,
			ModelName:  "Windows网页版",
		})
		// web api token maybe expired
		// check & refresh new one
		webUserInfo, err2 = webPanClient.GetUserInfo()
		if err2 != nil {
			if err2.Code == apierror.ApiCodeTokenExpiredCode && tryRefreshWebToken {
				tryRefreshWebToken = false
				wt, e := loginHelper.GetWebapiNewToken(ticketId, userId)
				if e != nil {
					logger.Verboseln("get web token from server error: ", e)
				}
				if wt != nil {
					webapiToken = &PanClientToken{
						AccessToken: wt.AccessToken,
						Expired:     wt.Expired,
					}
				}
				time.Sleep(time.Duration(1) * time.Second)
				goto doWebLoginAct
			}
			webPanClient = nil
			//return nil, err2
		}
		// web create session
		if webUserInfo != nil {
			appConfig.UserId = webUserInfo.UserId
			webPanClient.UpdateAppConfig(appConfig)
			r, e := webPanClient.CreateSession(nil)
			if e != nil {
				logger.Verboseln("call CreateSession error in SetupUserByCookie: " + e.Error())
			}
			if r != nil && !r.Result {
				logger.Verboseln("上传签名秘钥失败，可能是你账号登录的设备已超最大数量")
			}
		}
	}

	//
	// setup PanUser
	//
	u := &PanUser{
		WebapiToken:       webapiToken,
		OpenapiToken:      openapiToken,
		panClient:         NewPanClient(webPanClient, openPanClient),
		Workdir:           "/",
		WorkdirFileEntity: *aliyunpan.NewFileEntityForRootDir(),
	}
	u.PanClient().OpenapiPanClient().SetAccessTokenRefreshCallback(func(userId string, newToken openapi.ApiToken) error {
		logger.Verboseln("openapi token refresh, update for user")
		u.OpenapiToken = &PanClientToken{
			AccessToken: newToken.AccessToken,
			Expired:     newToken.ExpiredAt,
		}
		return nil
	})

	// setup user info
	name := "Unknown"
	if openUserInfo != nil {
		// fill userId for client
		u.PanClient().OpenapiPanClient().UpdateUserId(openUserInfo.UserId)
		if u.PanClient().WebapiPanClient() != nil {
			u.PanClient().WebapiPanClient().UpdateUserId(openUserInfo.UserId)
		}

		// update user
		if openUserInfo.Nickname != "" {
			name = openUserInfo.Nickname
		}
		u.UserId = openUserInfo.UserId
		u.Nickname = name
		u.AccountName = openUserInfo.UserName

		// default file drive
		if u.ActiveDriveId == "" {
			u.ActiveDriveId = openUserInfo.FileDriveId
		}

		// drive list
		u.DriveList = DriveInfoList{
			{DriveId: openUserInfo.FileDriveId, DriveTag: "File", DriveName: "备份盘"},
			{DriveId: openUserInfo.ResourceDriveId, DriveTag: "Resource", DriveName: "资源库"},
		}
		if webUserInfo != nil {
			u.AccountName = webUserInfo.UserName
			u.DriveList = append(u.DriveList, &DriveInfo{DriveId: webUserInfo.AlbumDriveId, DriveTag: "Album", DriveName: "相册"})
		}
	} else {
		// error, maybe the token has expired
		return nil, apierror.NewFailedApiError("cannot get user info, the token has expired. please login again")
	}

	// return user
	return u, nil
}

func (pu *PanUser) PanClient() *PanClient {
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
		fe, err := pu.PanClient().OpenapiPanClient().FileInfoById(pu.ActiveDriveId, pu.WorkdirFileEntity.FileId)
		if err != nil {
			logger.Verboseln("刷新工作目录信息失败")
			return
		}
		pu.WorkdirFileEntity = *fe
	} else if pu.IsAlbumDriveActive() {
		fe, err := pu.PanClient().OpenapiPanClient().FileInfoById(pu.ActiveDriveId, pu.AlbumWorkdirFileEntity.FileId)
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
