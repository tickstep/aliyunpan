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
	jsoniter "github.com/json-iterator/go"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan/cmder/cmdutil"
	"github.com/tickstep/aliyunpan/cmder/cmdutil/jsonhelper"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/aliyunpan/library/homedir"
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const (
	// EnvVerbose 启用调试环境变量
	EnvVerbose = "ALIYUNPAN_VERBOSE"
	// EnvConfigDir 配置路径环境变量
	EnvConfigDir = "ALIYUNPAN_CONFIG_DIR"
	// ConfigName 配置文件名
	ConfigName = "aliyunpan_config.json"
	// ConfigVersion 配置文件版本
	ConfigVersion string = "1.0"

	// DefaultFileUploadParallelNum 默认的文件上传并发数量
	DefaultFileUploadParallelNum = 10

	// MaxFileUploadParallelNum 最大文件上传并发数量。过大会被阿里云盘风控，导致无法上传
	MaxFileUploadParallelNum = 20

	// DefaultFileDownloadParallelNum 默认的文件下载并发数量
	DefaultFileDownloadParallelNum = 5

	// MaxFileDownloadParallelNum 最大文件下载并发数量。过大会被阿里云盘风控，导致无法下载
	MaxFileDownloadParallelNum = 20

	// DefaultTokenServiceWebHost 默认的token服务
	DefaultTokenServiceWebHost = "https://api.tickstep.com"
	//DefaultTokenServiceWebHost = "http://localhost:8977"

	// DefaultVideoFileExtensions 默认的视频文件后缀
	DefaultVideoFileExtensions = "mp4,flv,mkv,mov,rm,rmvb,wmv,wma,mv,asf,asx,mpg,mpeg,mpe,3gp,m4v,avi,vob"

	// DefaultDeviceName 默认客户端名称
	DefaultDeviceName = "Chrome浏览器"

	// DefaultClientId 默认的clientId
	DefaultClientId = "cf9f70e8fc61430f8ec5ab5cadf31375"
)

var (
	CmdConfigVerbose = logger.New("CONFIG", EnvVerbose)
	configFilePath   = filepath.Join(GetConfigDir(), ConfigName)

	// Config 配置信息, 由外部调用
	Config = NewConfig(configFilePath)
)

type UpdateCheckInfo struct {
	PreferUpdateSrv string `json:"preferUpdateSrv"` // 优先更新服务器，github | tickstep
	LatestVer       string `json:"latestVer"`       // 最后检测到的版本
	CheckTime       int64  `json:"checkTime"`       // 最后检测的时间戳，单位为秒
}

