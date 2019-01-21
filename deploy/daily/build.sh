#!/bin/bash

main_dir=../../src/gtas/main.go

rm -f ./gtas
go clean
go build  -o ./gtas $main_dir








