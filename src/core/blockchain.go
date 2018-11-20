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

package core

import (
	"common"
	"core/datasource"
	"core/rpc"
	"encoding/binary"
	"time"
	"os"
	"storage/core"
	"storage/tasdb"
	"math/big"
	vtypes "storage/core/types"
	"github.com/hashicorp/golang-lru"
	"fmt"
	"bytes"
	"consensus/groupsig"
	"log"
	"middleware"
	"middleware/types"
	"taslog"
	"middleware/notify"
	"network"
	"math"
)

const (
	BLOCK_STATUS_KEY = "bcurrent"

	CONFIG_SEC = "chain"

	CHAIN_BLOCK_HASH_INIT_LENGTH uint64 = 10

	BLOCK_CHAIN_ADJUST_TIME_OUT = 10 * time.Second
)

var BlockChainImpl *BlockChain

var Logger taslog.Logger

// 配置
type BlockChainConfig struct {
	block string

	blockHeight string

	state string

	//组内能出的最大QN值
	qn uint64
}

type BlockChain struct {
	config *BlockChainConfig

	// key: blockhash, value: block
	blocks tasdb.Database
	//key: height, value: blockHeader
	blockHeight tasdb.Database

	transactionPool *TransactionPool

	//已上链的最新块
	latestBlock   *types.BlockHeader
	topBlocks     *lru.Cache
	latestStateDB *core.AccountDB

	// 读写锁
	lock middleware.Loglock

	// 是否可以工作
	init bool

	statedb    tasdb.Database
	stateCache core.Database // State database to reuse between imports (contains state cache)

	executor      *TVMExecutor
	voteProcessor VoteProcessor

	blockCache *lru.Cache

	isAdujsting bool

	lastBlockHash *BlockHash
}

type castingBlock struct {
	state    *core.AccountDB
	receipts vtypes.Receipts
}

// 默认配置
func DefaultBlockChainConfig() *BlockChainConfig {
	return &BlockChainConfig{
		block: "block",

		blockHeight: "height",

		state: "state",

		qn: 4,
	}
}

func getBlockChainConfig() *BlockChainConfig {
	defaultConfig := DefaultBlockChainConfig()
	if nil == common.GlobalConf {
		return defaultConfig
	}

	return &BlockChainConfig{
		block: common.GlobalConf.GetString(CONFIG_SEC, "block", defaultConfig.block),

		blockHeight: common.GlobalConf.GetString(CONFIG_SEC, "blockHeight", defaultConfig.blockHeight),

		state: common.GlobalConf.GetString(CONFIG_SEC, "state", defaultConfig.state),

		qn: uint64(common.GlobalConf.GetInt(CONFIG_SEC, "qn", int(defaultConfig.qn))),
	}

}

func initBlockChain() error {

	Logger = taslog.GetLoggerByName("core" + common.GlobalConf.GetString("instance", "index", ""))

	chain := &BlockChain{
		config:          getBlockChainConfig(),
		transactionPool: NewTransactionPool(),
		latestBlock:     nil,

		lock:        middleware.NewLoglock("chain"),
		init:        true,
		isAdujsting: false,
	}

	var err error
	chain.blockCache, err = lru.New(20)
	chain.topBlocks, _ = lru.New(1000)
	if err != nil {
		return err
	}

	//从磁盘文件中初始化leveldb
	chain.blocks, err = datasource.NewDatabase(chain.config.block)
	if err != nil {
		//todo: 日志
		return err
	}

	chain.blockHeight, err = datasource.NewDatabase(chain.config.blockHeight)
	if err != nil {
		//todo: 日志
		return err
	}

	chain.statedb, err = datasource.NewDatabase(chain.config.state)
	if err != nil {
		//todo: 日志
		return err
	}
	chain.stateCache = core.NewDatabase(chain.statedb)

	chain.executor = NewTVMExecutor(chain)

	// 恢复链状态 height,latestBlock
	// todo:特殊的key保存最新的状态，当前写到了ldb，有性能损耗
	chain.latestBlock = chain.queryBlockHeaderByHeight([]byte(BLOCK_STATUS_KEY), false)
	if nil != chain.latestBlock {
		chain.buildCache(chain.topBlocks)
		Logger.Infof("initBlockChain chain.latestBlock.StateTree  Hash:%s",chain.latestBlock.StateTree.Hex())
		state, err := core.NewAccountDB(common.BytesToHash(chain.latestBlock.StateTree.Bytes()), chain.stateCache)
		if nil == err {
			chain.latestStateDB = state
		} else {
			panic("initBlockChain NewAccountDB fail:" + err.Error())
		}
	} else {
		// 创始块
		state, err := core.NewAccountDB(common.Hash{}, chain.stateCache)
		if nil == err {
			chain.latestStateDB = state
			block := GenesisBlock(state, chain.stateCache.TrieDB())
			chain.saveBlock(block)
		}
	}

	BlockChainImpl = chain
	return nil
}

