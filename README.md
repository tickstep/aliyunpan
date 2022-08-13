# 关于
阿里云盘CLI。仿 Linux shell 文件处理命令的阿里云盘命令行客户端，支持webdav文件协议，支持同步备份功能。

# 特色
1. 多平台支持, 支持 Windows, macOS, linux(x86/x64/arm), android, iOS 等
2. 阿里云盘多用户支持
3. 支持文件网盘，相册网盘无缝切换
4. [下载](docs/manual.md#下载文件目录)网盘内文件, 支持多个文件或目录下载, 支持断点续传和单文件并行下载。支持软链接(符号链接)文件。
5. [上传](docs/manual.md#上传文件目录)本地文件, 支持多个文件或目录上传，支持排除指定文件夹/文件（正则表达式）功能。支持软链接(符号链接)文件。
6. [同步备份功能](docs/manual.md#同步备份功能)支持备份本地文件到云盘，备份云盘文件到本地，双向同步备份保持本地文件和网盘文件同步。常用于嵌入式或者NAS等设备，支持docker镜像部署。
7. 命令和文件路径输入支持Tab键自动补全
8. 支持阿里云ECS环境下使用内网链接上传/下载，速度更快(只支持阿里经典网络，最高可达100MB/s)，还可以节省公网带宽流量(配置transfer_url_type=2即可)
9. 支持[webdav文件服务](docs/manual.md#webdav文件服务)，可以将阿里云盘当做webdav文件网盘挂载到Windows, macOS, linux的磁盘中进行使用。webdav部署支持docker镜像，镜像只有不到10MB非常小巧。
10. 支持[JavaScript插件](docs/manual.md#JavaScript插件)，你可以按照自己的需要定制上传/下载中关键步骤的行为，最大程度满足自己的个性化需求

# 相关说明
1. 本项目还处于开发阶段，未经过充分的测试，如有bug或者好的建议欢迎提交issue
2. 由于阿里云盘还在内测中，后面功能和接口随时会被修改，相对应的，本工具相关功能也会被影响
3. 目前阶段优先处理Bug，功能增强的开发会相对往后安排

# 版本标签说明
1. arm / armv5 / armv7 : 适用32位ARM系统
2. arm64 : 适用64位ARM系统
3. 386 / x86 : 适用32系统，包括Intel和AMD的CPU系统
4. amd64 / x64 : 适用64位系统，包括Intel和AMD的CPU系统
5. mips : 适用MIPS指令集的CPU，例如国产龙芯CPU
6. macOS amd64适用Intel CPU的机器，macOS arm64目前主要是适用苹果M1芯片的机器
7. iOS arm64适用iPhone手机，并且必须是越狱的手机才能正常运行

# 如何安装
## 直接下载
可以直接在本仓库 [发布页](https://github.com/tickstep/aliyunpan/releases) 下载，解压后使用。

# 如何使用
如果程序运行时输出乱码, 请检查下终端的编码方式是否为 `UTF-8`.   
使用本程序之前, 非常建议先学习一些 linux 基础命令知识.   
如果没有带任何参数运行程序, 程序将会进入仿Linux shell系统用户界面的cli交互模式, 可直接运行相关命令.   
cli交互模式下, 光标所在行的前缀应为 `aliyunpan >`, 如果登录了帐号则格式为 `aliyunpan:<工作目录> <用户昵称>$ `   
程序会提供相关命令的使用说明.   

## Windows
程序应在 命令提示符 (Command Prompt) 或 PowerShell 中运行.   
也可直接双击程序运行, 具体使用方法请参见 [命令列表及说明](docs/manual.md#命令列表及说明).   

## Linux / macOS
程序应在 终端 (Terminal) 运行.   
具体使用方法请参见 [命令列表及说明](docs/manual.md#命令列表及说明) .   

# 交流反馈
提交issue: [issues页面](https://github.com/tickstep/aliyunpan/issues)   
联系邮箱: tickstep@outlook.com

# 鸣谢
本项目大量借鉴了以下相关项目的功能&成果   
> [tickstep/cloudpan189-go](https://github.com/tickstep/cloudpan189-go)    
> [hacdias/webdav](https://github.com/hacdias/webdav)   