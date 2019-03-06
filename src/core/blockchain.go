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
	"fmt"
	"github.com/hashicorp/golang-lru"
	"bytes"
	"middleware"
	"middleware/types"
	"taslog"
	"math/big"
	"middleware/notify"
	"os"
	"storage/account"
	"storage/tasdb"
	"storage/vm"
	"time"
	"utility"
)

const (
	BLOCK_STATUS_KEY = "bcurrent"

	CONFIG_SEC = "chain"

	addBlockMark = "addBlockMark"

	removeBlockMark = "removeBlockMark"
)

var BlockChainImpl BlockChain

var Logger taslog.Logger

var consensusLogger taslog.Logger

type BlockChainConfig struct {
	block string

	blockHeight string

	state string

	bonus string

	heavy string

	light string

	check string
}

type FullBlockChain struct {
	prototypeChain
	config       *BlockChainConfig
	castedBlock  *lru.Cache
	bonusManager *BonusManager
}

type castingBlock struct {
	state    *account.AccountDB
	receipts types.Receipts
}

func getBlockChainConfig() *BlockChainConfig {
	defaultConfig := &BlockChainConfig{
		block: "block",

		blockHeight: "height",

		state: "state",

		bonus: "bonus",

		light: "light",

		heavy: "heavy",

		check: "check",
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

		check: common.GlobalConf.GetString(CONFIG_SEC, "check", defaultConfig.check),
	}

}