func (chain *BlockChain) SetLastBlockHash(bh *BlockHash) {
	chain.lastBlockHash = bh
}

func (chain *BlockChain) Height() uint64 {
	if nil == chain.latestBlock {
		return math.MaxUint64
	}
	return chain.latestBlock.Height
}

func (chain *BlockChain) TotalQN() uint64 {
	if nil == chain.latestBlock {
		return 0
	}
	return chain.latestBlock.TotalQN
}

func (chain *BlockChain) LatestStateDB() *core.AccountDB {
	return chain.latestStateDB
}

func (chain *BlockChain) IsAdujsting() bool {
	return chain.isAdujsting
}

func (chain *BlockChain) GetBalance(address common.Address) *big.Int {
	if nil == chain.latestStateDB {
		return nil
	}

	return chain.latestStateDB.GetBalance(common.BytesToAddress(address.Bytes()))
}

func (chain *BlockChain) GetNonce(address common.Address) uint64 {
	if nil == chain.latestStateDB {
		return 0
	}

	return chain.latestStateDB.GetNonce(common.BytesToAddress(address.Bytes()))
}

//清除链所有数据
func (chain *BlockChain) Clear() error {
	chain.lock.Lock("Clear")
	defer chain.lock.Unlock("Clear")

	chain.init = false
	chain.latestBlock = nil
	chain.topBlocks, _ = lru.New(1000)

	var err error

	chain.blocks.Close()
	chain.blockHeight.Close()
	chain.statedb.Close()

	os.RemoveAll(datasource.DEFAULT_FILE)

	chain.statedb, err = datasource.NewDatabase(chain.config.state)
	if err != nil {
		//todo: 日志
		return err
	}

	chain.stateCache = core.NewDatabase(chain.statedb)
	chain.executor = NewTVMExecutor(chain)

	// 创始块
	state, err := core.NewAccountDB(common.Hash{}, chain.stateCache)
	if nil == err {
		chain.latestStateDB = state
		block := GenesisBlock(state, chain.stateCache.TrieDB())

		chain.saveBlock(block)
	}

	chain.init = true

	chain.transactionPool.Clear()
	return err
}

func (chain *BlockChain) GenerateBlock(bh types.BlockHeader) *types.Block {
	block := &types.Block{
		Header: &bh,
	}

	block.Transactions = make([]*types.Transaction, len(bh.Transactions))
	for i, hash := range bh.Transactions {
		t, _ := chain.transactionPool.GetTransaction(hash)
		if t == nil {
			return nil
		}
		block.Transactions[i] = t
	}
	return block
}

//根据哈希取得某个交易
func (chain *BlockChain) GetTransactionByHash(h common.Hash) (*types.Transaction, error) {
	return chain.transactionPool.GetTransaction(h)
}

//辅助方法族
//查询最高块
func (chain *BlockChain) QueryTopBlock() *types.BlockHeader {
	chain.lock.RLock("QueryTopBlock")
	defer chain.lock.RUnlock("QueryTopBlock")
	result := *chain.latestBlock
	return &result
}

//根据指定哈希查询块
func (chain *BlockChain) QueryBlockByHash(hash common.Hash) *types.BlockHeader {
	return chain.queryBlockHeaderByHash(hash)
}

