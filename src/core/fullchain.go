package core

import (
	"storage/tasdb"
	"github.com/hashicorp/golang-lru"
	"middleware/types"
	"storage/core"
	"middleware"
	"common"
	vtypes "storage/core/types"
	"math"
	"time"
)

type FullChain struct {

	//key: height, value: blockHeader
	blockHeight tasdb.Database

	transactionPool ITransactionPool

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


func (chain *FullChain) SetAdujsting(isAjusting bool) {
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

//根据哈希取得某个交易
func (chain *FullChain) GetTransactionByHash(h common.Hash) (*types.Transaction, error) {
	return chain.transactionPool.GetTransaction(h)
}

func (chain *FullChain) GenerateBlock(bh types.BlockHeader) *types.Block {
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

func (chain *FullChain) CastingBlock(height uint64, nonce uint64, queueNumber uint64, castor []byte, groupid []byte) *types.Block{
	panic("not expect enter here")
}

func (chain *FullChain) VerifyCastingBlock(bh types.BlockHeader) ([]common.Hash, int8, *core.AccountDB, vtypes.Receipts){
	panic("not expect enter here")
}

func (chain *FullChain) AddBlockOnChain(b *types.Block) int8 {
	panic("not expect enter here")
}

func (chain *FullChain) QueryTopBlock() *types.BlockHeader {
	return nil
}

func (chain *FullChain) QueryBlockByHash(hash common.Hash) *types.BlockHeader {
	return nil
}

func (chain *FullChain) IsAdujsting() bool {
	return chain.isAdujsting
}

func (chain *FullChain) Clear() error {
	panic("not expect enter here")
}

func (chain *FullChain) QueryBlockBody(blockHash common.Hash) []*types.Transaction {
	return nil
}

func(chain *FullChain) QueryBlockByHeight(height uint64) *types.BlockHeader{
	return nil
}

func (chain *FullChain) GetTransactionPool() ITransactionPool {
	return chain.transactionPool
}

func (chain *FullChain) Close() {
	chain.statedb.Close()
}


func (chain *FullChain) getStartIndex(size uint64) uint64 {
	var start uint64
	height := chain.latestBlock.Height
	if height < size {
		start = 0
	} else {
		start = height - (size-1)
	}

	return start
}


func (chain *FullChain) buildCache(size uint64,cache *lru.Cache) {
	for i := chain.getStartIndex(size); i < chain.latestBlock.Height; i++ {
		chain.topBlocks.Add(i, chain.QueryBlockHeaderByHeight(i, false))
	}

}

// 根据指定高度查询块
func (chain *FullChain) QueryBlockHeaderByHeight(height interface{}, cache bool) *types.BlockHeader {
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

		key = GenerateHeightKey(height.(uint64))
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

func (chain *FullChain) SaveBlock(b *types.Block) int8 {
	panic("not expect enter here")
}


func (chain *FullChain) Height() uint64 {
	if nil == chain.latestBlock {
		return math.MaxUint64
	}
	return chain.latestBlock.Height
}


func (chain *FullChain) Remove(header *types.BlockHeader){
	panic("not expect enter here")
}