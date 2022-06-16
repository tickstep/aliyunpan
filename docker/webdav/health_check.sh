#!/bin/sh

# 检查webdav的服务端口是否存在
netstat -luntp | awk '{print $4}' | grep 23077
if [ $? == 0 ]
then
    echo $?
    exit 0
else
    echo $?
    exit 1
fi


