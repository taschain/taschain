////   Copyright (C) 2018 TASChain
////
////   This program is free software: you can redistribute it and/or modify
////   it under the terms of the GNU General Public License as published by
////   the Free Software Foundation, either version 3 of the License, or
////   (at your option) any later version.
////
////   This program is distributed in the hope that it will be useful,
////   but WITHOUT ANY WARRANTY; without even the implied warranty of
////   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
////   GNU General Public License for more details.
////
////   You should have received a copy of the GNU General Public License
////   along with this program.  If not, see <https://www.gnu.org/licenses/>.
//
package core

import (
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/middleware"
	"github.com/taschain/taschain/middleware/ticker"
	time2 "github.com/taschain/taschain/middleware/time"
	"github.com/taschain/taschain/middleware/types"
	"github.com/taschain/taschain/network"
	"github.com/taschain/taschain/storage/account"
	"github.com/taschain/taschain/storage/tasdb"
	"github.com/taschain/taschain/taslog"
	"math"
	"math/big"
	"os"
	"strconv"
	"testing"
)

var source = "100"



func TestBlockChain_AddBlock(t *testing.T) {
	initContext4Test()

	//BlockChainImpl.Clear()

	queryAddr := "0xf77fa9ca98c46d534bd3d40c3488ed7a85c314db0fd1e79c6ccc75d79bd680bd"
	b := BlockChainImpl.GetBalance(common.StringToAddress(queryAddr));
	addr := genHash("1")
	fmt.Printf("balance = %d \n",b)
	fmt.Printf("addr = %s \n",common.BytesToAddress(addr))

	// 查询创始块
	blockHeader := BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 0 != blockHeader.Height {
		t.Fatalf("clear data fail")
	}




	if BlockChainImpl.GetBalance(common.BytesToAddress(genHash("1"))).Int64() != 1000000 {
		//t.Fatalf("fail to init 1 balace to 100")
	}

	txpool := BlockChainImpl.GetTransactionPool()
	if nil == txpool {
		t.Fatalf("fail to get txpool")
	}
//	code := `
//import account
//def Test(a, b, c, d):
//	print("hehe")
//`
	// 交易1
	_, err := txpool.AddTransaction(genTestTx( 12345, "100", "2", 0, 1))
	if err != nil {
		t.Fatalf("fail to AddTransaction")
	}
	//txpool.AddTransaction(genContractTx(1, 20000000, "1", "", 1, 0, []byte(code), nil, 0))
	contractAddr := common.BytesToAddress(common.Sha256(common.BytesCombine([]byte("1"), common.Uint64ToByte(0))))
	//交易2
	_, err = txpool.AddTransaction(genTestTx( 123456, "2", "3", 0, 1))
	if err != nil {
		t.Fatalf("fail to AddTransaction")
	}

	//交易3 执行失败的交易
	_, err = txpool.AddTransaction(genTestTx( 123456, "2", "3", 0, 1))
	if err != nil {
		t.Fatalf("fail to AddTransaction")
	}
	castor := new([]byte)
	groupid := new([]byte)

	// 铸块1
	block := BlockChainImpl.CastBlock(1, common.Hex2Bytes("12"), 0, *castor, *groupid)
	if nil == block {
		t.Fatalf("fail to cast new block")
	}

	// 上链
	//time.Sleep(time.Second)
	if 0 != BlockChainImpl.AddBlockOnChain(source,block) {
		t.Fatalf("fail to add block")
	}

	//最新块是块1
	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 1 != blockHeader.Height {
		t.Fatalf("add block1 failed")
	}

	if BlockChainImpl.GetBalance(common.BytesToAddress(genHash("1"))).Int64() != 999999 {
		//t.Fatalf("fail to transfer 1 from 1  to 2")
	}

	// 池子中交易的数量为0

	if 0 != len(txpool.GetReceived()) {
		t.Fatalf("fail to remove transactions after addBlock")
	}
	
	//交易3
	_, err = txpool.AddTransaction(genTestTx( 1, "1", "2", 2, 10))
	if err != nil {
		t.Fatalf("fail to AddTransaction")
	}

	//txpool.AddTransaction(genContractTx(1, 20000000, "1", contractAddr.GetHexString(), 3, 0, []byte(`{"FuncName": "Test", "Args": [10.123, "ten", [1, 2], {"key":"value", "key2":"value2"}]}`), nil, 0))
	fmt.Println(contractAddr.GetHexString())
	// 铸块2
	block2 := BlockChainImpl.CastBlock(2,  common.Hex2Bytes("123"), 0, *castor, *groupid)
	//time.Sleep(time.Second)
	if 0 != BlockChainImpl.AddBlockOnChain(source,block2) {
		t.Fatalf("fail to add empty block")
	}

	if BlockChainImpl.GetBalance(common.BytesToAddress(genHash("1"))).Int64() != 999989 {
		//t.Fatalf("fail to transfer 10 from 1 to 2")
	}

	//最新块是块2
	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 2 != blockHeader.Height || blockHeader.Hash != block2.Header.Hash || block.Header.Hash != block2.Header.PreHash {
		t.Fatalf("add block2 failed")
	}
	blockHeader = BlockChainImpl.QueryBlockByHash(block2.Header.Hash).Header
	if nil == blockHeader {
		t.Fatalf("fail to QueryBlockByHash, hash: %x ", block2.Header.Hash)
	}

	blockHeader = BlockChainImpl.QueryBlockByHeight(2).Header
	if nil == blockHeader {
		t.Fatalf("fail to QueryBlockByHeight, height: %d ", 2)
	}

	// 铸块3 空块
	block3 := BlockChainImpl.CastBlock(3, common.Hex2Bytes("125"), 0, *castor, *groupid)
	//time.Sleep(time.Second)
	if 0 != BlockChainImpl.AddBlockOnChain(source,block3) {
		t.Fatalf("fail to add empty block")
	}
	//最新块是块3
	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 3 != blockHeader.Height || blockHeader.Hash != block3.Header.Hash {
		t.Fatalf("add block3 failed")
	}

	block4 := BlockChainImpl.CastBlock(4, common.Hex2Bytes("126"), 0, *castor, *groupid)
	//time.Sleep(time.Second)
	if 0 != BlockChainImpl.AddBlockOnChain(source,block4) {
		t.Fatalf("fail to add empty block")
	}
	//最新块是块3
	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 4 != blockHeader.Height || blockHeader.Hash != block4.Header.Hash {
		t.Fatalf("add block3 failed")
	}

	block5 := BlockChainImpl.CastBlock(5, common.Hex2Bytes("126"), 0, *castor, *groupid)
	//time.Sleep(time.Second)
	if 0 != BlockChainImpl.AddBlockOnChain(source,block5) {
		t.Fatalf("fail to add empty block")
	}
	//最新块是块5
	blockHeader = BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 5 != blockHeader.Height || blockHeader.Hash != block5.Header.Hash {
		t.Fatalf("add block3 failed")
	}

	// 铸块4 空块
	// 模拟分叉
	//block4 := BlockChainImpl.CastBlockAfter(block.Header, 2, 124, 0, *castor, *groupid)
	//
	//if 0 != BlockChainImpl.AddBlockOnChain(block4) {
	//	t.Fatalf("fail to add empty block")
	//}
	////最新块是块4
	//blockHeader = BlockChainImpl.QueryTopBlock()
	//if nil == blockHeader || 2 != blockHeader.Height || blockHeader.Hash != block4.Header.Hash {
	//	t.Fatalf("add block4 failed")
	//}
	//blockHeader = BlockChainImpl.QueryBlockByHeight(3)
	//if nil != blockHeader {
	//	t.Fatalf("failed to remove uncle blocks")
	//}
	//
	//if BlockChainImpl.GetBalance(c.BytesToAddress(genHash("1"))).Int64() != 999999 {
	//	t.Fatalf("fail to switch to main gchain. %d", BlockChainImpl.GetBalance(c.BytesToAddress(genHash("1"))))
	//}

	BlockChainImpl.Close()

}

