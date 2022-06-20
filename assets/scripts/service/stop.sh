#!/bin/sh

PROCESS=`ps -ef|grep aliyunpan|grep -v grep|grep -v PPID|awk '{ print $2}'`
for i in $PROCESS         
do
  # kill进程
  kill -9 $i
  break
done
