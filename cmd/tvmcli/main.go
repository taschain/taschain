package main

import (
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

var (
	app = kingpin.New("chat", "A command-line chat application.")

	deployContract = app.Command("deploy", "deploy contract.")
	contractName   = deployContract.Arg("name", "").Required().String()
	contractPath   = deployContract.Arg("path", "").Required().String()

	callContract    = app.Command("call", "call contract.")
	contractAddress = callContract.Arg("contractAddress", "contract address.").Required().String()
	contractAbi     = callContract.Arg("abiPath", "").Required().String()

	exportAbi             = app.Command("export", "export abi.")
	exportAbiContractName = exportAbi.Arg("name", "").Required().String()
	exportAbiContractPath = exportAbi.Arg("path", "").Required().String()
)

func main() {

	tvmCli := NewTvmCli()

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {

	// deploy Token ./cli/erc20.py
	case deployContract.FullCommand():
		f, err := ioutil.ReadFile(filepath.Dir(os.Args[0]) + "/" + *contractPath) //读取文件
		if err != nil {
			fmt.Println("read the ", *contractPath, " file failed ", err)
			return
		}
		tvmCli.Deploy(*contractName, string(f))

	// call ./cli/call_Token_abi.json
	case callContract.FullCommand():
		f, err := ioutil.ReadFile(filepath.Dir(os.Args[0]) + "/" + *contractAbi) //读取文件
		if err != nil {
			fmt.Println("read the ", *contractAbi, " file failed ", err)
			return
		}
		tvmCli.Call(*contractAddress, string(f))

	// export ./cli/erc20.py
	case exportAbi.FullCommand():
		f, err := ioutil.ReadFile(filepath.Dir(os.Args[0]) + "/" + *exportAbiContractPath) //读取文件
		if err != nil {
			fmt.Println("read the ", *exportAbiContractPath, " file failed ", err)
			return
		}
		tvmCli.ExportAbi(*exportAbiContractName, string(f))
	}
}
