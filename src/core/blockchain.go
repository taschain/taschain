package core

import (
	"common"
	"core/datasource"
	"encoding/binary"
	"encoding/json"
	"time"
	"hash"
	"crypto/sha256"
	"sync"
)

type BlockChain struct {
	blocks          datasource.Database // key: blockhash, value: block
	blockHeight     datasource.Database //key: height, value: blockHeader
	transactionPool *TransactionPool
	latestBlock     *BlockHeader
	height          uint64
	hs              hash.Hash
	lock            sync.RWMutex
}

func InitBlockChain() *BlockChain {
	//todo: 从磁盘文件中初始化leveldb
	return &BlockChain{
		transactionPool: NewTransactionPool(),
		height:          0,
		hs:              sha256.New(),
		lock:            sync.RWMutex{},
	}
}

//根据哈希取得某个交易
func (chain *BlockChain) GetTransactionByHash(h common.Hash) (*Transaction, error) {
	return chain.transactionPool.GetTransaction(h)
}

//辅助方法族
//查询最高块
func (chain *BlockChain) QueryTopBlock() BlockHeader {
	return *chain.latestBlock
}

//根据指定哈希查询块
func (chain *BlockChain) QueryBlockByHash(hash common.Hash) *BlockHeader {
	result, err := chain.blocks.Get(hash.Bytes())

	if err != nil {
		var block Block
		err = json.Unmarshal(result, &block)
		if err != nil || &block == nil {
			return nil
		}

		return block.header
	} else {
		return nil
	}
}

//根据指定高度查询块
func (chain *BlockChain) QueryBlockByHeight(height uint64) *BlockHeader {

	result, err := chain.blockHeight.Get(chain.generateHeightKey(height))
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

//构建一个铸块（组内当前铸块人同步操作）
func (chain *BlockChain) CastingBlock() Block {

	block := new(Block)
	block.transactions = chain.transactionPool.GetTransactionsForCasting()

	transactionHashes := make([]common.Hash, len(block.transactions))
	for i, tx := range block.transactions {
		transactionHashes[i] = tx.hash
	}

	block.header = &BlockHeader{
		Transactions: transactionHashes,
		CurTime:      time.Now(), //todo:时区问题
	}

	if chain.latestBlock != nil {
		block.header.PreHash = chain.latestBlock.Hash
		block.header.Height = chain.latestBlock.Height + 1
	}

	blockByte, _ := json.Marshal(block)
	block.header.Hash = common.BytesToHash(chain.hs.Sum(blockByte))

	return *block
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
func (chain *BlockChain) AddBlockOnChain(b Block) int8 {
	preHash := b.header.PreHash
	preBlock, error := chain.blocks.Has(preHash.Bytes())

	//本地无父块，暂不处理
	// todo:可以缓存，等父块到了再add
	if error != nil || !preBlock {
		return -1
	}

	// 验证块是否有问题
	status := chain.VerifyCastingBlock(*b.header)
	if status != 0 {
		return -1
	}

	// 检查高度
	height := b.header.Height

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
func (chain *BlockChain) saveBlock(b Block) int8 {
	chain.lock.Lock()
	defer chain.lock.Unlock()

	// 根据hash存block
	blockJson, err := json.Marshal(b)
	if err != nil {
		return -1
	}
	err = chain.blocks.Put(b.header.Hash.Bytes(), blockJson)
	if err != nil {
		return -1
	}

	// 根据height存blockheader
	headerJson, err := json.Marshal(b.header)
	if err != nil {
		return -1
	}

	err = chain.blockHeight.Put(chain.generateHeightKey(chain.height+1), headerJson)
	if err != nil {
		return -1
	}

	chain.latestBlock = b.header
	chain.height = b.header.Height

	return 0

}

// 链分叉，调整主链
// todo:错误回滚
func (chain *BlockChain) adjust(b Block) int8 {
	chain.lock.Lock()
	defer chain.lock.Unlock()

	header := chain.QueryBlockByHeight(b.header.Height)
	if header == nil {
		return -1
	}

	// todo:判断权重，决定是否要替换
	if chain.weight(header, b.header) {
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
	return true
}

// 删除块
func (chain *BlockChain) remove(header *BlockHeader) {
	chain.blocks.Delete(header.Hash.Bytes())
	chain.blockHeight.Delete(chain.generateHeightKey(header.Height))
}