func (chain *BlockChain) QueryBlockBody(blockHash common.Hash) []*types.Transaction {
	block := chain.queryBlockByHash(blockHash)
	if nil == block {
		return nil
	}
	return block.Transactions
}

func (chain *BlockChain) queryBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	block := chain.queryBlockByHash(hash)
	if nil == block {
		return nil
	}
	return block.Header
}

func (chain *BlockChain) queryBlockByHash(hash common.Hash) *types.Block {
	result, err := chain.blocks.Get(hash.Bytes())

	if result != nil {
		var block *types.Block
		block, err = types.UnMarshalBlock(result)
		if err != nil || &block == nil {
			return nil
		}
		return block
	} else {
		return nil
	}
}

// 根据指定高度查询块
// 带有缓存
func (chain *BlockChain) QueryBlockByHeight(height uint64) *types.BlockHeader {
	chain.lock.RLock("QueryBlockByHeight")
	defer chain.lock.RUnlock("QueryBlockByHeight")

	return chain.queryBlockHeaderByHeight(height, true)
}

// 根据指定高度查询块
func (chain *BlockChain) queryBlockHeaderByHeight(height interface{}, cache bool) *types.BlockHeader {
	var key []byte
	switch height.(type) {
	case []byte:
		key = height.([]byte)
	default:
		if cache {
			h := height.(uint64)
			if h > (chain.latestBlock.Height - 1000) {
				result, ok := chain.topBlocks.Get(h)
				if ok && nil != result {
					return result.(*types.BlockHeader)
				}

			}
		}

		key = chain.generateHeightKey(height.(uint64))
	}

	// 从持久化存储中查询
	result, err := chain.blockHeight.Get(key)
	if result != nil {
		var header *types.BlockHeader
		header, err = types.UnMarshalBlockHeader(result)
		if err != nil {
			return nil
		}

		return header
	} else {
		return nil
	}
}

//构建一个铸块（组内当前铸块人同步操作）
func (chain *BlockChain) CastingBlock(height uint64, nonce uint64, queueNumber uint64, castor []byte, groupid []byte) *types.Block {
	//beginTime := time.Now()
	latestBlock := chain.latestBlock
	//校验高度
	if latestBlock != nil && height <= latestBlock.Height {
		Logger.Debugf("[BlockChain] fail to cast block: height problem. height:%d, latest:%d", height, latestBlock.Height)
		return nil
	}

	block := new(types.Block)

	block.Transactions = chain.transactionPool.GetTransactionsForCasting()
	block.Header = &types.BlockHeader{
		CurTime:     time.Now(), //todo:时区问题
		Height:      height,
		Nonce:       nonce,
		QueueNumber: queueNumber,
		Castor:      castor,
		GroupId:     groupid,
		TotalQN:     latestBlock.TotalQN + queueNumber,//todo:latestBlock != nil?
	}

	if latestBlock != nil {
		block.Header.PreHash = latestBlock.Hash
		block.Header.PreTime = latestBlock.CurTime
	}
	//defer network.Logger.Debugf("casting block %d-%d cost %v,curtime:%v", height, queueNumber, time.Since(beginTime), block.Header.CurTime)

	//Logger.Infof("CastingBlock NewAccountDB height:%d StateTree Hash:%s",height,latestBlock.StateTree.Hex())
	preRoot := common.BytesToHash(latestBlock.StateTree.Bytes())
	if len(block.Transactions) > 0 {
		Logger.Infof("CastingBlock NewAccountDB height:%d preHash:%s preRoot:%s",
			height, latestBlock.Hash.Hex(), preRoot.Hex())
	}
	state, err := core.NewAccountDB(preRoot, chain.stateCache)
	if err != nil {
		var buffer bytes.Buffer
		buffer.WriteString("fail to new statedb, lateset height: ")
		buffer.WriteString(fmt.Sprintf("%d", latestBlock.Height))
		buffer.WriteString(", block height: ")
		buffer.WriteString(fmt.Sprintf("%d error:", block.Header.Height))
		buffer.WriteString(fmt.Sprint(err))
		panic(buffer.String())

	}

	// Process block using the parent state as reference point.
	statehash, receipts, err := chain.executor.Execute(state, block, chain.voteProcessor)

	// 准确执行了的交易，入块
	// 失败的交易也要从池子里，去除掉
	//block.Header.Transactions = make([]common.Hash, len(executed))
	//executedTxs := make([]*Transaction, len(executed))
	//for i, tx := range executed {
	//	if tx == nil {
	//		continue
	//	}
	//	block.Header.Transactions[i] = tx.Hash
	//	executedTxs[i] = tx
	//}
	//block.Transactions = executedTxs
	//block.Header.EvictedTxs = errTxs

	block.Header.Transactions = make([]common.Hash, len(block.Transactions))
	for i, tx := range block.Transactions {
		block.Header.Transactions[i] = tx.Hash
	}
	block.Header.EvictedTxs = []common.Hash{}
	block.Header.TxTree = calcTxTree(block.Transactions)
	Logger.Infof("CastingBlock block.Header.TxTree height:%d StateTree:%s TxTree:%s",height,statehash.Hex(),block.Header.TxTree.Hex())
	block.Header.StateTree = common.BytesToHash(statehash.Bytes())
	block.Header.ReceiptTree = calcReceiptsTree(receipts)
	block.Header.Hash = block.Header.GenHash()

	chain.blockCache.Add(block.Header.Hash, &castingBlock{
		state:    state,
		receipts: receipts,
	})

	chain.transactionPool.ReserveTransactions(block.Header.Hash, block.Transactions)
	return block
}

