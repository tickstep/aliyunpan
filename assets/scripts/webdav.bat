@echo off

REM 配置环境变量
REM set ALIYUNPAN_CONFIG_DIR=d:\path\to\your\aliyunpan\config

REM 指定refresh token用于登录
aliyunpan login -RefreshToken=9078907....adg9087

REM 上传下载链接类型：1-默认 2-阿里ECS环境
aliyunpan config set -transfer_url_type 1

REM 指定webdav启动参数并进行启动。你可以按照自己的需求更改以下参数，例如用户名和密码
aliyunpan webdav start -ip "0.0.0.0" -port 23077 -webdav_user "admin" -webdav_password "admin" -pan_dir_path "/" -bs 1024