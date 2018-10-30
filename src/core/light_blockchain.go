package core

import (
	"github.com/hashicorp/golang-lru"
	"middleware"
	"middleware/types"

	"taslog"

	"storage/core"
	"common"
	"consensus/groupsig"
	"middleware/notify"
	"log"
	"fmt"
	vtypes "storage/core/types"
	"math/big"
	"storage/core/vm"
	"storage/tasdb"
)

const (
	LightFullAccountBlockCacheSize = 20
	LIGHT_BLOCK_CACHE_SIZE         = 20
	LIGHT_BLOCKHEIGHT_CACHE_SIZE   = 100
	LIGHT_BLOCKBODY_CACHE_SIZE     = 100
	LIGHT_LRU_SIZE                 = 5000000
)

type LightChain struct {
	config                   *LightChainConfig
	prototypeChain
	missingAccountBlocks     map[common.Hash]*types.Block
	missingAccountBlocksLock middleware.Loglock

	fullAccountBlockCache *lru.Cache

	//todo 这里其实只存储轻节点冷启动的第一块的PRE，其实也可以去掉  待测试
	preBlockStateRoot     map[common.Hash]common.Hash
	preBlockStateRootLock middleware.Loglock
}

// 配置
type LightChainConfig struct {
	blockHeight string

	state string

	check string
}

func getLightChainConfig() *LightChainConfig {
	defaultConfig := DefaultLightChainConfig()
	if nil == common.GlobalConf {
		return defaultConfig
	}

	return &LightChainConfig{
		blockHeight: common.GlobalConf.GetString(CONFIG_SEC, "blockHeight", defaultConfig.blockHeight),

		state: common.GlobalConf.GetString(CONFIG_SEC, "state", defaultConfig.state),

		check: common.GlobalConf.GetString(CONFIG_SEC, "check", defaultConfig.check),
	}
}

