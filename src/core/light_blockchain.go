package core

import (
	"github.com/hashicorp/golang-lru"
	"middleware"
	"middleware/types"

	"taslog"
	"core/datasource"
	"storage/core"
	"common"
	"bytes"
	"consensus/groupsig"
	"middleware/notify"
	"network"
	"log"
	"fmt"
	vtypes "storage/core/types"
	"sync"
	"math/big"
	"storage/core/vm"
)

const (
	LIGHT_BLOCK_CACHE_SIZE       = 20
	LIGHT_BLOCKHEIGHT_CACHE_SIZE = 100
	LIGHT_BLOCKBODY_CACHE_SIZE   = 100
	LIGHT_LRU_SIZE               = 5000000
)

type LightChain struct {
	config      *LightChainConfig
	prototypeChain
	pending     map[uint64]*types.Block
	pendingLock sync.Mutex

	missingNodeState map[common.Hash]bool
	missingNodeLock  middleware.Loglock

	preBlockStateRoot     map[common.Hash]common.Hash
	preBlockStateRootLock middleware.Loglock
}

// 配置
type LightChainConfig struct {
	blockHeight string

	state string
}

func getLightChainConfig() *LightChainConfig {
	defaultConfig := DefaultLightChainConfig()
	if nil == common.GlobalConf {
		return defaultConfig
	}

	return &LightChainConfig{
		blockHeight: common.GlobalConf.GetString(CONFIG_SEC, "blockHeight", defaultConfig.blockHeight),

		state: common.GlobalConf.GetString(CONFIG_SEC, "state", defaultConfig.state),
	}
}

// 默认配置
func DefaultLightChainConfig() *LightChainConfig {
	return &LightChainConfig{
		blockHeight: "light_height",
		state:       "light_state",
	}
}

func initLightChain(helper types.ConsensusHelper) error {
	Logger = taslog.GetLoggerByName("core" + common.GlobalConf.GetString("instance", "index", ""))
	Logger.Debugf("in initLightChain")

	chain := &LightChain{
		config: getLightChainConfig(),
		prototypeChain: prototypeChain{
			transactionPool: NewTransactionPool(),
			latestBlock:     nil,

			lock:            middleware.NewLoglock("lightchain"),
			init:            true,
			isAdujsting:     false,
			isLightMiner:    true,
			consensusHelper: helper,
		},
		pending:               make(map[uint64]*types.Block),
		pendingLock:           sync.Mutex{},
		missingNodeState:      make(map[common.Hash]bool),
		missingNodeLock:       middleware.NewLoglock("lightchain"),
		preBlockStateRoot:     make(map[common.Hash]common.Hash),
		preBlockStateRootLock: middleware.NewLoglock("lightchain"),
	}

	var err error
	chain.blockCache, err = lru.New(LIGHT_BLOCK_CACHE_SIZE)
	chain.topBlocks, _ = lru.New(LIGHT_BLOCKHEIGHT_CACHE_SIZE)
	if err != nil {
		return err
	}
	chain.blocks, err = datasource.NewLRUMemDatabase(LIGHT_BLOCKBODY_CACHE_SIZE)
	if err != nil {
		Logger.Error("[LightChain initLightChain Error!Msg=%v]", err)
		return err
	}
	chain.blockHeight, err = datasource.NewDatabase(chain.config.blockHeight)
	if err != nil {
		Logger.Error("[LightChain initLightChain Error!Msg=%v]", err)
		return err
	}
	chain.statedb, err = datasource.NewLRUMemDatabase(LIGHT_LRU_SIZE)
	if err != nil {
		Logger.Error("[LightChain initLightChain Error!Msg=%v]", err)
		return err
	}

	chain.bonusManager = newBonusManager()
	chain.stateCache = core.NewLightDatabase(chain.statedb)
	chain.executor = NewTVMExecutor(chain)
	initMinerManager(chain)
	initTraceChain()
	// 恢复链状态 height,latestBlock
	// todo:特殊的key保存最新的状态，当前写到了ldb，有性能损耗
	chain.latestBlock = chain.queryBlockHeaderByHeight([]byte(BLOCK_STATUS_KEY), false)
	if nil != chain.latestBlock {
		chain.buildCache(LIGHT_BLOCKHEIGHT_CACHE_SIZE, chain.topBlocks)
		Logger.Infof("initLightChain chain.latestBlock.StateTree  Hash:%s", chain.latestBlock.StateTree.Hex())
	} else {
		//// 创始块
		state, err := core.NewAccountDB(common.Hash{}, chain.stateCache)
		if nil == err {
			block := GenesisBlock(state, chain.stateCache.TrieDB(), chain.consensusHelper.GenerateGenesisInfo())
			_, headerJson := chain.saveBlock(block)
			chain.updateLastBlock(state, block.Header, headerJson)
		}
	}

	BlockChainImpl = chain
	return nil
}

