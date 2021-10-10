# 关于 Windows EXE ICO 和应用信息编译
为了编译出来的windows的exe文件带有ico和应用程序信息，需要使用 github.com/josephspurrier/goversioninfo/cmd/goversioninfo 工具

工具安装，运行下面的命令即可生成工具。也可以直接用 bin/ 文件夹下面的编译好的
```
go get github.com/josephspurrier/goversioninfo/cmd/goversioninfo
```

versioninfo.json - 里面有exe程序信息以及ico的配置
使用 goversioninfo 工具运行以下命令
```
    goversioninfo -o=resource_windows_386.syso
    goversioninfo -64 -o=resource_windows_amd64.syso
```
即可编译出.syso资源库，再使用 go build 编译之后，exe文件就会拥有应用程序信息和ico图标