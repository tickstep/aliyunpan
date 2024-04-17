package syncdrive

import (
	"encoding/json"
	"fmt"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/log"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/library-go/logger"
	"io/ioutil"
	"path"
	"strings"
	"time"
)

type (
	// SyncOption 同步选项
	SyncOption struct {
		FileDownloadParallel  int   // 文件下载并发数
		FileUploadParallel    int   // 文件上传并发数
		FileDownloadBlockSize int64 // 文件下载分片大小
		FileUploadBlockSize   int64 // 文件上传分片大小

		MaxDownloadRate int64 // 限制最大下载速度
		MaxUploadRate   int64 // 限制最大上传速度

		// 优先级选项
		SyncPriority SyncPriorityOption

		// 本地文件修改检测间隔
		LocalFileModifiedCheckIntervalSec int

		// 文件记录器
		FileRecorder *log.FileRecorder
	}

	// SyncTaskManager 同步任务管理器
	SyncTaskManager struct {
		syncDriveConfig      *SyncDriveConfig
		syncOption           SyncOption
		PanUser              *config.PanUser
		PanClient            *config.PanClient
		SyncConfigFolderPath string

		// useConfigFile 是否使用配置文件启动
		useConfigFile bool
	}

	// SyncDriveConfig 同步盘配置文件
	SyncDriveConfig struct {
		ConfigVer    string      `json:"configVer"`
		SyncTaskList []*SyncTask `json:"syncTaskList"`
	}
)

var (
	ErrSyncTaskListEmpty error = fmt.Errorf("no sync task")
)

func NewSyncTaskManager(user *config.PanUser, panClient *config.PanClient, syncConfigFolderPath string,
	option SyncOption) *SyncTaskManager {
	return &SyncTaskManager{
		PanUser:              user,
		PanClient:            panClient,
		SyncConfigFolderPath: syncConfigFolderPath,
		syncOption:           option,
	}
}

func (m *SyncTaskManager) parseConfigFile() error {
	/** 样例
	{
	 "configVer": "1.0",
	 "syncTaskList": [
	  {
	   "name": "NS游戏备份",
	   "id": "5b2d7c10-e927-4e72-8f9d-5abb3bb04814",
	   "driveId": "19519111",
	   "localFolderPath": "D:\\smb\\datadisk\\game",
	   "panFolderPath": "/sync_drive/game",
	   "mode": "sync",
	   "driveName": "backup",
	   "lastSyncTime": ""
	  }
	 ]
	}
	*/
	configFilePath := m.ConfigFilePath()
	r := &SyncDriveConfig{
		ConfigVer:    "1.0",
		SyncTaskList: []*SyncTask{},
	}
	m.syncDriveConfig = r

	if b, _ := utils.PathExists(configFilePath); b != true {
		//text := utils.ObjectToJsonStr(r, true)
		//ioutil.WriteFile(ConfigFilePath, []byte(text), 0755)
		return fmt.Errorf("备份配置文件不存在：" + m.ConfigFilePath())
	}
	data, e := ioutil.ReadFile(configFilePath)
	if e != nil {
		return e
	}

	if len(data) > 0 {
		if err2 := json.Unmarshal(data, m.syncDriveConfig); err2 != nil {
			logger.Verboseln("parse sync drive config json error ", err2)
			return err2
		}
	}
	return nil
}

func (m *SyncTaskManager) ConfigFilePath() string {
	return path.Join(m.SyncConfigFolderPath, "sync_drive_config.json")
}

