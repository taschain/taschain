package core

import (
	"common"
	"core/datasource"
	"encoding/binary"
	"encoding/json"
	"time"
	"sync"
	"os"
	"vm/core/state"
	c "vm/common"
	"vm/ethdb"
	"vm/common/hexutil"
	"math/big"
	"vm/core/types"
	"github.com/hashicorp/golang-lru"
	"fmt"
	"vm/common/math"
)

const (
	STATUS_KEY = "current"
	CONFIG_SEC = "chain"
)

var BlockChainImpl *BlockChain

// 配置
type BlockChainConfig struct {
	block       string
	blockCache  int
	blockHandle int

	blockHeight       string
	blockHeightCache  int
	blockHeightHandle int

	state       string
	stateCache  int
	stateHandle int

	//组内能出的最大QN值
	qn uint64
}

type BlockChain struct {
	config *BlockChainConfig

	// key: blockhash, value: block
	blocks datasource.Database
	//key: height, value: blockHeader
	blockHeight datasource.Database

	transactionPool *TransactionPool
	//已上链的最新块
	latestBlock *BlockHeader

	latestStateDB *state.StateDB

	// 当前链的高度，其值等于当前链里有多少块（创始块不计入）
	// 与最高块的关系是：chain.height = latestBlock.Height
	//height uint64

	// 读写锁
	lock sync.RWMutex

	// 是否可以工作
	init bool

	statedb    ethdb.Database
	stateCache state.Database // State database to reuse between imports (contains state cache)

	executor      *EVMExecutor
	voteProcessor VoteProcessor

	blockCache *lru.Cache
}

type castingBlock struct {
	state    *state.StateDB
	receipts types.Receipts
}

// 默认配置
func DefaultBlockChainConfig() *BlockChainConfig {
	return &BlockChainConfig{
		block:       "block",
		blockCache:  128,
		blockHandle: 1024,

		blockHeight:       "height",
		blockHeightCache:  128,
		blockHeightHandle: 1024,

		state:       "state",
		stateCache:  128,
		stateHandle: 1024,

		qn: 4,
	}
}

func getBlockChainConfig() *BlockChainConfig {
	defaultConfig := DefaultBlockChainConfig()
	if nil == common.GlobalConf {
		return defaultConfig
	}

	config := &BlockChainConfig{
		block:       common.GlobalConf.GetString(CONFIG_SEC, "block", defaultConfig.block),
		blockCache:  common.GlobalConf.GetInt(CONFIG_SEC, "blockCache", defaultConfig.blockCache),
		blockHandle: common.GlobalConf.GetInt(CONFIG_SEC, "blockHandle", defaultConfig.blockHandle),

		blockHeight:       common.GlobalConf.GetString(CONFIG_SEC, "blockHeight", defaultConfig.blockHeight),
		blockHeightCache:  common.GlobalConf.GetInt(CONFIG_SEC, "blockHeightCache", defaultConfig.blockHeightCache),
		blockHeightHandle: common.GlobalConf.GetInt(CONFIG_SEC, "blockHeightHandle", defaultConfig.blockHeightHandle),

		state:       common.GlobalConf.GetString(CONFIG_SEC, "state", defaultConfig.state),
		stateCache:  common.GlobalConf.GetInt(CONFIG_SEC, "stateCache", defaultConfig.stateCache),
		stateHandle: common.GlobalConf.GetInt(CONFIG_SEC, "stateHandle", defaultConfig.stateHandle),

		qn: uint64(common.GlobalConf.GetInt(CONFIG_SEC, "qn", int(defaultConfig.qn))),
	}

	return config
}

