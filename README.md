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

## apt安装
适用于apt包管理器的系统，例如Ubuntu，国产deepin深度操作系统等。目前只支持amd64和arm64架构的机器。
```shell
sudo curl -fsSL http://file.tickstep.com/apt/pgp | gpg --dearmor | sudo tee /etc/apt/trusted.gpg.d/tickstep-packages-archive-keyring.gpg > /dev/null && echo "deb [signed-by=/etc/apt/trusted.gpg.d/tickstep-packages-archive-keyring.gpg] http://file.tickstep.com/apt aliyunpan main" | sudo tee /etc/apt/sources.list.d/tickstep-aliyunpan.list > /dev/null && sudo apt-get update && sudo apt-get install -y aliyunpan
 
```

## rpm安装
适用于rpm包管理器的系统，例如CentOS、RockyLinux等。目前只支持amd64和arm64架构的机器。
```shell
sudo curl -fsSL http://file.tickstep.com/rpm/aliyunpan/aliyunpan.repo | sudo tee /etc/yum.repos.d/tickstep-aliyunpan.repo > /dev/null && sudo yum install aliyunpan -y
 
```

## docker安装
### sync同步盘
同步备份功能，支持备份本地文件到云盘，备份云盘文件到本地，双向同步备份三种模式。支持JavaScript插件对备份文件进行过滤。
备份功能支持以下三种模式：
1. 备份本地文件，即上传本地文件到网盘，始终保持本地文件有一个完整的备份在网盘
2. 备份云盘文件，即下载网盘文件到本地，始终保持网盘的文件有一个完整的备份在本地
3. 双向备份，保持网盘文件和本地文件严格一致
```
docker run -d --name=aliyunpan-sync --restart=always -v "<your local dir>:/home/app/data" -e TZ="Asia/Shanghai" -e ALIYUNPAN_REFRESH_TOKEN="<your refreshToken>" -e ALIYUNPAN_PAN_DIR="<your drive pan dir>" -e ALIYUNPAN_SYNC_MODE="upload" tickstep/aliyunpan-sync:<tag>
 
  
<tag>: 版本号，例如：v0.1.8
<your local dir>：本地目录绝对路径，例如：/tickstep/Documents/设计文档
ALIYUNPAN_PAN_DIR：云盘目录
ALIYUNPAN_REFRESH_TOKEN：RefreshToken
ALIYUNPAN_SYNC_MODE：备份模式，支持三种: upload(备份本地文件到云盘),download(备份云盘文件到本地),sync(双向同步备份)
```
更详情文档请参考dockerhub网址：[tickstep/aliyunpan-sync](https://hub.docker.com/r/tickstep/aliyunpan-sync)

### webdav共享盘
让阿里云盘变身为webdav协议的文件服务器。这样你可以把阿里云盘挂载为Windows、Linux、Mac系统的磁盘，可以通过NAS系统做文件管理或文件同步等等。
```
docker run -d --name=aliyunpan-webdav --restart=always -p 23077:23077 -e TZ="Asia/Shanghai" -e ALIYUNPAN_REFRESH_TOKEN="<your refreshToken>" -e ALIYUNPAN_AUTH_USER="admin" -e ALIYUNPAN_AUTH_PASSWORD="admin" -e ALIYUNPAN_PAN_DRIVE="File" -e ALIYUNPAN_PAN_DIR="/" tickstep/aliyunpan-webdav:<tag>
 
 
<tag>: 版本号，例如：v0.1.8
ALIYUNPAN_REFRESH_TOKEN RefreshToken
ALIYUNPAN_AUTH_USER webdav登录用户名
ALIYUNPAN_AUTH_PASSWORD webdav登录密码
ALIYUNPAN_PAN_DRIVE 网盘类型，可选： File-文件 Album-相册
ALIYUNPAN_PAN_DIR 网盘文件夹的webdav服务根目录
```
更详情文档请参考dockerhub网址：[tickstep/aliyunpan-webdav](https://hub.docker.com/r/tickstep/aliyunpan-webdav)

# 如何使用
完整和详细的命令说明请查看手册：[命令手册](docs/manual.md)   

如果程序运行时输出乱码, 请检查下终端的编码方式是否为 `UTF-8`.   
使用本程序之前, 非常建议先学习一些 linux 基础命令知识.   
如果没有带任何参数运行程序, 程序将会进入仿Linux shell系统用户界面的cli交互模式, 可直接运行相关命令.   
cli交互模式下, 光标所在行的前缀应为 `aliyunpan >`, 如果登录了帐号则格式为 `aliyunpan:<工作目录> <用户昵称>$ `   
程序会提供相关命令的使用说明.   

1. Windows
程序应在 命令提示符 (Command Prompt) 或 PowerShell 中运行.   
也可直接双击程序运行, 具体使用方法请参见 [命令列表及说明](docs/manual.md#命令列表及说明)   

2. Linux / macOS
程序应在 终端 (Terminal) 运行.   
具体使用方法请参见 [命令列表及说明](docs/manual.md#命令列表及说明)   


## 基本使用
本程序支持阿里云盘大多数命令操作，这里只介绍基本的使用，更多更详细的命令请查看手册：[命令手册](docs/manual.md)。

### 启动程序
直接启动进入交互命令行
```shell
[tickstep@MacPro ~]$ aliyunpan
提示: 方向键上下可切换历史命令.
提示: Ctrl + A / E 跳转命令 首 / 尾.
提示: 输入 help 获取帮助.
aliyunpan > 
```

### 查看帮助
```shell
aliyunpan > help

...
   阿里云盘:
     album, abm   相簿(Beta)
     cd           切换工作目录
     download, d  下载文件/目录
     ls, l, ll    列出目录
     mkdir        创建目录
     mv           移动文件/目录
     pwd          输出工作目录
     recycle      回收站
     rename       重命名文件
     rm           删除文件/目录
     share        分享文件/目录
     sync         同步备份功能
     upload, u    上传文件/目录
     webdav       在线网盘服务
...
```

### 登录
需要先登录，已经登录过的可以跳过此步。   
RefreshToken获取教程请查看：[如何获取RefreshToken](#如何获取RefreshToken)
```shell
aliyunpan > login -RefreshToken=32994cd2c43...4d505fa79

阿里云盘登录成功:  tickstep
aliyunpan:/ tickstep$ 
```

### 查看文件列表
```shell
aliyunpan:/ tickstep$ ls

  #  文件大小       修改日期               文件(目录)                             
  0         -  2021-11-03 13:32:22  临时/                     
  1         -  2021-07-10 07:44:34  好友的分享/               
  2         -  2021-07-09 22:11:22  我的项目/                 
  3         -  2021-07-09 22:10:37  我的游戏/       
  4         -  2021-07-09 22:10:10  我的文档/       
  5  349.86KB  2021-06-06 11:46:02  使用统计.xls                                            
  6  503.57KB  2021-06-06 11:46:02  IMG_0098.JPG                                            
  7   72.20KB  2021-06-06 11:46:02  IMG_0103.PNG 
       总: 3.20MB                   文件总数: 3, 目录总数: 7  
----
```

### 下载文件
```shell
aliyunpan:/ tickstep$ download IMG_0106.JPG

[0] 当前文件下载最大并发量为: 5, 下载缓存为: 64.00KB
[1] 加入下载队列: /IMG_0106.JPG
[1] ----
文件ID: 60bc44f855814e19692a4958b4a8823a1a06e5de
文件名: IMG_0106.JPG
文件类型: 文件
文件路径: /IMG_0106.JPG

[1] 准备下载: /IMG_0106.JPG
[1] 将会下载到路径: /root/Downloads/4d001d48564f43b3bc5662874f04bbe6/IMG_0106.JPG
[1] 下载开始
[1] ↓ 704.00KB/1.48MB 0B/s(1.69MB/s) in 1.88s, left - ............
[1] 下载完成, 保存位置: /root/Downloads/4d001d48564f43b..62874f04bbe6/IMG_0106.JPG
[1] 检验文件有效性成功: /root/Downloads/4d001d48564f43b..62874f04bbe6/IMG_0106.JPG

下载结束, 时间: 4秒, 数据总量: 1.48MB
aliyunpan:/ 小妖粉粉$ 
```
下载支持两种链接类型：1-默认类型 2-阿里ECS环境类型   
在普通网络下，下载速度可以达到10MB/s，在阿里ECS（必须是"经典网络"类型的机器）环境下，下载速度单文件可以轻松达到20MB/s，多文件可以达到100MB/s   
![](./assets/images/download_file_ecs_speed_screenshot.gif)
![](./assets/images/download_file_speed_screenshot.gif)

### 上传文件
```shell
aliyunpan:/ tickstep$ upload /Users/tickstep/Downloads/apt.zip /tmp

[0] 当前文件上传最大并发量为: 10, 上传分片大小为: 10.00MB
[1] 加入上传队列: /Users/tickstep/Downloads/apt.zip
[1] 2022-08-13 13:41:22 准备上传: /Users/tickstep/Downloads/apt.zip => /tmp/apt.zip
[1] 2022-08-13 13:41:22 正在检测和创建云盘文件夹: /tmp
[1] 2022-08-13 13:41:22 正在计算文件SHA1: /Users/tickstep/Downloads/apt.zip
[1] 2022-08-13 13:41:22 检测秒传中, 请稍候...
[1] 2022-08-13 13:41:22 秒传失败，开始正常上传文件
[1] ↑ 21.00MB/21.00MB 702.53KB/s(702.70KB/s) in 15s ............
[1] 2022-08-13 13:41:22 上传文件成功, 保存到网盘路径: /tmp/apt.zip
[1] 2022-08-13 13:41:22 文件上传结果： 成功！ 耗时 18秒

上传结束, 时间: 18秒, 数据总量: 21.00MB
```
上传支持两种链接类型：1-默认类型 2-阿里ECS环境类型   
在阿里ECS（必须是"经典网络"类型的机器）环境下，上传速度单文件可以轻松达到30MB/s，多文件可以达到100MB/s   
![](./assets/images/upload_file_speed_screenshot.gif)

## 如何获取RefreshToken
需要通过浏览器获取refresh_token。这里以Chrome浏览器为例，其他浏览器类似。   
打开 [阿里云盘网页](https://www.aliyundrive.com/drive) 并进行登录，然后F12按键打开浏览器调试菜单，按照下面步骤进行
![](./assets/images/how-to-get-refresh-token.png)

或者直接在控制台输入以下命令获取
```
JSON.parse(localStorage.getItem("token")).refresh_token
```
![](./assets/images/how-to-get-refresh-token-cmd.png)

# 交流反馈
提交issue: [issues页面](https://github.com/tickstep/aliyunpan/issues)   
联系邮箱: tickstep@outlook.com

# 鸣谢
本项目大量借鉴了以下相关项目的功能&成果   
> [tickstep/cloudpan189-go](https://github.com/tickstep/cloudpan189-go)    
> [hacdias/webdav](https://github.com/hacdias/webdav)   