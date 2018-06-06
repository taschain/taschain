package core

import (
	"common"
	"core/datasource"
	"encoding/binary"
	"encoding/json"
	"time"
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
	"bytes"
	"consensus/groupsig"
	"log"
	"network/p2p"
	"middleware"
)

const (
	BLOCK_STATUS_KEY                    = "bcurrent"
	CONFIG_SEC                          = "chain"
	CHAIN_BLOCK_HASH_INIT_LENGTH uint64 = 10

	BLOCK_CHAIN_ADJUST_TIME_OUT = 10 * time.Second
)

var BlockChainImpl *BlockChain

// 配置
type BlockChainConfig struct {
	block string

	blockHeight string

	state string

	//组内能出的最大QN值
	qn uint64
}

type BlockChain struct {
	config *BlockChainConfig

	// key: blockhash, value: block
	blocks ethdb.Database
	//key: height, value: blockHeader
	blockHeight ethdb.Database

	transactionPool *TransactionPool

	//已上链的最新块
	latestBlock   *BlockHeader
	topBlocks     *lru.Cache
	latestStateDB *state.StateDB

	// 当前链的高度，其值等于当前链里有多少块（创始块不计入）
	// 与最高块的关系是：chain.height = latestBlock.Height
	//height uint64

	// 读写锁
	lock middleware.Loglock

	// 是否可以工作
	init bool

	statedb    ethdb.Database
	stateCache state.Database // State database to reuse between imports (contains state cache)

	executor      *EVMExecutor
	voteProcessor VoteProcessor

	blockCache *lru.Cache

	isAdujsting bool

	solitaryBlocks *lru.Cache
}

type castingBlock struct {
	state    *state.StateDB
	receipts types.Receipts
}

// 默认配置
func DefaultBlockChainConfig() *BlockChainConfig {
	return &BlockChainConfig{
		block: "block",

		blockHeight: "height",

		state: "state",

		qn: 4,
	}
}

func getBlockChainConfig() *BlockChainConfig {
	defaultConfig := DefaultBlockChainConfig()
	if nil == common.GlobalConf {
		return defaultConfig
	}

	return &BlockChainConfig{
		block: common.GlobalConf.GetString(CONFIG_SEC, "block", defaultConfig.block),

		blockHeight: common.GlobalConf.GetString(CONFIG_SEC, "blockHeight", defaultConfig.blockHeight),

		state: common.GlobalConf.GetString(CONFIG_SEC, "state", defaultConfig.state),

		qn: uint64(common.GlobalConf.GetInt(CONFIG_SEC, "qn", int(defaultConfig.qn))),
	}

}