func initBlockChain() error {

	chain := &BlockChain{
		config:          getBlockChainConfig(),
		transactionPool: NewTransactionPool(),
		latestBlock:     nil,

		lock: sync.RWMutex{},
		init: true,
	}

	var err error
	chain.blockCache, err = lru.New(1000)
	if err != nil {
		return err
	}

	//从磁盘文件中初始化leveldb
	chain.blocks, err = datasource.NewLDBDatabase(chain.config.block, chain.config.blockCache, chain.config.blockHandle)
	if err != nil {
		//todo: 日志
		return err
	}

	chain.blockHeight, err = datasource.NewLDBDatabase(chain.config.blockHeight, chain.config.blockHeightCache, chain.config.blockHeightHandle)
	if err != nil {
		//todo: 日志
		return err
	}

	chain.statedb, err = ethdb.NewLDBDatabase(chain.config.state, chain.config.stateCache, chain.config.stateHandle)
	if err != nil {
		//todo: 日志
		return err
	}
	chain.stateCache = state.NewDatabase(chain.statedb)

	chain.executor = NewEVMExecutor(chain)

	// 恢复链状态 height,latestBlock
	// todo:特殊的key保存最新的状态，当前写到了ldb，有性能损耗
	chain.latestBlock = chain.queryBlockHeaderByHeight([]byte(STATUS_KEY))
	if nil != chain.latestBlock {
		state, err := state.New(c.BytesToHash(chain.latestBlock.StateTree.Bytes()), chain.stateCache)
		if nil == err {
			chain.latestStateDB = state
		}
	} else {
		// 创始块
		state, err := state.New(c.Hash{}, chain.stateCache)
		if nil == err {
			chain.latestStateDB = state
			block := GenesisBlock(state, chain.stateCache.TrieDB())

			chain.saveBlock(block)

		}
	}

	BlockChainImpl = chain
	return nil
}

func Clear(config *BlockChainConfig) {
	os.RemoveAll(config.block)
	os.RemoveAll(config.blockHeight)
	os.RemoveAll(config.state)
}

func (chain *BlockChain) SetVoteProcessor(processor VoteProcessor) {
	chain.lock.Lock()
	defer chain.lock.Unlock()

	chain.voteProcessor = processor
}

func (chain *BlockChain) Height() uint64 {
	chain.lock.RLock()
	defer chain.lock.RUnlock()

	if nil == chain.latestBlock {
		return math.MaxUint64
	}
	return chain.latestBlock.Height
}

func (chain *BlockChain) LatestStateDB() *state.StateDB {
	return chain.latestStateDB
}

func (chain *BlockChain) GetBlockMessage(height uint64, hash common.Hash) *BlockMessage {
	chain.lock.RLock()
	defer chain.lock.RUnlock()

	localHeight := chain.latestBlock.Height
	if height >= localHeight {
		return nil
	}

	//todo: 当前简单处理，暂时不处理分叉问题
	blocks := make([]*Block, localHeight-height)

	for i := height + 1; i <= localHeight; i++ {
		bh := chain.queryBlockByHeight(i)
		if nil == bh {
			continue
		}
		b := chain.queryBlockByHash(bh.Hash)
		if nil == b {
			continue
		}

		blocks[i-height-1] = b
	}

	return &BlockMessage{
		Blocks: blocks,
	}
}

func (chain *BlockChain) AddBlockMessage(bm BlockMessage) error {
	blocks := bm.Blocks
	if nil == blocks || 0 == len(blocks) {
		return ErrNil
	}

	for _, block := range blocks {
		code := chain.AddBlockOnChain(block)
		if 0 != code {
			return fmt.Errorf("fail to add, code:%d", code)
		}
	}
	return nil
}

func (chain *BlockChain) GetBalance(address common.Address) *big.Int {
	if nil == chain.latestStateDB {
		return nil
	}

	return chain.latestStateDB.GetBalance(c.BytesToAddress(address.Bytes()))
}

func (chain *BlockChain) GetNonce(address common.Address) uint64 {
	if nil == chain.latestStateDB {
		return 0
	}

	return chain.latestStateDB.GetNonce(c.BytesToAddress(address.Bytes()))
}

