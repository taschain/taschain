package core

import (
	"storage/tasdb"
	"github.com/hashicorp/golang-lru"
	"middleware/types"
	"storage/account"
	"middleware"
	"common"
	"encoding/binary"
	"math"
	"math/big"
	"middleware/statistics"
	"utility"
	"consensus/groupsig"
	"bytes"
	"time"
)

const BLOCK_CHAIN_ADJUST_TIME_OUT = 5 * time.Second

type prototypeChain struct {
	isLightMiner bool
	blocks       tasdb.Database
	//key: height, value: blockHeader
	blockHeight tasdb.Database

	transactionPool TransactionPool

	//已上链的最新块
	latestBlock   *types.BlockHeader
	latestStateDB *account.AccountDB

	topBlocks *lru.Cache

	// 读写锁
	lock middleware.Loglock

	// 是否可以工作
	init bool

	statedb    tasdb.Database
	stateCache account.AccountDatabase // State database to reuse between imports (contains state cache)

	executor      *TVMExecutor
	voteProcessor VoteProcessor

	futureBlocks   *lru.Cache
	verifiedBlocks *lru.Cache

	isAdujsting bool

	consensusHelper types.ConsensusHelper

	bonusManager *BonusManager

	checkdb tasdb.Database
}

func (chain *prototypeChain) PutCheckValue(height uint64, hash []byte) error {
	key := utility.UInt64ToByte(height)
	return chain.checkdb.Put(key, hash)
}

func (chain *prototypeChain) GetCheckValue(height uint64) (common.Hash, error) {
	key := utility.UInt64ToByte(height)
	raw, err := chain.checkdb.Get(key)
	return common.BytesToHash(raw), err
}

func (chain *prototypeChain) IsLightMiner() bool {
	return chain.isLightMiner
}

