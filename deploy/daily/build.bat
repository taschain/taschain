@echo off

rd /S /Q pid
rd /S /Q logs

for /d %%s in ( d* ) do (
    rd /S /Q %%s
)

del /F /Q gtas.exe

set GOPATH=%cd%\..\..\thirdparty;%cd%\..\..
go clean
go build -o ./gtas.exe ../../src/gtas/main.go
