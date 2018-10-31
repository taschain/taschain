#!/bin/bash

#此脚本用于打包页面静态文件，编译前执行
#执行前确保当前处于工程目录下

#需要先安装go-bindata，使用下面命令：
#go get -u github.com/jteeuwen/go-bindata/...

cd src
go-bindata -o asset/asset.go -pkg=asset gtas/fronted/...