func initBlockChain(helper types.ConsensusHelper) error {
	Logger = taslog.GetLoggerByIndex(taslog.CoreLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	consensusLogger = taslog.GetLoggerByIndex(taslog.ConsensusLogConfig, common.GlobalConf.GetString("instance", "index", ""))
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
	chain.futureBlocks, err = lru.New(10)
	if err != nil {
		return err
	}
	chain.verifiedBlocks, err = lru.New(10)
	if err != nil {
		return err
	}
	chain.topBlocks, _ = lru.New(10)
	if err != nil {
		return err
	}
	chain.castedBlock, err = lru.New(10)
	if err != nil {
		return err
	}

	chain.verifiedBodyCache, _ = lru.New(50)

	chain.blocks, err = tasdb.NewDatabase(chain.config.block)
	if err != nil {
		//todo: 日志
		Logger.Debugf("Init block chain error! Error:%s", err.Error())
		return err
	}

	chain.blockHeight, err = tasdb.NewDatabase(chain.config.blockHeight)
	if err != nil {
		Logger.Debugf("Init block chain error! Error:%s", err.Error())
		//todo: 日志
		return err
	}

	chain.statedb, err = tasdb.NewDatabase(chain.config.state)
	if err != nil {
		Logger.Debugf("Init block chain error! Error:%s", err.Error())
		//todo: 日志
		return err
	}

	chain.checkdb, err = tasdb.NewDatabase(chain.config.state)
	if err != nil {
		Logger.Debugf("Init block chain error! Error:%s", err.Error())
		//todo: 日志
		return err
	}

	chain.bonusManager = newBonusManager()
	chain.stateCache = account.NewDatabase(chain.statedb)

	chain.executor = NewTVMExecutor(chain)
	initMinerManager()
	// 恢复链状态 height,latestBlock
	// todo:特殊的key保存最新的状态，当前写到了ldb，有性能损耗
	chain.latestBlock = chain.queryBlockHeaderByHeight([]byte(BLOCK_STATUS_KEY), false)
	if nil != chain.latestBlock {
		chain.ensureChainConsistency()
		if !chain.versionValidate() {
			fmt.Println("Illegal data version! Please delete the directory d0 and restart the program!")
			os.Exit(0)
		}
		chain.buildCache(10, chain.topBlocks)
		Logger.Debugf("initBlockChain chain.latestBlock.StateTree  Hash:%s", chain.latestBlock.StateTree.Hex())
		state, err := account.NewAccountDB(common.BytesToHash(chain.latestBlock.StateTree.Bytes()), chain.stateCache)
		if nil == err {
			chain.latestStateDB = state
		} else {
			panic("initBlockChain NewAccountDB fail:" + err.Error())
		}
	} else {
		chain.insertGenesisBlock()
	}

	chain.forkProcessor = initforkProcessor()
	BlockChainImpl = chain
	return nil
}

//构建一个铸块（组内当前铸块人同步操作）
func (chain *FullBlockChain) CastBlock(height uint64, proveValue *big.Int, proveRoot common.Hash, qn uint64, castor []byte, groupid []byte) *types.Block {
	chain.lock.Lock("CastBlock")
	defer chain.lock.Unlock("CastBlock")
	latestBlock := chain.QueryTopBlock()
	if latestBlock != nil && height <= latestBlock.Height {
		Logger.Info("[BlockChain] fail to cast block: height problem. height:%d, latest:%d", height, latestBlock.Height)
		return nil
	}

	block := new(types.Block)

	block.Transactions = chain.transactionPool.PackForCast()
	block.Header = &types.BlockHeader{
		CurTime:    time.Now(), //todo:时区问题
		Height:     height,
		ProveValue: proveValue,
		Castor:     castor,
		GroupId:    groupid,
		TotalQN:    latestBlock.TotalQN + qn, //todo:latestBlock != nil?
		StateTree:  common.BytesToHash(latestBlock.StateTree.Bytes()),
		ProveRoot:  proveRoot,
	}

	if latestBlock != nil {
		block.Header.PreHash = latestBlock.Hash
		block.Header.PreTime = latestBlock.CurTime
	}

	preRoot := common.BytesToHash(latestBlock.StateTree.Bytes())

	state, err := account.NewAccountDB(preRoot, chain.stateCache)
	if err != nil {
		var buffer bytes.Buffer
		buffer.WriteString("fail to new statedb, lateset height: ")
		buffer.WriteString(fmt.Sprintf("%d", latestBlock.Height))
		buffer.WriteString(", block height: ")
		buffer.WriteString(fmt.Sprintf("%d error:", block.Header.Height))
		buffer.WriteString(fmt.Sprint(err))
		panic(buffer.String())

	}

	statehash, evictedTxs, transactions, receipts, err, _ := chain.executor.Execute(state, block, height, "casting")

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

	transactionHashes := make([]common.Hash, len(transactions))

	block.Transactions = transactions
	for i, transaction := range transactions {
		transactionHashes[i] = transaction.Hash
	}
	block.Header.Transactions = transactionHashes
	block.Header.TxTree = calcTxTree(block.Transactions)
	block.Header.EvictedTxs = evictedTxs

	block.Header.StateTree = common.BytesToHash(statehash.Bytes())
	block.Header.ReceiptTree = calcReceiptsTree(receipts)
	block.Header.Hash = block.Header.GenHash()
	defer Logger.Infof("casting block %d,hash:%v,qn:%d,tx:%d,TxTree:%v,proValue:%v,stateTree:%s,prestatetree:%s",
		height, block.Header.Hash.String(), block.Header.TotalQN, len(block.Transactions), block.Header.TxTree.Hex(),
		chain.consensusHelper.VRFProve2Value(block.Header.ProveValue), block.Header.StateTree.String(), preRoot.String())
	//自己铸的块 自己不需要验证
	chain.verifiedBlocks.Add(block.Header.Hash, &castingBlock{
		state:    state,
		receipts: receipts,
	})
	chain.castedBlock.Add(block.Header.Hash, block)

	if len(block.Transactions) != 0 {
		chain.verifiedBlocks.Add(block.Header.Hash, block.Transactions)
	}
	return block
}

//验证一个铸块（如本地缺少交易，则异步网络请求该交易）
//返回值:
// 0, 验证通过
// -1，验证失败
// 1 无法验证（缺少交易，已异步向网络模块请求）
// 2 无法验证（前一块在链上不存存在）
func (chain *FullBlockChain) VerifyBlock(bh types.BlockHeader) ([]common.Hash, int8) {
	chain.lock.Lock("VerifyCastingBlock")
	defer chain.lock.Unlock("VerifyCastingBlock")

	return chain.verifyBlock(bh, nil)
}

func (chain *FullBlockChain) verifyBlock(bh types.BlockHeader, txs []*types.Transaction) ([]common.Hash, int8) {
	Logger.Infof("verifyBlock hash:%v,height:%d,totalQn:%d,preHash:%v,len header tx:%d,len tx:%d", bh.Hash.String(), bh.Height, bh.TotalQN, bh.PreHash.String(), len(bh.Transactions), len(txs))

	if bh.Hash != bh.GenHash() {
		Logger.Debugf("Validate block hash error!")
		return nil, -1
	}

	if !chain.hasPreBlock(bh) {
		if txs != nil {
			chain.futureBlocks.Add(bh.PreHash, &types.Block{Header: &bh, Transactions: txs})
		}
		return nil, 2
	}

	miss, missingTx, transactions := chain.missTransaction(bh, txs)
	if miss {
		return missingTx, 1
	}

	Logger.Debugf("validateTxRoot,tx tree root:%v,len txs:%d,miss len:%d", bh.TxTree.Hex(), len(transactions), len(missingTx))
	if !chain.validateTxRoot(bh.TxTree, transactions) {
		return nil, -1
	}

	block := types.Block{Header: &bh, Transactions: transactions}
	executeTxResult, _, _ := chain.executeTransaction(&block)
	if !executeTxResult {
		return nil, -1
	}
	if len(block.Transactions) != 0 {
		chain.verifiedBlocks.Add(block.Header.Hash, block.Transactions)
	}
	return nil, 0
}

func (chain *FullBlockChain) hasPreBlock(bh types.BlockHeader) bool {
	pre := chain.queryBlockHeaderByHash(bh.PreHash)
	if pre == nil {
		return false
	}
	return true
}

//铸块成功，上链
//返回值: 0,上链成功
//       -1，验证失败
//        1, 丢弃该块(链上已存在该块）
//        2,丢弃该块（链上存在QN值更大的相同高度块)
//        3,分叉调整
func (chain *FullBlockChain) AddBlockOnChain(source string, b *types.Block, situation types.AddBlockOnChainSituation) types.AddBlockResult {
	if validateCode, result := chain.validateBlock(source, b); !result {
		return validateCode
	}
	chain.lock.Lock("AddBlockOnChain")
	defer chain.lock.Unlock("AddBlockOnChain")
	return chain.addBlockOnChain(source, b, situation)
}

func (chain *FullBlockChain) validateBlock(source string, b *types.Block) (types.AddBlockResult, bool) {
	if b == nil {
		return types.AddBlockFailed, false
	}

	if !chain.hasPreBlock(*b.Header) {
		Logger.Debugf("coming block %s,%d has no pre on local chain.Forking...", b.Header.Hash.String(), b.Header.Height)
		chain.futureBlocks.Add(b.Header.PreHash, b)
		go chain.forkProcessor.requestChainPieceInfo(source, chain.latestBlock.Height)
		return types.Forking, false
	}

	if chain.queryBlockHeaderByHash(b.Header.Hash) != nil {
		return types.BlockExisted, false
	}

	if check, err := chain.GetConsensusHelper().CheckProveRoot(b.Header); !check {
		Logger.Errorf("checkProveRoot fail, err=%v", err.Error())
		return types.AddBlockFailed, false
	}

	groupValidateResult, err := chain.validateGroupSig(b.Header)
	if !groupValidateResult {
		if err == common.ErrSelectGroupNil || err == common.ErrSelectGroupInequal {
			Logger.Infof("Add block on chain failed: depend on group!")
		} else {
			Logger.Errorf("Fail to validate group sig!Err:%s", err.Error())
		}
		return types.AddBlockFailed, false
	}
	return types.ValidateBlockOk, true
}

func (chain *FullBlockChain) addBlockOnChain(source string, b *types.Block, situation types.AddBlockOnChainSituation) types.AddBlockResult {
	topBlock := chain.latestBlock
	Logger.Debugf("coming block:hash=%v, preH=%v, height=%v,totalQn:%d", b.Header.Hash.Hex(), b.Header.PreHash.Hex(), b.Header.Height, b.Header.TotalQN)
	Logger.Debugf("Local tophash=%v, topPreHash=%v, height=%v,totalQn:%d", topBlock.Hash.Hex(), topBlock.PreHash.Hex(), topBlock.Height, topBlock.TotalQN)

	if _, verifyResult := chain.verifyBlock(*b.Header, b.Transactions); verifyResult != 0 {
		Logger.Errorf("Fail to VerifyCastingBlock, reason code:%d \n", verifyResult)
		if verifyResult == 2 {
			Logger.Debugf("coming block  has no pre on local chain.Forking...", )
			go chain.forkProcessor.requestChainPieceInfo(source, chain.latestBlock.Height)
		}
		return types.AddBlockFailed
	}

	if b.Header.PreHash == topBlock.Hash {
		result, _ := chain.insertBlock(b)
		return result
	}
	if b.Header.Hash == topBlock.Hash || chain.queryBlockHeaderByHash(b.Header.Hash) != nil {
		return types.BlockExisted
	}

	if b.Header.TotalQN < topBlock.TotalQN {
		if situation == types.Sync {
			go chain.forkProcessor.requestChainPieceInfo(source, chain.latestBlock.Height)
		}
		return types.BlockTotalQnLessThanLocal
	}
	commonAncestor := chain.queryBlockHeaderByHash(b.Header.PreHash)
	Logger.Debugf("commonAncestor hash:%s height:%d", commonAncestor.Hash.Hex(), commonAncestor.Height)
	if b.Header.TotalQN > topBlock.TotalQN {
		chain.removeFromCommonAncestor(commonAncestor)
		return chain.addBlockOnChain(source, b, situation)
	}
	if b.Header.TotalQN == topBlock.TotalQN {
		if chain.compareValue(commonAncestor, b.Header) {
			if situation == types.Sync {
				go chain.forkProcessor.requestChainPieceInfo(source, chain.latestBlock.Height)
			}
			return types.BlockTotalQnLessThanLocal
		}
		chain.removeFromCommonAncestor(commonAncestor)
		return chain.addBlockOnChain(source, b, situation)
	}
	go chain.forkProcessor.requestChainPieceInfo(source, chain.latestBlock.Height)
	return types.Forking
}

func (chain *FullBlockChain) executeTransaction(block *types.Block) (bool, *account.AccountDB, types.Receipts) {
	preBlock := chain.queryBlockHeaderByHash(block.Header.PreHash)
	if preBlock == nil {
		panic("Pre block nil !!")
	}
	preRoot := common.BytesToHash(preBlock.StateTree.Bytes())
	if len(block.Transactions) > 0 {
		Logger.Debugf("NewAccountDB height:%d StateTree:%s preHash:%s preRoot:%s", block.Header.Height, block.Header.StateTree.Hex(), preBlock.Hash.Hex(), preRoot.Hex())
	}
	state, err := account.NewAccountDB(preRoot, chain.stateCache)
	if err != nil {
		Logger.Errorf("Fail to new statedb, error:%s", err)
		return false, state, nil
	}

	statehash, _, _, receipts, err, _ := chain.executor.Execute(state, block, block.Header.Height, "fullverify")
	if common.ToHex(statehash.Bytes()) != common.ToHex(block.Header.StateTree.Bytes()) {
		Logger.Errorf("Fail to verify statetree, hash1:%x hash2:%x", statehash.Bytes(), block.Header.StateTree.Bytes())
		return false, state, receipts
	}
	receiptsTree := calcReceiptsTree(receipts).Bytes()
	if common.ToHex(receiptsTree) != common.ToHex(block.Header.ReceiptTree.Bytes()) {
		Logger.Errorf("fail to verify receipt, hash1:%s hash2:%s", common.ToHex(receiptsTree), common.ToHex(block.Header.ReceiptTree.Bytes()))
		return false, state, receipts
	}

	chain.verifiedBlocks.Add(block.Header.Hash, &castingBlock{state: state, receipts: receipts,})
	return true, state, receipts
}

func (chain *FullBlockChain) insertBlock(remoteBlock *types.Block) (types.AddBlockResult, []byte) {
	Logger.Debugf("Insert block hash:%s,height:%d", remoteBlock.Header.Hash.Hex(), remoteBlock.Header.Height)
	blockByte, err := types.MarshalBlock(remoteBlock)
	if err != nil {
		Logger.Errorf("Fail to json Marshal, error:%s", err.Error())
		return types.AddBlockFailed, nil
	}

	chain.markAddBlock(blockByte)
	if !chain.saveBlockByHash(remoteBlock.Header.Hash, blockByte) {
		return types.AddBlockFailed, nil
	}

	headerByte, err := types.MarshalBlockHeader(remoteBlock.Header)
	if err != nil {
		Logger.Errorf("Fail to json Marshal header, error:%s", err.Error())
		return types.AddBlockFailed, nil
	}
	if !chain.saveBlockByHeight(remoteBlock.Header.Height, headerByte) {
		return types.AddBlockFailed, nil
	}

	saveStateResult, accountDB, receipts := chain.saveBlockState(remoteBlock)
	if !saveStateResult {
		return types.AddBlockFailed, nil
	}

	if !chain.updateLastBlock(accountDB, remoteBlock.Header, headerByte) {
		return types.AddBlockFailed, headerByte
	}

	chain.updateCheckValue(remoteBlock)

	chain.updateTxPool(remoteBlock, receipts)
	chain.topBlocks.Add(remoteBlock.Header.Height, remoteBlock.Header)

	chain.eraseAddBlockMark()
	chain.successOnChainCallBack(remoteBlock, headerByte)
	return types.AddBlockSucc, headerByte
}

func (chain *FullBlockChain) saveBlockByHash(hash common.Hash, blockByte []byte) bool {
	err := chain.blocks.Put(hash.Bytes(), blockByte)
	if err != nil {
		Logger.Errorf("Fail to put block hash %s  error:%s", hash.String(), err.Error())
		return false
	}
	return true
}

func (chain *FullBlockChain) saveBlockByHeight(height uint64, headerByte []byte) bool {
	err := chain.blockHeight.Put(utility.UInt64ToByte(height), headerByte)
	if err != nil {
		Logger.Errorf("Fail to put block height:%d  error:%s", height, err.Error())
		return false
	}
	return true
}

func (chain *FullBlockChain) saveBlockState(b *types.Block) (bool, *account.AccountDB, types.Receipts) {
	var state *account.AccountDB
	var receipts types.Receipts
	if value, exit := chain.verifiedBlocks.Get(b.Header.Hash); exit {
		b := value.(*castingBlock)
		state = b.state
		receipts = b.receipts
	} else {
		var executeTxResult bool
		executeTxResult, state, receipts = chain.executeTransaction(b)
		if !executeTxResult {
			Logger.Errorf("Fail to execute txs!")
			return false, state, receipts
		}
	}
	root, err := state.Commit(true)
	if err != nil {
		Logger.Errorf("State commit error:%s", err.Error())
		return false, state, receipts
	}

	triedb := chain.stateCache.TrieDB()
	err = triedb.Commit(root, false)
	if err != nil {
		Logger.Errorf("Trie commit error:%s", err.Error())
		return false, state, receipts
	}
	return true, state, receipts
}

func (chain *FullBlockChain) updateLastBlock(state *account.AccountDB, header *types.BlockHeader, headerJson []byte) bool {
	err := chain.blockHeight.Put([]byte(BLOCK_STATUS_KEY), headerJson)
	if err != nil {
		Logger.Errorf("Fail to put %s, error:%s", BLOCK_STATUS_KEY, err.Error())
		return false
	}
	chain.latestStateDB = state
	chain.latestBlock = header
	Logger.Debugf("Update latestStateDB:%s height:%d", header.StateTree.Hex(), header.Height)
	return true
}

func (chain *FullBlockChain) updateCheckValue(block *types.Block) {
	verifyHash := chain.consensusHelper.VerifyHash(block)
	chain.PutCheckValue(block.Header.Height, verifyHash.Bytes())
}

func (chain *FullBlockChain) updateTxPool(block *types.Block, receipts types.Receipts) {
	chain.transactionPool.MarkExecuted(receipts, block.Transactions)
}

func (chain *FullBlockChain) successOnChainCallBack(remoteBlock *types.Block, headerJson []byte) {
	Logger.Infof("ON chain succ! height=%d,hash=%s", remoteBlock.Header.Height, remoteBlock.Header.Hash.Hex())
	notify.BUS.Publish(notify.BlockAddSucc, &notify.BlockOnChainSuccMessage{Block: *remoteBlock,})
	if value, _ := chain.futureBlocks.Get(remoteBlock.Header.Hash); value != nil {
		block := value.(*types.Block)
		Logger.Debugf("Get block from future blocks,hash:%s,height:%d", block.Header.Hash.String(), block.Header.Height)
		//todo 这里为了避免死锁只能调用这个方法，但是没办法调用CheckProveRoot全量账本验证了
		chain.addBlockOnChain("", block, types.FutureBlockCache)
		return
	}
	//GroupChainImpl.RemoveDismissGroupFromCache(b.Header.Height)
	if BlockSyncer != nil {
		topBlockInfo := TopBlockInfo{Hash: chain.latestBlock.Hash, TotalQn: chain.latestBlock.TotalQN, Height: chain.latestBlock.Height, PreHash: chain.latestBlock.PreHash}
		go BlockSyncer.sendTopBlockInfoToNeighbor(topBlockInfo)
	}
}

//根据指定哈希查询块
func (chain *FullBlockChain) QueryBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	chain.lock.RLock("QueryBlockHeaderByHash")
	defer chain.lock.RUnlock("QueryBlockHeaderByHash")
	return chain.queryBlockHeaderByHash(hash)
}

