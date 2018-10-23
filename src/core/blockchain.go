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
	"time"
	"os"
	"storage/core"
	vtypes "storage/core/types"
	"github.com/hashicorp/golang-lru"
	"fmt"
	"bytes"
	"consensus/groupsig"
	"log"
	"middleware"
	"middleware/types"
	"taslog"
	"math/big"
	"storage/core/vm"
	"middleware/notify"
)

const (
	BLOCK_STATUS_KEY = "bcurrent"

	CONFIG_SEC = "chain"
)

var BlockChainImpl BlockChain

var Logger taslog.Logger

// 配置
type BlockChainConfig struct {
	block string

	blockHeight string

	state string

	bonus string

	heavy string

	light string
}

type FullBlockChain struct {
	prototypeChain
	config       *BlockChainConfig
	castedBlock  *lru.Cache
	bonusManager *BonusManager
}

type castingBlock struct {
	state    *core.AccountDB
	receipts vtypes.Receipts
}

func getBlockChainConfig() *BlockChainConfig {
	defaultConfig := &BlockChainConfig{
		block: "block",

		blockHeight: "height",

		state: "state",


		bonus: "bonus",

		light: "light",

		heavy: "heavy",
	}

	if nil == common.GlobalConf {
		return defaultConfig
	}

	return &BlockChainConfig{
		block: common.GlobalConf.GetString(CONFIG_SEC, "block", defaultConfig.block),

		blockHeight: common.GlobalConf.GetString(CONFIG_SEC, "blockHeight", defaultConfig.blockHeight),

		state: common.GlobalConf.GetString(CONFIG_SEC, "state", defaultConfig.state),


		bonus: common.GlobalConf.GetString(CONFIG_SEC, "bonus", defaultConfig.bonus),

		heavy: common.GlobalConf.GetString(CONFIG_SEC, "heavy", defaultConfig.heavy),

		light: common.GlobalConf.GetString(CONFIG_SEC, "light", defaultConfig.light),
	}

}

func initBlockChain(helper types.ConsensusHelper) error {

	Logger = taslog.GetLoggerByName("core" + common.GlobalConf.GetString("instance", "index", ""))

	chain := &FullBlockChain{
		config: getBlockChainConfig(),
		prototypeChain: prototypeChain{
			transactionPool: NewTransactionPool(),
			latestBlock:     nil,
			lock:            middleware.NewLoglock("chain"),
			init:            true,
			isAdujsting:     false,
			isLightMiner:    false,
			consensusHelper: helper,
		},
	}

	var err error
	chain.verifiedBlocks, err = lru.New(20)
	chain.topBlocks, _ = lru.New(1000)
	if err != nil {
		return err
	}

	chain.castedBlock, err = lru.New(20)

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

	chain.bonusManager = newBonusManager()
	chain.stateCache = core.NewDatabase(chain.statedb)

	chain.executor = NewTVMExecutor(chain)
	initMinerManager(chain)
	initTraceChain()
	// 恢复链状态 height,latestBlock
	// todo:特殊的key保存最新的状态，当前写到了ldb，有性能损耗
	chain.latestBlock = chain.queryBlockHeaderByHeight([]byte(BLOCK_STATUS_KEY), false)
	if nil != chain.latestBlock {
		chain.buildCache(1000, chain.topBlocks)
		Logger.Infof("initBlockChain chain.latestBlock.StateTree  Hash:%s", chain.latestBlock.StateTree.Hex())
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
			block := GenesisBlock(state, chain.stateCache.TrieDB(), chain.consensusHelper.GenerateGenesisInfo())
			Logger.Infof("GenesisBlock StateTree:%s", block.Header.StateTree.Hex())
			_, headerJson := chain.saveBlock(block)
			chain.updateLastBlock(state, block.Header, headerJson)
		}
	}

	BlockChainImpl = chain
	return nil
}

