#!/bin/sh

# 请更改成你自己电脑上aliyunpan执行文件所在的目录
#cd /path/to/aliyunpan/folder

# 配置环境变量
#export ALIYUNPAN_CONFIG_DIR=/path/to/your/aliyunpan/config

chmod +x ./aliyunpan

# 指定refresh token用于登录
./aliyunpan login -RefreshToken=9078907....adg9087

# 上传下载链接类型：1-默认 2-阿里ECS环境
./aliyunpan config set -transfer_url_type 1

# 指定webdav启动参数并进行启动。你可以按照自己的需求更改以下参数，例如用户名、密码和网盘目录
./aliyunpan webdav start -ip "0.0.0.0" -port 23077 -webdav_user "admin" -webdav_password "admin" -pan_dir_path "/" -bs 1024


