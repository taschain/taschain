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

package cli

import (
	"common"
	"core"
	"errors"
	"fmt"
	"gopkg.in/alecthomas/kingpin.v2"
	"network"
	"os"

	"core/net/handler"
	chandler "consensus/net"
	"consensus/mediator"

	"encoding/json"
	"consensus/groupsig"
	"taslog"
	"core/net/sync"
	_ "net/http/pprof"
	"net/http"
	"middleware"
	"time"
	"middleware/types"
	"runtime"
	"log"
	"strconv"
	"consensus/model"
	"redis"
)

const (
	// Section 默认section配置
	Section = "gtas"
	// RemoteHost 默认host
	RemoteHost = "127.0.0.1"
	// RemotePort 默认端口
	RemotePort = 8088

	instanceSection = "instance"

	indexKey = "index"

	chainSection = "chain"

	databaseKey = "database"

	statisticsSection = "statistics"

	nodetypeSection = "nodetype"

	redis_prefix = "aliyun_"
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

func (gtas *Gtas) waitingUtilSyncFinished() {
	log.Println("waiting for block and group sync finished....")
	for {
		if sync.BlockSyncer.IsInit() && sync.GroupSyncer.IsInit() {
			break
		}
		time.Sleep(time.Millisecond * 500)
	}
	log.Println("block and group sync finished!!")
}

// miner 起旷工节点
func (gtas *Gtas) miner(rpc, super, testMode bool, rpcAddr, seedIp string, rpcPort uint,light bool) {
	middleware.SetupStackTrap("/Users/daijia/stack.log") //todo: absolute path?
	err := gtas.fullInit(super, testMode, seedIp,light)
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

	gtas.waitingUtilSyncFinished()
	redis.NodeOnline(mediator.Proc.GetPubkeyInfo().ID.Serialize(), mediator.Proc.GetPubkeyInfo().PK.Serialize())
	ok := mediator.StartMiner()

	gtas.inited = true
	if !ok {
		return
	}
}

func (gtas *Gtas) exit(ctrlC <-chan bool, quit chan<- bool) {
	<-ctrlC
	fmt.Println("exiting...")
	core.BlockChainImpl.Close()
	taslog.Close()
	mediator.StopMiner()
	if gtas.inited {
		redis.NodeOffline(mediator.Proc.GetMinerID().Serialize())
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
	_ = app.Flag("metrics", "enable metrics").Bool()
	_ = app.Flag("dashboard", "enable metrics dashboard").Bool()
	pprofPort := app.Flag("pprof", "enable pprof").Default("8080").Uint()
	statisticsEnable := app.Flag("statistics", "enable statistics").Bool()
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
	instanceIndex := mineCmd.Flag("instance", "instance index").Short('i').Default("0").Int()
	//light node
	light := mineCmd.Flag("light", "light node").Bool()

	//在测试模式下 P2P的NAT关闭
	testMode := mineCmd.Flag("test", "test mode").Bool()
	seedIp := mineCmd.Flag("seed", "seed ip").String()

	prefix := mineCmd.Flag("prefix", "redis key prefix temp").String()
	nat := mineCmd.Flag("nat", "nat server address").String()

	clearCmd := app.Command("clear", "Clear the data of blockchain")

	command, err := app.Parse(os.Args[1:])
	if err != nil {
		kingpin.Fatalf("%s, try --help", err)
	}
	common.InstanceIndex = *instanceIndex
	go func() {
		http.ListenAndServe(fmt.Sprintf(":%d", *pprofPort), nil)
		runtime.SetBlockProfileRate(1)
		runtime.SetMutexProfileFraction(1)
	}()
	gtas.simpleInit(*configFile)

	common.GlobalConf.SetInt(instanceSection, indexKey, *instanceIndex)
	databaseValue := "d" + strconv.Itoa(*instanceIndex)
	common.GlobalConf.SetString(chainSection, databaseKey, databaseValue)
	common.GlobalConf.SetBool(statisticsSection, "enable", *statisticsEnable)
	if *prefix == "" {
		common.GlobalConf.SetString("test", "prefix", redis_prefix)
	} else {
		common.GlobalConf.SetString("test", "prefix", *prefix)
	}

	if *nat != "" {
		network.NatServerIp = *nat
		log.Printf("NAT server ip:%s", *nat)
	}
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
		gtas.miner(*rpc, *super, *testMode, addrRpc.String(), *seedIp, *portRpc,*light)
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

func (gtas *Gtas) fullInit(isSuper, testMode bool, seedIp string,light bool) error {
	var err error
	// 椭圆曲线初始化
	//groupsig.Init(1)

	// 初始化中间件
	middleware.InitMiddleware()

	// block初始化
	err = core.InitCore(light)
	if err != nil {
		return err
	}

	id, err := network.Init(*configManager, isSuper, handler.NewChainHandler(), chandler.MessageHandler, testMode, seedIp)
	if err != nil {
		return err
	}

	sync.InitGroupSyncer()
	sync.InitBlockSyncer()

	// TODO gov, ConsensusInit? StartMiner?
	//ok := global.InitGov(core.BlockChainImpl)
	//if !ok {
	//	return errors.NewAccountDB("gov module error")
	//}

	if isSuper {
		//超级节点启动前先把Redis数据清空
		redis.CleanRedisData()
	}

	secret := (*configManager).GetString(Section, "secret", "")
	if secret == "" {
		secret = getRandomString(5)
		(*configManager).SetString(Section, "secret", secret)
	}
	minerInfo := model.NewMinerInfo(id, secret)
	// 打印相关
	ShowPubKeyInfo(minerInfo, id)
	ok := mediator.ConsensusInit(minerInfo)
	if !ok {
		return errors.New("consensus module error")
	}

	mediator.Proc.BeginGenesisGroupMember()
	return nil
}

func LoadPubKeyInfo(key string) ([]model.PubKeyInfo) {
	var infos []PubKeyInfo
	keys := (*configManager).GetString(Section, key, "")
	err := json.Unmarshal([]byte(keys), &infos)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	var pubKeyInfos []model.PubKeyInfo
	for _, v := range infos {
		var pub = groupsig.Pubkey{}
		fmt.Println(v.PubKey)
		err := pub.SetHexString(v.PubKey)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		pubKeyInfos = append(pubKeyInfos, model.NewPubKeyInfo(*groupsig.NewIDFromString(v.ID), pub))
	}
	return pubKeyInfos
}

func ShowPubKeyInfo(info model.MinerInfo, id string) {
	pubKey := info.GetDefaultPubKey().GetHexString()
	fmt.Printf("Miner PubKey: %s;\n", pubKey)
	js, _ := json.Marshal(PubKeyInfo{pubKey, id})
	fmt.Printf("pubkey_info json: %s\n", js)
}

func NewGtas() *Gtas {
	return &Gtas{}
}

func genTestTx(hash string, price uint64, source string, target string, nonce uint64, value uint64) *types.Transaction {

	sourcebyte := common.BytesToAddress(common.Sha256([]byte(source)))
	targetbyte := common.BytesToAddress(common.Sha256([]byte(target)))

	//byte: 84,104,105,115,32,105,115,32,97,32,116,114,97,110,115,97,99,116,105,111,110
	data := []byte("This is a transaction")
	return &types.Transaction{
		Data:     data,
		Value:    value,
		Nonce:    nonce,
		Source:   &sourcebyte,
		Target:   &targetbyte,
		GasPrice: price,
		GasLimit: 3,
		Hash:     common.BytesToHash(common.Sha256([]byte(hash))),
	}
}
