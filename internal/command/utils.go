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
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan/cmder/cmdliner"
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
	files, err := acUser.PanClient().MatchPathByShellPattern(driveId, GetActiveUser().PathJoin(driveId, pattern))
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
		ps, err := acUser.PanClient().MatchPathByShellPattern(driveId, acUser.PathJoin(driveId, patterns[k]))
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

// RefreshTokenInNeed 刷新refresh token
func RefreshTokenInNeed(activeUser *config.PanUser) bool {
	if activeUser == nil {
		return false
	}

	// refresh expired token
	if activeUser.PanClient() != nil {
		if len(activeUser.WebToken.RefreshToken) > 0 {
			cz := time.FixedZone("CST", 8*3600) // 东8区
			expiredTime, _ := time.ParseInLocation("2006-01-02 15:04:05", activeUser.WebToken.ExpireTime, cz)
			now := time.Now()
			if (expiredTime.Unix() - now.Unix()) <= (20 * 60) { // 20min
				pluginManger := plugins.NewPluginManager(config.GetPluginDir())
				plugin, _ := pluginManger.GetPlugin()
				params := &plugins.UserTokenRefreshFinishParams{
					Result:    "success",
					Message:   "",
					OldToken:  "",
					NewToken:  "",
					UpdatedAt: utils.NowTimeStr(),
				}

				// need update refresh token
				logger.Verboseln("access token expired, get new from refresh token")
				if wt, er := aliyunpan.GetAccessTokenFromRefreshToken(activeUser.RefreshToken); er == nil {
					params.Result = "success"
					params.OldToken = activeUser.RefreshToken
					params.NewToken = wt.RefreshToken

					activeUser.RefreshToken = wt.RefreshToken
					activeUser.WebToken = *wt
					activeUser.PanClient().UpdateToken(*wt)
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
						params.Message = er.Error()
						params.OldToken = activeUser.RefreshToken
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

// DoLoginHelper 登录助手，使用token进行登录
func DoLoginHelper(refreshToken string) (refreshTokenStr string, webToken aliyunpan.WebLoginToken, error error) {
	line := cmdliner.NewLiner()
	defer line.Close()

	if refreshToken == "" {
		refreshToken, error = line.State.Prompt("请输入RefreshToken, 回车键提交 > ")
		if error != nil {
			return
		}
	}

	// app login
	atoken, apperr := aliyunpan.GetAccessTokenFromRefreshToken(refreshToken)
	if apperr != nil {
		if apperr.Code == apierror.ApiCodeTokenExpiredCode || apperr.Code == apierror.ApiCodeRefreshTokenExpiredCode {
			fmt.Println("Token过期，需要重新登录")
		} else {
			fmt.Println("Token登录失败：", apperr)
		}
		return "", webToken, fmt.Errorf("登录失败")
	}
	refreshTokenStr = refreshToken
	return refreshTokenStr, *atoken, nil
}

// TryLogin 尝试登录，基础应用支持的各类登录方式进行尝试成功登录
func TryLogin() *config.PanUser {
	// 获取当前插件
	pluginManger := plugins.NewPluginManager(config.GetPluginDir())
	plugin, _ := pluginManger.GetPlugin()
	params := &plugins.UserTokenRefreshFinishParams{
		Result:    "success",
		Message:   "",
		OldToken:  "",
		NewToken:  "",
		UpdatedAt: utils.NowTimeStr(),
	}

	// can do automatically login?
	for _, u := range config.Config.UserList {
		if u.UserId == config.Config.ActiveUID {
			// login
			_, webToken, err := DoLoginHelper(u.RefreshToken)
			if err != nil {
				logger.Verboseln("automatically login use saved refresh token error ", err)
				if u.TokenId != "" {
					logger.Verboseln("try to login use tokenId")
					h := panlogin.NewLoginHelper(config.DefaultTokenServiceWebHost)
					r, e := h.GetRefreshToken(u.TokenId)
					if e != nil {
						logger.Verboseln("try to login use tokenId error", e)
						// login fail plugin callback
						params.Result = "fail"
						params.OldToken = u.RefreshToken
						params.NewToken = ""
						if er := plugin.UserTokenRefreshFinishCallback(plugins.GetContext(u), params); er != nil {
							logger.Verbosef("UserTokenRefreshFinishCallback error: " + er.Error())
						}
						break
					}
					refreshToken, e := h.ParseSecureRefreshToken("", r.SecureRefreshToken)
					if e != nil {
						logger.Verboseln("try to parse refresh token error", e)
						break
					}
					_, webToken, err = DoLoginHelper(refreshToken)
					if err != nil {
						logger.Verboseln("try to use refresh token from tokenId error", e)
						break
					}
					fmt.Println("Token重新自动登录成功")
					// save new refresh token
					u.RefreshToken = refreshToken
				} else {
					// login fail plugin callback
					params.Result = "fail"
					params.OldToken = u.RefreshToken
					params.NewToken = ""
					if er := plugin.UserTokenRefreshFinishCallback(plugins.GetContext(u), params); er != nil {
						logger.Verbosef("UserTokenRefreshFinishCallback error: " + er.Error())
					}
				}
				break
			}
			// plugin param
			params.Result = "success"
			params.OldToken = u.RefreshToken
			params.NewToken = webToken.RefreshToken

			// success login, save new token and access token
			u.RefreshToken = webToken.RefreshToken
			u.WebToken = webToken

			// save
			SaveConfigFunc(nil)
			// reload
			ReloadConfigFunc(nil)

			// do plugin callback
			if er := plugin.UserTokenRefreshFinishCallback(plugins.GetContext(u), params); er != nil {
				logger.Verbosef("UserTokenRefreshFinishCallback error: " + er.Error())
			}

			return config.Config.ActiveUser()
		}
	}
	return nil
}