//构建一个铸块（组内当前铸块人同步操作）
func (chain *FullBlockChain) CastBlock(height uint64, proveValue *big.Int, qn uint64, castor []byte, groupid []byte) *types.Block {
	//beginTime := time.Now()
	latestBlock := chain.QueryTopBlock()
	//校验高度
	if latestBlock != nil && height <= latestBlock.Height {
		Logger.Debugf("[BlockChain] fail to cast block: height problem. height:%d, latest:%d", height, latestBlock.Height)
		return nil
	}

	block := new(types.Block)

	block.Transactions = chain.transactionPool.GetTransactionsForCasting()
	block.Header = &types.BlockHeader{
		CurTime:    time.Now(), //todo:时区问题
		Height:     height,
		ProveValue: proveValue,
		Castor:     castor,
		GroupId:    groupid,
		TotalQN:    latestBlock.TotalQN + qn, //todo:latestBlock != nil?
		StateTree:  common.BytesToHash(latestBlock.StateTree.Bytes()),
	}

	if latestBlock != nil {
		block.Header.PreHash = latestBlock.Hash
		block.Header.PreTime = latestBlock.CurTime
	}

	//Logger.Infof("CastingBlock NewAccountDB height:%d StateTree Hash:%s",height,latestBlock.StateTree.Hex())
	preRoot := common.BytesToHash(latestBlock.StateTree.Bytes())
	//if len(block.Transactions) > 0 {
	//	Logger.Infof("CastingBlock NewAccountDB height:%d preHash:%s preRoot:%s", height, latestBlock.Hash.Hex(), preRoot.Hex())
	//}
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
	statehash, receipts, err := chain.executor.Execute(state, block, height, "casting")

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
	block.Header.TxTree = calcTxTree(block.Transactions)
	//Logger.Infof("CastingBlock block.Header.TxTree height:%d StateTree Hash:%s",height,statehash.Hex())
	block.Header.StateTree = common.BytesToHash(statehash.Bytes())
	block.Header.ReceiptTree = calcReceiptsTree(receipts)
	block.Header.Hash = block.Header.GenHash()
	defer Logger.Infof("casting block %d,hash:%v,qn:%d", height, block.Header.Hash.String(), block.Header.TotalQN)

	//自己铸的块 自己不需要验证
	chain.verifiedBlocks.Add(block.Header.Hash, &castingBlock{
		state:    state,
		receipts: receipts,
	})
	chain.castedBlock.Add(block.Header.Hash, block)
	Logger.Debugf("CastingBlock into cache! Height:%d-%d,Hash:%x,stateHash:%x,len tx:%d", height, block.Header.ProveValue, block.Header.Hash, block.Header.StateTree, len(block.Transactions))

	chain.transactionPool.ReserveTransactions(block.Header.Hash, block.Transactions)
	return block
}

//验证一个铸块（如本地缺少交易，则异步网络请求该交易）
//返回值:
// 0, 验证通过
// -1，验证失败
// 1 无法验证（缺少交易，已异步向网络模块请求）
// 2 无法验证（前一块在链上不存存在）
func (chain *FullBlockChain) VerifyBlock(bh types.BlockHeader) ([]common.Hash, int8, *core.AccountDB, vtypes.Receipts) {
	chain.lock.Lock("VerifyCastingBlock")
	defer chain.lock.Unlock("VerifyCastingBlock")

	return chain.verifyCastingBlock(bh, nil)
}

func (chain *FullBlockChain) verifyCastingBlock(bh types.BlockHeader, txs []*types.Transaction) ([]common.Hash, int8, *core.AccountDB, vtypes.Receipts) {
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
			BlockPv:           bh.ProveValue,
		}
		go RequestTransaction(*m, castorId.String())
		return missing, 1, nil, nil
	}

	txtree := calcTxTree(transactions)

	if !bytes.Equal(txtree.Bytes(), bh.TxTree.Bytes()) {
		Logger.Debugf("[BlockChain]fail to verify txtree, hash1:%s hash2:%s", txtree.Hex(), bh.TxTree.Hex())
		return missing, -1, nil, nil
	}

	//执行交易
	Logger.Debugf("verifyCastingBlock NewAccountDB hash:%s, height:%d", preBlock.StateTree.Hex(), bh.Height)
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

	Logger.Infof("verifyCastingBlock height:%d StateTree Hash:%s", b.Header.Height, b.Header.StateTree.Hex())
	statehash, receipts, err := chain.executor.Execute(state, b, bh.Height, "fullverify")
	if common.ToHex(statehash.Bytes()) != common.ToHex(bh.StateTree.Bytes()) {
		Logger.Debugf("[BlockChain]fail to verify statetree, hash1:%x hash2:%x", statehash.Bytes(), b.Header.StateTree.Bytes())
		return nil, -1, nil, nil
	}
	receiptsTree := calcReceiptsTree(receipts).Bytes()
	if common.ToHex(receiptsTree) != common.ToHex(b.Header.ReceiptTree.Bytes()) {
		Logger.Debugf("[BlockChain]fail to verify receipt, hash1:%s hash2:%s", receiptsTree, b.Header.ReceiptTree.Bytes())
		return nil, 1, nil, nil
	}

	chain.verifiedBlocks.Add(bh.Hash, &castingBlock{
		state:    state,
		receipts: receipts,
	})
	//return nil, 0, state, receipts
	return nil, 0, state, receipts
}

