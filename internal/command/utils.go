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
package command

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan_web"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/functions/panlogin"
	"github.com/tickstep/aliyunpan/internal/plugins"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/library-go/logger"
	"github.com/urfave/cli"
	"math/rand"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	panCommandVerbose = logger.New("PANCOMMAND", config.EnvVerbose)

	saveConfigMutex *sync.Mutex = new(sync.Mutex)

	ReloadConfigFunc = func(c *cli.Context) error {
		err := config.Config.Reload()
		if err != nil {
			fmt.Printf("重载配置错误: %s\n", err)
		}
		return nil
	}

	SaveConfigFunc = func(c *cli.Context) error {
		saveConfigMutex.Lock()
		defer saveConfigMutex.Unlock()
		err := config.Config.Save()
		if err != nil {
			fmt.Printf("保存配置错误: %s\n", err)
		}
		return nil
	}
)

// RunTestShellPattern 执行测试通配符
func RunTestShellPattern(driveId string, pattern string) {
	acUser := GetActiveUser()
	files, err := acUser.PanClient().OpenapiPanClient().MatchPathByShellPattern(driveId, GetActiveUser().PathJoin(driveId, pattern))
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, f := range *files {
		fmt.Printf("%s\n", f.Path)
	}
	return
}

// matchPathByShellPattern 通配符匹配路径，允许返回多个匹配结果
func matchPathByShellPattern(driveId string, patterns ...string) (files []*aliyunpan.FileEntity, e error) {
	acUser := GetActiveUser()
	for k := range patterns {
		ps, err := acUser.PanClient().OpenapiPanClient().MatchPathByShellPattern(driveId, acUser.PathJoin(driveId, patterns[k]))
		if err != nil {
			return nil, err
		}
		files = append(files, *ps...)
	}
	return files, nil
}

// makePathAbsolute 拼接路径，确定返回路径为绝对路径
func makePathAbsolute(driveId string, patterns ...string) (panpaths []string, err error) {
	acUser := GetActiveUser()
	for k := range patterns {
		ps := acUser.PathJoin(driveId, patterns[k])
		panpaths = append(panpaths, ps)
	}
	return panpaths, nil
}

func RandomStr(count int) string {
	//STR_SET := "abcdefjhijklmnopqrstuvwxyzABCDEFJHIJKLMNOPQRSTUVWXYZ1234567890"
	STR_SET := "abcdefjhijklmnopqrstuvwxyz1234567890"
	rand.Seed(time.Now().UnixNano())
	str := strings.Builder{}
	for i := 0; i < count; i++ {
		str.WriteByte(byte(STR_SET[rand.Intn(len(STR_SET))]))
	}
	return str.String()
}

func GetAllPathFolderByPath(pathStr string) []string {
	dirNames := strings.Split(pathStr, "/")
	dirs := []string{}
	p := "/"
	dirs = append(dirs, p)
	for _, s := range dirNames {
		p = path.Join(p, s)
		dirs = append(dirs, p)
	}
	return dirs
}

// EscapeStr 转义字符串
func EscapeStr(s string) string {
	return url.PathEscape(s)
}

// UnescapeStr 反转义字符串
func UnescapeStr(s string) string {
	r, _ := url.PathUnescape(s)
	return r
}

func NewWebLoginToken(accessToken string, expired int64) aliyunpan_web.WebLoginToken {
	webapiToken := &config.PanClientToken{
		AccessToken: accessToken,
		Expired:     expired,
	}
	return aliyunpan_web.WebLoginToken{
		AccessTokenType: "Bearer",
		AccessToken:     webapiToken.AccessToken,
		RefreshToken:    "",
		ExpiresIn:       7200,
		ExpireTime:      webapiToken.GetExpiredTimeCstStr(),
	}
}

// RefreshWebTokenInNeed 刷新 webapi access token
func RefreshWebTokenInNeed(activeUser *config.PanUser, deviceName string) bool {
	if activeUser == nil {
		return false
	}

	// refresh expired web token
	if activeUser.PanClient().WebapiPanClient() != nil {
		if activeUser.WebapiToken != nil && len(activeUser.WebapiToken.AccessToken) > 0 {
			cz := time.FixedZone("CST", 8*3600) // 东8区
			expiredTime := time.Unix(activeUser.WebapiToken.Expired, 0).In(cz)
			now := time.Now()
			if (expiredTime.Unix() - now.Unix()) <= (2 * 60) { // 有效期小于2min就刷新
				pluginManger := plugins.NewPluginManager(config.GetPluginDir())
				plugin, _ := pluginManger.GetPlugin()
				params := &plugins.UserTokenRefreshFinishParams{
					Result:    "success",
					Message:   "webapi",
					OldToken:  "",
					NewToken:  "",
					UpdatedAt: utils.NowTimeStr(),
				}

				// need update refresh token
				logger.Verboseln("web access token expired, get new from server")
				loginHelper := panlogin.NewLoginHelper(config.DefaultTokenServiceWebHost)
				wt, e := loginHelper.GetWebapiNewToken(activeUser.TicketId, activeUser.UserId, activeUser.PanClient().WebapiPanClient().GetAccessToken())
				if e != nil {
					logger.Verboseln("get web token from server error: ", e)
				}
				if wt != nil {
					params.Result = "success"
					params.OldToken = activeUser.WebapiToken.AccessToken
					params.NewToken = wt.AccessToken

					// update for user & client
					userWebToken := NewWebLoginToken(wt.AccessToken, wt.Expired)
					activeUser.WebapiToken = &config.PanClientToken{
						AccessToken: wt.AccessToken,
						Expired:     wt.Expired,
					}
					activeUser.PanClient().WebapiPanClient().UpdateToken(userWebToken)
					logger.Verboseln("get new access token success")

					// plugin callback
					if er1 := plugin.UserTokenRefreshFinishCallback(plugins.GetContext(activeUser), params); er1 != nil {
						logger.Verbosef("UserTokenRefreshFinishCallback error: " + er1.Error())
					}

					// create new signature
					_, e := activeUser.PanClient().WebapiPanClient().CreateSession(nil)
					if e != nil {
						logger.Verboseln("call CreateSession error in RefreshWebTokenInNeed: " + e.Error())
					}
					return true
				} else {
					// token refresh error
					// if token has expired, callback plugin api for notify
					if now.Unix() >= expiredTime.Unix() {
						params.Result = "fail"
						params.Message = e.Error()
						params.OldToken = activeUser.WebapiToken.AccessToken
						if er1 := plugin.UserTokenRefreshFinishCallback(plugins.GetContext(activeUser), params); er1 != nil {
							logger.Verbosef("UserTokenRefreshFinishCallback error: " + er1.Error())
						}
					}
				}
			}
		}
	}
	return false
}

