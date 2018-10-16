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
	"encoding/binary"
)

type prototypeChain struct {
	isLightMiner bool
	blocks       tasdb.Database
	//key: height, value: blockHeader
	blockHeight tasdb.Database

	transactionPool TransactionPool

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
	genesisInfo *types.GenesisInfo

	bonusManager *BonusManager

}

func (chain *prototypeChain) IsLightMiner() bool {
	return chain.isLightMiner
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

func (chain *prototypeChain) TotalQN()*big.Int {
	if nil == chain.latestBlock {
		return nil
	}
	return chain.latestBlock.ProveValue
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

	return chain.queryBlockHeaderByHeight(height, true)
}

// 根据指定高度查询块
func (chain *prototypeChain) queryBlockHeaderByHeight(height interface{}, cache bool) *types.BlockHeader {
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

		key = generateHeightKey(height.(uint64))
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
		bh := chain.queryBlockHeaderByHeight(height, true)
		if bh != nil {
			cbh := BlockHash{Hash: bh.Hash, Height: bh.Height, Pv: bh.ProveValue}
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

func (chain *prototypeChain) GetTransactionPool() TransactionPool {
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

func (chain *prototypeChain) GetSateCache() core.Database {
	return chain.stateCache
}
func (chain *prototypeChain) IsAdujsting() bool {
	return chain.isAdujsting
}

func (chain *prototypeChain) SetAdujsting(isAjusting bool) {
	Logger.Debugf("aaaaaaaaa SetAdujsting %v, topHash=%v, height=%v", isAjusting, chain.latestBlock.Hash.Hex(), chain.latestBlock.Height)
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
		start = height - (size - 1)
	}

	return start
}

func (chain *prototypeChain) buildCache(size uint64, cache *lru.Cache) {
	for i := chain.getStartIndex(size); i < chain.latestBlock.Height; i++ {
		chain.topBlocks.Add(i, chain.queryBlockHeaderByHeight(i, false))
	}
}


func (chain *prototypeChain) SetLastBlockHash(bh *BlockHash) {
	chain.lastBlockHash = bh
}
func (chain *prototypeChain) LatestStateDB() *core.AccountDB {
	return chain.latestStateDB
}

func generateHeightKey(height uint64) []byte {
	h := make([]byte, 8)
	binary.BigEndian.PutUint64(h, height)
	return h
}

func (chain *prototypeChain) AddBonusTrasanction(transaction *types.Transaction){
	chain.GetTransactionPool().AddTransaction(transaction)
}

func (chain *prototypeChain) GetBonusManager() *BonusManager{
	return chain.bonusManager
}