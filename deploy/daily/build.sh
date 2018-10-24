#!/bin/bash

main_dir=/Users/daijia/code/tas/src/gtas/main.go

rm ./gtas
go build  -o ./gtas $main_dir
rm -rf ./d*
rm -rf ./logs
rm -rf pid_tas*

#如果需要更换新的结点进行启动，使用如下代码将对应的配置文件删除
#for v in {4..9}
#do
#  rm -rf 'tas'$v'.ini'
#  rm -rf 'joined_group.config.'$v
#done