//铸块成功，上链
//返回值: 0,上链成功
//       -1，验证失败
//        1, 丢弃该块(链上已存在该块或链上存在QN值更大的相同高度块)
//        2，未上链(异步进行分叉调整)
func (chain *FullBlockChain) AddBlockOnChain(b *types.Block) int8 {
	if b == nil {
		return -1
	}
	chain.lock.Lock("AddBlockOnChain")
	defer chain.lock.Unlock("AddBlockOnChain")
	//defer network.Logger.Debugf("add on chain block %d-%d,cast+verify+io+onchain cost%v", b.Header.Height, b.Header.ProveValue, time.Since(b.Header.CurTime))

	return chain.addBlockOnChain(b)
}

func (chain *FullBlockChain) addBlockOnChain(b *types.Block) int8 {
	var (
		state    *core.AccountDB
		receipts vtypes.Receipts
		status   int8
	)

	// 自己铸块的时候，会将块临时存放到blockCache里
	// 当组内其他成员验证通过后，自己上链就无需验证、执行交易，直接上链即可
	cache, _ := chain.verifiedBlocks.Get(b.Header.Hash)
	//if false {
	if cache != nil {
		status = 0
		state = cache.(*castingBlock).state
		receipts = cache.(*castingBlock).receipts
	} else {
		// 验证块是否有问题
		_, status, state, receipts = chain.verifyCastingBlock(*b.Header, b.Transactions)
		if status != 0 {
			Logger.Errorf("[BlockChain]fail to VerifyCastingBlock, reason code:%d \n", status)
			return -1
		}
	}
	trace := &TraceHeader{Hash: b.Header.Hash, PreHash: b.Header.PreHash, Value: chain.consensusHelper.VRFProve2Value(b.Header.ProveValue), TotalQn: b.Header.TotalQN, Height: b.Header.Height}
	TraceChainImpl.AddTrace(trace)

	topBlock := chain.latestBlock
	Logger.Debugf("coming block:hash=%v, preH=%v, height=%v,totalQn:%d", b.Header.Hash.Hex(), b.Header.PreHash.Hex(), b.Header.Height, b.Header.TotalQN)
	Logger.Debugf("Local tophash=%v, topPreH=%v, height=%v,totalQn:%d", topBlock.Hash.Hex(), topBlock.PreHash.Hex(), b.Header.Height, topBlock.TotalQN)

	if b.Header.PreHash == topBlock.Hash {
		result, headerByte := chain.insertBlock(b, state, receipts)
		if result == 0 {
			chain.successOnChainCallBack(b, headerByte)
		}
		return result
	}

	if b.Header.TotalQN < topBlock.TotalQN || b.Header.Hash == topBlock.Hash || chain.queryBlockHeaderByHash(b.Header.Hash) != nil {
		return 1
	}
	result, headerJson := chain.processFork(topBlock, b, state, receipts)
	if result == 0 {
		chain.successOnChainCallBack(b, headerJson)
	}
	return result
}

func (chain *FullBlockChain) insertBlock(remoteBlock *types.Block, state *core.AccountDB, receipts vtypes.Receipts) (int8, []byte) {
	result, headerByte := chain.saveBlock(remoteBlock)
	if result != 0 {
		return -1, headerByte
	}
	root, _ := state.Commit(true)
	triedb := chain.stateCache.TrieDB()
	triedb.Commit(root, false)
	if chain.updateLastBlock(state, remoteBlock.Header, headerByte) == -1 {
		return -1, headerByte
	}
	chain.transactionPool.Remove(remoteBlock.Header.Hash, remoteBlock.Header.Transactions)
	chain.transactionPool.MarkExecuted(receipts, remoteBlock.Transactions)
	chain.successOnChainCallBack(remoteBlock, headerByte)
	return 0, headerByte
}