// 默认配置
func DefaultLightChainConfig() *LightChainConfig {
	return &LightChainConfig{
		blockHeight: "light_height",
		state:       "light_state",
		check:       "light_check",
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

		missingAccountBlocks:     make(map[common.Hash]*types.Block),
		missingAccountBlocksLock: middleware.NewLoglock("lightchain"),
		preBlockStateRoot:        make(map[common.Hash]common.Hash),
		preBlockStateRootLock:    middleware.NewLoglock("lightchain"),
	}

	var err error
	chain.futureBlocks, err = lru.New(20)
	chain.fullAccountBlockCache, _ = lru.New(LightFullAccountBlockCacheSize)
	chain.verifiedBlocks, err = lru.New(LIGHT_BLOCK_CACHE_SIZE)
	chain.topBlocks, _ = lru.New(LIGHT_BLOCKHEIGHT_CACHE_SIZE)
	if err != nil {
		return err
	}
	chain.blocks, err = tasdb.NewLRUMemDatabase(LIGHT_BLOCKBODY_CACHE_SIZE)
	if err != nil {
		Logger.Error("[LightChain initLightChain Error!Msg=%v]", err)
		return err
	}
	chain.blockHeight, err = tasdb.NewDatabase(chain.config.blockHeight)
	if err != nil {
		Logger.Error("[LightChain initLightChain Error!Msg=%v]", err)
		return err
	}
	chain.statedb, err = tasdb.NewLRUMemDatabase(LIGHT_LRU_SIZE)
	if err != nil {
		Logger.Error("[LightChain initLightChain Error!Msg=%v]", err)
		return err
	}
	chain.checkdb, err = tasdb.NewDatabase(chain.config.check)
	if err != nil {
		Logger.Error("[LightChain initLightChain Error!Msg=%v]", err)
		return err
	}

	chain.bonusManager = newBonusManager()
	chain.stateCache = core.NewLightDatabase(chain.statedb)
	chain.executor = NewTVMExecutor(chain)
	initMinerManager(chain)
	// 恢复链状态 height,latestBlock
	// todo:特殊的key保存最新的状态，当前写到了ldb，有性能损耗
	chain.latestBlock = chain.queryBlockHeaderByHeight([]byte(BLOCK_STATUS_KEY), false)
	if nil != chain.latestBlock {
		chain.buildCache(LIGHT_BLOCKHEIGHT_CACHE_SIZE, chain.topBlocks)
		Logger.Infof("initLightChain chain.latestBlock.StateTree  Hash:%s", chain.latestBlock.StateTree.Hex())
		state, err := core.NewAccountDB(chain.latestBlock.StateTree, chain.stateCache)
		if nil == err {
			chain.latestStateDB = state
		} else {
			panic(err)
		}
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
func (chain *LightChain) CastBlock(height uint64, proveValue *big.Int, proveRoot common.Hash, qn uint64, castor []byte, groupid []byte) *types.Block {
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
func (chain *LightChain) VerifyBlock(bh types.BlockHeader) ([]common.Hash, int8) {
	chain.lock.Lock("VerifyCastingLightChainBlock")
	defer chain.lock.Unlock("VerifyCastingLightChainBlock")

	return chain.verifyBlock(bh, nil)
}

func (chain *LightChain) verifyBlock(bh types.BlockHeader, txs []*types.Transaction) ([]common.Hash, int8) {
	Logger.Debugf("Verify block height:%d,preHash:%v,tx len:%d", bh.Height, bh.PreHash.String(), len(txs))

	if bh.Hash != bh.GenHash() {
		Logger.Debugf("Validate block hash error!")
		return nil, -1
	}

	hasPreBlock, preBlock := chain.hasPreBlock(bh)
	if !hasPreBlock {
		if txs != nil {
			chain.futureBlocks.Add(bh.PreHash, types.Block{Header: &bh, Transactions: txs})
		}
		return nil, 2
	}

	miss, missingTx, transactions := chain.missTransaction(bh, txs)
	if miss {
		return missingTx, 1
	}

	if !chain.validateTxRoot(bh.TxTree, transactions) {
		return nil, -1
	}

	b := &types.Block{Header: &bh, Transactions: txs}
	var preBlockStateTree []byte
	if preBlock == nil {
		preBlockStateTree = chain.getPreBlockStateRoot(b.Header.Hash).Bytes()
	} else {
		preBlockStateTree = preBlock.StateTree.Bytes()
	}
	preRoot := common.BytesToHash(preBlockStateTree)
	//todo 这里分叉如何处理？
	missingAccountTxs := chain.getMissingAccountTransactions(preRoot, b)
	if len(missingAccountTxs) != 0 {
		return nil, 3
	}

	return nil, 0
}

func (chain *LightChain) hasPreBlock(bh types.BlockHeader) (bool, *types.BlockHeader) {
	preBlock := chain.queryBlockHeaderByHash(bh.PreHash)

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

func (chain *LightChain) getMissingAccountTransactions(preStateRoot common.Hash, b *types.Block) []*types.Transaction {
	state, err := core.NewAccountDB(preStateRoot, chain.stateCache)
	if err != nil {
		panic("Fail to new statedb, error:%s" + err.Error())
	}
	var missingAccouts []common.Address
	var missingAccountTxs []*types.Transaction
	_, ok := chain.fullAccountBlockCache.Get(b.Header.Hash)
	if chain.Height() == 0 && !ok {
		missingAccountTxs = b.Transactions
		castor := common.BytesToAddress(b.Header.Castor)

		missingAccouts = append(missingAccouts, castor)
		missingAccouts = append(missingAccouts, common.BonusStorageAddress)
		missingAccouts = append(missingAccouts, common.LightDBAddress)
		missingAccouts = append(missingAccouts, common.HeavyDBAddress)
	} else if !ok {
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
		chain.missingAccountBlocks[b.Header.Hash] = b
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

	// 自己铸块的时候，会将块临时存放到blockCache里
	// 当组内其他成员验证通过后，自己上链就无需验证、执行交易，直接上链即可
	//todo 需要先验证HASH 是否一致然后才能使用该缓存
	cache, _ := chain.verifiedBlocks.Get(b.Header.Hash)
	//if false {
	if cache == nil {
		// 验证块是否有问题
		_, status := chain.verifyBlock(*b.Header, b.Transactions)
		if status == 3 {
			return 3
		}
		if status != 0 {
			Logger.Errorf("[LightChain]fail to VerifyCastingBlock, reason code:%d \n", status)
			return -1
		}
	}

	//if !chain.validateGroupSig(b.Header) {
	//	Logger.Debugf("Fail to validate group sig!")
	//	return -1
	//}

	topBlock := chain.latestBlock
	Logger.Debugf("coming block:hash=%v, preH=%v, height=%v,totalQn:%d", b.Header.Hash.Hex(), b.Header.PreHash.Hex(), b.Header.Height, b.Header.TotalQN)
	Logger.Debugf("Local tophash=%v, topPreH=%v, height=%v,totalQn:%d", topBlock.Hash.Hex(), topBlock.PreHash.Hex(), b.Header.Height, topBlock.TotalQN)

	//轻节点第一次同步直接上链
	if b.Header.PreHash == topBlock.Hash || chain.Height() == 0 {
		result, headerByte := chain.insertBlock(b)
		if result == 0 {
			chain.successOnChainCallBack(b, headerByte)
		}
		return result
	}

	if b.Header.TotalQN < topBlock.TotalQN || b.Header.Hash == topBlock.Hash || chain.queryBlockHeaderByHash(b.Header.Hash) != nil {
		return 1

	}
	//var castorId groupsig.ID
	//error := castorId.Deserialize(b.Header.Castor)
	//if error != nil {
	//	log.Printf("[BlockChain]Give up ajusting bolck chain because of groupsig id deserialize error:%s", error.Error())
	//	return -1
	//}
	//BlockChainImpl.SetAdujsting(true)
	//RequestChainPiece(castorId.String(), b.Header.Height)
	return 2
}

func (chain *LightChain) insertBlock(remoteBlock *types.Block) (int8, []byte) {
	executeTxResult, state, receipts := chain.executeTransaction(remoteBlock)
	if !executeTxResult {
		return -1, nil
	}

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
	verifyHash := chain.consensusHelper.VerifyHash(remoteBlock)
	chain.PutCheckValue(remoteBlock.Header.Height, verifyHash.Bytes())
	chain.transactionPool.Remove(remoteBlock.Header.Hash, remoteBlock.Header.Transactions)
	chain.transactionPool.MarkExecuted(receipts, remoteBlock.Transactions)
	chain.successOnChainCallBack(remoteBlock, headerByte)
	return 0, headerByte
}

func (chain *LightChain) executeTransaction(block *types.Block) (bool, *core.AccountDB, vtypes.Receipts) {
	preBlock := chain.queryBlockHeaderByHash(block.Header.PreHash)
	var preBlockStateTree []byte
	if preBlock == nil {
		preBlockStateTree = chain.getPreBlockStateRoot(block.Header.Hash).Bytes()
	} else {
		preBlockStateTree = preBlock.StateTree.Bytes()
	}
	preRoot := common.BytesToHash(preBlockStateTree)
	state, err := core.NewAccountDB(preRoot, chain.stateCache)
	if err != nil {
		panic("Fail to new statedb, error:%s" + err.Error())
		return false, state, nil
	}
	statehash, receipts, err := chain.executor.Execute(state, block, block.Header.Height, "lightverify")

	if common.ToHex(statehash.Bytes()) != common.ToHex(block.Header.StateTree.Bytes()) {
		Logger.Debugf("[LightChain]fail to verify statetree, hash1:%x hash2:%x", statehash.Bytes(), block.Header.StateTree.Bytes())
		return false, state, receipts
	}

	chain.verifiedBlocks.Add(block.Header.Hash, &castingBlock{state: state, receipts: receipts,})
	return true, state, receipts
}

func (chain *LightChain) successOnChainCallBack(remoteBlock *types.Block, headerJson []byte) {
	Logger.Debugf("ON chain succ! height=%d,hash=%s", remoteBlock.Header.Height, remoteBlock.Header.Hash.Hex())
	notify.BUS.Publish(notify.BlockAddSucc, &notify.BlockMessage{Block: *remoteBlock,})
	if value, _ := chain.futureBlocks.Get(remoteBlock.Header.PreHash); value != nil {
		block := value.(types.Block)
		//todo 这里为了避免死锁只能调用这个方法，但是没办法调用CheckProveRoot全量账本验证了
		chain.addBlockOnChain(&block)
		return
	}
	//GroupChainImpl.RemoveDismissGroupFromCache(b.Header.Height)
	BlockSyncer.Sync()
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
	chain.statedb, err = tasdb.NewLRUMemDatabase(LIGHT_LRU_SIZE)
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

func (chain *LightChain) MarkFullAccountBlock(blockHash common.Hash) *types.Block {
	chain.missingAccountBlocksLock.Lock("MarkFullAccountBlock")
	defer chain.missingAccountBlocksLock.Unlock("MarkFullAccountBlock")

	b := chain.missingAccountBlocks[blockHash]
	chain.fullAccountBlockCache.Add(blockHash, nil)
	delete(chain.missingAccountBlocks, blockHash)
	return b
}

func (chain *LightChain) SetPreBlockStateRoot(blockHash common.Hash, preBlockStateRoot common.Hash) {
	chain.preBlockStateRootLock.Lock("SetPreBlockStateRoot")
	defer chain.preBlockStateRootLock.Unlock("SetPreBlockStateRoot")

	chain.preBlockStateRoot[blockHash] = preBlockStateRoot
}

func (chain *LightChain) getPreBlockStateRoot(blockHash common.Hash) common.Hash {
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