func (chain *prototypeChain) GenerateBlock(bh types.BlockHeader) *types.Block {
	block := &types.Block{
		Header: &bh,
	}

	transactions, missing, _ := chain.transactionPool.GetTransactions(bh.Hash, bh.Transactions)
	if len(missing) != 0 {
		panic("GenerateBlock:should not has missing txs!")
	}
	block.Transactions = transactions

	//block.Transactions = make([]*types.Transaction, len(bh.Transactions))
	//for i, hash := range bh.Transactions {
	//	t, _ := chain.transactionPool.GetTransaction(hash)
	//	if t == nil {
	//		return nil
	//	}
	//	block.Transactions[i] = t
	//}
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
			result, ok := chain.topBlocks.Get(h)
			if ok && nil != result {
				return result.(*types.BlockHeader)
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

func (chain *prototypeChain) GetSateCache() account.AccountDatabase {
	return chain.stateCache
}
func (chain *prototypeChain) IsAdujsting() bool {
	return chain.isAdujsting
}

func (chain *prototypeChain) SetAdujsting(isAjusting bool) {
	Logger.Debugf("SetAdujsting %v, topHash=%v, height=%v", isAjusting, chain.latestBlock.Hash.Hex(), chain.latestBlock.Height)
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

func (chain *prototypeChain) LatestStateDB() *account.AccountDB {
	//chain.lock.RLock("LatestStateDB")
	//defer chain.lock.RUnlock("LatestStateDB")
	return chain.latestStateDB
}

func generateHeightKey(height uint64) []byte {
	h := make([]byte, 8)
	binary.BigEndian.PutUint64(h, height)
	return h
}

func (chain *prototypeChain) AddBonusTrasanction(transaction *types.Transaction) {
	chain.GetTransactionPool().AddTransaction(transaction)
}

func (chain *prototypeChain) GetBonusManager() *BonusManager {
	return chain.bonusManager
}

func (chain *prototypeChain) GetConsensusHelper() types.ConsensusHelper {
	return chain.consensusHelper
}

func (chain *prototypeChain) missTransaction(bh types.BlockHeader, txs []*types.Transaction) (bool, []common.Hash, []*types.Transaction) {
	var missing []common.Hash
	var transactions []*types.Transaction
	if nil == txs {
		transactions, missing, _ = chain.transactionPool.GetTransactions(bh.Hash, bh.Transactions)
	} else {
		transactions = txs
	}

	if 0 != len(missing) {
		var castorId groupsig.ID
		error := castorId.Deserialize(bh.Castor)
		if error != nil {
			panic("Groupsig id deserialize error:" + error.Error())
		}
		//向CASTOR索取交易
		m := &TransactionRequestMessage{TransactionHashes: missing, CurrentBlockHash: bh.Hash, BlockHeight: bh.Height, BlockPv: bh.ProveValue,}
		go RequestTransaction(*m, castorId.String())
		return true, missing, transactions
	}
	return false, missing, transactions
}

func (chain *prototypeChain) validateTxRoot(txMerkleTreeRoot common.Hash, txs []*types.Transaction) bool {
	txTree := calcTxTree(txs)

	if !bytes.Equal(txTree.Bytes(), txMerkleTreeRoot.Bytes()) {
		Logger.Errorf("Fail to verify txTree, hash1:%s hash2:%s", txTree.Hex(), txMerkleTreeRoot.Hex())
		return false
	}
	return true
}

func (chain *prototypeChain) validateGroupSig(bh *types.BlockHeader) bool {
	if chain.Height() == 0 {
		return true
	}
	pre := BlockChainImpl.QueryBlockByHash(bh.PreHash)
	if pre == nil {
		time.Sleep(time.Second)
		panic("Pre should not be nil")
	}
	result, err := chain.GetConsensusHelper().VerifyNewBlock(bh, pre.Header)
	if err != nil {
		Logger.Errorf("validateGroupSig error:%s", err.Error())
		return false
	}
	return result
}

func (chain *prototypeChain) GetTraceHeader(hash []byte) *types.BlockHeader {
	traceHeader := TraceChainImpl.GetTraceHeaderByHash(hash)
	if traceHeader == nil {
		statistics.VrfLogger.Debugf("TraceHeader is nil")
		return nil
	}
	if traceHeader.Random == nil || len(traceHeader.Random) == 0 {
		statistics.VrfLogger.Debugf("PreBlock Random is nil")
	} else {
		statistics.VrfLogger.Debugf("PreBlock Random : %v", common.Bytes2Hex(traceHeader.Random))
	}
	return &types.BlockHeader{PreHash: traceHeader.PreHash, Hash: traceHeader.Hash, Random: traceHeader.Random, TotalQN: traceHeader.TotalQn, Height: traceHeader.Height}
}

func (chain *prototypeChain) ProcessChainPiece(id string, chainPiece []*types.BlockHeader, topHeader *types.BlockHeader) {
	chain.lock.Lock("ProcessChainPiece")
	defer chain.lock.Unlock("ProcessChainPiece")

	if topHeader.TotalQN <= chain.latestBlock.TotalQN {
		return
	}

	if chainPiece[0].Height < chain.latestBlock.Height {
		return
	}

	if !chain.verifyChainPiece(chainPiece, topHeader) {
		return
	}
	Logger.Debugf("ProcessChainPiece id:%s,chainPiece %d-%d,topHeader height:%d,totalQn:%d,hash:%v",
		id, chainPiece[len(chainPiece)-1].Height, chainPiece[0].Height, topHeader.Height, topHeader.TotalQN, topHeader.Hash.Hex())
	commonAncestor, hasCommonAncestor, _ := chain.findCommonAncestor(chainPiece, 0, len(chainPiece)-1)
	if hasCommonAncestor {
		Logger.Debugf("[BlockChain]Got common ancestor! Height:%d,localHeight:%d", commonAncestor.Height, chain.Height())
		chain.removeFromCommonAncestor(commonAncestor)
		RequestBlock(id, commonAncestor.Height+1)

		//if topHeader.TotalQN == chain.latestBlock.TotalQN {
		//	if chain.compareValue(commonAncestor, topHeader) {
		//		chain.SetAdujsting(false)
		//		return
		//	}
		//	chain.removeFromCommonAncestor(commonAncestor)
		//	RequestBlock(id, commonAncestor.Height+1)
		//}
	} else {
		Logger.Debugf("[BlockChain]Do not find common ancestor!Request hashes form node:%s,base height:%d", id, chainPiece[len(chainPiece)-1].Height-1, )
		RequestChainPiece(id, chainPiece[len(chainPiece)-1].Height)
	}
}

func (ch prototypeChain) verifyChainPiece(chainPiece []*types.BlockHeader, topHeader *types.BlockHeader) bool {
	//todo
	return true
}

// 删除块 只删除最高块
func (chain *prototypeChain) remove(header *types.BlockHeader) bool {
	if header.Hash != chain.latestBlock.Hash {
		return false
	}

	Logger.Debugf("remove hash:%s height:%d ", header.Hash.Hex(), header.Height)
	hash := header.Hash
	block := BlockChainImpl.QueryBlockByHash(hash)
	chain.topBlocks.Remove(header.Height)
	chain.blocks.Delete(hash.Bytes())
	chain.blockHeight.Delete(generateHeightKey(header.Height))

	chain.latestBlock = BlockChainImpl.QueryBlockHeaderByHash(chain.latestBlock.PreHash)
	chain.latestStateDB, _ = account.NewAccountDB(chain.latestBlock.StateTree, chain.stateCache)

	// 删除块的交易，返回transactionpool
	if nil == block {
		return true
	}
	txs := block.Transactions
	if 0 == len(txs) {
		return true
	}
	chain.transactionPool.UnMarkExecuted(txs)
	return true
}

func (chain *prototypeChain) Remove(header *types.BlockHeader) bool {
	chain.lock.Lock("Remove Top")
	defer chain.lock.Unlock("Remove Top")
	return chain.remove(header)
}

func (chain *prototypeChain) removeFromCommonAncestor(commonAncestor *types.BlockHeader) {
	Logger.Debugf("removeFromCommonAncestor hash:%s height:%d latestheight:%d", commonAncestor.Hash.Hex(), commonAncestor.Height, chain.latestBlock.Height)
	for height := chain.latestBlock.Height; height > commonAncestor.Height; height-- {
		header := chain.queryBlockHeaderByHeight(height, true)
		if header == nil {
			Logger.Debugf("removeFromCommonAncestor nil height:%d", height)
			continue
		}
		chain.remove(header)
		Logger.Debugf("Remove local chain headers %d", header.Height)
	}
}

func (chain *prototypeChain) compareValue(commonAncestor *types.BlockHeader, remoteHeader *types.BlockHeader) bool {
	var localValue *big.Int
	remoteValue := chain.consensusHelper.VRFProve2Value(remoteHeader.ProveValue)
	Logger.Debugf("compareValue hash:%s height:%d latestheight:%d", commonAncestor.Hash.Hex(), commonAncestor.Height, chain.latestBlock.Height)
	for height := commonAncestor.Height + 1; height <= chain.latestBlock.Height; height++ {
		Logger.Debugf("compareValue queryBlockHeaderByHeight height:%d ", height)
		header := chain.queryBlockHeaderByHeight(height, true)
		if header == nil {
			Logger.Debugf("compareValue queryBlockHeaderByHeight nil !height:%d ", height)
			continue
		}
		localValue = chain.consensusHelper.VRFProve2Value(header.ProveValue)
	}
	if localValue == nil {
		time.Sleep(time.Second)
	}
	if localValue.Cmp(remoteValue) >= 0 {
		return true
	}
	return false
}

func (chain *prototypeChain) findCommonAncestor(chainPiece []*types.BlockHeader, l int, r int) (*types.BlockHeader, bool, int) {

	if l > r || r < 0 || l >= len(chainPiece) {
		return nil, false, -1
	}
	m := (l + r) / 2
	result := chain.isCommonAncestor(chainPiece, m)
	if result == 0 {
		return chainPiece[m], true, m
	}

	if result == 1 {
		return chain.findCommonAncestor(chainPiece, l, m-1)
	}

	if result == -1 {
		return chain.findCommonAncestor(chainPiece, m+1, r)
	}
	return nil, false, -1
}

//bhs 中没有空值
//返回值
// 0  当前HASH相等，后面一块HASH不相等 是共同祖先
//1   当前HASH相等，后面一块HASH相等
//-1  当前HASH不相等
//-100 参数不合法
func (chain *prototypeChain) isCommonAncestor(chainPiece []*types.BlockHeader, index int) int {
	if index < 0 || index >= len(chainPiece) {
		return -100
	}
	he := chainPiece[index]

	bh := chain.queryBlockHeaderByHeight(he.Height, true)
	if bh == nil {
		Logger.Debugf("[BlockChain]isCommonAncestor:Height:%d,local hash:%x,coming hash:%x\n", he.Height, nil, he.Hash)
		return -1
	}
	Logger.Debugf("[BlockChain]isCommonAncestor:Height:%d,local hash:%x,coming hash:%x\n", he.Height, bh.Hash, he.Hash)
	if index == 0 && bh.Hash == he.Hash {
		return 0
	}
	if index == 0 {
		return -1
	}
	//判断链更后面的一块
	afterHe := chainPiece[index-1]
	afterbh := chain.queryBlockHeaderByHeight(afterHe.Height, true)
	if afterbh == nil {
		Logger.Debugf("[BlockChain]isCommonAncestor:after block height:%d,local hash:%s,coming hash:%x\n", afterHe.Height, "null", afterHe.Hash)
		if afterHe != nil && bh.Hash == he.Hash {
			return 0
		}
		return -1
	}
	Logger.Debugf("[BlockChain]isCommonAncestor:after block height:%d,local hash:%x,coming hash:%x\n", afterHe.Height, afterbh.Hash, afterHe.Hash)
	if afterHe.Hash != afterbh.Hash && bh.Hash == he.Hash {
		return 0
	}
	if afterHe.Hash == afterbh.Hash && bh.Hash == he.Hash {
		return 1
	}
	return -1
}

func (chain *prototypeChain) updateLastBlock(state *account.AccountDB, header *types.BlockHeader, headerJson []byte) int8 {
	err := chain.blockHeight.Put([]byte(BLOCK_STATUS_KEY), headerJson)
	if err != nil {
		Logger.Errorf("[block]fail to put current, error:%s \n", err)
		return -1
	}
	chain.latestStateDB = state
	chain.latestBlock = header

	Logger.Debugf("blockchain update latestStateDB:%s height:%d", header.StateTree.Hex(), header.Height)
	return 0
}