//清除链所有数据
func (chain *BlockChain) Clear() error {
	chain.lock.Lock()
	defer chain.lock.Unlock()

	chain.init = false
	chain.latestBlock = nil

	err := chain.blockHeight.Clear()
	if nil != err {
		return err
	}
	err = chain.blocks.Clear()
	if nil != err {
		return err
	}

	os.RemoveAll(chain.config.state)
	chain.statedb, err = ethdb.NewLDBDatabase(chain.config.state, chain.config.stateCache, chain.config.stateHandle)
	if err != nil {
		//todo: 日志
		return err
	}

	chain.stateCache = state.NewDatabase(chain.statedb)
	chain.executor = NewEVMExecutor(chain)

	// 创始块
	state, err := state.New(c.Hash{}, chain.stateCache)
	if nil == err {
		chain.latestStateDB = state
		block := GenesisBlock(state, chain.stateCache.TrieDB())

		chain.saveBlock(block)
	}

	chain.init = true

	chain.transactionPool.Clear()
	return err
}

func (chain *BlockChain) GenerateBlock(bh BlockHeader) *Block {
	block := &Block{
		Header: &bh,
	}

	block.Transactions = make([]*Transaction, len(bh.Transactions))
	for i, hash := range bh.Transactions {
		block.Transactions[i], _ = chain.transactionPool.GetTransaction(hash)
	}
	return block
}

//根据哈希取得某个交易
func (chain *BlockChain) GetTransactionByHash(h common.Hash) (*Transaction, error) {
	return chain.transactionPool.GetTransaction(h)
}

//辅助方法族
//查询最高块
func (chain *BlockChain) QueryTopBlock() *BlockHeader {
	chain.lock.RLock()
	defer chain.lock.RUnlock()

	return chain.latestBlock
}

//根据指定哈希查询块
func (chain *BlockChain) QueryBlockByHash(hash common.Hash) *BlockHeader {
	chain.lock.RLock()
	defer chain.lock.RUnlock()

	return chain.queryBlockHeaderByHash(hash)
}

func (chain *BlockChain) queryBlockByHash(hash common.Hash) *Block {
	result, err := chain.blocks.Get(hash.Bytes())

	if result != nil {
		var block *Block
		err = json.Unmarshal(result, &block)
		if err != nil || &block == nil {
			return nil
		}

		return block
	} else {
		return nil
	}
}

func (chain *BlockChain) queryBlockHeaderByHash(hash common.Hash) *BlockHeader {
	block := chain.queryBlockByHash(hash)
	if nil == block {
		return nil
	}

	return block.Header
}

//根据指定高度查询块
func (chain *BlockChain) queryBlockHeaderByHeight(height []byte) *BlockHeader {

	result, err := chain.blockHeight.Get(height)
	if result != nil {
		var header BlockHeader
		err = json.Unmarshal(result, &header)
		if err != nil {
			return nil
		}

		return &header
	} else {
		return nil
	}
}

//根据指定高度查询块
func (chain *BlockChain) QueryBlockByHeight(height uint64) *BlockHeader {
	chain.lock.RLock()
	defer chain.lock.RUnlock()

	return chain.queryBlockHeaderByHeight(chain.generateHeightKey(height))
}

func (chain *BlockChain) queryBlockByHeight(height uint64) *BlockHeader {
	return chain.queryBlockHeaderByHeight(chain.generateHeightKey(height))
}

