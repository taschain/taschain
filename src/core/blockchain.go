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
)

const STATUS_KEY = "current"

// 配置文件，暂时写死
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
	//当前链的高度
	height uint64

	// 读写锁
	lock sync.RWMutex

	// 是否可以工作
	init bool

	statedb    ethdb.Database
	stateCache state.Database // State database to reuse between imports (contains state cache)

	executor *EVMExecutor
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

func InitBlockChain() *BlockChain {

	chain := &BlockChain{
		config:          DefaultBlockChainConfig(),
		transactionPool: NewTransactionPool(),
		height:          0,
		latestBlock:     nil,

		lock: sync.RWMutex{},
		init: true,
	}

	//从磁盘文件中初始化leveldb
	var err error

	chain.blocks, err = datasource.NewLDBDatabase(chain.config.block, chain.config.blockCache, chain.config.blockHandle)
	if err != nil {
		//todo: 日志
		return nil
	}

	chain.blockHeight, err = datasource.NewLDBDatabase(chain.config.blockHeight, chain.config.blockHeightCache, chain.config.blockHeightHandle)
	if err != nil {
		//todo: 日志
		return nil
	}

	chain.statedb, err = ethdb.NewLDBDatabase(chain.config.state, chain.config.stateCache, chain.config.stateHandle)
	if err != nil {
		//todo: 日志
		return nil
	}
	chain.stateCache = state.NewDatabase(chain.statedb)

	chain.executor = NewEVMExecutor(chain)

	// 恢复链状态 height,latestBlock
	// todo:特殊的key保存最新的状态，当前写到了ldb，有性能损耗
	chain.latestBlock = chain.getBlockHeaderByHeight([]byte(STATUS_KEY))
	if nil != chain.latestBlock {
		chain.height = chain.latestBlock.Height
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
			chain.height = 1

		}
	}

	return chain
}

func Clear(config *BlockChainConfig) {
	os.RemoveAll(config.block)
	os.RemoveAll(config.blockHeight)
	os.RemoveAll(config.state)
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
	chain.height = 0

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
		chain.height = 1

	}

	chain.init = true
	return err
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

	return chain.queryBlockByHash(hash)
}

func (chain *BlockChain) queryBlockByHash(hash common.Hash) *BlockHeader {
	result, err := chain.blocks.Get(hash.Bytes())

	if result != nil {
		var block Block
		err = json.Unmarshal(result, &block)
		if err != nil || &block == nil {
			return nil
		}

		return block.Header
	} else {
		return nil
	}
}

//根据指定高度查询块
func (chain *BlockChain) getBlockHeaderByHeight(height []byte) *BlockHeader {

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

	return chain.getBlockHeaderByHeight(chain.generateHeightKey(height))
}

func (chain *BlockChain) queryBlockByHeight(height uint64) *BlockHeader {
	return chain.getBlockHeaderByHeight(chain.generateHeightKey(height))
}

func (chain *BlockChain) CastingBlockAfter(latestBlock *BlockHeader) *Block {
	block := new(Block)

	block.Transactions = chain.transactionPool.GetTransactionsForCasting()
	transactionHashes := make([]common.Hash, len(block.Transactions))
	for i, tx := range block.Transactions {
		transactionHashes[i] = tx.Hash
	}

	block.Header = &BlockHeader{
		Transactions: transactionHashes,
		CurTime:      time.Now(), //todo:时区问题
	}

	if latestBlock != nil {
		block.Header.PreHash = latestBlock.Hash
		block.Header.Height = latestBlock.Height + 1
	}

	state, err := state.New(c.BytesToHash(latestBlock.StateTree.Bytes()), chain.stateCache)
	if err != nil {
		return nil
	}

	// Process block using the parent state as reference point.
	receipts, statehash, _, err := chain.executor.Execute(state, block)
	if err != nil {
		return nil
	}
	block.Header.StateTree = common.BytesToHash(statehash.Bytes())

	block.Header.TxTree = block.calcTxTree()
	block.Header.ReceiptTree = block.calcReceiptsTree(receipts)

	block.Header.Hash = block.Header.GenHash()
	return block
}

//构建一个铸块（组内当前铸块人同步操作）
func (chain *BlockChain) CastingBlock() *Block {
	return chain.CastingBlockAfter(chain.latestBlock)

}

//验证一个铸块（如本地缺少交易，则异步网络请求该交易）
//返回:=0, 验证通过；=-1，验证失败；=1，缺少交易，已异步向网络模块请求
func (chain *BlockChain) VerifyCastingBlock(bh BlockHeader) int8 {
	missing := false
	transactions := make([]*Transaction, len(bh.Transactions))
	for i, hash := range bh.Transactions {
		transaction, err := chain.transactionPool.GetTransaction(hash)
		if err != nil {
			missing = true
		} else {
			transactions[i] = transaction
		}

	}

	if missing {
		return 1
	}

	return 0
}

//铸块成功，上链
//返回:=0,上链成功；=-1，验证失败；=1,上链成功，上链过程中发现分叉并进行了权重链调整
func (chain *BlockChain) AddBlockOnChain(b *Block) int8 {
	chain.lock.Lock()
	defer chain.lock.Unlock()

	preHash := b.Header.PreHash
	preBlock := chain.queryBlockByHash(preHash)

	//本地无父块，暂不处理
	// todo:可以缓存，等父块到了再add
	if preBlock == nil {
		return -1
	}

	// 验证块是否有问题
	status := chain.VerifyCastingBlock(*b.Header)
	if status != 0 {
		return -1
	}
	if hexutil.Encode(b.calcTxTree().Bytes()) != hexutil.Encode(b.Header.TxTree.Bytes()) {
		return -1
	}

	// 验证交易执行结果
	state, err := state.New(c.BytesToHash(preBlock.StateTree.Bytes()), chain.stateCache)
	if err != nil {
		return -1
	}
	receipts, statehash, _, err := chain.executor.Execute(state, b)
	if err != nil {
		return -1
	}
	if hexutil.Encode(statehash.Bytes()) != hexutil.Encode(b.Header.StateTree.Bytes()) {
		return -1
	}

	if hexutil.Encode(b.calcReceiptsTree(receipts).Bytes()) != hexutil.Encode(b.Header.ReceiptTree.Bytes()) {
		return -1
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
	chain.height = b.Header.Height
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
		for height := header.Height + 1; height <= chain.height; height++ {
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
	chain.blocks.Delete(header.Hash.Bytes())
	chain.blockHeight.Delete(chain.generateHeightKey(header.Height))

	// todo: 删除块的交易，是否要回transactionpool
}

func (chain *BlockChain) GetTransactionPool() *TransactionPool {
	return chain.transactionPool
}