func initBlockChain() error {

	chain := &BlockChain{
		config:          getBlockChainConfig(),
		transactionPool: NewTransactionPool(),
		latestBlock:     nil,

		lock:        middleware.NewLoglock("chain"),
		init:        true,
		isAdujsting: false,
	}

	var err error
	chain.blockCache, err = lru.New(1000)
	chain.topBlocks, _ = lru.New(1000)
	if err != nil {
		return err
	}
	chain.solitaryBlocks, _ = lru.New(100)

	//从磁盘文件中初始化leveldb
	chain.blocks, err = datasource.NewDatabase(chain.config.block)
	if err != nil {
		//todo: 日志
		return err
	}

	chain.blockHeight, err = datasource.NewDatabase(chain.config.blockHeight)
	if err != nil {
		//todo: 日志
		return err
	}

	chain.statedb, err = datasource.NewDatabase(chain.config.state)
	if err != nil {
		//todo: 日志
		return err
	}
	chain.stateCache = state.NewDatabase(chain.statedb)

	chain.executor = NewEVMExecutor(chain)

	// 恢复链状态 height,latestBlock
	// todo:特殊的key保存最新的状态，当前写到了ldb，有性能损耗
	chain.latestBlock = chain.queryBlockHeaderByHeight([]byte(BLOCK_STATUS_KEY), false)
	if nil != chain.latestBlock {
		chain.buildCache(chain.topBlocks)
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

func Clear() {
	path := datasource.DEFAULT_FILE
	if nil != common.GlobalConf {
		path = common.GlobalConf.GetString(CONFIG_SEC, "database", datasource.DEFAULT_FILE)
	}
	os.RemoveAll(path)

}

func (chain *BlockChain) Close() {
	chain.statedb.Close()
}

func (chain *BlockChain) SetVoteProcessor(processor VoteProcessor) {
	chain.lock.Lock("SetVoteProcessor")
	defer chain.lock.Unlock("SetVoteProcessor")

	chain.voteProcessor = processor
}

func (chain *BlockChain) Height() uint64 {
	if nil == chain.latestBlock {
		return math.MaxUint64
	}
	return chain.latestBlock.Height
}

func (chain *BlockChain) TotalQN() uint64 {
	if nil == chain.latestBlock {
		return 0
	}
	return chain.latestBlock.TotalQN
}

func (chain *BlockChain) LatestStateDB() *state.StateDB {
	return chain.latestStateDB
}

func (chain *BlockChain) IsAdujsting() bool {
	return chain.isAdujsting
}

func (chain *BlockChain) SetAdujsting(isAjusting bool) {
	chain.isAdujsting = isAjusting
}

//获取当前从height到本地最新的所有的块
//进行HASH校验，如果请求结点和当前结点在同一条链上面 则返回对应的块，否则返回本地block hash信息 通知请求结点进行链调整
func (chain *BlockChain) GetBlockMessage(height uint64, hash common.Hash) *BlockMessage {
	chain.lock.RLock("GetBlockMessage")
	defer chain.lock.RUnlock("GetBlockMessage")
	if chain.isAdujsting {
		return nil
	}
	localHeight := chain.latestBlock.Height

	bh := chain.QueryBlockByHeight(height)
	if bh != nil && bh.Hash == hash {
		//当前结点和请求结点在同一条链上
		log.Printf("[BlockChain]GetBlockMessage:Self is on the same branch with request node!\n")
		blocks := make([]*Block, 0)

		for i := height + 1; i <= localHeight; i++ {
			bh := chain.queryBlockHeaderByHeight(i, true)
			if nil == bh {
				continue
			}
			b := chain.queryBlockByHash(bh.Hash)
			if nil == b {
				continue
			}

			blocks = append(blocks, b)
		}

		return &BlockMessage{
			Blocks: blocks,
		}
	} else {
		//当前结点和请求结点不在同一条链
		log.Printf("[BlockChain]GetBlockMessage:Self is not on the same branch with request node!\n")
		var cbh []*ChainBlockHash
		if height > localHeight {
			cbh = GetBlockHashesFromLocalChain(localHeight, CHAIN_BLOCK_HASH_INIT_LENGTH)
		} else {
			cbh = GetBlockHashesFromLocalChain(height, CHAIN_BLOCK_HASH_INIT_LENGTH)
		}
		return &BlockMessage{
			BlockHashes: cbh,
		}
	}
}

//从当前链上获取block hash
//height 起始高度
//length 起始高度向下算非空块的长度
func GetBlockHashesFromLocalChain(height uint64, length uint64) []*ChainBlockHash {
	if BlockChainImpl.isAdujsting {
		return nil
	}
	var i uint64
	r := make([]*ChainBlockHash, 0)
	for i = 0; i < length; {
		if height < 0 {
			break
		}
		bh := BlockChainImpl.queryBlockHeaderByHeight(height, true)
		if bh == nil {
			height--
			continue
		} else {
			cbh := ChainBlockHash{Hash: bh.Hash, Height: bh.Height}
			r = append(r, &cbh)
			i++
		}
	}
	return r
}

//func (chain *BlockChain) AddBlockMessage(bm BlockMessage) error {
//	blocks := bm.Blocks
//	if blocks != nil || len(blocks) != 0 {
//		for _, block := range blocks {
//			code := chain.AddBlockOnChain(block)
//			if code < 0 {
//				return fmt.Errorf("fail to add, code:%d", code)
//			}
//		}
//	} else {
//		blockHashes := bm.BlockHashes
//		if blockHashes == nil{
//			return ErrNil
//		}
//		height, hasCommonAncestor := FindCommonAncestor(blockHashes,0,len(blockHashes)-1)
//		if hasCommonAncestor{
//
//		}else {
//			RequestBlockChainHashes()
//		}
//	}
//	return nil
//}

func FindCommonAncestor(cbhr []*ChainBlockHash, l int, r int) (*ChainBlockHash, bool) {

	if l > r || r < 0 || l >= len(cbhr) {
		return nil, false
	}
	m := (l + r) / 2
	result := isCommonAncestor(cbhr, m)
	if result == 0 {
		return cbhr[m], true
	}

	if result == 1 {
		return FindCommonAncestor(cbhr, l, m-1)
	}

	if result == -1 {
		return FindCommonAncestor(cbhr, m+1, r)
	}
	return nil, false
}

//返回值
// 0  当前HASH相等，后面一块HASH不相等
//1   当前HASH相等，后面一块HASH相等
//-1  当前HASH不相等
//-100 参数不合法
func isCommonAncestor(cbhr []*ChainBlockHash, index int) int {
	if index < 0 || index >= len(cbhr) {
		return -100
	}
	he := cbhr[index]
	bh := BlockChainImpl.queryBlockHeaderByHeight(he.Height, true)
	if bh == nil {
		log.Printf("[BlockChain]isCommonAncestor:Height:%d,local hash:%s,coming hash:%s\n", he.Height, "null", he.Hash)
		return -1
	}
	log.Printf("[BlockChain]isCommonAncestor:Height:%d,local hash:%s,coming hash:%s\n", he.Height, bh.Hash, he.Hash)
	if index == 0 && bh.Hash == he.Hash {
		return 0
	}
	//判断更高的一块
	afterHe := cbhr[index-1]
	afterbh := BlockChainImpl.queryBlockHeaderByHeight(afterHe.Height, true)
	if afterbh == nil {
		log.Printf("[BlockChain]isCommonAncestor:after block height:%d,local hash:%s,coming hash:%s\n", afterHe.Height, "null", afterHe.Hash)
		return -1
	}
	log.Printf("[BlockChain]isCommonAncestor:after block height:%d,local hash:%s,coming hash:%s\n", afterHe.Height, afterbh.Hash, afterHe.Hash)
	if afterHe.Hash != afterbh.Hash && bh.Hash == he.Hash {
		return 0
	}
	if afterHe.Hash == afterbh.Hash && bh.Hash == he.Hash {
		return 1
	}
	return -1
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
	chain.lock.Lock("Clear")
	defer chain.lock.Unlock("Clear")

	chain.init = false
	chain.latestBlock = nil
	chain.topBlocks, _ = lru.New(1000)

	//err := chain.blockHeight.Clear()
	//if nil != err {
	//	return err
	//}
	//err = chain.blocks.Clear()
	//if nil != err {
	//	return err
	//}
	var err error

	chain.blocks.Close()
	chain.blockHeight.Close()
	chain.statedb.Close()

	os.RemoveAll(datasource.DEFAULT_FILE)

	chain.statedb, err = datasource.NewDatabase(chain.config.state)
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
		t, _ := chain.transactionPool.GetTransaction(hash)
		if t == nil {
			return nil
		}
		block.Transactions[i] = t
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
	chain.lock.RLock("QueryTopBlock")
	defer chain.lock.RUnlock("QueryTopBlock")

	return chain.latestBlock
}

//根据指定哈希查询块
func (chain *BlockChain) QueryBlockByHash(hash common.Hash) *BlockHeader {
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

// 根据指定高度查询块
func (chain *BlockChain) queryBlockHeaderByHeight(height interface{}, cache bool) *BlockHeader {
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
					return result.(*BlockHeader)
				}

			}
		}

		key = chain.generateHeightKey(height.(uint64))
	}

	// 从持久化存储中查询
	result, err := chain.blockHeight.Get(key)
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