func (chain *BlockChain) CastingBlockAfter(latestBlock *BlockHeader, height uint64, nonce uint64, queueNumber uint64, castor []byte, groupid []byte) *Block {
	//todo: 校验高度

	block := new(Block)

	block.Transactions = chain.transactionPool.GetTransactionsForCasting()
	transactionHashes := make([]common.Hash, len(block.Transactions))
	for i, tx := range block.Transactions {
		transactionHashes[i] = tx.Hash
	}

	block.Header = &BlockHeader{
		Transactions: transactionHashes,
		CurTime:      time.Now(), //todo:时区问题
		Height:       height,
		Nonce:        nonce,
		QueueNumber:  queueNumber,
		Castor:       castor,
		GroupId:      groupid,
	}

	if latestBlock != nil {
		block.Header.PreHash = latestBlock.Hash
		block.Header.Height = latestBlock.Height + 1
		block.Header.PreTime = latestBlock.CurTime
	}

	state, err := state.New(c.BytesToHash(latestBlock.StateTree.Bytes()), chain.stateCache)
	if err != nil {
		return nil
	}

	// Process block using the parent state as reference point.
	receipts, statehash, _, err := chain.executor.Execute(state, block, chain.voteProcessor)
	if err != nil {
		return nil
	}
	block.Header.StateTree = common.BytesToHash(statehash.Bytes())

	block.Header.TxTree = calcTxTree(block.Transactions)
	block.Header.ReceiptTree = calcReceiptsTree(receipts)

	block.Header.Hash = block.Header.GenHash()

	chain.blockCache.Add(block.Header.Hash, &castingBlock{
		state:    state,
		receipts: receipts,
	})
	return block
}

//构建一个铸块（组内当前铸块人同步操作）
func (chain *BlockChain) CastingBlock(height uint64, nonce uint64, queueNumber uint64, castor []byte, groupid []byte) *Block {
	return chain.CastingBlockAfter(chain.latestBlock, height, nonce, queueNumber, castor, groupid)

}

//验证一个铸块（如本地缺少交易，则异步网络请求该交易）
//返回:=0, 验证通过；=-1，验证失败；=1，缺少交易，已异步向网络模块请求
func (chain *BlockChain) VerifyCastingBlock(bh BlockHeader) ([]common.Hash, int8, *state.StateDB, types.Receipts) {

	// 校验父亲块
	preHash := bh.PreHash
	preBlock := chain.queryBlockHeaderByHash(preHash)

	//本地无父块，暂不处理
	// todo:可以缓存，等父块到了再add
	if preBlock == nil {
		return nil, -1, nil, nil
	}

	// 验证交易
	missing := make([]common.Hash, 0)
	transactions := make([]*Transaction, len(bh.Transactions))
	for i, hash := range bh.Transactions {
		transaction, err := chain.transactionPool.GetTransaction(hash)
		if err != nil {
			missing = append(missing, hash)
		} else {
			transactions[i] = transaction
		}

	}
	if 0 != len(missing) {
		//广播，索取交易
		m := &TransactionRequestMessage{
			TransactionHashes: missing,
			RequestTime:       time.Now(),
		}
		BroadcastTransactionRequest(*m)
		return missing, 1, nil, nil
	}
	if hexutil.Encode(calcTxTree(transactions).Bytes()) != hexutil.Encode(bh.TxTree.Bytes()) {
		return missing, -1, nil, nil
	}

	//执行交易
	state, err := state.New(c.BytesToHash(preBlock.StateTree.Bytes()), chain.stateCache)
	if err != nil {
		return nil, -1, nil, nil
	}

	b := new(Block)
	b.Header = &bh
	b.Transactions = transactions
	receipts, statehash, _, err := chain.executor.Execute(state, b, chain.voteProcessor)
	if err != nil {
		return nil, -1, nil, nil
	}
	if hexutil.Encode(statehash.Bytes()) != hexutil.Encode(b.Header.StateTree.Bytes()) {
		return nil, -1, nil, nil
	}

	if hexutil.Encode(calcReceiptsTree(receipts).Bytes()) != hexutil.Encode(b.Header.ReceiptTree.Bytes()) {
		return nil, 1, nil, nil
	}

	return nil, 0, state, receipts
}

