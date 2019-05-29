package core

import (
	"common"
	"fmt"
	"math/rand"
	"middleware"
	"middleware/ticker"
	time2 "middleware/time"
	"middleware/types"
	"storage/account"
	"storage/tasdb"
	"strconv"
	"taslog"
	"testing"
)

/*
**  Creator: pxf
**  Date: 2019/3/19 下午2:46
**  Description: 
*/

func TestFullBlockChain_HasBlock(t *testing.T) {
	common.InitConf("/Users/pxf/workspace/tas_develop/tas/deploy/daily/tas1.ini")
	types.InitMiddleware()
	middleware.InitMiddleware()
	initBlockChain(nil)

	hasBLock := BlockChainImpl.HasBlock(common.HexToHash("0x7f57774109cad543d9acfbcfa3630b30ca652d2310470341b78c62ee7463633b"))
	t.Log(hasBLock)
}

func TestFullBlockChain_QueryBlockFloor(t *testing.T) {
	common.InitConf("/Users/pxf/workspace/tas_develop/test9/tas9.ini")
	middleware.InitMiddleware()
	initBlockChain(nil)

	chain := BlockChainImpl.(*FullBlockChain)

	fmt.Println("=====")
	bh := chain.queryBlockHeaderByHeight(0)
	fmt.Println(bh, bh.Hash.String())
	//top := gchain.latestBlock
	//t.Log(top.Height, top.Hash.String())
	//
	//for h := uint64(4460); h <= 4480; h++ {
	//	bh := gchain.queryBlockHeaderByHeightFloor(h)
	//	t.Log(bh.Height, bh.Hash.String())
	//}

	bh = chain.queryBlockHeaderByHeightFloor(0)
	fmt.Println(bh)
}

type CommitContext struct {
	fullChain *FullBlockChain
	block *types.Block
}

var ctx *CommitContext
func init(){
	fc:= newFullChain()
	blk := initBlock()

	ctx = &CommitContext{
		fullChain:fc,
		block:blk,
	}
}

func BenchmarkCommitBlock(b *testing.B){
	db := ctx.fullChain.getDB()
	//block := initBlock()
	ctx.block.Header.Hash=common.BytesToHash([]byte(strconv.Itoa(rand.Int())))
	ctx.fullChain.commitBlock(ctx.block,&executePostState{state:db})
}

func newFullChain()*FullBlockChain{
	middleware.InitMiddleware()
	common.InitConf("tas1.ini")
	common.DefaultLogger = taslog.GetLoggerByIndex(taslog.DefaultConfig, common.GlobalConf.GetString("instance", "index", ""))
	types.InitMiddleware()
	instance := common.GlobalConf.GetString("instance", "index", "")
	Logger = taslog.GetLoggerByIndex(taslog.CoreLogConfig, instance)
	consensusLogger = taslog.GetLoggerByIndex(taslog.ConsensusLogConfig, instance)
	chain := &FullBlockChain{
		config:          getBlockChainConfig(),
		latestBlock:     nil,
		init:            true,
		isAdujsting:     false,
		consensusHelper: nil,
		ticker:          ticker.NewGlobalTicker("chain"),
		ts: 			time2.TSInstance,
		futureBlocks: 	common.MustNewLRUCache(10),
		verifiedBlocks: common.MustNewLRUCache(10),
		topBlocks: 		common.MustNewLRUCache(20),
	}
	ds, err := tasdb.NewDataSource(chain.config.dbfile)
	if err != nil {
		fmt.Errorf("init db error")
	}

	chain.blocks, err = ds.NewPrefixDatabase(chain.config.block)
	if err != nil {
		fmt.Errorf("init db error")
	}

	chain.blockHeight, err = ds.NewPrefixDatabase(chain.config.blockHeight)
	if err != nil {
		fmt.Errorf("init db error")
	}
	chain.txdb, err = ds.NewPrefixDatabase(chain.config.tx)
	if err != nil {
		fmt.Errorf("init db error")
	}
	chain.statedb, err = ds.NewPrefixDatabase(chain.config.state)
	if err != nil {
		fmt.Errorf("init db error")
	}

	receiptdb, err := ds.NewPrefixDatabase(chain.config.receipt)
	if err != nil {
		fmt.Errorf("init db error")
	}
	chain.stateCache = account.NewDatabase(chain.statedb)

	chain.bonusManager = newBonusManager()
	chain.batch = chain.blocks.CreateLDBBatch()
	chain.transactionPool = NewTransactionPool(chain, receiptdb)
	return chain
}

func (chain *FullBlockChain) getDB()*account.AccountDB{
	stateDB, _ := account.NewAccountDB(common.Hash{}, chain.stateCache)
	return stateDB
}

func initBlock() *types.Block{
	hash := common.BytesToHash([]byte(strconv.Itoa(rand.Int())))
	preHash := common.BytesToHash([]byte(strconv.Itoa(rand.Int())))
	bh := &types.BlockHeader{
		Hash:hash,
		Height:uint64(rand.Int()),
		PreHash:preHash,
		Elapsed:int32(rand.Int()),
		ProveValue:[]byte(strconv.Itoa(rand.Int())),
		TotalQN:uint64(rand.Int()),
		CurTime:time2.Int64ToTimeStamp(int64(rand.Int())),
		Castor:[]byte(strconv.Itoa(rand.Int())),
		GroupId:[]byte(strconv.Itoa(rand.Int())),
		Signature:[]byte(strconv.Itoa(rand.Int())),
		Nonce:int32(rand.Int()),
		TxTree:common.BytesToHash([]byte(strconv.Itoa(rand.Int()))),
		ReceiptTree:common.BytesToHash([]byte(strconv.Itoa(rand.Int()))),
		StateTree:common.BytesToHash([]byte(strconv.Itoa(rand.Int()))),
		ExtraData:[]byte(strconv.Itoa(rand.Int())),
		Random:[]byte(strconv.Itoa(rand.Int())),
	}

	txs := []*types.Transaction{}
	for i:=0;i<5000;i++{
		addr := common.BytesToAddress([]byte(strconv.Itoa(rand.Int())))
		sc:=common.BytesToAddress([]byte(strconv.Itoa(rand.Int())))
		tx := &types.Transaction{
			Data:[]byte(strconv.Itoa(rand.Int())),
			Value:uint64(rand.Int()),
			Nonce:uint64(rand.Int()),
			Target:&addr,
			Type:int8(rand.Int()),
			GasLimit:111111111,
			GasPrice:11111111111,
			Hash:common.BytesToHash([]byte(strconv.Itoa(rand.Int()))),
			ExtraData:[]byte(strconv.Itoa(rand.Int())),
			Sign:[]byte(strconv.Itoa(rand.Int())),
			Source:&sc,
		}
		txs = append(txs, tx)
	}
	return &types.Block{
		Header:bh,
		Transactions:txs,
	}
}