func (chain *FullBlockChain) QueryBlockByHash(hash common.Hash) *types.Block {
	chain.lock.RLock("QueryBlockByHash")
	defer chain.lock.RUnlock("QueryBlockByHash")
	return chain.queryBlockByHash(hash)
}

func (chain *FullBlockChain) QueryBlock(height uint64) *types.Block {
	chain.lock.RLock("QueryBlock")
	defer chain.lock.RUnlock("QueryBlock")

	var b *types.Block
	for i := height; i <= chain.Height(); i++ {
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
	return b
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
	chain.checkdb.Close()

	os.RemoveAll(tasdb.DEFAULT_FILE)

	chain.statedb, err = tasdb.NewDatabase(chain.config.state)
	if err != nil {
		//todo: 日志
		return err
	}

	chain.stateCache = account.NewDatabase(chain.statedb)
	chain.executor = NewTVMExecutor(chain)

	chain.insertGenesisBlock()
	chain.init = true
	chain.transactionPool.Clear()
	return err
}

func (chain *FullBlockChain) GetCastingBlock(hash common.Hash) *types.Block {
	v, ok := chain.castedBlock.Get(hash)
	if !ok {
		return nil
	}
	return v.(*types.Block)
}

func Clear() {
	path := tasdb.DEFAULT_FILE
	if nil != common.GlobalConf {
		path = common.GlobalConf.GetString(CONFIG_SEC, "database", tasdb.DEFAULT_FILE)
	}
	os.RemoveAll(path)
}

func (chain *FullBlockChain) GetAccountDBByHash(hash common.Hash) (vm.AccountDB, error) {
	header := chain.QueryBlockHeaderByHash(hash)
	return account.NewAccountDB(header.StateTree, chain.stateCache)
}

func (chain *FullBlockChain) GetAccountDBByHeight(height uint64) (vm.AccountDB, error) {
	var header *types.BlockHeader
	h := height
	for {
		header = chain.queryBlockHeaderByHeight(h, true)
		if header != nil || h == 0 {
			break
		}
		h--
	}
	if header == nil {
		return nil, fmt.Errorf("no data at height %v-%v", h, height)
	}
	return account.NewAccountDB(header.StateTree, chain.stateCache)
}

func (chain *FullBlockChain) versionValidate() bool {
	genesisHeader := chain.queryBlockHeaderByHeight(uint64(0), true)
	if genesisHeader == nil {
		return false
	}
	version := genesisHeader.Nonce
	if version != ChainDataVersion {
		return false
	}
	return true
}

func (chain *FullBlockChain) insertGenesisBlock() {
	state, err := account.NewAccountDB(common.Hash{}, chain.stateCache)
	if nil == err {
		genesisBlock := GenesisBlock(state, chain.stateCache.TrieDB(), chain.consensusHelper.GenerateGenesisInfo())
		Logger.Debugf("GenesisBlock Hash:%s,StateTree:%s", genesisBlock.Header.Hash.String(), genesisBlock.Header.StateTree.Hex())
		blockByte, _ := types.MarshalBlock(genesisBlock)
		chain.saveBlockByHash(genesisBlock.Header.Hash, blockByte)

		headerByte, _ := types.MarshalBlockHeader(genesisBlock.Header)
		chain.saveBlockByHeight(genesisBlock.Header.Height, headerByte)

		chain.updateLastBlock(state, genesisBlock.Header, headerByte)
		chain.updateCheckValue(genesisBlock)
	} else {
		panic("Init block chain error:" + err.Error())
	}
}
