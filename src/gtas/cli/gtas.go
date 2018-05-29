package cli

import (
	"common"
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

	"encoding/json"
	"consensus/groupsig"
	"taslog"
	"core/net/sync"
	"time"
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
	inited bool
}

// vote 投票操作
func (gtas *Gtas) vote(from, modelNum string, configVote VoteConfigKvs) {
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

// miner 起旷工节点
func (gtas *Gtas) miner(rpc, super bool, rpcAddr string, rpcPort uint) {
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
	if super {
		keys1 := LoadPubKeyInfo("pubkeys1")
		keys2 := LoadPubKeyInfo("pubkeys2")
		keys3 := LoadPubKeyInfo("pubkeys3")
		fmt.Println("Waiting node to connect...")
		for {
			if len(p2p.Server.GetConnInfo()) >= 20 {
				fmt.Println("Connection:")
				for _, c := range p2p.Server.GetConnInfo() {
					fmt.Println(c.Id)
				}
				//time.Sleep(time.Second * 10)
				break
			}
		}
		createGroup(keys3, "gtas3")
		time.Sleep(time.Second*30)
		createGroup(keys2, "gtas2")
		time.Sleep(time.Second*30)
		createGroup(keys1, "gtas1")

	}
	gtas.inited = true
	//测试SendTransactions
	//peer1Id := "QmPf7ArTTxDqd1znC9LF5r73YR85sbEU1t1SzTvt2fRry2"
	//txs := mockTxs()
	//core.SendTransactions(txs, peer1Id)

	//测试BroadcastTransactions
	//txs := mockTxs()
	//core.BroadcastTransactions(txs)

	//测试BroadcastTransactionRequest
	//time.Sleep(10*time.Second)
	//m := core.TransactionRequestMessage{SourceId:p2p.Server.SelfNetInfo.Id,RequestTime:time.Now()}
	//m.TransactionHashes = []common.Hash{common.BytesToHash(core.Sha256([]byte("tx1"))), common.BytesToHash(core.Sha256([]byte("tx3")))}
	//core.BroadcastTransactionRequest(m)

	//测试mock block
	//txpool := core.BlockChainImpl.GetTransactionPool()
	//// 交易1
	//txpool.Add(genTestTx("jdai1", 12345, "1", "2", 0, 1))
	//
	////交易2
	//txpool.Add(genTestTx("jdai2", 123456, "2", "3", 0, 1))
	//castor:=[]byte{1,2,3,4}
	//groupid:=[]byte{5,6,7,8}
	//
	//// 铸块1
	//block := core.BlockChainImpl.CastingBlock(1, 12, 0, castor, groupid)
	//if 0 != core.BlockChainImpl.AddBlockOnChain(block){
	//	fmt.Printf("fail to add block\n")
	//}else{
	//	fmt.Printf("now height: %d\n",core.BlockChainImpl.Height())
	//}
	//
	//fmt.Printf("local height: %d\n",core.BlockChainImpl.Height())

	// 截获ctrl+c中断信号，退出

	//txpool := core.BlockChainImpl.GetTransactionPool()
	//// 交易1
	//txpool.Add(genTestTx("jdai1", 12345, "1", "2", 0, 1))
	//
	////交易2
	//txpool.Add(genTestTx("jdai2", 123456, "2", "3", 0, 1))
	//castor := []byte{1, 2, 3, 4}
	//groupid := []byte{5, 6, 7, 8}
	//
	//// 铸块1
	//block := core.BlockChainImpl.CastingBlock(1, 12, 0, castor, groupid)
	//if 0 != core.BlockChainImpl.AddBlockOnChain(block) {
	//	fmt.Printf("fail to add block\n")
	//} else {
	//	fmt.Printf("now height: %d\n", core.BlockChainImpl.Height())
	//}
	//
	//block = core.BlockChainImpl.CastingBlockAfter(block.Header,2, 123, 0, castor, groupid)
	//if 0 != core.BlockChainImpl.AddBlockOnChain(block) {
	//	fmt.Printf("fail to add block\n")
	//} else {
	//	fmt.Printf("now height: %d\n", core.BlockChainImpl.Height())
	//}
	//
	//block = core.BlockChainImpl.CastingBlockAfter(block.Header,3, 124, 0, castor, groupid)
	//if 0 != core.BlockChainImpl.AddBlockOnChain(block) {
	//	fmt.Printf("fail to add block\n")
	//} else {
	//	fmt.Printf("now height: %d\n", core.BlockChainImpl.Height())
	//}
	//
	//fmt.Printf("local height: %d\n", core.BlockChainImpl.Height())

}

func createGroup(keys []logical.PubKeyInfo,name string) {
	zero := mediator.CreateGroup(keys, name)
	if zero != 0 {
		fmt.Printf("create %s group failed\n", name)
	}
	fmt.Printf("create %s group success\n", name)
}

func (gtas *Gtas) exit(ctrlC <-chan bool, quit chan<- bool) {
	<-ctrlC
	fmt.Println("exiting...")
	core.BlockChainImpl.Close()
	taslog.Close()
	if gtas.inited {
		fmt.Println("exit success")
		quit <- true
	} else {
		fmt.Println("exit before inited")
		os.Exit(0)
	}
}

func (gtas *Gtas) Run() {
	var err error
	// control+c 信号
	ctrlC := signals()
	quitChan := make(chan bool)
	go gtas.exit(ctrlC, quitChan)
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
	addrRpc := mineCmd.Flag("rpcaddr", "rpc host").Short('r').Default("0.0.0.0").IP()
	portRpc := mineCmd.Flag("rpcport", "rpc port").Short('p').Default("8088").Uint()
	super := mineCmd.Flag("super", "start super node").Bool()

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
		gtas.miner(*rpc, *super, addrRpc.String(), *portRpc)
	case clearCmd.FullCommand():
		err := ClearBlock()
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("clear blockchain successfully")
		}
	}
	<-quitChan
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
	// 椭圆曲线初始化
	groupsig.Init(1)
	// block初始化
	err = core.InitCore()
	if err != nil {
		return err
	}

	//TODO 初始化日志， network初始化
	p2p.SetChainHandler(new(handler.ChainHandler))
	p2p.SetConsensusHandler(new(chandler.ConsensusHandler))

	err = network.InitNetwork(configManager)
	if err != nil {
		return err
	}
	sync.InitBlockSyncer()
	sync.InitGroupSyncer()

	// TODO gov, ConsensusInit? StartMiner?
	ok := global.InitGov(core.BlockChainImpl)
	if !ok {
		return errors.New("gov module error")
	}

	id := p2p.Server.SelfNetInfo.Id
	secret := (*configManager).GetString(Section, "secret", "")
	if secret == "" {
		secret = getRandomString(5)
		(*configManager).SetString(Section, "secret", secret)
	}
	minerInfo := logical.NewMinerInfo(id, secret)
	// 打印相关
	ShowPubKeyInfo(minerInfo, id)
	ok = mediator.ConsensusInit(minerInfo)
	if !ok {
		return errors.New("consensus module error")
	}
	ok = mediator.StartMiner()
	if !ok {
		return errors.New("start miner error")
	}
	mediator.Proc.BeginGenesisGroupMember()
	return nil
}