// PanConfig 配置详情
type PanConfig struct {
	ConfigVer string `json:"configVer"`
	ActiveUID string `json:"activeUID"`

	UserList PanUserList `json:"userList"`

	CacheSize           int `json:"cacheSize"`           // 下载缓存
	MaxDownloadParallel int `json:"maxDownloadParallel"` // 最大下载并发量，即同时下载文件最大数量
	MaxUploadParallel   int `json:"maxUploadParallel"`   // 最大上传并发量，即同时上传文件最大数量

	MaxDownloadRate int64 `json:"maxDownloadRate"` // 限制最大下载速度，单位 B/s, 即字节/每秒
	MaxUploadRate   int64 `json:"maxUploadRate"`   // 限制最大上传速度，单位 B/s, 即字节/每秒
	TransferUrlType int   `json:"transferUrlType"` // 上传/下载URL类别，1-默认，2-阿里云ECS

	SaveDir string `json:"saveDir"` // 下载储存路径

	Proxy           string          `json:"proxy"`        // 代理
	LocalAddrs      string          `json:"localAddrs"`   // 本地网卡地址
	PreferIPType    string          `json:"preferIPType"` // 优先IP类型，IPv4或者IPv6
	UpdateCheckInfo UpdateCheckInfo `json:"updateCheckInfo"`

	VideoFileExtensions string `json:"videoFileExtensions"`
	FileRecordConfig    string `json:"fileRecordConfig"` // 上传、下载、同步文件的记录，包括失败和成功的

	DeviceId   string `json:"deviceId"`   // 客户端ID，用于标识登录客户端，阿里单个账号最多允许10个客户端同时登录
	DeviceName string `json:"deviceName"` // 客户端名称，默认为：Chrome浏览器

	// Openapi客户端信息
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`

	configFilePath string
	configFile     *os.File
	fileMu         sync.Mutex
	activeUser     *PanUser
}

// NewConfig 返回 PanConfig 指针对象
func NewConfig(configFilePath string) *PanConfig {
	c := &PanConfig{
		configFilePath: configFilePath,
	}
	return c
}

// Init 初始化配置
func (c *PanConfig) Init() error {
	return c.init()
}

// Reload 从文件重载配置
func (c *PanConfig) Reload() error {
	return c.init()
}

// Close 关闭配置文件
func (c *PanConfig) Close() error {
	if c.configFile != nil {
		err := c.configFile.Close()
		c.configFile = nil
		return err
	}
	return nil
}

// Save 保存配置信息到配置文件
func (c *PanConfig) Save() error {
	// 检测配置项是否合法, 不合法则自动修复
	c.fix()

	err := c.lazyOpenConfigFile()
	if err != nil {
		return err
	}

	c.fileMu.Lock()
	defer c.fileMu.Unlock()

	data, err := jsoniter.MarshalIndent(c, "", " ")
	if err != nil {
		// json数据生成失败
		panic(err)
	}

	// 减掉多余的部分
	err = c.configFile.Truncate(int64(len(data)))
	if err != nil {
		return err
	}

	_, err = c.configFile.Seek(0, os.SEEK_SET)
	if err != nil {
		return err
	}

	_, err = c.configFile.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func (c *PanConfig) init() error {
	if c.configFilePath == "" {
		return ErrConfigFileNotExist
	}

	c.initDefaultConfig()
	err := c.loadConfigFromFile()
	if err != nil {
		return err
	}

	// 设置全局代理
	if c.Proxy != "" {
		requester.SetGlobalProxy(c.Proxy)
	}
	// 设置本地网卡地址
	if c.LocalAddrs != "" {
		requester.SetLocalTCPAddrList(strings.Split(c.LocalAddrs, ",")...)
	}

	// 设置域名解析策略 IPv4 or IPv6
	t := requester.IPAny
	if strings.ToLower(c.PreferIPType) == "ipv4" {
		t = requester.IPv4
	} else if strings.ToLower(c.PreferIPType) == "ipv6" {
		t = requester.IPv6
	}
	requester.SetPreferIPType(t)

	return nil
}

// lazyOpenConfigFile 打开配置文件
func (c *PanConfig) lazyOpenConfigFile() (err error) {
	if c.configFile != nil {
		return nil
	}

	c.fileMu.Lock()
	os.MkdirAll(filepath.Dir(c.configFilePath), 0755)
	c.configFile, err = os.OpenFile(c.configFilePath, os.O_CREATE|os.O_RDWR, 0755)
	c.fileMu.Unlock()

	if err != nil {
		if os.IsPermission(err) {
			return ErrConfigFileNoPermission
		}
		if os.IsExist(err) {
			return ErrConfigFileNotExist
		}
		return err
	}
	return nil
}

// loadConfigFromFile 载入配置
func (c *PanConfig) loadConfigFromFile() (err error) {
	err = c.lazyOpenConfigFile()
	if err != nil {
		return err
	}

	// 未初始化
	info, err := c.configFile.Stat()
	if err != nil {
		return err
	}

	if info.Size() == 0 {
		err = c.Save()
		return err
	}

	c.fileMu.Lock()
	defer c.fileMu.Unlock()

	_, err = c.configFile.Seek(0, os.SEEK_SET)
	if err != nil {
		return err
	}

	err = jsonhelper.UnmarshalData(c.configFile, c)
	if err != nil {
		return ErrConfigContentsParseError
	}
	if c.DeviceName == "" {
		c.DeviceName = DefaultDeviceName
	}
	if c.ClientId == "" {
		c.ClientId = DefaultClientId
	}
	return nil
}

func (c *PanConfig) initDefaultConfig() {
	// 设置默认的下载路径
	switch runtime.GOOS {
	case "windows":
		c.SaveDir = cmdutil.ExecutablePathJoin("Downloads")
	case "android":
		// TODO: 获取完整的的下载路径
		c.SaveDir = "/sdcard/Download"
	default:
		dataPath, ok := os.LookupEnv("HOME")
		if !ok {
			CmdConfigVerbose.Warn("Environment HOME not set")
			c.SaveDir = cmdutil.ExecutablePathJoin("Downloads")
		} else {
			c.SaveDir = filepath.Join(dataPath, "Downloads")
		}
	}
	c.ConfigVer = ConfigVersion
	c.VideoFileExtensions = DefaultVideoFileExtensions
	c.DeviceId = RandomDeviceId() // 生成默认客户端ID
	c.DeviceName = DefaultDeviceName
	c.ClientId = DefaultClientId
	c.FileRecordConfig = "1" // 默认开启
	c.PreferIPType = "ipv4"  // 默认优先IPv4
}

// GetConfigDir 获取配置路径
func GetConfigDir() string {
	// 按照以下顺序依次获取配置目录
	// 1.环境变量ALIYUNPAN_CONFIG_DIR => 2. /etc/aliyunpan/ => 3. ~/.aliyunpan/ => 4.当前程序目录

	// 1. 从环境变量读取
	configDir, ok := os.LookupEnv(EnvConfigDir)
	if ok {
		if filepath.IsAbs(configDir) {
			logger.Verboseln("use config dir from ALIYUNPAN_CONFIG_DIR env: ", configDir)
			return configDir
		}
		// 如果不是绝对路径, 从程序目录寻找
		configDir = cmdutil.ExecutablePathJoin(configDir)
		logger.Verboseln("use config dir from ALIYUNPAN_CONFIG_DIR env: ", configDir)
		return configDir
	} else {
		// 2. /etc/aliyunpan/
		if runtime.GOOS == "linux" {
			cd := "/etc/aliyunpan"
			if IsFolderExist(cd) {
				logger.Verboseln("use config dir: ", cd)
				return cd
			}
		}

		// 3. ~/.aliyunpan/
		if runtime.GOOS == "linux" || runtime.GOOS == "windows" {
			cd, er := homedir.Expand("~/.aliyunpan")
			if er == nil {
				if IsFolderExist(cd) {
					logger.Verboseln("use config dir: ", cd)
					return cd
				}
			}
		}
	}

	// 4.当前程序所在目录
	configDir = cmdutil.ExecutablePathJoin("")
	logger.Verboseln("use config dir: ", configDir)
	return configDir
}

// GetPluginDir 获取插件文件夹路径
func GetPluginDir() string {
	return strings.TrimSuffix(GetConfigDir(), "/") + "/plugin"
}

// GetPluginKvFile 获取插件KV键值对存储文件
func GetPluginKvFile() string {
	return GetPluginDir() + "/kv.bolt"
}

// GetSyncDriveDir 获取同步备份的文件夹路径
func GetSyncDriveDir() string {
	return strings.TrimSuffix(GetConfigDir(), "/") + "/sync_drive"
}

// GetLogDir 获取日志文件目录路径
func GetLogDir() string {
	return strings.TrimSuffix(GetConfigDir(), "/") + "/logs"
}

// GetLogFilePath 获取日志文件路径
func GetLogFilePath() string {
	dirPath := GetLogDir()
	if b, e := utils.PathExists(dirPath); e == nil {
		if !b {
			os.MkdirAll(dirPath, 0755)
		}
	}
	return dirPath + "/" + "aliyunpan_verbose.log"
}

// GetLockerDir 获取文件锁路径
func GetLockerDir() string {
	return strings.TrimSuffix(GetConfigDir(), "/")
}

func (c *PanConfig) ActiveUser() *PanUser {
	if c.activeUser == nil {
		if c.UserList == nil {
			return nil
		}
		if c.ActiveUID == "" {
			return nil
		}
		for _, u := range c.UserList {
			if u.UserId == c.ActiveUID {
				if u.PanClient() == nil {
					// restore client
					user, err := SetupUserByCookie(u.OpenapiToken, u.WebapiToken,
						u.TicketId, u.UserId,
						c.DeviceId, c.DeviceName,
						c.ClientId, c.ClientSecret)
					if err != nil {
						logger.Verboseln("setup user error")
						return nil
					}
					u.panClient = user.panClient
					u.Nickname = user.Nickname

					u.DriveList = user.DriveList
					// check workdir valid or not
					if user.IsFileDriveActive() {
						fe, err1 := u.PanClient().OpenapiPanClient().FileInfoByPath(u.ActiveDriveId, u.Workdir)
						if err1 != nil {
							// default to root
							u.Workdir = "/"
							u.WorkdirFileEntity = *aliyunpan.NewFileEntityForRootDir()
						} else {
							u.WorkdirFileEntity = *fe
						}
					} else if user.IsResourceDriveActive() {
						fe, err1 := u.PanClient().OpenapiPanClient().FileInfoByPath(u.ActiveDriveId, u.ResourceWorkdir)
						if err1 != nil {
							// default to root
							u.ResourceWorkdir = "/"
							u.ResourceWorkdirFileEntity = *aliyunpan.NewFileEntityForRootDir()
						} else {
							u.ResourceWorkdirFileEntity = *fe
						}
					} else if user.IsAlbumDriveActive() {
						fe, err1 := u.PanClient().WebapiPanClient().FileInfoByPath(u.ActiveDriveId, u.AlbumWorkdir)
						if err1 != nil {
							// default to root
							u.AlbumWorkdir = "/"
							u.AlbumWorkdirFileEntity = *aliyunpan.NewFileEntityForRootDir()
						} else {
							u.AlbumWorkdirFileEntity = *fe
						}
					}
				}
				c.activeUser = u
				return u
			}
		}
		return &PanUser{}
	}
	return c.activeUser
}

func (c *PanConfig) SetActiveUser(user *PanUser) *PanUser {
	needToInsert := true
	for _, u := range c.UserList {
		if u.UserId == user.UserId {
			// update user info
			u.Nickname = user.Nickname
			u.WebapiToken = user.WebapiToken
			u.OpenapiToken = user.OpenapiToken
			u.TicketId = user.TicketId
			u.PanClient().UpdateClient(user.PanClient().OpenapiPanClient(), user.PanClient().WebapiPanClient())
			needToInsert = false
			break
		}
	}
	if needToInsert {
		// insert
		c.UserList = append(c.UserList, user)
	}

	// setup active user
	c.ActiveUID = user.UserId
	// clear active user cache
	c.activeUser = nil
	// reload
	return c.ActiveUser()
}

func (c *PanConfig) fix() {

}

// NumLogins 获取登录的用户数量
func (c *PanConfig) NumLogins() int {
	return len(c.UserList)
}

// SwitchUser 切换登录用户
func (c *PanConfig) SwitchUser(uid, username string) (*PanUser, error) {
	for _, u := range c.UserList {
		if u.UserId == uid || u.AccountName == username {
			return c.SetActiveUser(u), nil
		}
	}
	return nil, fmt.Errorf("未找到指定的账号")
}

// DeleteUser 删除用户，并自动切换登录用户为用户列表第一个
func (c *PanConfig) DeleteUser(uid string) (*PanUser, error) {
	for idx, u := range c.UserList {
		if u.UserId == uid {
			// delete user from user list
			c.UserList = append(c.UserList[:idx], c.UserList[idx+1:]...)
			c.ActiveUID = ""
			c.activeUser = nil
			if len(c.UserList) > 0 {
				c.SwitchUser(c.UserList[0].UserId, "")
			}
			return u, nil
		}
	}
	return nil, fmt.Errorf("未找到指定的账号")
}

// HTTPClient 返回设置好的 HTTPClient
func (c *PanConfig) HTTPClient(ua string) *requester.HTTPClient {
	client := requester.NewHTTPClient()
	if ua != "" {
		client.SetUserAgent(ua)
	}
	return client
}

func (c *PanConfig) GetVideoExtensionList() []string {
	if c.VideoFileExtensions == "" {
		return []string{}
	}
	exts := strings.Split(c.VideoFileExtensions, ",")
	r := []string{}
	for _, ex := range exts {
		ex = strings.TrimSpace(ex)
		if ex != "" {
			r = append(r, strings.ToLower(ex))
		}
	}
	return r
}
