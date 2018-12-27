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

	chandler "consensus/net"
	"consensus/mediator"

	"encoding/json"
	"consensus/groupsig"
	"taslog"
	_ "net/http/pprof"
	"net/http"
	"middleware"
	"time"
	"middleware/types"
	"runtime"
	"strconv"
	"consensus/model"
	"runtime/debug"
	"core/net/handler"
	"github.com/vmihailenco/msgpack"
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

	redis_prefix = "aliyun_"
)

var walletManager wallets
var lightMiner bool

type Gtas struct {
	inited bool
	account Account
}

// vote 投票操作
func (gtas *Gtas) vote(from, modelNum string, configVote VoteConfigKvs) {
	if from == "" {
		// 本地钱包同时无钱包地址
		if len(walletManager) == 0 {
			common.DefaultLogger.Infof("Please new account or assign a account")
			return
		} else {
			from = walletManager[0].Address
		}
	}
	config, err := configVote.ToVoteConfig()
	if err != nil {
		common.DefaultLogger.Errorf("translate vote config error:%v", err)
		return
	}
	if err != nil {
		common.DefaultLogger.Errorf("serialize config error:%v", err)
		return
	}
	msg, err := getMessage(RemoteHost, RemotePort, "GTAS_vote", from, modelNum, config)
	if err != nil {
		common.DefaultLogger.Errorf("rpc get message error:%v", err)
		return
	}
	common.DefaultLogger.Infof(msg)
}

// miner 起旷工节点
func (gtas *Gtas) miner(rpc, super, testMode bool, rpcAddr, seedIp string, rpcPort uint, light bool, apply string) {
	gtas.runtimeInit()
	err := gtas.fullInit(super, testMode, seedIp, light)
	if err != nil {
		common.DefaultLogger.Error(err.Error())
		return
	}
	if rpc {
		err = StartRPC(rpcAddr, rpcPort)
		if err != nil {
			common.DefaultLogger.Infof(err.Error())
			return
		}
	}

	ok := mediator.StartMiner()

	core.InitGroupSyncer()
	//common.DefaultLogger.Infof("Waiting for group init done!")
	//for {
	//	if core.GroupSyncer.IsInit() {
	//		break
	//	}
	//	time.Sleep(time.Millisecond * 500)
	//}
	//common.DefaultLogger.Infof("Group first init done!\nStart to init block!")
	core.InitBlockSyncer()
	if len(apply) > 0 {
		go func() {
			timer := time.NewTimer(time.Second * 10)
			for {
				<-timer.C
				if core.BlockSyncer.IsInit() {
					break
				} else {
					timer.Reset(time.Second * 5)
				}
			}
			switch apply {
			case "heavy":
				gtas.autoApplyMiner(types.MinerTypeHeavy)
				//result, _ := GtasAPIImpl.MinerApply(500, types.MinerTypeHeavy)
				//common.DefaultLogger.Infof("initial apply heavy result:%v", result)
			case "light":
				gtas.autoApplyMiner(types.MinerTypeLight)
				//result, _ := GtasAPIImpl.MinerApply(500, types.MinerTypeLight)
				//common.DefaultLogger.Infof("initial apply light result:%v", result)
			}
		}()
	}
	gtas.inited = true
	if !ok {
		return
	}
}

func (gtas *Gtas) runtimeInit() {
	debug.SetGCPercent(100)
	debug.SetMaxStack(2 * 1000000000)
	common.DefaultLogger.Infof("setting gc 100%, max memory 2g")

}