// 根据指定高度查询块
// 带有缓存
func (chain *BlockChain) QueryBlockByHeight(height uint64) *BlockHeader {
	chain.lock.RLock("QueryBlockByHeight")
	defer chain.lock.RUnlock("QueryBlockByHeight")

	return chain.queryBlockHeaderByHeight(height, true)
}

func (chain *BlockChain) CastingBlockAfter(latestBlock *BlockHeader, height uint64, nonce uint64, queueNumber uint64, castor []byte, groupid []byte) *Block {

	//校验高度
	if latestBlock != nil && height <= latestBlock.Height {
		log.Printf("[block] fail to cast block: height problem. height:%d, latest:%d", height, latestBlock.Height)
		return nil
	}

	block := new(Block)

	block.Transactions = chain.transactionPool.GetTransactionsForCasting()
	block.Header = &BlockHeader{
		CurTime:     time.Now(), //todo:时区问题
		Height:      height,
		Nonce:       nonce,
		QueueNumber: queueNumber,
		Castor:      castor,
		GroupId:     groupid,
		TotalQN:     latestBlock.TotalQN + queueNumber,
	}

	if latestBlock != nil {
		block.Header.PreHash = latestBlock.Hash
		block.Header.PreTime = latestBlock.CurTime
	}

	state, err := state.New(c.BytesToHash(latestBlock.StateTree.Bytes()), chain.stateCache)
	if err != nil {
		var buffer bytes.Buffer
		buffer.WriteString("fail to new statedb, lateset height: ")
		buffer.WriteString(fmt.Sprintf("%d", latestBlock.Height))
		buffer.WriteString(", block height: ")
		buffer.WriteString(fmt.Sprintf("%d", block.Header.Height))
		panic(buffer.String())

	}

	// Process block using the parent state as reference point.
	receipts, _, _, statehash, _ := chain.executor.Execute(state, block, chain.voteProcessor)

	// 准确执行了的交易，入块
	// 失败的交易也要从池子里，去除掉
	//block.Header.Transactions = make([]common.Hash, len(executed))
	//executedTxs := make([]*Transaction, len(executed))
	//for i, tx := range executed {
	//	if tx == nil {
	//		continue
	//	}
	//	block.Header.Transactions[i] = tx.Hash
	//	executedTxs[i] = tx
	//}
	//block.Transactions = executedTxs
	//block.Header.EvictedTxs = errTxs

	block.Header.Transactions = make([]common.Hash, len(block.Transactions))
	for i, tx := range block.Transactions {
		block.Header.Transactions[i] = tx.Hash
	}
	block.Header.EvictedTxs = []common.Hash{}
	block.Header.TxTree = calcTxTree(block.Transactions)
	block.Header.StateTree = common.BytesToHash(statehash.Bytes())
	block.Header.ReceiptTree = calcReceiptsTree(receipts)
	block.Header.Hash = block.Header.GenHash()

	chain.blockCache.Add(block.Header.Hash, &castingBlock{
		state:    state,
		receipts: receipts,
	})

	log.Printf("[block]cast block success. blockheader: %x\n", block.Header.Hash)
	return block
}

