#!/bin/sh
# 检查aliyunpan进程是否存在
ps | awk '{print $4}' | grep aliyunpan
if [ $? == 0 ]
then
    echo $?
    exit 0
else
    echo $?
    exit 1
fi