func TestBlockChain_CastingBlock(t *testing.T) {
	initContext4Test()


	castor := []byte{1, 2}
	group := []byte{3, 4}
	block1 := BlockChainImpl.CastBlock(1, common.Hex2Bytes("1"), 1, castor, group)
	if nil == block1 {
		t.Fatalf("fail to cast block1")
	}

	BlockChainImpl.Close()
}

func TestBlockChain_GetBlockMessage(t *testing.T) {
	initContext4Test()

	castor := new([]byte)
	groupid := new([]byte)
	block1 := BlockChainImpl.CastBlock(1, common.Hex2Bytes("125"), 0, *castor, *groupid)
	//time.Sleep(time.Second)
	if 0 != BlockChainImpl.AddBlockOnChain(source,block1) {
		t.Fatalf("fail to add empty block")
	}

	block2 := BlockChainImpl.CastBlock(2, common.Hex2Bytes("1256"), 0, *castor, *groupid)
	//time.Sleep(time.Second)
	if 0 != BlockChainImpl.AddBlockOnChain(source,block2) {
		t.Fatalf("fail to add empty block")
	}

	block3 := BlockChainImpl.CastBlock(3, common.Hex2Bytes("1257"), 0, *castor, *groupid)
	//time.Sleep(time.Second)
	if 0 != BlockChainImpl.AddBlockOnChain(source,block3) {
		t.Fatalf("fail to add empty block")
	}

	if 3 != BlockChainImpl.Height() {
		t.Fatalf("fail to add 3 blocks")
	}
	chain := BlockChainImpl.(*FullBlockChain)

	header1 := chain.queryBlockHeaderByHeight(uint64(1))
	header2 := chain.queryBlockHeaderByHeight(uint64(2))
	header3 := chain.queryBlockHeaderByHeight(uint64(3))

	b1 := chain.queryBlockByHash(header1.Hash)
	b2 := chain.queryBlockByHash(header2.Hash)
	b3 := chain.queryBlockByHash(header3.Hash)

	fmt.Printf("1: %d\n", b1.Header.Nonce)
	fmt.Printf("2: %d\n", b2.Header.Nonce)
	fmt.Printf("3: %d\n", b3.Header.Nonce)

}