//验证一个铸块（如本地缺少交易，则异步网络请求该交易）
//返回值:
// 0, 验证通过
// -1，验证失败
// 1 无法验证（缺少交易，已异步向网络模块请求）
// 2 无法验证（前一块在链上不存存在）
func (chain *BlockChain) VerifyCastingBlock(bh types.BlockHeader) ([]common.Hash, int8, *core.AccountDB, vtypes.Receipts) {
	chain.lock.Lock("VerifyCastingBlock")
	defer chain.lock.Unlock("VerifyCastingBlock")

	return chain.verifyCastingBlock(bh, nil)
}

func (chain *BlockChain) verifyCastingBlock(bh types.BlockHeader, txs []*types.Transaction) ([]common.Hash, int8, *core.AccountDB, vtypes.Receipts) {
	// 校验父亲块
	preHash := bh.PreHash
	preBlock := chain.queryBlockHeaderByHash(preHash)

	if preBlock == nil {
		return nil, 2, nil, nil
	}

	// 验证交易
	var (
		transactions []*types.Transaction
		missing      []common.Hash
	)
	if nil == txs {
		transactions, missing, _ = chain.transactionPool.GetTransactions(bh.Hash, bh.Transactions)
	} else {
		transactions = txs
	}

	if 0 != len(missing) {

		var castorId groupsig.ID
		error := castorId.Deserialize(bh.Castor)
		if error != nil {
			log.Printf("[BlockChain]Give up request txs because of groupsig id deserialize error:%s", error.Error())
			return nil, 1, nil, nil
		}

		//向CASTOR索取交易
		m := &TransactionRequestMessage{
			TransactionHashes: missing,
			CurrentBlockHash:  bh.Hash,
			BlockHeight:       bh.Height,
			BlockQn:           bh.QueueNumber,
		}
		go RequestTransaction(*m, castorId.String())
		return missing, 1, nil, nil
	}

	txtree := calcTxTree(transactions)

	if !bytes.Equal(txtree.Bytes(),bh.TxTree.Bytes()) {
		Logger.Debugf("[BlockChain]fail to verify txtree, hash1:%s hash2:%s", txtree.Hex(), bh.TxTree.Hex())
		return missing, -1, nil, nil
	}

	//执行交易
	preRoot := common.BytesToHash(preBlock.StateTree.Bytes())
	if len(txs) > 0 {
		Logger.Infof("NewAccountDB height:%d StateTree:%s preHash:%s preRoot:%s",
			bh.Height, bh.StateTree.Hex(), preHash.Hex(), preRoot.Hex())
	}
	state, err := core.NewAccountDB(preRoot, chain.stateCache)
	if err != nil {
		Logger.Errorf("[BlockChain]fail to new statedb, error:%s", err)
		return nil, -1, nil, nil
	}
	//else {
	//	log.Printf("[BlockChain]state.new %d\n", preBlock.StateTree.Bytes())
	//}

	b := new(types.Block)
	b.Header = &bh
	b.Transactions = transactions


	begin := time.Now()
	Logger.Infof("verifyCastingBlock height:%d StateTree Hash:%s",b.Header.Height,b.Header.StateTree.Hex())

	statehash, receipts, err := chain.executor.Execute(state, b, chain.voteProcessor)
	if common.ToHex(statehash.Bytes()) != common.ToHex(bh.StateTree.Bytes()) {
		Logger.Debugf("[BlockChain]fail to verify statetree, hash1:%x hash2:%x", statehash.Bytes(), b.Header.StateTree.Bytes())
		return nil, -1, nil, nil
	}
	Logger.Debugf("verifyCastingBlock executor Execute cost time:%v", time.Since(begin))
		//receiptsTree := calcReceiptsTree(receipts).Bytes()
	//if common.ToHex(receiptsTree) != common.ToHex(b.Header.ReceiptTree.Bytes()) {
	//	Logger.Debugf("[BlockChain]fail to verify receipt, hash1:%s hash2:%s", receiptsTree, b.Header.ReceiptTree.Bytes())
	//	return nil, 1, nil, nil
	//}

	chain.blockCache.Add(bh.Hash, &castingBlock{
		state:    state,
		receipts: receipts,
	})
	//return nil, 0, state, receipts
	return nil, 0, state, nil
}

