package core

import (
	"github.com/hashicorp/golang-lru"
	"middleware"
	"middleware/types"

	"taslog"
	"core/datasource"
	"storage/core"
	"common"
	"bytes"
	"consensus/groupsig"
	"middleware/notify"
	"network"
)

type LightChain struct {
	config *LightChainConfig
	FullChain
}

// 配置
type LightChainConfig struct {
	blockHeight string

	state string

	//组内能出的最大QN值
	qn uint64
}

func getLightChainConfig() *LightChainConfig {
	defaultConfig := DefaultLightChainConfig()
	if nil == common.GlobalConf {
		return defaultConfig
	}

	return &LightChainConfig{
		blockHeight: common.GlobalConf.GetString(CONFIG_SEC, "blockHeight", defaultConfig.blockHeight),

		state: common.GlobalConf.GetString(CONFIG_SEC, "state", defaultConfig.state),

		qn: uint64(common.GlobalConf.GetInt(CONFIG_SEC, "qn", int(defaultConfig.qn))),
	}
}

// 默认配置
func DefaultLightChainConfig() *LightChainConfig {
	return &LightChainConfig{
		blockHeight: "height",
		state: "state",
		qn: 4,
	}
}

func initLightChain() error {
	Logger = taslog.GetLoggerByName("core" + common.GlobalConf.GetString("instance", "index", ""))

	chain := &LightChain{
		config:          getLightChainConfig(),
		FullChain:FullChain{
			transactionPool: NewLightTransactionPool(),
			latestBlock:     nil,

			lock:        middleware.NewLoglock("lightchain"),
			init:        true,
			isAdujsting: false,
		},
	}

	var err error
	chain.blockCache, err = lru.New(20)
	chain.topBlocks, _ = lru.New(100)
	if err != nil {
		return err
	}
	chain.blockHeight, err = datasource.NewDatabase(chain.config.blockHeight)
	if err != nil {
		Logger.Error("[LightChain initLightChain Error!Msg=%v]",err)
		return err
	}
	chain.statedb, err = datasource.NewLRUMemDatabase(5000000)
	if err != nil {
		Logger.Error("[LightChain initLightChain Error!Msg=%v]",err)
		return err
	}
	chain.stateCache = core.NewLightDatabase(chain.statedb)
	chain.executor = NewTVMExecutor(chain)

	// 恢复链状态 height,latestBlock
	// todo:特殊的key保存最新的状态，当前写到了ldb，有性能损耗
	chain.latestBlock = chain.QueryBlockHeaderByHeight([]byte(BLOCK_STATUS_KEY), false)
	if nil != chain.latestBlock {
		chain.buildCache(100,chain.topBlocks)
		Logger.Infof("initLightChain chain.latestBlock.StateTree  Hash:%s",chain.latestBlock.StateTree.Hex())
	} else {
		// 创始块
		state, err := core.NewAccountDB(common.Hash{}, chain.stateCache)
		if nil == err {
			chain.latestStateDB = state
			block := GenesisBlock(state, chain.stateCache.TrieDB())
			chain.saveBlock(block)
		}
	}

	BlockChainImpl = chain
	return nil
}




//清除链所有数据
func (chain *LightChain) Clear() error {
	chain.lock.Lock("Clear")
	defer chain.lock.Unlock("Clear")

	chain.init = false
	chain.latestBlock = nil
	chain.topBlocks, _ = lru.New(1000)

	var err error

	chain.blockHeight.Close()
	chain.statedb.Close()


	chain.statedb, err = datasource.NewDatabase(chain.config.state)
	if err != nil {
		//todo: 日志
		return err
	}

	chain.stateCache = core.NewLightDatabase(chain.statedb)
	chain.executor = NewTVMExecutor(chain)

	// 创始块
	state, err := core.NewAccountDB(common.Hash{}, chain.stateCache)
	if nil == err {
		chain.latestStateDB = state
		block := GenesisBlock(state, chain.stateCache.TrieDB())

		chain.saveBlock(block)
	}

	chain.init = true

	chain.transactionPool.Clear()
	return err
}