func TestBlockChain_GetTopBlocks(t *testing.T) {
	initContext4Test()

	castor := new([]byte)
	groupid := new([]byte)

	var i uint64
	for i = 1; i < 2000; i++ {
		block := BlockChainImpl.CastBlock(i, common.Hex2Bytes( strconv.FormatInt(int64(i),10)), 0, *castor, *groupid)
		//time.Sleep(time.Second)
		if 0 != BlockChainImpl.AddBlockOnChain(source,block) {
			t.Fatalf("fail to add empty block")
		}
	}
	chain := BlockChainImpl.(*FullBlockChain)
	lent := chain.topBlocks.Len()
	fmt.Printf("len = %d \n",lent)
	if 20 != chain.topBlocks.Len() {
		t.Fatalf("error for size:20")
	}


	for i = BlockChainImpl.Height() - 19; i < 2000; i++ {
		lowestLDB := chain.queryBlockHeaderByHeight(i)
		if nil == lowestLDB {
			t.Fatalf("fail to get lowest block from ldb,%d", i)
		}

		lowest, ok := chain.topBlocks.Get(lowestLDB.Hash)
		if !ok || nil == lowest {
			t.Fatalf("fail to get lowest block from cache,%d", i)
		}
	}
}

func TestBlockChain_StateTree(t *testing.T) {
	initContext4Test()
	chain := BlockChainImpl.(*FullBlockChain)

	// 查询创始块
	blockHeader := BlockChainImpl.QueryTopBlock()
	if nil == blockHeader || 0 != blockHeader.Height {
		t.Fatalf("clear data fail")
	}

	if BlockChainImpl.GetBalance(common.BytesToAddress(genHash("1"))).Int64() != 100 {
		//t.Fatalf("fail to init 1 balace to 100")
	}

	txpool := BlockChainImpl.GetTransactionPool()
	if nil == txpool {
		t.Fatalf("fail to get txpool")
	}

	castor := new([]byte)
	groupid := new([]byte)

	block0 := BlockChainImpl.CastBlock(1, common.Hex2Bytes("12"), 0, *castor, *groupid)
	// 上链
	//time.Sleep(time.Second)
	if 0 != BlockChainImpl.AddBlockOnChain(source,block0) {
		t.Fatalf("fail to add block0")
	}

	// 交易1
	txpool.AddTransaction(genTestTx( 12345, "1", "2", 1, 1))

	//交易2
	txpool.AddTransaction(genTestTx( 123456, "2", "3", 2, 2))

	// 交易3 失败的交易
	txpool.AddTransaction(genTestTx(123457, "1", "2", 2, 3))

	// 铸块1
	block := BlockChainImpl.CastBlock(2, common.Hex2Bytes("12"), 0, *castor, *groupid)
	if nil == block {
		t.Fatalf("fail to cast new block")
	}

	// 上链
	//time.Sleep(time.Second)
	if 0 != BlockChainImpl.AddBlockOnChain(source,block) {
		t.Fatalf("fail to add block")
	}

	// 铸块2
	block2 := BlockChainImpl.CastBlock(3, common.Hex2Bytes("22"), 0, *castor, *groupid)
	if nil == block2 {
		t.Fatalf("fail to cast new block")
	}

	// 上链
	//time.Sleep(time.Second)
	if 0 != BlockChainImpl.AddBlockOnChain(source,block2) {
		t.Fatalf("fail to add block")
	}

	fmt.Printf("state: %d\n", chain.latestBlock.StateTree)

	// 铸块3
	block3 := BlockChainImpl.CastBlock(4, common.Hex2Bytes("12"), 0, *castor, *groupid)
	if nil == block3 {
		t.Fatalf("fail to cast new block")
	}

	// 上链
	//time.Sleep(time.Second)
	if 0 != BlockChainImpl.AddBlockOnChain(source,block3) {
		t.Fatalf("fail to add block")
	}


	fmt.Printf("state: %d\n", chain.getLatestBlock().StateTree)
}

