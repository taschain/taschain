package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestTvm(t *testing.T) {
	tvmCli := NewTvmCli()
	f, err := ioutil.ReadFile(filepath.Dir(os.Args[0]) + "/" + "erc20.py") //读取文件
	if err != nil {
		fmt.Println("read the erc20.py file failed ", err)
		t.Fail()
	}
	contractAddress := tvmCli.Deploy("Token", string(f))
	tvmCli.DeleteTvmCli()

	tvmCli = NewTvmCli()
	abiJson := `{
	"FuncName": "balance_of",
		"Args": ["0x6c63b15aac9b94927681f5fb1a7343888dece14e3160b3633baa9e0d540228cd"]
}`
	tvmCli.Call(contractAddress, abiJson)
	tvmCli.DeleteTvmCli()
}