//构建一个铸块（组内当前铸块人同步操作）
func (chain *LightChain) CastBlock(height uint64, proveValue *big.Int, qn uint64, castor []byte, groupid []byte) *types.Block {
	//panic("Not support!")
	return nil
}

//验证一个铸块（如本地缺少交易，则异步网络请求该交易）
//返回值:
// 0, 验证通过
// -1，验证失败
// 1 无法验证（缺少交易，已异步向网络模块请求）
// 2 无法验证（前一块在链上不存存在）
//3 无法验证(缺少账户状态信息) 只有轻节点有
func (chain *LightChain) VerifyBlock(bh types.BlockHeader) ([]common.Hash, int8, *core.AccountDB, vtypes.Receipts) {
	chain.lock.Lock("VerifyCastingLightChainBlock")
	defer chain.lock.Unlock("VerifyCastingLightChainBlock")

	return chain.verifyCastingBlock(bh, nil)
}

func (chain *LightChain) verifyCastingBlock(bh types.BlockHeader, txs []*types.Transaction) ([]common.Hash, int8, *core.AccountDB, vtypes.Receipts) {

	Logger.Debugf("Verify block height:%d,preHash:%v,tx len:%d", bh.Height, bh.PreHash.String(), len(txs))
	hasPreBlock, preBlock := chain.hasPreBlock(bh, txs)
	if !hasPreBlock {
		return nil, 2, nil, nil
	}

	if miss, missingTx := chain.missTransaction(bh, txs); miss {
		return missingTx, 1, nil, nil
	}

	if !chain.validateTxRoot(bh.TxTree, txs) {
		return nil, -1, nil, nil
	}

	b := &types.Block{Header: &bh, Transactions: txs}
	var preBlockStateTree []byte
	if preBlock == nil {
		preBlockStateTree = chain.GetPreBlockStateRoot(b.Header.Hash).Bytes()
	} else {
		preBlockStateTree = preBlock.StateTree.Bytes()
	}
	preRoot := common.BytesToHash(preBlockStateTree)

	missingAccountTxs := chain.getMissingAccountTransactions(preRoot, b)
	if len(missingAccountTxs) != 0 {
		return nil, 3, nil, nil
	}

	state, err := core.NewAccountDB(preRoot, chain.stateCache)
	if err != nil {
		panic("Fail to new statedb, error:%s" + err.Error())
		return nil, -1, nil, nil
	}
	statehash, receipts, err := chain.executor.Execute(state, b, bh.Height, "lightverify")

	chain.FreeMissNodeState(bh.Hash)
	if common.ToHex(statehash.Bytes()) != common.ToHex(bh.StateTree.Bytes()) {
		Logger.Debugf("[LightChain]fail to verify statetree, hash1:%x hash2:%x", statehash.Bytes(), b.Header.StateTree.Bytes())
		return nil, -1, nil, nil
	}

	chain.blockCache.Add(bh.Hash, &castingBlock{state: state, receipts: receipts,})
	//return nil, 0, state, receipts
	return nil, 0, state, nil
}

func (chain *LightChain) hasPreBlock(bh types.BlockHeader, txs []*types.Transaction) (bool, *types.BlockHeader) {
	preHash := bh.PreHash
	preBlock := chain.queryBlockHeaderByHash(preHash)

	var isEmpty bool
	if chain.Height() == 0 {
		isEmpty = true
	} else {
		isEmpty = false
	}
	//轻节点初始化同步的时候不是从第一块开始同步，因此同步第一块的时候不验证preblock
	if !isEmpty && preBlock == nil {
		return false, preBlock
	}
	return true, preBlock
}

func (chain *LightChain) missTransaction(bh types.BlockHeader, txs []*types.Transaction) (bool, []common.Hash) {
	// 验证交易
	var missing []common.Hash
	if nil == txs {
		_, missing, _ = chain.transactionPool.GetTransactions(bh.Hash, bh.Transactions)
	}

	if 0 != len(missing) {
		var castorId groupsig.ID
		error := castorId.Deserialize(bh.Castor)
		if error != nil {
			panic("[LightChain]Groupsig id deserialize error:" + error.Error())
		}
		//向CASTOR索取交易
		m := &TransactionRequestMessage{TransactionHashes: missing, CurrentBlockHash: bh.Hash, BlockHeight: bh.Height, BlockPv: bh.ProveValue,}
		go RequestTransaction(*m, castorId.String())
		return true, missing
	}
	return false, missing
}

