#!/bin/bash

#export GOPATH=$GOPATH:/Users/fenglei108/git/tas/taschain
export GOPATH=$GOPATH:/var/lib/jenkins/workspace/tas_develop/

main_dir=tool/main.go

go clean
go build  -o ./performance_tool $main_dir









