set version=0.8.0
set GOPATH=%cd%\..\..\..\..\taschain_thirdparty;%cd%\..\..\..
del gtas_windows_v%version%.zip
cd gtas_windows
go clean
go build -o ./gtas.exe ../../../../src/gtas/main.go
md  gtas_windows\\py
copy ..\..\..\..\src\tvm\py\time.py .\gtas_windows\py\
copy ..\..\..\..\src\tvm\py\coin.py .\gtas_windows\py\
copy ..\..\..\..\src\tvm\py\tns.py .\gtas_windows\py\
"C:\Program Files\7-Zip\7z.exe" a -tzip ..\gtas_windows_v%version%.zip *
del gtas.exe
cd ..