func (chain *LightChain) validateTxRoot(txMerkleTreeRoot common.Hash, txs []*types.Transaction) bool {
	txTree := calcTxTree(txs)

	if !bytes.Equal(txTree.Bytes(), txMerkleTreeRoot.Bytes()) {
		Logger.Errorf("Fail to verify txTree, hash1:%s hash2:%s", txTree.Hex(), txMerkleTreeRoot.Hex())
		return false
	}
	return true
}

func (chain *LightChain) getMissingAccountTransactions(preStateRoot common.Hash, b *types.Block) []*types.Transaction {
	state, err := core.NewAccountDB(preStateRoot, chain.stateCache)
	if err != nil {
		panic("Fail to new statedb, error:%s" + err.Error())
	}
	var missingAccouts []common.Address
	var missingAccountTxs []*types.Transaction
	if chain.Height() == 0 && chain.GetCachedBlock(b.Header.Height) == nil {
		missingAccountTxs = b.Transactions
		castor := common.BytesToAddress(b.Header.Castor)

		missingAccouts = append(missingAccouts, castor)
		missingAccouts = append(missingAccouts, common.BonusStorageAddress)
		missingAccouts = append(missingAccouts, common.LightDBAddress)
		missingAccouts = append(missingAccouts, common.HeavyDBAddress)
	} else {
		missingAccountTxs, missingAccouts = chain.executor.FilterMissingAccountTransaction(state, b)
	}

	if len(missingAccountTxs) != 0 || len(missingAccouts) != 0 {
		Logger.Debugf("len(noExecuteTxs):%d,len(missingAccoutns):%d", len(missingAccountTxs), len(missingAccouts))
		var castorId groupsig.ID
		error := castorId.Deserialize(b.Header.Castor)
		if error != nil {
			Logger.Errorf("castorId.Deserialize error!", error.Error())
		}
		ReqStateInfo(castorId.String(), b.Header.Height, b.Header.ProveValue, missingAccountTxs, missingAccouts, b.Header.Hash)
	}
	return missingAccountTxs
}

//铸块成功，上链
//返回值: 0,上链成功
//       -1，验证失败
//        1, 丢弃该块(链上已存在该块或链上存在QN值更大的相同高度块)
//        2，未上链(异步进行分叉调整)
//        3,未上链(缺少账户状态信息)只有轻节点有
func (chain *LightChain) AddBlockOnChain(b *types.Block) int8 {
	if b == nil {
		return -1
	}
	chain.lock.Lock("LightChain:AddBlockOnChain")
	defer chain.lock.Unlock("LightChain:AddBlockOnChain")
	//defer network.Logger.Debugf("add on chain block %d-%d,cast+verify+io+onchain cost%v", b.Header.Height, b.Header.QueueNumber, time.Since(b.Header.CurTime))

	return chain.addBlockOnChain(b)
}

func (chain *LightChain) addBlockOnChain(b *types.Block) int8 {
	var (
		state  *core.AccountDB
		status int8
	)

	// 自己铸块的时候，会将块临时存放到blockCache里
	// 当组内其他成员验证通过后，自己上链就无需验证、执行交易，直接上链即可
	cache, _ := chain.blockCache.Get(b.Header.Hash)
	//if false {
	if cache != nil {
		status = 0
		state = cache.(*castingBlock).state
		chain.blockCache.Remove(b.Header.Hash)
	} else {
		// 验证块是否有问题
		_, status, state, _ = chain.verifyCastingBlock(*b.Header, b.Transactions)
		Logger.Errorf("[LightChain]verifyCastingBlock,status:%d \n", status)
		if status == 3 {
			return 3
		}
		if status != 0 {
			Logger.Errorf("[LightChain]fail to VerifyCastingBlock, reason code:%d \n", status)
			return -1
		}
	}

	var headerJson []byte
	if chain.Height() == 0 || b.Header.PreHash == chain.latestBlock.Hash {
		status, headerJson = chain.saveBlock(b)
	} else if b.Header.TotalQN-chain.latestBlock.TotalQN <= 0 || b.Header.Hash == chain.latestBlock.Hash {
		return 1
	} else if b.Header.PreHash == chain.latestBlock.PreHash {
		chain.Remove(chain.latestBlock)
		status, headerJson = chain.saveBlock(b)
	} else {
		//b.Header.TotalQN > chain.latestBlock.TotalQN
		if chain.isAdujsting {
			return 2
		}
		var castorId groupsig.ID
		error := castorId.Deserialize(b.Header.Castor)
		if error != nil {
			log.Printf("[LightChain]Give up ajusting bolck chain because of groupsig id deserialize error:%s", error.Error())
			return -1
		}
		chain.SetAdujsting(true)
		RequestBlockInfoByHeight(castorId.String(), chain.latestBlock.Height, chain.latestBlock.Hash, true)
		status = 2
	}

	// 上链成功，移除pool中的交易
	if 0 == status {
		Logger.Debugf("ON chain succ! Height:%d,Hash:%x", b.Header.Height, b.Header.Hash)

		chain.latestStateDB = state
		root, _ := state.Commit(true)
		triedb := chain.stateCache.TrieDB()
		triedb.Commit(root, false)
		if chain.updateLastBlock(state, b.Header, headerJson) == -1 {
			return -1
		}
		chain.transactionPool.Remove(b.Header.Hash, b.Header.Transactions)
		notify.BUS.Publish(notify.BlockAddSucc, &notify.BlockMessage{Block: *b,})

		headerMsg := network.Message{Code: network.NewBlockHeaderMsg, Body: headerJson}
		network.GetNetInstance().Relay(headerMsg, 1)
		network.Logger.Debugf("After add on chain,spread block %d-%d header to neighbor,header size %d,hash:%v", b.Header.Height, b.Header.ProveValue, len(headerJson), b.Header.Hash)
	}
	return status

}

