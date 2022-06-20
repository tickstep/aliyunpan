@echo off

REM 配置环境变量
REM set ALIYUNPAN_CONFIG_DIR=d:\path\to\your\aliyunpan\config

REM 指定refresh token用于登录
aliyunpan login -RefreshToken=9078907....adg9087

REM 上传下载链接类型：1-默认 2-阿里ECS环境
aliyunpan config set -transfer_url_type 1

REM 指定配置参数并进行启动
REM 支持的模式：upload(备份本地文件到云盘),download(备份云盘文件到本地),sync(双向同步备份)
aliyunpan sync start -ldir "D:\tickstep\Documents\设计文档" -pdir "/备份盘/我的文档" -mode "upload"