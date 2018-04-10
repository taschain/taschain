package core

import (
	"common"
	"core/datasource"
	"encoding/binary"
	"encoding/json"
	"time"
	"sync"
	"os"
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
	//当前链的高度
	height uint64

	// 读写锁
	lock sync.RWMutex

	// 是否可以工作
	init bool
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

	// 恢复链状态 height,latestBlock
	// todo:特殊的key保存最新的状态，当前写到了ldb，有性能损耗
	chain.latestBlock = chain.getBlockHeaderByHeight([]byte(STATUS_KEY))
	if nil != chain.latestBlock {
		chain.height = chain.latestBlock.Height
	} else {
		// 创始块
		chain.saveBlock(GenesisBlock())
		chain.height = 1
	}
	return chain
}

func Clear(config *BlockChainConfig) {
	os.RemoveAll(config.block)
	os.RemoveAll(config.blockHeight)
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

	// 创始块
	chain.saveBlock(GenesisBlock())
	chain.height = 1
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

	result, err := chain.blocks.Get(hash.Bytes())

	if err != nil {
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
	chain.lock.RLock()
	defer chain.lock.RUnlock()

	result, err := chain.blockHeight.Get(height)
	if err != nil {
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

//构建一个铸块（组内当前铸块人同步操作）
func (chain *BlockChain) CastingBlock() *Block {

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

	if chain.latestBlock != nil {
		block.Header.PreHash = chain.latestBlock.Hash
		block.Header.Height = chain.latestBlock.Height + 1
	}

	blockByte, _ := json.Marshal(block)
	block.Header.Hash = common.BytesToHash(Sha256(blockByte))

	return block
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
	preHash := b.Header.PreHash
	preBlock, error := chain.blocks.Has(preHash.Bytes())

	//本地无父块，暂不处理
	// todo:可以缓存，等父块到了再add
	if error != nil || !preBlock {
		return -1
	}

	// 验证块是否有问题
	status := chain.VerifyCastingBlock(*b.Header)
	if status != 0 {
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

	return status

}

// 保存block到ldb
// todo:错误回滚
func (chain *BlockChain) saveBlock(b *Block) int8 {
	chain.lock.Lock()
	defer chain.lock.Unlock()

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
	chain.lock.Lock()
	defer chain.lock.Unlock()

	header := chain.QueryBlockByHeight(b.Header.Height)
	if header == nil {
		return -1
	}

	// todo:判断权重，决定是否要替换
	if chain.weight(header, b.Header) {
		chain.remove(header)
		// 替换
		for height := header.Height + 1; height < chain.height; height++ {
			header = chain.QueryBlockByHeight(height)
			if header == nil {
				continue
			}
			chain.remove(header)
		}

		if chain.saveBlock(b) == 0 {
			return 1
		} else {
			return -1
		}
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
}

func (chain *BlockChain) GetTransactionPool() *TransactionPool {
	return chain.transactionPool
}