//铸块成功，上链
//返回值: 0,上链成功
//       -1，验证失败
//        1, 丢弃该块(链上已存在该块或链上存在QN值更大的相同高度块)
//        2，未上链(异步进行分叉调整)
func (chain *BlockChain) AddBlockOnChain(b *types.Block) int8 {
	if b == nil {
		return -1
	}
	chain.lock.Lock("AddBlockOnChain")
	defer chain.lock.Unlock("AddBlockOnChain")
	//defer network.Logger.Debugf("add on chain block %d-%d,cast+verify+io+onchain cost%v", b.Header.Height, b.Header.QueueNumber, time.Since(b.Header.CurTime))

	return chain.addBlockOnChain(b)
}

func (chain *BlockChain) addBlockOnChain(b *types.Block) int8 {

	var (
		state    *core.AccountDB
		receipts vtypes.Receipts
		status   int8
	)

	// 自己铸块的时候，会将块临时存放到blockCache里
	// 当组内其他成员验证通过后，自己上链就无需验证、执行交易，直接上链即可
	cache, _ := chain.blockCache.Get(b.Header.Hash)
	//if false {
	if cache != nil {
		status = 0
		state = cache.(*castingBlock).state
		receipts = cache.(*castingBlock).receipts
		chain.blockCache.Remove(b.Header.Hash)
	} else {
		// 验证块是否有问题
		_, status, state, receipts = chain.verifyCastingBlock(*b.Header, b.Transactions)
		if status != 0 {
			Logger.Errorf("[BlockChain]fail to VerifyCastingBlock, reason code:%d \n", status)
			return -1
		}
	}

	if b.Header.PreHash == chain.latestBlock.Hash {
		status = chain.saveBlock(b)
	} else if b.Header.TotalQN <= chain.latestBlock.TotalQN || b.Header.Hash == chain.latestBlock.Hash {
		return 1
	} else if b.Header.PreHash == chain.latestBlock.PreHash {
		chain.remove(chain.latestBlock)
		status = chain.saveBlock(b)
	} else {
		//b.Header.TotalQN > chain.latestBlock.TotalQN
		if chain.isAdujsting {
			return 2
		}
		var castorId groupsig.ID
		error := castorId.Deserialize(b.Header.Castor)
		if error != nil {
			log.Printf("[BlockChain]Give up ajusting bolck chain because of groupsig id deserialize error:%s", error.Error())
			return -1
		}
		chain.SetAdujsting(true)
		go RequestBlockInfoByHeight(castorId.String(), chain.latestBlock.Height, chain.latestBlock.Hash)
		status = 2
	}

	// 上链成功，移除pool中的交易
	if 0 == status {
		chain.transactionPool.Remove(b.Header.Hash, b.Header.Transactions)
		chain.transactionPool.AddExecuted(receipts, b.Transactions)
		chain.latestStateDB = state
		root, _ := state.Commit(true)
		triedb := chain.stateCache.TrieDB()
		triedb.Commit(root, false)

		notify.BUS.Publish(notify.BlockAddSucc, &notify.BlockMessage{Block: *b,})
		for _,receipt := range receipts{
			if receipt.Logs != nil {
				for _, log := range receipt.Logs{
					rpc.EventPublisher.PublishEvent(log)
				}
			}
		}
		h, e := types.MarshalBlockHeader(b.Header)
		if e != nil {
			headerMsg := network.Message{Code:network.NewBlockHeaderMsg,Body:h}
			go network.GetNetInstance().Relay(headerMsg,1)
			network.Logger.Debugf("After add on chain,spread block %d-%d header to neighbor,header size %d,hash:%v", b.Header.Height, b.Header.QueueNumber, len(h), b.Header.Hash)
		}

	}
	return status

}

