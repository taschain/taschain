package cli

import (
	"common"
	"core"
	"errors"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"os"
)

const (
	// Section 默认section配置
	Section    = "gtas"
	// RemoteHost 默认host
	RemoteHost = "127.0.0.1"
	// RemotePort 默认端口
	RemotePort = 8088
)

var configManager = &common.GlobalConf
var walletManager wallets
var blockChain *core.BlockChain

type Gtas struct {
}

func (gtas *Gtas) Run() {
	var err error
	app := kingpin.New("GTAS", "A blockchain application.")
	app.HelpFlag.Short('h')
	// TODO config file的默认位置以及相关问题
	configFile := app.Flag("config", "Config file").Default("tas.ini").String()
	//remoteAddr := app.Flag("remoteaddr", "rpc host").Short('r').Default("127.0.0.1").IP()
	//remotePort := app.Flag("remoteport", "rpc port").Short('p').Default("8080").Uint()

	// 投票解析
	voteCmd := app.Command("vote", "new vote")
	//modelNum =
	// from
	configVote := VoteConfigParams(voteCmd.Arg("config", voteConfigHelp()))

	// 交易解析
	tCmd := app.Command("t", "create a transaction")
	fromT := tCmd.Flag("from", "from acc").Short('f').String()
	toT := tCmd.Flag("to", "to acc").Short('t').String()
	valueT := tCmd.Flag("value", "value").Short('v').Uint64()
	codeT := tCmd.Flag("code", "code").Short('c').String()

	// 查询余额解析
	balanceCmd := app.Command("balance", "get the balance of account")
	accountBalance := balanceCmd.Flag("account", "account address").String()

	// 新建账户解析
	newCmd := app.Command("newaccount", "new account")

	// mine
	mineCmd := app.Command("miner", "miner start")
	// rpc解析
	rpc := mineCmd.Flag("rpc", "start rpc server").Bool()
	rpcAddr := mineCmd.Flag("rpcaddr", "rpc host").Short('r').Default("127.0.0.1").IP()
	rpcPort := mineCmd.Flag("rpcport", "rpc port").Short('p').Default("8088").Uint()

	clearCmd := app.Command("clear", "Clear the data of blockchain")

	command, err := app.Parse(os.Args[1:])
	if err != nil {
		kingpin.Fatalf("%s, try --help", err)
	}
	gtas.simpleInit(*configFile)
	switch command {
	case voteCmd.FullCommand():
		vConfig, _ := configVote.ToVoteConfig()
		vBytes, err := vConfig.AbiEncode()
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("%v", vBytes)
		}
		return
	case tCmd.FullCommand():
		msg, err := getMessage(RemoteHost, RemotePort, "GTAS_t", *fromT, *toT, *valueT, *codeT)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(msg)
		return
	case balanceCmd.FullCommand():
		msg, err := getMessage(RemoteHost, RemotePort, "GTAS_getBalance", *accountBalance)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(msg)
		}
		return
	case newCmd.FullCommand():
		privKey, address := walletManager.newWallet()
		fmt.Println("Please Remember Your PrivateKey!")
		fmt.Printf("PrivateKey: %s\n WalletAddress: %s", privKey, address)
		return
	case mineCmd.FullCommand():
		err = gtas.fullInit()
		if err != nil {
			fmt.Println(err)
			return
		}
		if *rpc {
			err = StartRPC(rpcAddr.String(), *rpcPort)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	case clearCmd.FullCommand():
		err := ClearBlock()
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("clear blockchain successfully")
		}
		return
	}

	// 截获ctrl+c中断信号，退出
	quit := signals()
	<-quit

}

// ClearBlock 删除本地的chainblock数据。
func ClearBlock() error {
	if blockChain == nil {
		blockChain = core.InitBlockChain()
	}
	return blockChain.Clear()
}

func (gtas *Gtas) simpleInit(configPath string) {
	common.InitConf(configPath)
	walletManager = newWallets()
}

func (gtas *Gtas) fullInit() error {
	blockChain = core.InitBlockChain()
	if blockChain == nil {
		return errors.New("InitBlockChain failed")
	}
	// TODO 初始化日志， network初始化
	//err := network.InitNetwork(configManager)
	//if err != nil {
	//	return err
	//}

	return nil
}

func NewGtas() *Gtas {
	return &Gtas{}
}