// RefreshOpenTokenInNeed 刷新 openapi access token
func RefreshOpenTokenInNeed(activeUser *config.PanUser) bool {
	if activeUser == nil {
		return false
	}

	// refresh expired openapi token
	if activeUser.PanClient().OpenapiPanClient() != nil {
		if len(activeUser.OpenapiToken.AccessToken) > 0 {
			cz := time.FixedZone("CST", 8*3600) // 东8区
			expiredTime := time.Unix(activeUser.OpenapiToken.Expired, 0).In(cz)
			now := time.Now()
			if (expiredTime.Unix() - now.Unix()) <= (2 * 60) { // 有效期小于2min就刷新
				pluginManger := plugins.NewPluginManager(config.GetPluginDir())
				plugin, _ := pluginManger.GetPlugin()
				params := &plugins.UserTokenRefreshFinishParams{
					Result:    "success",
					Message:   "openapi",
					OldToken:  "",
					NewToken:  "",
					UpdatedAt: utils.NowTimeStr(),
				}

				// need update refresh token
				logger.Verboseln("openapi access token expired, get new from server")
				loginHelper := panlogin.NewLoginHelper(config.DefaultTokenServiceWebHost)
				wt, e := loginHelper.GetOpenapiNewToken(activeUser.TicketId, activeUser.UserId, activeUser.PanClient().OpenapiPanClient().GetAccessToken())
				if e != nil {
					logger.Verboseln("get openapi token from server error: ", e)
				}
				if wt != nil {
					params.Result = "success"
					params.OldToken = activeUser.WebapiToken.AccessToken
					params.NewToken = wt.AccessToken

					// update for user
					activeUser.OpenapiToken = &config.PanClientToken{
						AccessToken: wt.AccessToken,
						Expired:     wt.Expired,
					}
					logger.Verboseln("get new access token success")

					// plugin callback
					if er1 := plugin.UserTokenRefreshFinishCallback(plugins.GetContext(activeUser), params); er1 != nil {
						logger.Verbosef("UserTokenRefreshFinishCallback error: " + er1.Error())
					}

					return true
				} else {
					// token refresh error
					// if token has expired, callback plugin api for notify
					if now.Unix() >= expiredTime.Unix() {
						params.Result = "fail"
						params.Message = e.Error()
						params.OldToken = activeUser.WebapiToken.AccessToken
						if er1 := plugin.UserTokenRefreshFinishCallback(plugins.GetContext(activeUser), params); er1 != nil {
							logger.Verbosef("UserTokenRefreshFinishCallback error: " + er1.Error())
						}
					}
				}
			}
		}
	}
	return false
}

func isIncludeFile(pattern string, fileName string) bool {
	b, er := filepath.Match(pattern, fileName)
	if er != nil {
		return false
	}
	return b
}

// isMatchWildcardPattern 是否是统配符字符串
func isMatchWildcardPattern(name string) bool {
	return strings.ContainsAny(name, "*") || strings.ContainsAny(name, "?") || strings.ContainsAny(name, "[")
}

// TryLogin 尝试登录，基础应用支持的各类登录方式进行尝试成功登录
func TryLogin() *config.PanUser {
	// can do automatically login?
	for _, u := range config.Config.UserList {
		if u.UserId == config.Config.ActiveUID {
			// login
			cloudUser, err := config.SetupUserByCookie(u.OpenapiToken, u.WebapiToken,
				u.TicketId, u.UserId,
				config.Config.DeviceId, config.Config.DeviceName,
				config.Config.ClientId, config.Config.ClientSecret)
			if cloudUser == nil {
				logger.Verboseln("尝试登录失败: ", err)
				fmt.Println("尝试登录失败，请使用 login 命令进行重新登录")
				return nil
			}

			// save
			SaveConfigFunc(nil)
			// reload
			ReloadConfigFunc(nil)

			return config.Config.ActiveUser()
		}
	}
	return nil
}