func (chain *BlockChain) CompareChainPiece(bhs []*BlockHash, sourceId string) {
	if bhs == nil || len(bhs) == 0 {
		return
	}
	chain.lock.Lock("CompareChainPiece")
	defer chain.lock.Unlock("CompareChainPiece")
	//Logger.Debugf("[BlockChain] CompareChainPiece get block hashes,length:%d,lowest height:%d", len(bhs), bhs[len(bhs)-1].Height)
	blockHash, hasCommonAncestor, _ := FindCommonAncestor(bhs, 0, len(bhs)-1)
	if hasCommonAncestor {
		Logger.Debugf("[BlockChain]Got common ancestor! Height:%d,localHeight:%d", blockHash.Height, chain.Height())
		//删除自身链的结点
		for height := blockHash.Height + 1; height <= chain.latestBlock.Height; height++ {
			header := chain.queryBlockHeaderByHeight(height, true)
			if header == nil {
				continue
			}
			chain.remove(header)
			chain.topBlocks.Remove(header.Height)
		}
		for h := blockHash.Height; h >= 0; h-- {
			header := chain.queryBlockHeaderByHeight(h, true)
			if header != nil {
				chain.latestBlock = header
				break
			}
		}
		RequestBlockInfoByHeight(sourceId, blockHash.Height, blockHash.Hash)
	} else {
		chain.SetLastBlockHash(bhs[0])
		cbhr := BlockHashesReq{Height: bhs[len(bhs)-1].Height, Length: uint64(len(bhs) * 10)}
		Logger.Debugf("[BlockChain]Do not find common ancestor!Request hashes form node:%s,base height:%d,length:%d", sourceId, cbhr.Height, cbhr.Length)
		RequestBlockHashes(sourceId, cbhr)
	}

}

// 保存block到ldb
// todo:错误回滚
//result code:
// -1 保存失败
// 0 保存成功
func (chain *BlockChain) saveBlock(b *types.Block) int8 {
	// 根据hash存block
	blockJson, err := types.MarshalBlock(b)
	if err != nil {
		log.Printf("[block]fail to json Marshal, error:%s \n", err)
		return -1
	}
	err = chain.blocks.Put(b.Header.Hash.Bytes(), blockJson)
	if err != nil {
		log.Printf("[block]fail to put key:hash value:block, error:%s \n", err)
		return -1
	}

	// 根据height存blockheader
	headerJson, err := types.MarshalBlockHeader(b.Header)
	if err != nil {

		log.Printf("[block]fail to json Marshal header, error:%s \n", err)
		return -1
	}

	err = chain.blockHeight.Put(chain.generateHeightKey(b.Header.Height), headerJson)
	if err != nil {
		log.Printf("[block]fail to put key:height value:headerjson, error:%s \n", err)
		return -1
	}

	// 持久化保存最新块信息
	chain.latestBlock = b.Header
	chain.topBlocks.Add(b.Header.Height, b.Header)
	err = chain.blockHeight.Put([]byte(BLOCK_STATUS_KEY), headerJson)
	if err != nil {
		fmt.Printf("[block]fail to put current, error:%s \n", err)
		return -1
	}

	return 0
}

