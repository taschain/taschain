set version=0.5.1
set GOPATH=%cd%\..\..\thirdparty;%cd%\..\..
cd gtas_windows
go clean
go build -o ./gtas.exe ../../../src/gtas/main.go
"C:\Program Files\7-Zip\7z.exe" a -tzip ..\gtas_windows_v%version%.zip *
del gtas.exe
cd ..
