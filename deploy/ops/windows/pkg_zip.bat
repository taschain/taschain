set version=0.5.2
set GOPATH=%cd%\..\..\..\..\taschain_thirdparty;%cd%\..\..\..
del gtas_windows_v%version%.zip
cd gtas_windows
go clean
go build -o ./gtas.exe ../../../../src/gtas/main.go
"C:\Program Files\7-Zip\7z.exe" a -tzip ..\gtas_windows_v%version%.zip *
del gtas.exe
cd ..