//进行HASH校验，如果请求结点和当前结点在同一条链上面 返回height到本地高度之间所有的块
//否则返回本地链从height向前开始一定长度的非空块hash 用于查找公公祖先
func (chain *BlockChain) GetBlockInfo(height uint64, hash common.Hash) *BlockInfo {
	chain.lock.RLock("GetBlockInfo")
	defer chain.lock.RUnlock("GetBlockInfo")
	localHeight := chain.latestBlock.Height

	bh := chain.queryBlockHeaderByHeight(height,true)
	if bh != nil && bh.Hash == hash {
		//当前结点和请求结点在同一条链上
		//Logger.Debugf("[BlockChain]Self is on the same branch with request node!")
		var b *types.Block
		for i := height + 1; i <= chain.Height(); i++ {
			bh := chain.queryBlockHeaderByHeight(i, true)
			if nil == bh {
				continue
			}
			b = chain.queryBlockByHash(bh.Hash)
			if nil == b {
				continue
			}
			break
		}
		if b == nil {
			return nil
		}

		var isTopBlock bool
		if b.Header.Height == chain.Height() {
			isTopBlock = true
		} else {
			isTopBlock = false
		}
		return &BlockInfo{Block: b, IsTopBlock: isTopBlock}
	} else {
		//当前结点和请求结点不在同一条链
		//Logger.Debugf("[BlockChain]GetBlockMessage:Self is not on the same branch with request node!")
		var bhs []*BlockHash
		if height >= localHeight {
			bhs = chain.getBlockHashesFromLocalChain(localHeight-1, CHAIN_BLOCK_HASH_INIT_LENGTH)
		} else {
			bhs = chain.getBlockHashesFromLocalChain(height, CHAIN_BLOCK_HASH_INIT_LENGTH)
		}
		return &BlockInfo{ChainPiece: bhs}
	}
}

//从当前链上获取block hash
//height 起始高度
//length 起始高度向下算非空块的长度
func (chain *BlockChain) GetBlockHashesFromLocalChain(height uint64, length uint64) []*BlockHash {
	chain.lock.RLock("GetBlockHashesFromLocalChain")
	defer chain.lock.RUnlock("GetBlockHashesFromLocalChain")
	return chain.getBlockHashesFromLocalChain(height, length)
}

func (chain *BlockChain) getBlockHashesFromLocalChain(height uint64, length uint64) []*BlockHash {
	var i uint64
	r := make([]*BlockHash, 0)
	for i = 0; i < length; {
		bh := BlockChainImpl.queryBlockHeaderByHeight(height, true)
		if bh != nil {
			cbh := BlockHash{Hash: bh.Hash, Height: bh.Height, Qn: bh.QueueNumber}
			r = append(r, &cbh)
			i++
		}
		if height == 0 {
			break
		}
		height--
	}
	return r
}
func FindCommonAncestor(bhs []*BlockHash, l int, r int) (*BlockHash, bool, int) {

	if l > r || r < 0 || l >= len(bhs) {
		return nil, false, -1
	}
	m := (l + r) / 2
	result := isCommonAncestor(bhs, m)
	if result == 0 {
		return bhs[m], true, m
	}

	if result == 1 {
		return FindCommonAncestor(bhs, l, m-1)
	}

	if result == -1 {
		return FindCommonAncestor(bhs, m+1, r)
	}
	return nil, false, -1
}