func (gtas *Gtas) exit(ctrlC <-chan bool, quit chan<- bool) {
	<-ctrlC
	if core.BlockChainImpl == nil {
		return
	}
	fmt.Println("exiting...")
	core.BlockChainImpl.Close()
	taslog.Close()
	mediator.StopMiner()
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
	_ = app.Flag("metrics", "enable metrics").Bool()
	_ = app.Flag("dashboard", "enable metrics dashboard").Bool()
	pprofPort := app.Flag("pprof", "enable pprof").Default("8080").Uint()
	statisticsEnable := app.Flag("statistics", "enable statistics").Bool()
	keystore := app.Flag("keystore", "the keystore path, default is current path").Default("keystore").Short('k').String()
	*statisticsEnable = false
	//remoteAddr := app.Flag("remoteaddr", "rpc host").Short('r').Default("127.0.0.1").IP()
	//remotePort := app.Flag("remoteport", "rpc port").Short('p').Default("8080").Uint()

	// 投票解析
	//voteCmd := app.Command("vote", "new vote")
	//modelNumVote := voteCmd.Flag("modelnum", "model number").Default("").String()
	//fromVote := voteCmd.Flag("from", "the wallet address who polled").Default("").String()
	//configVote := VoteConfigParams(voteCmd.Arg("config", voteConfigHelp()))

	//控制台
	consoleCmd := app.Command("console", "start gtas console")
	showRequest := consoleCmd.Flag("show", "show the request json").Short('v').Bool()
	remoteHost := consoleCmd.Flag("host", "the node host address to connect").Short('i').String()
	remotePort := consoleCmd.Flag("port", "the node host port to connect").Short('p').Int()


	// 交易解析
	//tCmd := app.Command("t", "create a transaction")
	//fromT := tCmd.Flag("from", "from acc").Short('f').String()
	//toT := tCmd.Flag("to", "to acc").Short('t').String()
	//valueT := tCmd.Flag("value", "value").Short('v').Uint64()
	//codeT := tCmd.Flag("code", "code").Short('c').String()

	// 查询余额解析
	//balanceCmd := app.Command("balance", "get the balance of account")
	//accountBalance := balanceCmd.Flag("account", "account address").String()

	// 新建账户解析
	//newCmd := app.Command("newaccount", "new account")

	// mine
	mineCmd := app.Command("miner", "miner start")
	// rpc解析
	rpc := mineCmd.Flag("rpc", "start rpc server").Bool()
	addrRpc := mineCmd.Flag("rpcaddr", "rpc host").Short('r').Default("0.0.0.0").IP()
	portRpc := mineCmd.Flag("rpcport", "rpc port").Short('p').Default("8088").Uint()
	super := mineCmd.Flag("super", "start super node").Bool()
	instanceIndex := mineCmd.Flag("instance", "instance index").Short('i').Default("0").Int()
	apply := mineCmd.Flag("apply", "apply heavy or light miner").String()
	//address := mineCmd.Flag("addr", "start miner with address ").String()
	//light 废弃
	light := mineCmd.Flag("light", "light node").Bool()

	//在测试模式下 P2P的NAT关闭
	testMode := mineCmd.Flag("test", "test mode").Bool()
	seedIp := mineCmd.Flag("seed", "seed ip").String()
	nat := mineCmd.Flag("nat", "nat server address").String()

	clearCmd := app.Command("clear", "Clear the data of blockchain")

	command, err := app.Parse(os.Args[1:])
	if err != nil {
		kingpin.Fatalf("%s, try --help", err)
	}

	//初始化账号操作接口
	if err := InitAccountManager(*keystore, command == mineCmd.FullCommand()); err != nil {
		fmt.Println(err)
		return
	}

	switch command {
	//case voteCmd.FullCommand():
	//	gtas.vote(*fromVote, *modelNumVote, *configVote)
	case consoleCmd.FullCommand():
		err := ConsoleInit(*remoteHost, *remotePort, *showRequest)
		if err != nil {
			fmt.Errorf(err.Error())
		}

	//case tCmd.FullCommand():
	//	msg, err := getMessage(RemoteHost, RemotePort, "GTAS_tx", *fromT, *toT, *valueT, *codeT)
	//	if err != nil {
	//		common.DefaultLogger.Error(err.Error())
	//	}
	//	common.DefaultLogger.Info(msg)
	//case balanceCmd.FullCommand():
	//	msg, err := getMessage(RemoteHost, RemotePort, "GTAS_getBalance", *accountBalance)
	//	if err != nil {
	//		common.DefaultLogger.Error(err.Error())
	//	} else {
	//		common.DefaultLogger.Info(msg)
	//	}
	//case newCmd.FullCommand():
	//	privKey, address := walletManager.newWallet()
	//	common.DefaultLogger.Info("Please Remember Your PrivateKey!")
	//	common.DefaultLogger.Infof("PrivateKey: %s\n WalletAddress: %s", privKey, address)
	case mineCmd.FullCommand():
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
		common.DefaultLogger = taslog.GetLoggerByIndex(taslog.DefaultConfig, common.GlobalConf.GetString("instance", "index", ""))
		BonusLogger = taslog.GetLoggerByIndex(taslog.BonusStatConfig, common.GlobalConf.GetString("instance", "index", ""))
		types.InitMiddleware()
		if *nat != "" {
			network.NatServerIp = *nat
			common.DefaultLogger.Infof("NAT server ip:%s", *nat)
		}
		lightMiner = *light
		//轻重节点一样
		gtas.miner(*rpc, *super, *testMode, addrRpc.String(), *seedIp, *portRpc, *light, *apply)
	case clearCmd.FullCommand():
		err := ClearBlock(*light)
		if err != nil {
			common.DefaultLogger.Error(err.Error())
		} else {
			common.DefaultLogger.Infof("clear blockchain successfully")
		}
	}
	<-quitChan
}

