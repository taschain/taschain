//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

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
