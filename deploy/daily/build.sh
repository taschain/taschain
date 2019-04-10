#!/bin/bash

export GOPATH=$GOPATH:/Users/pxf/workspace/tas_develop/tas
main_dir=../../src/gtas/main.go

rm -f ./gtas
go clean
go build  -o ./gtas $main_dir