func LoadPubKeyInfo(key string) ([]logical.PubKeyInfo) {
	infos := []PubKeyInfo{}
	keys := (*configManager).GetString(Section, key, "")
	err := json.Unmarshal([]byte(keys), &infos)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	pubKeyInfos := []logical.PubKeyInfo{}
	for _, v := range infos {
		var pub = groupsig.Pubkey{}
		fmt.Println(v.PubKey)
		err := pub.SetHexString(v.PubKey)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		pubKeyInfos = append(pubKeyInfos, logical.PubKeyInfo{*groupsig.NewIDFromString(v.ID), pub})
	}
	return pubKeyInfos
}

func ShowPubKeyInfo(info logical.MinerInfo, id string) {
	pubKey := info.GetDefaultPubKey().GetHexString()
	fmt.Printf("PubKey: %s;\nID: %s;\nIDHex:%s;\n", pubKey, id, groupsig.NewIDFromString(id).GetHexString())
	js, _ := json.Marshal(PubKeyInfo{pubKey, id})
	fmt.Printf("pubkey_info json: %s", js)
}

func NewGtas() *Gtas {
	return &Gtas{}
}

func mockTxs() []*core.Transaction {
	//source byte: 138,170,12,235,193,42,59,204,152,26,146,154,213,207,129,10,9,14,17,174
	//target byte: 93,174,34,35,176,3,97,163,150,23,122,156,180,16,255,97,242,0,21,173
	//hash : 112,155,85,189,61,160,245,168,56,18,91,208,238,32,197,191,221,124,171,161,115,145,45,66,129,202,232,22,183,154,32,27
	t1 := genTestTx("tx1", 123, "111", "abc", 0, 1)
	t2 := genTestTx("tx2", 456, "222", "ddd", 0, 1)
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