//铸块成功，上链
//返回:=0,上链成功；=-1，验证失败；=1,上链成功，上链过程中发现分叉并进行了权重链调整
func (chain *BlockChain) AddBlockOnChain(b *Block) int8 {
	chain.lock.Lock()
	defer chain.lock.Unlock()

	var (
		state    *state.StateDB
		receipts types.Receipts
		status   int8
	)

	// 自己铸块的时候，会将块临时存放到blockCache里
	// 当组内其他成员验证通过后，自己上链就无需验证、执行交易，直接上链即可
	cache, _ := chain.blockCache.Get(b.Header.Hash)
	if cache != nil {
		status = 0
		state = cache.(*castingBlock).state
		receipts = cache.(*castingBlock).receipts
		chain.blockCache.Remove(b.Header.Hash)
	} else {
		// 验证块是否有问题
		_, status, state, receipts = chain.VerifyCastingBlock(*b.Header)
		if status != 0 {
			return -1
		}
	}

	// 检查高度
	height := b.Header.Height

	// 完美情况
	if height == (chain.latestBlock.Height + 1) {
		status = chain.saveBlock(b)
	} else if height > (chain.latestBlock.Height + 1) {
		//todo:高度超出链当前链的最大高度，这种case是否等价于父块没有?
		return -1
	} else {
		status = chain.adjust(b)
	}

	// 上链成功，移除pool中的交易
	if 0 == status {
		chain.transactionPool.Remove(b.Header.Transactions)
		chain.transactionPool.AddExecuted(receipts, b.Transactions)
		chain.latestStateDB = state
		root, _ := state.Commit(true)
		triedb := chain.stateCache.TrieDB()
		triedb.Commit(root, false)

	}
	return status

}

// 保存block到ldb
// todo:错误回滚
func (chain *BlockChain) saveBlock(b *Block) int8 {
	// 根据hash存block
	blockJson, err := json.Marshal(b)
	if err != nil {
		return -1
	}
	err = chain.blocks.Put(b.Header.Hash.Bytes(), blockJson)
	if err != nil {
		return -1
	}

	// 根据height存blockheader
	headerJson, err := json.Marshal(b.Header)
	if err != nil {
		return -1
	}

	err = chain.blockHeight.Put(chain.generateHeightKey(b.Header.Height), headerJson)
	if err != nil {
		return -1
	}

	// 持久化保存最新块信息
	chain.latestBlock = b.Header
	err = chain.blockHeight.Put([]byte(STATUS_KEY), headerJson)
	if err != nil {
		return -1
	}

	return 0
}

// 链分叉，调整主链
// todo:错误回滚
func (chain *BlockChain) adjust(b *Block) int8 {
	header := chain.queryBlockByHeight(b.Header.Height)
	if header == nil {
		return -1
	}

	// todo:判断权重，决定是否要替换
	if chain.weight(header, b.Header) {
		chain.remove(header)
		// 替换
		for height := header.Height + 1; height <= chain.latestBlock.Height; height++ {
			header = chain.queryBlockByHeight(height)
			if header == nil {
				continue
			}
			chain.remove(header)
		}

		return chain.saveBlock(b)
	} else {
		return -1
	}

}

func (chain *BlockChain) generateHeightKey(height uint64) []byte {
	h := make([]byte, 8)
	binary.BigEndian.PutUint64(h, height)
	return h
}

// 判断权重
func (chain *BlockChain) weight(current *BlockHeader, candidate *BlockHeader) bool {

	return chain.getWeight(current.QueueNumber) > chain.getWeight(candidate.QueueNumber)
}

//取得铸块权重
//第一顺为权重1，第二顺位权重2，第三顺位权重4...，即权重越低越好（但0为无效）
func (chain *BlockChain) getWeight(number uint64) uint64 {
	if number <= chain.config.qn {
		return uint64(number) << 1
	} else {
		return 0
	}
}

// 删除块
func (chain *BlockChain) remove(header *BlockHeader) {
	hash := header.Hash
	block := chain.queryBlockByHash(hash)
	chain.blocks.Delete(hash.Bytes())
	chain.blockHeight.Delete(chain.generateHeightKey(header.Height))

	// 删除块的交易，返回transactionpool
	if nil == block {
		return
	}
	txs := block.Transactions
	chain.transactionPool.RemoveExecuted(txs)
	chain.transactionPool.addTxs(txs)
}

func (chain *BlockChain) GetTransactionPool() *TransactionPool {
	return chain.transactionPool
}
