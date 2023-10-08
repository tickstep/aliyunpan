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
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/library-go/converter"
	"github.com/urfave/cli"
)

type QuotaInfo struct {
	// 已使用个人空间大小
	UsedSize int64
	// 个人空间总大小
	Quota int64
}

func CmdQuota() cli.Command {
	return cli.Command{
		Name:        "quota",
		Usage:       "获取当前帐号空间配额",
		Description: "获取网盘的总储存空间, 和已使用的储存空间",
		Category:    "阿里云盘账号",
		Before:      ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if config.Config.ActiveUser() == nil {
				fmt.Println("未登录账号")
				return nil
			}
			q, err := RunGetQuotaInfo()
			if err == nil {
				fmt.Printf("账号: %s, uid: %s, 个人空间总额: %s, 个人空间已使用: %s, 比率: %.2f%%\n",
					config.Config.ActiveUser().Nickname, config.Config.ActiveUser().UserId,
					converter.ConvertFileSize(q.Quota, 2), converter.ConvertFileSize(q.UsedSize, 2),
					100*float64(q.UsedSize)/float64(q.Quota))
			}
			return nil
		},
	}
}

func RunGetQuotaInfo() (quotaInfo *QuotaInfo, error error) {
	user, err := GetActivePanClient().GetUserInfo()
	if err != nil {
		return nil, err
	}
	return &QuotaInfo{
		UsedSize: int64(user.UsedSize),
		Quota:    int64(user.TotalSize),
	}, nil
}