//构建一个铸块（组内当前铸块人同步操作）
func (chain *BlockChain) CastingBlock(height uint64, nonce uint64, queueNumber uint64, castor []byte, groupid []byte) *Block {
	return chain.CastingBlockAfter(chain.latestBlock, height, nonce, queueNumber, castor, groupid)

}

//验证一个铸块（如本地缺少交易，则异步网络请求该交易）
//返回:=0, 验证通过；=-1，验证失败；=1，缺少交易，已异步向网络模块请求;=2 当前链正在调整，无法检验
func (chain *BlockChain) VerifyCastingBlock(bh BlockHeader) ([]common.Hash, int8, *state.StateDB, types.Receipts) {
	chain.lock.Lock("VerifyCastingBlock")
	defer chain.lock.Unlock("VerifyCastingBlock")

	if chain.isAdujsting {
		return nil, 2, nil, nil
	}
	return chain.verifyCastingBlock(bh)
}

func (chain *BlockChain) verifyCastingBlock(bh BlockHeader) ([]common.Hash, int8, *state.StateDB, types.Receipts) {

	log.Printf("[block] start to verifyCastingBlock, %x\n", bh.Hash)
	// 校验父亲块
	preHash := bh.PreHash
	preBlock := chain.queryBlockHeaderByHash(preHash)

	//本地无父块，暂不处理
	// todo:可以缓存，等父块到了再add
	if preBlock == nil {

		log.Printf("[block]fail to query preBlock, hash:%s \n", preHash)

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
	txtree := calcTxTree(transactions).Bytes()
	if hexutil.Encode(txtree) != hexutil.Encode(bh.TxTree.Bytes()) {
		log.Printf("[block]fail to verify txtree, hash1:%s hash2:%s \n", txtree, bh.TxTree.Bytes())

		return missing, -1, nil, nil
	}

	//执行交易
	state, err := state.New(c.BytesToHash(preBlock.StateTree.Bytes()), chain.stateCache)
	if err != nil {
		log.Printf("[block]fail to new statedb, error:%s \n", err)

		return nil, -1, nil, nil
	} else {
		log.Printf("[block]state.new %d\n", preBlock.StateTree.Bytes())
	}

	b := new(Block)
	b.Header = &bh
	b.Transactions = transactions

	receipts, _, _, statehash, _ := chain.executor.Execute(state, b, chain.voteProcessor)
	if hexutil.Encode(statehash.Bytes()) != hexutil.Encode(bh.StateTree.Bytes()) {
		log.Printf("[block]fail to verify statetree, hash1:%x hash2:%x \n", statehash.Bytes(), b.Header.StateTree.Bytes())

		return nil, -1, nil, nil
	}
	receiptsTree := calcReceiptsTree(receipts).Bytes()
	if hexutil.Encode(receiptsTree) != hexutil.Encode(b.Header.ReceiptTree.Bytes()) {
		log.Printf("[block]fail to verify receipt, hash1:%s hash2:%s \n", receiptsTree, b.Header.ReceiptTree.Bytes())

		return nil, 1, nil, nil
	}

	chain.blockCache.Add(bh.Hash, &castingBlock{
		state:    state,
		receipts: receipts,
	})
	return nil, 0, state, receipts
}

//铸块成功，上链
//返回值: 0,上链成功
//       -1，验证失败
//        1,链上已存在该块，丢弃该块
//        2，未上链，缓存该块
//        3 未上链，异步进行分叉调整
func (chain *BlockChain) addBlockOnChain(b *Block) int8 {

	// 验证块是否已经在链上
	existed := chain.queryBlockByHash(b.Header.Hash)
	if nil != existed {
		return 1
	}

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
		_, status, state, receipts = chain.verifyCastingBlock(*b.Header)
		if status != 0 {
			log.Printf("[BlockChain]fail to VerifyCastingBlock, reason code:%d \n", status)
			return -1
		}
	}

	// 完美情况
	if b.Header.PreHash == chain.latestBlock.Hash {
		status = chain.saveBlock(b)
	} else {
		height := b.Header.Height
		if height > chain.Height() && chain.queryBlockByHash(b.Header.PreHash) == nil {
			chain.solitaryBlocks.Add(b.Header.PreHash, b)
			return 2
		} else {
			//出现分叉，进行链调整
			chain.isAdujsting = true
			log.Printf("[BlockChain]Local block chain has fork,adjusting...\n")
			go chain.adjust(b)

			go func() {
				t := time.NewTimer(BLOCK_CHAIN_ADJUST_TIME_OUT)

				<-t.C
				log.Printf("[BlockChain]Local block adjusting time up.change the state!\n")
				chain.isAdujsting = false
			}()
			status = 3
		}
	}

	// 上链成功，移除pool中的交易
	if 0 == status {
		chain.transactionPool.Remove(b.Header.Transactions)
		chain.transactionPool.Remove(b.Header.EvictedTxs)
		chain.transactionPool.AddExecuted(receipts, b.Transactions)
		chain.latestStateDB = state
		root, _ := state.Commit(true)
		triedb := chain.stateCache.TrieDB()
		triedb.Commit(root, false)

		solitaryBlock, r := chain.solitaryBlocks.Get(b.Header.Hash)
		if r {
			chain.addBlockOnChain(solitaryBlock.(*Block))
		}
	}
	return status

}