func (chain *FullBlockChain) processFork(localTopBlock *types.BlockHeader, remoteBlock *types.Block, state *core.AccountDB, receipts vtypes.Receipts) (int8,[]byte) {
	replace, commonAncestor, err := TraceChainImpl.FindCommonAncestor(localTopBlock.Hash.Bytes(), remoteBlock.Header.Hash.Bytes())
	Logger.Debugf("TraceChain height=%d, hash=%v, replace=%t, err=%v", remoteBlock.Header.Height, commonAncestor.Hash.Hex(), replace, err)
	if err == ErrMissingTrace {
		//分叉分支缺结点
		panic("Local trace chain miss block!!")
	}

	if remoteBlock.Header.TotalQN > localTopBlock.TotalQN || replace {
		if remoteBlock.Header.PreHash == commonAncestor.Hash {
			Logger.Debugf("TraceChain Hash:%s Replace Latest:%s", remoteBlock.Header.Hash.Hex(), chain.latestBlock.Hash.Hex())
			chain.Remove(chain.latestBlock)
			result, headerJson := chain.insertBlock(remoteBlock, state, receipts)
			return result, headerJson
		}

		for i := commonAncestor.Height + 1; i <= chain.Height(); i++ {
			header := chain.queryBlockHeaderByHeight(i, true)
			chain.Remove(header)
		}
		Logger.Debugf("processFork trigger by height=%d hash=%s, remove from height=%d to height=%d",remoteBlock.Header.Height,
			remoteBlock.Header.Hash.Hex(),commonAncestor.Height + 1, chain.Height())
		chain.isAdujsting = true
		BlockSyncer.Sync()
		return 2,nil
	}
	return 1,nil
}

func (chain *FullBlockChain) successOnChainCallBack(remoteBlock *types.Block, headerJson []byte) {
	Logger.Debugf("ON chain succ! height=%d,hash=%s", remoteBlock.Header.Height, remoteBlock.Header.Hash.Hex())
	notify.BUS.Publish(notify.BlockAddSucc, &notify.BlockMessage{Block: *remoteBlock,})
	//GroupChainImpl.RemoveDismissGroupFromCache(b.Header.Height)
	BlockSyncer.Sync()
}

func (chain *FullBlockChain) updateLastBlock(state *core.AccountDB, header *types.BlockHeader, headerJson []byte) int8 {
	err := chain.blockHeight.Put([]byte(BLOCK_STATUS_KEY), headerJson)
	if err != nil {
		Logger.Errorf("[block]fail to put current, error:%s \n", err)
		return -1
	}
	chain.latestStateDB = state
	chain.latestBlock = header
	Logger.Debugf("blockchain update latestStateDB:%s height:%d", header.StateTree.Hex(), header.Height)
	return 0
}


//根据指定哈希查询块
func (chain *FullBlockChain) QueryBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	return chain.queryBlockHeaderByHash(hash)
}

func (chain *FullBlockChain) QueryBlockBody(blockHash common.Hash) []*types.Transaction {
	block := chain.QueryBlockByHash(blockHash)
	if nil == block {
		return nil
	}
	return block.Transactions
}

func (chain *FullBlockChain) queryBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	block := chain.QueryBlockByHash(hash)
	if nil == block {
		return nil
	}
	return block.Header
}