// ClearBlock 删除本地的chainblock数据。
func ClearBlock(light bool) error {
	err := core.InitCore(light, mediator.NewConsensusHelper(groupsig.ID{}))
	if err != nil {
		return err
	}
	return core.BlockChainImpl.Clear()
}

func (gtas *Gtas) simpleInit(configPath string) {
	common.InitConf(configPath)
	walletManager = newWallets()
}

func (gtas *Gtas) checkAddress(address string) (error) {
	acm := AccountOp.(*AccountManager)
	if address != "" {
		if aci, err := acm.getAccountInfo(address); err != nil {
			return fmt.Errorf("cannot get miner, err:%v", err.Error())
		} else {
			if aci.Miner == nil {
				return fmt.Errorf("the address is not a miner account: %v", address)
			}
			gtas.account = aci.Account
			return nil
		}
	} else {
		aci := acm.getFirstMinerAccount()
		if aci != nil {
			gtas.account = *aci
			return nil
		} else {
			return fmt.Errorf("please create a miner account first")
		}
	}
}

func (gtas *Gtas) fullInit(isSuper, testMode bool, seedIp string, light bool) error {
	var err error

	// 椭圆曲线初始化
	//groupsig.Init(1)
	// 初始化中间件
	middleware.InitMiddleware()
	// block初始化
	//secret := (*configManager).GetString(Section, "secret", "")
	//if secret == "" {
	//	secret = getRandomString(5)
	//	(*configManager).SetString(Section, "secret", secret)
	//}
	addressConfig := common.GlobalConf.GetString(Section, "miner", "")
	err = gtas.checkAddress(addressConfig)
	if err != nil {
		return err
	}

	common.GlobalConf.SetString(Section, "miner", gtas.account.Address)
	fmt.Println("address:", gtas.account.Address)
	fmt.Println("sk:", gtas.account.Sk)

	minerInfo := model.NewSelfMinerDO(common.HexToAddress(gtas.account.Address))

	err = core.InitCore(light, mediator.NewConsensusHelper(minerInfo.ID))
	if err != nil {
		return err
	}
	id := minerInfo.ID.GetHexString()
	err = network.Init(common.GlobalConf, isSuper, handler.NewChainHandler(), chandler.MessageHandler, testMode, seedIp, id)
	if err != nil {
		return err
	}

	// TODO gov, ConsensusInit? StartMiner?
	//ok := global.InitGov(core.BlockChainImpl)
	//if !ok {
	//	return errors.NewAccountDB("gov module error")
	//}

	if isSuper {
		//超级节点启动前先把Redis数据清空
		//redis.CleanRedisData()
	}

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
	keys := common.GlobalConf.GetString(Section, key, "")
	err := json.Unmarshal([]byte(keys), &infos)
	if err != nil {
		common.DefaultLogger.Infof(err.Error())
		return nil
	}
	var pubKeyInfos []model.PubKeyInfo
	for _, v := range infos {
		var pub = groupsig.Pubkey{}
		common.DefaultLogger.Info(v.PubKey)
		err := pub.SetHexString(v.PubKey)
		if err != nil {
			common.DefaultLogger.Info(err)
			return nil
		}
		pubKeyInfos = append(pubKeyInfos, model.NewPubKeyInfo(*groupsig.NewIDFromString(v.ID), pub))
	}
	return pubKeyInfos
}

func ShowPubKeyInfo(info model.SelfMinerDO, id string) {
	pubKey := info.GetDefaultPubKey().GetHexString()
	common.DefaultLogger.Infof("Miner PubKey: %s;\n", pubKey)
	js, _ := json.Marshal(PubKeyInfo{pubKey, id})
	common.DefaultLogger.Infof("pubkey_info json: %s\n", js)
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

func (gtas *Gtas) autoApplyMiner(mtype int)  {
	miner := mediator.Proc.GetMinerInfo()
	if miner.ID.GetHexString() != gtas.account.Address {
		panic(fmt.Errorf("id error %v %v", miner.ID.GetHexString(), gtas.account.Address))
	}

	tm := &types.Miner{
		Id:           miner.ID.Serialize(),
		PublicKey:    miner.PK.Serialize(),
		VrfPublicKey: miner.VrfPK,
		Stake:        500,
		Type:         byte(mtype),
	}
	data, err := msgpack.Marshal(tm)
	if err != nil {
		fmt.Println("err marhsal types.Miner", err)
		return
	}

	ret, err := GtasAPIImpl.TxUnSafe(gtas.account.Sk, "", 0, 100, 100, 1, types.TransactionTypeMinerApply, common.ToHex(data))
	fmt.Println("apply result", ret, err)

}