func (chain *BlockChain) AddBlockOnChain(b *Block) int8 {
	chain.lock.Lock("AddBlockOnChain")
	defer chain.lock.Unlock("AddBlockOnChain")

	return chain.addBlockOnChain(b)
}

// 保存block到ldb
// todo:错误回滚
func (chain *BlockChain) saveBlock(b *Block) int8 {
	// 根据hash存block
	blockJson, err := json.Marshal(b)
	if err != nil {
		log.Printf("[block]fail to json Marshal, error:%s \n", err)
		return -1
	}
	err = chain.blocks.Put(b.Header.Hash.Bytes(), blockJson)
	if err != nil {
		log.Printf("[block]fail to put key:hash value:block, error:%s \n", err)
		return -1
	}

	// 根据height存blockheader
	headerJson, err := json.Marshal(b.Header)
	if err != nil {

		log.Printf("[block]fail to json Marshal header, error:%s \n", err)
		return -1
	}

	err = chain.blockHeight.Put(chain.generateHeightKey(b.Header.Height), headerJson)
	if err != nil {
		log.Printf("[block]fail to put key:height value:headerjson, error:%s \n", err)
		return -1
	}

	// 持久化保存最新块信息
	chain.latestBlock = b.Header
	chain.topBlocks.Add(b.Header.Height, b.Header)
	err = chain.blockHeight.Put([]byte(BLOCK_STATUS_KEY), headerJson)
	if err != nil {
		fmt.Printf("[block]fail to put current, error:%s \n", err)
		return -1
	}

	return 0
}