//轻节点不出块
func (chain *LightChain) CastingBlock(height uint64, nonce uint64, queueNumber uint64, castor []byte, groupid []byte) *types.Block {
	return nil
}

//验证一个铸块（如本地缺少交易，则异步网络请求该交易）
//返回值:
// 0, 验证通过
// -1，验证失败
// 1 无法验证（缺少交易，已异步向网络模块请求）
// 2 无法验证（前一块在链上不存存在）
func (chain *LightChain) VerifyCastingBlock(bh types.BlockHeader) ([]common.Hash, int8, *core.AccountDB, vtypes.Receipts) {
	chain.lock.Lock("VerifyCastingBlock")
	defer chain.lock.Unlock("VerifyCastingBlock")

	return chain.verifyCastingBlock(bh, nil)
}

func (chain *LightChain) verifyCastingBlock(bh types.BlockHeader, txs []*types.Transaction) ([]common.Hash, int8, *core.AccountDB, vtypes.Receipts) {
	// 校验父亲块
	preHash := bh.PreHash
	preBlock := chain.queryBlockHeaderByHash(preHash)

	if preBlock == nil {
		return nil, 2, nil, nil
	}

	// 验证交易
	var (
		transactions []*types.Transaction
		missing      []common.Hash
	)
	if nil == txs {
		transactions, missing, _ = chain.transactionPool.GetTransactions(bh.Hash, bh.Transactions)
	} else {
		transactions = txs
	}

	if 0 != len(missing) {

		var castorId groupsig.ID
		error := castorId.Deserialize(bh.Castor)
		if error != nil {
			log.Printf("[BlockChain]Give up request txs because of groupsig id deserialize error:%s", error.Error())
			return nil, 1, nil, nil
		}

		//向CASTOR索取交易
		m := &TransactionRequestMessage{
			TransactionHashes: missing,
			CurrentBlockHash:  bh.Hash,
			BlockHeight:       bh.Height,
			BlockQn:           bh.QueueNumber,
		}
		go RequestTransaction(*m, castorId.String())
		return missing, 1, nil, nil
	}

	txtree := calcTxTree(transactions)

	if !bytes.Equal(txtree.Bytes(),bh.TxTree.Bytes()) {
		Logger.Debugf("[BlockChain]fail to verify txtree, hash1:%s hash2:%s", txtree.Hex(), bh.TxTree.Hex())
		return missing, -1, nil, nil
	}

	//执行交易
	preRoot := common.BytesToHash(preBlock.StateTree.Bytes())
	if len(txs) > 0 {
		Logger.Infof("NewAccountDB height:%d StateTree:%s preHash:%s preRoot:%s",
			bh.Height, bh.StateTree.Hex(), preHash.Hex(), preRoot.Hex())
	}
	state, err := core.NewAccountDB(preRoot, chain.stateCache)
	if err != nil {
		Logger.Errorf("[BlockChain]fail to new statedb, error:%s", err)
		return nil, -1, nil, nil
	}
	//else {
	//	log.Printf("[BlockChain]state.new %d\n", preBlock.StateTree.Bytes())
	//}

	b := new(types.Block)
	b.Header = &bh
	b.Transactions = transactions

	Logger.Infof("verifyCastingBlock height:%d StateTree Hash:%s",b.Header.Height,b.Header.StateTree.Hex())
	statehash, receipts, err := chain.executor.Execute(state, b, chain.voteProcessor)
	if common.ToHex(statehash.Bytes()) != common.ToHex(bh.StateTree.Bytes()) {
		Logger.Debugf("[BlockChain]fail to verify statetree, hash1:%x hash2:%x", statehash.Bytes(), b.Header.StateTree.Bytes())
		return nil, -1, nil, nil
	}
	//receiptsTree := calcReceiptsTree(receipts).Bytes()
	//if common.ToHex(receiptsTree) != common.ToHex(b.Header.ReceiptTree.Bytes()) {
	//	Logger.Debugf("[BlockChain]fail to verify receipt, hash1:%s hash2:%s", receiptsTree, b.Header.ReceiptTree.Bytes())
	//	return nil, 1, nil, nil
	//}

	chain.blockCache.Add(bh.Hash, &castingBlock{
		state:    state,
		receipts: receipts,
	})
	//return nil, 0, state, receipts
	return nil, 0, state, nil
}