func (chain *FullBlockChain) QueryBlockByHash(hash common.Hash) *types.Block {
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

func (chain *FullBlockChain) QueryBlock(height uint64) *types.Block {
	chain.lock.RLock("QueryBlock")
	defer chain.lock.RUnlock("QueryBlock")

	var b *types.Block
	for i := height + 1; i <= chain.Height(); i++ {
		bh := chain.queryBlockHeaderByHeight(i, true)
		if nil == bh {
			continue
		}
		b = chain.QueryBlockByHash(bh.Hash)
		if nil == b {
			continue
		}
		break
	}
	return b
}

// 保存block到ldb
// todo:错误回滚
//result code:
// -1 保存失败
// 0 保存成功
func (chain *FullBlockChain) saveBlock(b *types.Block) (int8, []byte) {
	// 根据hash存block
	blockJson, err := types.MarshalBlock(b)
	if err != nil {
		log.Printf("[block]fail to json Marshal, error:%s \n", err)
		return -1, nil
	}
	err = chain.blocks.Put(b.Header.Hash.Bytes(), blockJson)
	if err != nil {
		log.Printf("[block]fail to put key:hash value:block, error:%s \n", err)
		return -1, nil
	}

	// 根据height存blockheader
	headerJson, err := types.MarshalBlockHeader(b.Header)
	if err != nil {

		log.Printf("[block]fail to json Marshal header, error:%s \n", err)
		return -1, nil
	}

	err = chain.blockHeight.Put(generateHeightKey(b.Header.Height), headerJson)
	if err != nil {
		log.Printf("[block]fail to put key:height value:headerjson, error:%s \n", err)
		return -1, nil
	}

	// 持久化保存最新块信息

	chain.topBlocks.Add(b.Header.Height, b.Header)

	return 0, headerJson
}

// 删除块
func (chain *FullBlockChain) Remove(header *types.BlockHeader) {
	hash := header.Hash
	block := chain.QueryBlockByHash(hash)
	chain.blocks.Delete(hash.Bytes())
	chain.blockHeight.Delete(generateHeightKey(header.Height))

	// 删除块的交易，返回transactionpool
	if nil == block {
		return
	}
	txs := block.Transactions
	if 0 == len(txs) {
		return
	}
	chain.transactionPool.UnMarkExecuted(txs)
}

//清除链所有数据
func (chain *FullBlockChain) Clear() error {
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
		block := GenesisBlock(state, chain.stateCache.TrieDB(), chain.consensusHelper.GenerateGenesisInfo())

		_, headerJson := chain.saveBlock(block)
		chain.updateLastBlock(state, block.Header, headerJson)
	}

	chain.init = true

	chain.transactionPool.Clear()
	return err
}

// 删除块
func (chain *FullBlockChain) remove(header *types.BlockHeader) {
	hash := header.Hash
	block := chain.QueryBlockByHash(hash)
	chain.blocks.Delete(hash.Bytes())
	chain.blockHeight.Delete(generateHeightKey(header.Height))

	// 删除块的交易，返回transactionpool
	if nil == block {
		return
	}
	txs := block.Transactions
	if 0 == len(txs) {
		return
	}
	chain.transactionPool.UnMarkExecuted(txs)
}

func (chain *FullBlockChain) GetTrieNodesByExecuteTransactions(header *types.BlockHeader, transactions []*types.Transaction, addresses []common.Address) *[]types.StateNode {
	Logger.Debugf("GetTrieNodesByExecuteTransactions height:%d,stateTree:%v", header.Height, header.StateTree)
	var nodesOnBranch = make(map[string]*[]byte)
	state, err := core.NewAccountDBWithMap(header.StateTree, chain.stateCache, nodesOnBranch)
	if err != nil {
		Logger.Infof("GetTrieNodesByExecuteTransactions error,height=%d,hash=%v \n", header.Height, header.StateTree)
		return nil
	}
	chain.executor.GetBranches(state, transactions, addresses, nodesOnBranch)

	data := []types.StateNode{}
	for key, value := range nodesOnBranch {
		data = append(data, types.StateNode{Key: ([]byte)(key), Value: *value})
	}
	return &data
}

func (chain *FullBlockChain) InsertStateNode(nodes *[]types.StateNode) {
	panic("Not support!")
}

func (chain *FullBlockChain) GetCastingBlock(hash common.Hash) *types.Block {
	v, ok := chain.castedBlock.Get(hash)
	if !ok {
		return nil
	}
	return v.(*types.Block)
}

func Clear() {
	path := datasource.DEFAULT_FILE
	if nil != common.GlobalConf {
		path = common.GlobalConf.GetString(CONFIG_SEC, "database", datasource.DEFAULT_FILE)
	}
	os.RemoveAll(path)

}

func (chain *FullBlockChain) SetVoteProcessor(processor VoteProcessor) {
	chain.lock.Lock("SetVoteProcessor")
	defer chain.lock.Unlock("SetVoteProcessor")

	chain.voteProcessor = processor
}

func (chain *FullBlockChain) GetAccountDBByHash(hash common.Hash) (vm.AccountDB, error) {
	header := chain.QueryBlockHeaderByHash(hash)
	return core.NewAccountDB(header.StateTree, chain.stateCache)
}