// 链分叉，调整主链
// todo:错误回滚
func (chain *BlockChain) adjust(b *Block) {
	chain.lock.Lock("adjust")
	defer chain.lock.Unlock("adjust")
	localTotalQN := chain.TotalQN()
	bTotalQN := b.Header.TotalQN
	if localTotalQN >= bTotalQN {
		log.Printf("[BlockChain]do  not adjust, local chain's total qn: %d,is greater than coming total qn:%d \n", localTotalQN, bTotalQN)
		return
	} else {
		//删除自身链的结点
		for height := b.Header.Height; height <= chain.latestBlock.Height; height++ {
			header := chain.queryBlockHeaderByHeight(height, true)
			if header == nil {
				continue
			}
			chain.remove(header)
			chain.topBlocks.Remove(header.Height)
		}

		var castorId groupsig.ID
		error := castorId.Deserialize(b.Header.Castor)
		if error != nil {
			log.Printf("[BlockChain]Give up ajusting bolck chain because of groupsig id deserialize error:%s", error.Error())
			return
		}
		cbhr := ChainBlockHashesReq{Height: b.Header.Height, Length: CHAIN_BLOCK_HASH_INIT_LENGTH}
		log.Printf("[BlockChain]adjust block chain from %s,Base height:%d,Length:%d", p2p.ConvertToPeerID(castorId.GetString()), b.Header.Height, CHAIN_BLOCK_HASH_INIT_LENGTH)
		RequestBlockChainHashes(castorId.GetString(), cbhr)
	}

}

func (chain *BlockChain) generateHeightKey(height uint64) []byte {
	h := make([]byte, 8)
	binary.BigEndian.PutUint64(h, height)
	return h
}

// 判断两个块所在的链的权重，取权重最大的块所在的链进行同步
func (chain *BlockChain) weight(current *BlockHeader, candidate *BlockHeader) bool {
	return current.TotalQN > candidate.TotalQN
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

func (chain *BlockChain) buildCache(cache *lru.Cache) {
	for i := chain.getStartIndex(); i < chain.latestBlock.Height; i++ {
		chain.topBlocks.Add(i, chain.queryBlockHeaderByHeight(i, false))
	}

}

func (chain *BlockChain) getStartIndex() uint64 {
	var start uint64
	height := chain.latestBlock.Height
	if height < 1000 {
		start = 0
	} else {
		start = height - 999
	}

	return start
}

func (chain *BlockChain) GetTopBlocks() []*BlockHeader {
	chain.lock.RLock("GetTopBlocks")
	defer chain.lock.RUnlock("GetTopBlocks")

	start := chain.getStartIndex()
	result := make([]*BlockHeader, chain.latestBlock.Height-start+1)

	for i := start; i <= chain.latestBlock.Height; i++ {
		bh, _ := chain.topBlocks.Get(i)
		result[i-start] = bh.(*BlockHeader)
	}

	return result
}