//铸块成功，上链
//返回值: 0,上链成功
//       -1，验证失败
//        1, 丢弃该块(链上已存在该块或链上存在QN值更大的相同高度块)
//        2，未上链(异步进行分叉调整)
func (chain *LightChain) AddBlockOnChain(b *types.Block) int8 {
	if b == nil {
		return -1
	}
	chain.lock.Lock("AddBlockOnChain")
	defer chain.lock.Unlock("AddBlockOnChain")
	//defer network.Logger.Debugf("add on chain block %d-%d,cast+verify+io+onchain cost%v", b.Header.Height, b.Header.QueueNumber, time.Since(b.Header.CurTime))

	return chain.addBlockOnChain(b)
}

func (chain *LightChain) addBlockOnChain(b *types.Block) int8 {

	var (
		state    *core.AccountDB
		receipts vtypes.Receipts
		status   int8
	)

	// 自己铸块的时候，会将块临时存放到blockCache里
	// 当组内其他成员验证通过后，自己上链就无需验证、执行交易，直接上链即可
	cache, _ := chain.blockCache.Get(b.Header.Hash)
	//if false {
	if cache != nil {
		status = 0
		state = cache.(*castingBlock).state
		receipts = cache.(*castingBlock).receipts
		chain.blockCache.Remove(b.Header.Hash)
	} else {
		// 验证块是否有问题
		_, status, state, receipts = chain.verifyCastingBlock(*b.Header, b.Transactions)
		if status != 0 {
			Logger.Errorf("[BlockChain]fail to VerifyCastingBlock, reason code:%d \n", status)
			return -1
		}
	}

	if b.Header.PreHash == chain.latestBlock.Hash {
		status = chain.saveBlock(b)
	} else if b.Header.TotalQN <= chain.latestBlock.TotalQN || b.Header.Hash == chain.latestBlock.Hash {
		return 1
	} else if b.Header.PreHash == chain.latestBlock.PreHash {
		chain.remove(chain.latestBlock)
		status = chain.saveBlock(b)
	} else {
		//b.Header.TotalQN > chain.latestBlock.TotalQN
		if chain.isAdujsting {
			return 2
		}
		var castorId groupsig.ID
		error := castorId.Deserialize(b.Header.Castor)
		if error != nil {
			log.Printf("[BlockChain]Give up ajusting bolck chain because of groupsig id deserialize error:%s", error.Error())
			return -1
		}
		chain.SetAdujsting(true)
		RequestBlockInfoByHeight(castorId.String(), chain.latestBlock.Height, chain.latestBlock.Hash)
		status = 2
	}

	// 上链成功，移除pool中的交易
	if 0 == status {
		chain.transactionPool.Remove(b.Header.Hash, b.Header.Transactions)
		chain.transactionPool.AddExecuted(receipts, b.Transactions)
		chain.latestStateDB = state
		root, _ := state.Commit(true)
		triedb := chain.stateCache.TrieDB()
		triedb.Commit(root, false)

		notify.BUS.Publish(notify.BlockAddSucc, &notify.BlockMessage{Block: *b,})

		h, e := types.MarshalBlockHeader(b.Header)
		if e != nil {
			headerMsg := network.Message{Code:network.NewBlockHeaderMsg,Body:h}
			network.GetNetInstance().Relay(headerMsg,1)
			network.Logger.Debugf("After add on chain,spread block %d-%d header to neighbor,header size %d,hash:%v", b.Header.Height, b.Header.QueueNumber, len(h), b.Header.Hash)
		}

	}
	return status

}


func (chain *LightChain) QueryBlockByHeight(height uint64) *types.BlockHeader {
	return nil
}
