package core
//
//import (
//	"testing"
//	"common"
//	"os"
//	"network"
//	"taslog"
//	"middleware/types"
//	"fmt"
//	"github.com/vmihailenco/msgpack"
//)
//
//func TestMinerManager_AddMiner(t *testing.T) {
//	common.InitConf(os.Getenv("HOME") + "/TasProject/work/1g3n/test1.ini")
//	network.Logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))
//	//Clear()
//	config := getBlockChainConfig()
//	initMinerManager(config)
//	miner1 := &types.Miner{Id:[]byte{1},Stake:100,Type:types.MinerTypeHeavy,AbortHeight:0,Status:types.MinerStatusNormal}
//	miner2 := &types.Miner{Id:[]byte{2},Stake:100,Type:types.MinerTypeHeavy,AbortHeight:0,Status:types.MinerStatusNormal}
//	MinerManagerImpl.AddGenesesMiner([]*types.Miner{miner1,miner2})
//	//miner,_ := MinerManagerImpl.GetMinerById([]byte{1},types.MinerTypeHeavy)
//	//fmt.Println(miner)
//	total := MinerManagerImpl.GetTotalStakeByHeight(0)
//	fmt.Println(total)
//	iter := MinerManagerImpl.heavyDB.NewIterator()
//	for ;iter.Next();{
//		var miner types.Miner
//		msgpack.Unmarshal(iter.Value(),&miner)
//		fmt.Printf("%+v\n",miner)
//	}
//}