func (chain *LightChain) updateLastBlock(state *core.AccountDB, header *types.BlockHeader, headerJson []byte) int8 {
	err := chain.blockHeight.Put([]byte(BLOCK_STATUS_KEY), headerJson)
	if err != nil {
		fmt.Printf("[block]fail to put current, error:%s \n", err)
		return -1
	}
	chain.latestStateDB = state
	chain.latestBlock = header
	Logger.Debugf("blockchain update latestStateDB:%s height:%d", header.StateTree.Hex(), header.Height)
	return 0
}

//根据指定哈希查询块
func (chain *LightChain) QueryBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	return chain.queryBlockHeaderByHash(hash)
}

func (chain *LightChain) queryBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	block := chain.QueryBlockByHash(hash)
	if nil == block {
		return nil
	}
	return block.Header
}

func (chain *LightChain) QueryBlockBody(blockHash common.Hash) []*types.Transaction {
	return nil
}

func (chain *LightChain) QueryBlock(height uint64) *types.Block {
	panic("Not support!")
}

func (chain *LightChain) QueryBlockByHash(hash common.Hash) *types.Block {
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

// 保存block到ldb
// todo:错误回滚
//result code:
// -1 保存失败
// 0 保存成功
func (chain *LightChain) saveBlock(b *types.Block) (int8, []byte) {
	// 根据hash存block
	blockJson, err := types.MarshalBlock(b)
	if err != nil {
		log.Printf("[lightblock]fail to json Marshal, error:%s \n", err)
		return -1, nil
	}
	err = chain.blocks.Put(b.Header.Hash.Bytes(), blockJson)
	if err != nil {
		log.Printf("[lightblock]fail to put key:hash value:block, error:%s \n", err)
		return -1, nil
	}
	// 根据height存blockheader
	headerJson, err := types.MarshalBlockHeader(b.Header)
	if err != nil {
		log.Printf("[lightblock]fail to json Marshal header, error:%s \n", err)
		return -1, nil
	}
	err = chain.blockHeight.Put(generateHeightKey(b.Header.Height), headerJson)
	if err != nil {
		log.Printf("[lightblock]fail to put key:height value:headerjson, error:%s \n", err)
		return -1, nil
	}

	// 持久化保存最新块信息
	chain.topBlocks.Add(b.Header.Height, b.Header)

	return 0, headerJson
}

// 删除块
func (chain *LightChain) Remove(header *types.BlockHeader) {
	hash := header.Hash
	chain.blocks.Delete(hash.Bytes())
	chain.blockHeight.Delete(generateHeightKey(header.Height))
}

//清除链所有数据
func (chain *LightChain) Clear() error {
	chain.lock.Lock("Clear")
	defer chain.lock.Unlock("Clear")
	chain.init = false
	chain.latestBlock = nil
	chain.topBlocks, _ = lru.New(LIGHT_BLOCKHEIGHT_CACHE_SIZE)
	var err error
	chain.blockHeight.Close()
	chain.statedb.Close()
	chain.statedb, err = datasource.NewLRUMemDatabase(LIGHT_LRU_SIZE)
	if err != nil {
		Logger.Error("[LightChain initLightChain Error!Msg=%v]", err)
		return err
	}
	chain.stateCache = core.NewLightDatabase(chain.statedb)
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

func (chain *LightChain) CompareChainPiece(bhs []*BlockHash, sourceId string) {
	if bhs == nil || len(bhs) == 0 {
		return
	}
	chain.lock.Lock("CompareChainPiece")
	defer chain.lock.Unlock("CompareChainPiece")
	//Logger.Debugf("[BlockChain] CompareChainPiece get block hashes,length:%d,lowest height:%d", len(bhs), bhs[len(bhs)-1].Height)
	blockHash, hasCommonAncestor, _ := chain.FindCommonAncestor(bhs, 0, len(bhs)-1)
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
		RequestBlockInfoByHeight(sourceId, blockHash.Height, blockHash.Hash, true)
	} else {
		chain.SetLastBlockHash(bhs[0])
		cbhr := BlockHashesReq{Height: bhs[len(bhs)-1].Height, Length: uint64(len(bhs) * 10)}
		Logger.Debugf("[BlockChain]Do not find common ancestor!Request hashes form node:%s,base height:%d,length:%d", sourceId, cbhr.Height, cbhr.Length)
		RequestBlockHashes(sourceId, cbhr)
	}

}

// 删除块
func (chain *LightChain) remove(header *types.BlockHeader) {
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
	chain.transactionPool.AddTransactions(txs)
}

func (chain *LightChain) GetTrieNodesByExecuteTransactions(header *types.BlockHeader, transactions []*types.Transaction, addresses []common.Address) *[]types.StateNode {
	panic("Not support!")
}

func (chain *LightChain) InsertStateNode(nodes *[]types.StateNode) {
	//TODO:put里面的索粒度太小了。增加putwithnolock方法
	for _, node := range *nodes {
		err := chain.statedb.Put(node.Key, node.Value)
		if err != nil {
			panic("InsertStateNode error:" + err.Error())
		}
	}
}

func (chain *LightChain) Cache(b *types.Block) {
	chain.pendingLock.Lock()
	defer chain.pendingLock.Unlock()

	chain.pending[b.Header.Height] = b
}

func (chain *LightChain) GetCachedBlock(blockHeight uint64) *types.Block {
	chain.pendingLock.Lock()
	defer chain.pendingLock.Unlock()
	return chain.pending[blockHeight]
}

func (chain *LightChain) RemoveFromCache(b *types.Block) {
	chain.pendingLock.Lock()
	defer chain.pendingLock.Unlock()

	delete(chain.pending, b.Header.Height)
}

func (chain *LightChain) MarkMissNodeState(blockHash common.Hash) {
	chain.missingNodeLock.Lock("MarkMissNodeState")
	defer chain.missingNodeLock.Unlock("MarkMissNodeState")

	chain.missingNodeState[blockHash] = true
}

func (chain *LightChain) GetNodeState(blockHash common.Hash) bool {
	chain.missingNodeLock.RLock("GetNodeState")
	defer chain.missingNodeLock.RUnlock("GetNodeState")

	return chain.missingNodeState[blockHash]
}

func (chain *LightChain) FreeMissNodeState(blockHash common.Hash) {
	chain.missingNodeLock.Lock("FreeMissNodeState")
	defer chain.missingNodeLock.Unlock("FreeMissNodeState")

	delete(chain.missingNodeState, blockHash)
}

func (chain *LightChain) SetPreBlockStateRoot(blockHash common.Hash, preBlockStateRoot common.Hash) {
	chain.preBlockStateRootLock.Lock("SetPreBlockStateRoot")
	defer chain.preBlockStateRootLock.Unlock("SetPreBlockStateRoot")

	chain.preBlockStateRoot[blockHash] = preBlockStateRoot
}

func (chain *LightChain) GetPreBlockStateRoot(blockHash common.Hash) common.Hash {
	chain.preBlockStateRootLock.Lock("GetPreBlockStateRoot")
	defer chain.preBlockStateRootLock.Unlock("GetPreBlockStateRoot")

	return chain.preBlockStateRoot[blockHash]
}

func (chain *LightChain) FreePreBlockStateRoot(blockHash common.Hash) {
	chain.preBlockStateRootLock.Lock("FreePreBlockStateRoot")
	defer chain.preBlockStateRootLock.Unlock("FreePreBlockStateRoot")

	delete(chain.preBlockStateRoot, blockHash)
}

func (chain *LightChain) GetAccountDBByHash(hash common.Hash) (vm.AccountDB, error) {
	header := chain.QueryBlockHeaderByHash(hash)
	return core.NewAccountDB(header.StateTree, chain.stateCache)
}