// Start 启动同步进程
func (m *SyncTaskManager) Start(tasks []*SyncTask) (bool, error) {
	if tasks != nil && len(tasks) > 0 {
		m.syncDriveConfig = &SyncDriveConfig{
			ConfigVer:    "1.0",
			SyncTaskList: []*SyncTask{},
		}
		m.syncDriveConfig.SyncTaskList = tasks
		m.useConfigFile = false
	} else {
		if er := m.parseConfigFile(); er != nil {
			return false, er
		}
		m.useConfigFile = true
	}
	if m.syncDriveConfig.SyncTaskList == nil || len(m.syncDriveConfig.SyncTaskList) == 0 {
		return false, ErrSyncTaskListEmpty
	}

	// start the sync task one by one
	for _, task := range m.syncDriveConfig.SyncTaskList {
		if len(task.Id) == 0 {
			task.Id = utils.UuidStr()
		}
		// check driveId
		if strings.ToLower(task.DriveName) == "backup" {
			task.DriveId = m.PanUser.DriveList.GetFileDriveId()
		} else if strings.ToLower(task.DriveName) == "resource" {
			task.DriveId = m.PanUser.DriveList.GetResourceDriveId()
		}
		if len(task.DriveId) == 0 {
			task.DriveId = m.PanUser.DriveList.GetFileDriveId()
		}

		// check userId
		if len(task.UserId) > 0 {
			if task.UserId != m.PanUser.UserId {
				// not this user task, skip
				logger.Verboseln("skip sync task: " + task.NameLabel())
				continue
			}
		} else {
			task.UserId = m.PanUser.UserId
		}

		// check pan path
		if !utils.IsPanAbsPath(task.PanFolderPath) {
			task.PanFolderPath = "/" + task.PanFolderPath
		}

		// check local path
		if !utils.IsLocalAbsPath(task.LocalFolderPath) {
			fmt.Println("任务启动失败，本地路径不是绝对路径: ", task.LocalFolderPath)
			continue
		}
		task.panUser = m.PanUser
		task.syncDbFolderPath = m.SyncConfigFolderPath
		task.panClient = m.PanClient
		task.syncOption = m.syncOption
		if task.Priority != "" {
			task.syncOption.SyncPriority = task.Priority
		} else {
			task.Priority = SyncPriorityTimestampFirst
			task.syncOption.SyncPriority = SyncPriorityTimestampFirst
		}
		if task.Policy == "" {
			task.Policy = SyncPolicyIncrement
		}
		task.LocalFolderPath = path.Clean(task.LocalFolderPath)
		task.PanFolderPath = path.Clean(task.PanFolderPath)
		if e := task.Start(); e != nil {
			logger.Verboseln(e)
			fmt.Printf("启动同步任务[%s]出错: %s\n", task.Id, e.Error())
			continue
		}
		fmt.Println("\n启动同步任务")
		fmt.Println(task)
		time.Sleep(200 * time.Millisecond)
	}
	// save config file
	if m.useConfigFile {
		ioutil.WriteFile(m.ConfigFilePath(), []byte(utils.ObjectToJsonStr(m.syncDriveConfig, true)), 0755)
	}
	return true, nil
}

// Stop 停止同步进程
func (m *SyncTaskManager) Stop() (bool, error) {
	// stop task one by one
	for _, task := range m.syncDriveConfig.SyncTaskList {
		var e error
		e = task.Stop()
		if e != nil {
			logger.Verboseln(e)
			fmt.Println("stop sync task error: ", task.NameLabel())
			continue
		}
		fmt.Println("正在停止同步任务: ", task.NameLabel())
	}

	// save config file
	if m.useConfigFile {
		ioutil.WriteFile(m.ConfigFilePath(), []byte(utils.ObjectToJsonStr(m.syncDriveConfig, true)), 0755)
	}
	return true, nil
}

// IsOneTime 判断是否是单次运行，此处判断只要是有一个任务是需要单次运行的，那么全部任务执行结束后退出程序
func (m *SyncTaskManager) IsOneTime() bool {
	for _, task := range m.syncDriveConfig.SyncTaskList {
		if task.CycleModeType == CycleOneTime {
			return true
		}
	}
	return false
}

// IsAllTaskDone 任务是否全部完成
func (m *SyncTaskManager) IsAllTaskDone() bool {
	var b = 0
	for _, v := range m.syncDriveConfig.SyncTaskList {
		if v.IsExecuteLoopIsDone() {
			b += 1
		}
	}
	if b == len(m.syncDriveConfig.SyncTaskList) {
		return true
	}
	return false
}
