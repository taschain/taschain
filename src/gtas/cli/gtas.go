package cli

import (
	"common"
	"consensus/groupsig"
	"core"
	"errors"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"network"
	"os"

	"network/p2p"
	"core/net/handler"
	chandler "consensus/net/handler"
	"consensus/mediator"
	"consensus/logical"
	"governance/global"
)

const (
	// Section 默认section配置
	Section = "gtas"
	// RemoteHost 默认host
	RemoteHost = "127.0.0.1"
	// RemotePort 默认端口
	RemotePort = 8088
)

var configManager = &common.GlobalConf
var walletManager wallets

type Gtas struct {
}

func (gtas *Gtas) vote(from, modelNum string, configVote VoteConfigKvs)  {
	if from == "" {
		// 本地钱包同时无钱包地址
		if len(walletManager) == 0 {
			fmt.Println("Please new account or assign a account")
			return
		} else {
			from = walletManager[0].Address
		}
	}
	config, err := configVote.ToVoteConfig()
	if err != nil {
		fmt.Println("translate vote config error: ", err)
		return
	}
	if err != nil {
		fmt.Println("serialize config error: ", err)
		return
	}
	msg, err := getMessage(RemoteHost, RemotePort, "GTAS_vote", from, modelNum, config)
	if err != nil {
		fmt.Println("rpc get message error: ", err)
		return
	}
	fmt.Println(msg)
}

func (gtas *Gtas) miner(rpc bool, rpcAddr string, rpcPort uint) {
	err := gtas.fullInit()
	if err != nil {
		fmt.Println(err)
		return
	}
	if rpc {
		err = StartRPC(rpcAddr, rpcPort)
		if err != nil {
			fmt.Println(err)
			return
		}
	}




	// 截获ctrl+c中断信号，退出
	quit := signals()
	<-quit
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
	modelNumVote := voteCmd.Flag("modelnum", "model number").Default("").String()
	fromVote := voteCmd.Flag("from", "the wallet address who polled").Default("").String()
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
	addrRpc := mineCmd.Flag("rpcaddr", "rpc host").Short('r').Default("127.0.0.1").IP()
	portRpc := mineCmd.Flag("rpcport", "rpc port").Short('p').Default("8088").Uint()

	clearCmd := app.Command("clear", "Clear the data of blockchain")

	command, err := app.Parse(os.Args[1:])
	if err != nil {
		kingpin.Fatalf("%s, try --help", err)
	}
	gtas.simpleInit(*configFile)
	switch command {
	case voteCmd.FullCommand():
		gtas.vote(*fromVote, *modelNumVote, *configVote)
	case tCmd.FullCommand():
		msg, err := getMessage(RemoteHost, RemotePort, "GTAS_t", *fromT, *toT, *valueT, *codeT)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println(msg)
	case balanceCmd.FullCommand():
		msg, err := getMessage(RemoteHost, RemotePort, "GTAS_getBalance", *accountBalance)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(msg)
		}
	case newCmd.FullCommand():
		privKey, address := walletManager.newWallet()
		fmt.Println("Please Remember Your PrivateKey!")
		fmt.Printf("PrivateKey: %s\n WalletAddress: %s", privKey, address)
	case mineCmd.FullCommand():
		gtas.miner(*rpc, addrRpc.String(), *portRpc)
	case clearCmd.FullCommand():
		err := ClearBlock()
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("clear blockchain successfully")
		}
	}

}

// ClearBlock 删除本地的chainblock数据。
func ClearBlock() error {
	err := core.InitCore()
	if err != nil {
		return err
	}
	return core.BlockChainImpl.Clear()
}

func (gtas *Gtas) simpleInit(configPath string) {
	common.InitConf(configPath)
	walletManager = newWallets()
}

func (gtas *Gtas) fullInit() error {
	var err error
	groupsig.Init(1)
	err = core.InitCore()
	if err != nil {
		return errors.New("InitBlockChain failed")
	}
	//TODO 初始化日志， network初始化
	p2p.SetChainHandler(new(handler.ChainHandler))
	p2p.SetConsensusHandler(new(chandler.ConsensusHandler))

	err = network.InitNetwork(configManager)
	if err != nil {
		return err
	}

	// TODO gov, ConsensusInit? StartMiner?
	ok := global.InitGov(core.BlockChainImpl)
	if !ok {
		return errors.New("")
	}

	id := p2p.Server.SelfNetInfo.Id
	secret := getRandomString(5)
	(*configManager).SetString(Section, "secret", secret)
	mediator.ConsensusInit(logical.NewMinerInfo(id, secret))
	mediator.StartMiner()
	return nil
}

func NewGtas() *Gtas {
	return &Gtas{}
}


func mockTxs() []*core.Transaction {
	//source byte: 138,170,12,235,193,42,59,204,152,26,146,154,213,207,129,10,9,14,17,174
	//target byte: 93,174,34,35,176,3,97,163,150,23,122,156,180,16,255,97,242,0,21,173
	//hash : 112,155,85,189,61,160,245,168,56,18,91,208,238,32,197,191,221,124,171,161,115,145,45,66,129,202,232,22,183,154,32,27
	t1 := genTestTx("tx1", 123, "111", "abc", 0, 1)
	t2 := genTestTx("tx1", 456, "222", "ddd", 0, 1)
	s := []*core.Transaction{t1, t2}
	return s
}

func genTestTx(hash string, price uint64, source string, target string, nonce uint64, value uint64) *core.Transaction {

	sourcebyte := common.BytesToAddress(core.Sha256([]byte(source)))
	targetbyte := common.BytesToAddress(core.Sha256([]byte(target)))

	//byte: 84,104,105,115,32,105,115,32,97,32,116,114,97,110,115,97,99,116,105,111,110
	data := []byte("This is a transaction")
	return &core.Transaction{
		Data:     data,
		Value:    value,
		Nonce:    nonce,
		Source:   &sourcebyte,
		Target:   &targetbyte,
		GasPrice: price,
		GasLimit: 3,
		Hash:     common.BytesToHash(core.Sha256([]byte(hash))),
	}
}
