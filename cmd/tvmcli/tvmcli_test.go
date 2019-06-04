package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"testing"
)

func TestTvm(t *testing.T) {
	tvmCli := NewTvmCli()
	f, err := os.Open("erc20.py") //读取文件
	if err != nil {
		t.Fail()
	}
	defer f.Close()
	codeStr := ""
	buf := bufio.NewReader(f)
	for {
		line, err := buf.ReadString('\n')
		codeStr = fmt.Sprintf("%s%s \n", codeStr, line)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				t.Fail()
				return
			}
		}
	}
	contractAddress := tvmCli.Deploy("Token", codeStr)
	tvmCli.DeleteTvmCli()

	tvmCli = NewTvmCli()
	abiJson := `{
	"FuncName": "balance_of",
		"Args": ["0x6c63b15aac9b94927681f5fb1a7343888dece14e3160b3633baa9e0d540228cd"]
}`
	tvmCli.Call(contractAddress, abiJson)
	tvmCli.DeleteTvmCli()
}