//bhs 中没有空值
//返回值
// 0  当前HASH相等，后面一块HASH不相等 是共同祖先
//1   当前HASH相等，后面一块HASH相等
//-1  当前HASH不相等
//-100 参数不合法
func isCommonAncestor(bhs []*BlockHash, index int) int {
	if index < 0 || index >= len(bhs) {
		return -100
	}
	he := bhs[index]
	bh := BlockChainImpl.queryBlockHeaderByHeight(he.Height, true)
	if bh == nil {
		Logger.Debugf("[BlockChain]isCommonAncestor:Height:%d,local hash:%s,coming hash:%x\n", he.Height, "null", he.Hash)
		return -1
	}
	Logger.Debugf("[BlockChain]isCommonAncestor:Height:%d,local hash:%x,coming hash:%x\n", he.Height, bh.Hash, he.Hash)
	if index == 0 && bh.Hash == he.Hash {
		return 0
	}
	if index == 0 {
		return -1
	}
	//判断链更后面的一块
	afterHe := bhs[index-1]
	afterbh := BlockChainImpl.queryBlockHeaderByHeight(afterHe.Height, true)
	if afterbh == nil {
		Logger.Debugf("[BlockChain]isCommonAncestor:after block height:%d,local hash:%s,coming hash:%x\n", afterHe.Height, "null", afterHe.Hash)
		if afterHe != nil && bh.Hash == he.Hash {
			return 0
		}
		return -1
	}
	Logger.Debugf("[BlockChain]isCommonAncestor:after block height:%d,local hash:%x,coming hash:%x\n", afterHe.Height, afterbh.Hash, afterHe.Hash)
	if afterHe.Hash != afterbh.Hash && bh.Hash == he.Hash {
		return 0
	}
	if afterHe.Hash == afterbh.Hash && bh.Hash == he.Hash {
		return 1
	}
	return -1
}

func (chain *BlockChain) generateHeightKey(height uint64) []byte {
	h := make([]byte, 8)
	binary.BigEndian.PutUint64(h, height)
	return h
}

// 删除块
func (chain *BlockChain) remove(header *types.BlockHeader) {
	hash := header.Hash
	block := chain.queryBlockByHash(hash)
	chain.blocks.Delete(hash.Bytes())
	chain.blockHeight.Delete(chain.generateHeightKey(header.Height))

	// 删除块的交易，返回transactionpool
	if nil == block {
		return
	}
	txs := block.Transactions
	if 0 == len(txs) {
		return
	}
	chain.transactionPool.lock.Lock("remove block")
	defer chain.transactionPool.lock.Unlock("remove block")
	chain.transactionPool.RemoveExecuted(txs)
	chain.transactionPool.addTxs(txs)
}

func (chain *BlockChain) GetTransactionPool() *TransactionPool {
	return chain.transactionPool
}

func (chain *BlockChain) buildCache(cache *lru.Cache) {
	for i := chain.getStartIndex(); i < chain.latestBlock.Height; i++ {
		chain.topBlocks.Add(i, chain.queryBlockHeaderByHeight(i, false))
	}

}

func (chain *BlockChain) getStartIndex() uint64 {
	var start uint64
	height := chain.latestBlock.Height
	if height < 1000 {
		start = 0
	} else {
		start = height - 999
	}

	return start
}

func Clear() {
	path := datasource.DEFAULT_FILE
	if nil != common.GlobalConf {
		path = common.GlobalConf.GetString(CONFIG_SEC, "database", datasource.DEFAULT_FILE)
	}
	os.RemoveAll(path)

}

func (chain *BlockChain) Close() {
	chain.statedb.Close()
}

func (chain *BlockChain) SetVoteProcessor(processor VoteProcessor) {
	chain.lock.Lock("SetVoteProcessor")
	defer chain.lock.Unlock("SetVoteProcessor")

	chain.voteProcessor = processor
}

func (chain *BlockChain) SetAdujsting(isAjusting bool) {
	chain.isAdujsting = isAjusting
	if isAjusting == true {
		go func() {
			t := time.NewTimer(BLOCK_CHAIN_ADJUST_TIME_OUT)

			<-t.C
			Logger.Debugf("[BlockChain]Local block adjusting time up.change the state!")
			chain.isAdujsting = false
		}()
	}
}
