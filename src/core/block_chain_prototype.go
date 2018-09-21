package core

import (
	"storage/tasdb"
	"github.com/hashicorp/golang-lru"
	"middleware/types"
	"storage/core"
	"middleware"
	"common"
	"math"
	"time"
	"math/big"
)

type prototypeChain struct {
	blocks tasdb.Database
	//key: height, value: blockHeader
	blockHeight tasdb.Database

	transactionPool TransactionPoolI

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

func (chain *prototypeChain) GenerateBlock(bh types.BlockHeader) *types.Block {
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


func (chain *prototypeChain) Height() uint64 {
	if nil == chain.latestBlock {
		return math.MaxUint64
	}
	return chain.latestBlock.Height
}

func (chain *prototypeChain) TotalQN() uint64 {
	if nil == chain.latestBlock {
		return 0
	}
	return chain.latestBlock.TotalQN
}

//查询最高块
func (chain *prototypeChain) QueryTopBlock() *types.BlockHeader {
	chain.lock.RLock("QueryTopBlock")
	defer chain.lock.RUnlock("QueryTopBlock")
	result := *chain.latestBlock
	return &result
}


// 根据指定高度查询块
// 带有缓存
func (chain *prototypeChain) QueryBlockByHeight(height uint64) *types.BlockHeader {
	chain.lock.RLock("QueryBlockByHeight")
	defer chain.lock.RUnlock("QueryBlockByHeight")

	return chain.QueryBlockHeaderByHeight(height, true)
}

// 根据指定高度查询块
func (chain *prototypeChain) QueryBlockHeaderByHeight(height interface{}, cache bool) *types.BlockHeader {
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

//从当前链上获取block hash
//height 起始高度
//length 起始高度向下算非空块的长度
func (chain *prototypeChain) QueryBlockHashes(height uint64, length uint64) []*BlockHash {
	chain.lock.RLock("GetBlockHashesFromLocalChain")
	defer chain.lock.RUnlock("GetBlockHashesFromLocalChain")
	return chain.getBlockHashesFromLocalChain(height, length)
}

func (chain *prototypeChain) getBlockHashesFromLocalChain(height uint64, length uint64) []*BlockHash {
	var i uint64
	r := make([]*BlockHash, 0)
	for i = 0; i < length; {
		bh := BlockChainImpl.QueryBlockHeaderByHeight(height, true)
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


//根据哈希取得某个交易
func (chain *prototypeChain) GetTransactionByHash(h common.Hash) (*types.Transaction, error) {
	return chain.transactionPool.GetTransaction(h)
}

func (chain *prototypeChain) GetTransactionPool() TransactionPoolI {
	return chain.transactionPool
}

//todo 轻节点如何处理？
func (chain *prototypeChain) GetBalance(address common.Address) *big.Int {
	if nil == chain.latestStateDB {
		return nil
	}

	return chain.latestStateDB.GetBalance(common.BytesToAddress(address.Bytes()))
}

//todo 轻节点如何处理？
func (chain *prototypeChain) GetNonce(address common.Address) uint64 {
	if nil == chain.latestStateDB {
		return 0
	}

	return chain.latestStateDB.GetNonce(common.BytesToAddress(address.Bytes()))
}

func (chain *prototypeChain) IsAdujsting() bool {
	return chain.isAdujsting
}


func (chain *prototypeChain) SetAdujsting(isAjusting bool) {
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


func (chain *prototypeChain) Close() {
	chain.statedb.Close()
}




func (chain *prototypeChain) getStartIndex(size uint64) uint64 {
	var start uint64
	height := chain.latestBlock.Height
	if height < size {
		start = 0
	} else {
		start = height - (size-1)
	}

	return start
}


func (chain *prototypeChain) buildCache(size uint64,cache *lru.Cache) {
	for i := chain.getStartIndex(size); i < chain.latestBlock.Height; i++ {
		chain.topBlocks.Add(i, chain.QueryBlockHeaderByHeight(i, false))
	}
}