var privateKey = "0x045c8153e5a849eef465244c0f6f40a43feaaa6855495b62a400cc78f9a6d61c76c09c3aaef393aa54bd2adc5633426e9645dfc36723a75af485c5f5c9f2c94658562fcdfb24e943cf257e25b9575216c6647c4e75e264507d2d57b3c8bc00b361"
func genTestTx(price uint64, source string, target string, nonce uint64, value uint64) *types.Transaction {

	sourcebyte := common.BytesToAddress(genHash(source))
	targetbyte := common.BytesToAddress(genHash(target))

	tx := &types.Transaction{
		GasPrice: price,

		Source:   &sourcebyte,
		Target:   &targetbyte,
		Nonce:    nonce,
		Value:    value,
	}
	tx.Hash = tx.GenHash()
	sk := common.HexStringToSecKey(privateKey)
	sign := sk.Sign(tx.Hash.Bytes())
	tx.Sign = sign.Bytes()

	return tx
}


func genHash(hash string) []byte {
	bytes3 := []byte(hash)
	return common.Sha256(bytes3)
}

//func TestMinerOnChain(t *testing.T)  {
//	Clear()
//	code := tvm.Read0("/Users/guangyujing/workspace/tas/src/tvm/py/miner/miner.py")
//
//	contract := tvm.Contract{code, "miner", nil}
//	jsonString, _ := json.Marshal(contract)
//	fmt.Println(string(jsonString))
//	contractAddress := common.HexToAddress("0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
//	OnChainFunc(string(jsonString), contractAddress.GetHexString())
//}
//

func resetDb(){
	_ =os.RemoveAll("d_b")
	_ =os.RemoveAll("d_g")
	_ =os.RemoveAll("d_txidx")
}

func initContext4Test(){
	_ = os.Chdir("../deploy/daily")
	common.InitConf("tas1.ini")
	resetDb()
	network.Logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))
	_ = middleware.InitMiddleware()
	//_ = initBlockChain(NewConsensusHelper4Test(groupsig.ID{}))
	_ = InitCore(false,NewConsensusHelper4Test(groupsig.ID{}))
	BlockChainImpl.GetTransactionPool().Clear()
}


func NewConsensusHelper4Test(id groupsig.ID) types.ConsensusHelper {
	return &ConsensusHelperImpl4Test{ID: id}
}

type ConsensusHelperImpl4Test struct {
	ID groupsig.ID
}

func (helper *ConsensusHelperImpl4Test) GenerateGenesisInfo() *types.GenesisInfo {
	return &types.GenesisInfo{}
}

func (helper *ConsensusHelperImpl4Test) VRFProve2Value(prove []byte) *big.Int {
	if len(prove) == 0 {
		return big.NewInt(0)
	}
	return big.NewInt(1)
}

func (helper *ConsensusHelperImpl4Test) ProposalBonus() *big.Int {
	return new(big.Int).SetUint64(model.Param.ProposalBonus)
}

func (helper *ConsensusHelperImpl4Test) PackBonus() *big.Int {
	return new(big.Int).SetUint64(model.Param.PackBonus)
}

func (helper *ConsensusHelperImpl4Test) CheckProveRoot(bh *types.BlockHeader) (bool, error) {
	//return Proc.CheckProveRoot(bh)
	return true, nil	//上链时不再校验，只在共识时校验（update：2019-04-23）
}

func (helper *ConsensusHelperImpl4Test) VerifyNewBlock(bh *types.BlockHeader, preBH *types.BlockHeader) (bool, error) {
	return true,nil
}

func (helper *ConsensusHelperImpl4Test) VerifyBlockHeader(bh *types.BlockHeader) (bool, error) {
	return true, nil
}

func (helper *ConsensusHelperImpl4Test) CheckGroup(g *types.Group) (ok bool, err error) {
	return true, nil
}



func (helper *ConsensusHelperImpl4Test) VerifyBonusTransaction(tx *types.Transaction) (ok bool, err error) {
	return true, nil
}


func (helper *ConsensusHelperImpl4Test) EstimatePreHeight(bh *types.BlockHeader) uint64 {
	height := bh.Height
	if height == 1 {
		return 0
	}
	return height - uint64(math.Ceil(float64(bh.Elapsed)/float64(model.Param.MaxGroupCastTime)))
}



func (helper *ConsensusHelperImpl4Test) CalculateQN(bh *types.BlockHeader) uint64 {
	return uint64(11)
}





func (chain *FullBlockChain) getDB()*account.AccountDB{
	stateDB, _ := account.NewAccountDB(common.Hash{}, chain.stateCache)
	return stateDB
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
	//options := &opt.Options{
	//	OpenFilesCacheCapacity: 100,
	//	BlockCacheCapacity:     16 * opt.MiB,
	//	WriteBuffer:            256 * opt.MiB, // Two of these are used internally
	//	Filter:                 filter.NewBloomFilter(10),
	//	CompactionTableSize: 	4*opt.MiB,
	//	CompactionTableSizeMultiplier: 2,
	//	CompactionTotalSize: 	16*opt.MiB,
	//	BlockSize: 				2*opt.MiB,
	//}
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

	pool := &TxPool{
		receiptdb:          receiptdb,
		batch:              chain.batch,
		asyncAdds:          common.MustNewLRUCache(txCountPerBlock * maxReqBlockCount),
		chain:              chain,
		gasPriceLowerBound: uint64(common.GlobalConf.GetInt("chain", "gasprice_lower_bound", 1)),
	}
	pool.received = newSimpleContainer(maxTxPoolSize)
	pool.bonPool = newBonusPool(chain.bonusManager, bonusTxMaxSize)

	chain.transactionPool = pool

	s := &txSyncer{
		rctNotifiy:    common.MustNewLRUCache(1000),
		indexer:       buildTxSimpleIndexer(),
		pool:          pool,
		ticker:        ticker.NewGlobalTicker("tx_syncer"),
		candidateKeys: common.MustNewLRUCache(100),
		chain:         chain,
		logger:        taslog.GetLoggerByIndex(taslog.TxSyncLogConfig, common.GlobalConf.GetString("instance", "index", "")),
	}

	TxSyncer = s

	return chain
}
