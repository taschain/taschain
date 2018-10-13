//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package core

import (
	"common"
	"core/datasource"
	"time"
	"os"
	"storage/core"
	vtypes "storage/core/types"
	"github.com/hashicorp/golang-lru"
	"fmt"
	"bytes"
	"consensus/groupsig"
	"log"
	"middleware"
	"middleware/types"
	"taslog"
	"middleware/notify"
	"network"
	"math/big"
)


//非组内信息签名，改成使用ECDSA算法（目前都是使用bn曲线）  @飞鼠
//铸块及奖励分红查看及验证  优先级：0    @小甫
//安全性验证：模拟节点发送请求给组内组外成员，优先级低       @飞鼠
//分叉频率
//块全网蔓延时间统计
//组链去掉成员公钥信息   @小甫
//矿工申请、撤销、退款等验证   优先级：0  @问勤
//vrf阈值   优先级：0  @花生
//铸块次数与权益是否正相关验证     @小甫
//验证组验证次数是否均匀分布    @小甫
//建组过程需要把矿工申请高度考虑进去，以防把同一时间申请的矿工分到一个组
//公链联盟链可通过配置灵活切换
//实现多种共识算法，并可以灵活切换

const (
	BLOCK_STATUS_KEY = "bcurrent"

	CONFIG_SEC = "chain"

	CHAIN_BLOCK_HASH_INIT_LENGTH uint64 = 10

	BLOCK_CHAIN_ADJUST_TIME_OUT = 10 * time.Second
)

var BlockChainImpl BlockChain

var Logger taslog.Logger

// 配置
type BlockChainConfig struct {
	block string

	blockHeight string

	state string

	bonus string

	heavy string

	light string
}

type FullBlockChain struct {
	prototypeChain
	config       *BlockChainConfig
	castedBlock  *lru.Cache
	bonusManager *BonusManager
}

type castingBlock struct {
	state    *core.AccountDB
	receipts vtypes.Receipts
}

func getBlockChainConfig() *BlockChainConfig {
	defaultConfig := &BlockChainConfig{
		block: "block",

		blockHeight: "height",

		state: "state",


		bonus: "bonus",

		light: "light",

		heavy: "heavy",
	}

	if nil == common.GlobalConf {
		return defaultConfig
	}

	return &BlockChainConfig{
		block: common.GlobalConf.GetString(CONFIG_SEC, "block", defaultConfig.block),

		blockHeight: common.GlobalConf.GetString(CONFIG_SEC, "blockHeight", defaultConfig.blockHeight),

		state: common.GlobalConf.GetString(CONFIG_SEC, "state", defaultConfig.state),


		bonus: common.GlobalConf.GetString(CONFIG_SEC, "bonus", defaultConfig.bonus),

		heavy: common.GlobalConf.GetString(CONFIG_SEC, "heavy", defaultConfig.heavy),

		light: common.GlobalConf.GetString(CONFIG_SEC, "light", defaultConfig.light),
	}

}

func initBlockChain(genesisInfo *types.GenesisInfo) error {

	Logger = taslog.GetLoggerByName("core" + common.GlobalConf.GetString("instance", "index", ""))

	chain := &FullBlockChain{
		config: getBlockChainConfig(),
		prototypeChain: prototypeChain{
			transactionPool: NewTransactionPool(),
			latestBlock:     nil,
			lock:            middleware.NewLoglock("chain"),
			init:            true,
			isAdujsting:     false,
			isLightMiner:    false,
			genesisInfo:     genesisInfo,
		},
	}

	var err error
	chain.blockCache, err = lru.New(20)
	chain.topBlocks, _ = lru.New(1000)
	if err != nil {
		return err
	}

	chain.castedBlock, err = lru.New(20)

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

	chain.bonusManager = newBonusManager()
	chain.stateCache = core.NewDatabase(chain.statedb)

	chain.executor = NewTVMExecutor(chain)
	initMinerManager(chain)

	// 恢复链状态 height,latestBlock
	// todo:特殊的key保存最新的状态，当前写到了ldb，有性能损耗
	chain.latestBlock = chain.queryBlockHeaderByHeight([]byte(BLOCK_STATUS_KEY), false)
	if nil != chain.latestBlock {
		chain.buildCache(1000, chain.topBlocks)
		Logger.Infof("initBlockChain chain.latestBlock.StateTree  Hash:%s", chain.latestBlock.StateTree.Hex())
		state, err := core.NewAccountDB(common.BytesToHash(chain.latestBlock.StateTree.Bytes()), chain.stateCache)
		if nil == err {
			chain.latestStateDB = state
		} else {
			panic("initBlockChain NewAccountDB fail:" + err.Error())
		}
	} else {
		// 创始块
		state, err := core.NewAccountDB(common.Hash{}, chain.stateCache)
		if nil == err {
			block := GenesisBlock(state, chain.stateCache.TrieDB(), genesisInfo)
			Logger.Infof("GenesisBlock StateTree:%s", block.Header.StateTree.Hex())
			_, headerJson := chain.saveBlock(block)
			chain.updateLastBlock(state, block.Header, headerJson)
		}
	}

	BlockChainImpl = chain
	return nil
}

//构建一个铸块（组内当前铸块人同步操作）
func (chain *FullBlockChain) CastBlock(height uint64, nonce uint64, proveValue *big.Int, castor []byte, groupid []byte) *types.Block {
	//beginTime := time.Now()
	latestBlock := chain.QueryTopBlock()
	//校验高度
	if latestBlock != nil && height <= latestBlock.Height {
		Logger.Debugf("[BlockChain] fail to cast block: height problem. height:%d, latest:%d", height, latestBlock.Height)
		return nil
	}

	block := new(types.Block)

	block.Transactions = chain.transactionPool.GetTransactionsForCasting()
	totalPV := &big.Int{}
	totalPV.Add(latestBlock.TotalPV, proveValue)
	block.Header = &types.BlockHeader{
		CurTime:    time.Now(), //todo:时区问题
		Height:     height,
		Nonce:      nonce,
		ProveValue: proveValue,
		Castor:     castor,
		GroupId:    groupid,
		TotalPV:    totalPV, //todo:latestBlock != nil?
		StateTree:  common.BytesToHash(latestBlock.StateTree.Bytes()),
	}

	if latestBlock != nil {
		block.Header.PreHash = latestBlock.Hash
		block.Header.PreTime = latestBlock.CurTime
	}
	//defer network.Logger.Debugf("casting block %d-%d cost %v,curtime:%v", height, queueNumber, time.Since(beginTime), block.Header.CurTime)

	//Logger.Infof("CastingBlock NewAccountDB height:%d StateTree Hash:%s",height,latestBlock.StateTree.Hex())
	preRoot := common.BytesToHash(latestBlock.StateTree.Bytes())
	if len(block.Transactions) > 0 {
		Logger.Infof("CastingBlock NewAccountDB height:%d preHash:%s preRoot:%s",
			height, latestBlock.Hash.Hex(), preRoot.Hex())
	}
	state, err := core.NewAccountDB(preRoot, chain.stateCache)
	if err != nil {
		var buffer bytes.Buffer
		buffer.WriteString("fail to new statedb, lateset height: ")
		buffer.WriteString(fmt.Sprintf("%d", latestBlock.Height))
		buffer.WriteString(", block height: ")
		buffer.WriteString(fmt.Sprintf("%d error:", block.Header.Height))
		buffer.WriteString(fmt.Sprint(err))
		panic(buffer.String())

	}

	// Process block using the parent state as reference point.
	statehash, receipts, err := chain.executor.Execute(state, block, height, "casting")

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
	//Logger.Infof("CastingBlock block.Header.TxTree height:%d StateTree Hash:%s",height,statehash.Hex())
	block.Header.StateTree = common.BytesToHash(statehash.Bytes())
	block.Header.ReceiptTree = calcReceiptsTree(receipts)
	block.Header.Hash = block.Header.GenHash()

	chain.blockCache.Add(block.Header.Hash, &castingBlock{
		state:    state,
		receipts: receipts,
	})
	chain.castedBlock.Add(block.Header.Hash, block)
	Logger.Debugf("CastingBlock into cache! Height:%d-%d,Hash:%x,stateHash:%x,len tx:%d", height, block.Header.ProveValue, block.Header.Hash, block.Header.StateTree, len(block.Transactions))

	chain.transactionPool.ReserveTransactions(block.Header.Hash, block.Transactions)
	return block
}

//验证一个铸块（如本地缺少交易，则异步网络请求该交易）
//返回值:
// 0, 验证通过
// -1，验证失败
// 1 无法验证（缺少交易，已异步向网络模块请求）
// 2 无法验证（前一块在链上不存存在）
func (chain *FullBlockChain) VerifyBlock(bh types.BlockHeader) ([]common.Hash, int8, *core.AccountDB, vtypes.Receipts) {
	chain.lock.Lock("VerifyCastingBlock")
	defer chain.lock.Unlock("VerifyCastingBlock")

	return chain.verifyCastingBlock(bh, nil)
}

func (chain *FullBlockChain) verifyCastingBlock(bh types.BlockHeader, txs []*types.Transaction) ([]common.Hash, int8, *core.AccountDB, vtypes.Receipts) {
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
			BlockPv:           bh.ProveValue,
		}
		go RequestTransaction(*m, castorId.String())
		return missing, 1, nil, nil
	}

	txtree := calcTxTree(transactions)

	if !bytes.Equal(txtree.Bytes(), bh.TxTree.Bytes()) {
		Logger.Debugf("[BlockChain]fail to verify txtree, hash1:%s hash2:%s", txtree.Hex(), bh.TxTree.Hex())
		return missing, -1, nil, nil
	}

	//执行交易
	Logger.Debugf("verifyCastingBlock NewAccountDB hash:%s, height:%d", preBlock.StateTree.Hex(), bh.Height)
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

	Logger.Infof("verifyCastingBlock height:%d StateTree Hash:%s", b.Header.Height, b.Header.StateTree.Hex())
	statehash, receipts, err := chain.executor.Execute(state, b, bh.Height, "fullverify")
	if common.ToHex(statehash.Bytes()) != common.ToHex(bh.StateTree.Bytes()) {
		Logger.Debugf("[BlockChain]fail to verify statetree, hash1:%x hash2:%x", statehash.Bytes(), b.Header.StateTree.Bytes())
		return nil, -1, nil, nil
	}
	receiptsTree := calcReceiptsTree(receipts).Bytes()
	if common.ToHex(receiptsTree) != common.ToHex(b.Header.ReceiptTree.Bytes()) {
		Logger.Debugf("[BlockChain]fail to verify receipt, hash1:%s hash2:%s", receiptsTree, b.Header.ReceiptTree.Bytes())
		return nil, 1, nil, nil
	}

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
func (chain *FullBlockChain) AddBlockOnChain(b *types.Block) int8 {
	if b == nil {
		return -1
	}
	chain.lock.Lock("AddBlockOnChain")
	defer chain.lock.Unlock("AddBlockOnChain")
	//defer network.Logger.Debugf("add on chain block %d-%d,cast+verify+io+onchain cost%v", b.Header.Height, b.Header.ProveValue, time.Since(b.Header.CurTime))

	return chain.addBlockOnChain(b)
}

func (chain *FullBlockChain) addBlockOnChain(b *types.Block) int8 {
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
		defer chain.blockCache.Remove(b.Header.Hash)
	} else {
		// 验证块是否有问题
		_, status, state, receipts = chain.verifyCastingBlock(*b.Header, b.Transactions)
		if status != 0 {
			Logger.Errorf("[BlockChain]fail to VerifyCastingBlock, reason code:%d \n", status)
			return -1
		}
	}

	var headerJson []byte
	if b.Header.PreHash == chain.latestBlock.Hash {
		status, headerJson = chain.saveBlock(b)
	} else if b.Header.TotalPV.Cmp(chain.latestBlock.TotalPV) <= 0 || b.Header.Hash == chain.latestBlock.Hash {
		return 1
	} else if b.Header.PreHash == chain.latestBlock.PreHash {
		chain.Remove(chain.latestBlock)
		status, headerJson = chain.saveBlock(b)
	} else {
		//b.Header.TotalPV > chain.latestBlock.TotalPV
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
		RequestBlockInfoByHeight(castorId.String(), chain.latestBlock.Height, chain.latestBlock.Hash, true)
		status = 2
	}

	// 上链成功，移除pool中的交易
	if 0 == status {
		Logger.Debugf("ON chain succ! Height:%d,Hash:%x", b.Header.Height, b.Header.Hash)

		root, _ := state.Commit(true)
		triedb := chain.stateCache.TrieDB()
		triedb.Commit(root, false)
		if chain.updateLastBlock(state, b.Header, headerJson) == -1 {
			return -1
		}
		chain.transactionPool.Remove(b.Header.Hash, b.Header.Transactions)
		chain.transactionPool.MarkExecuted(receipts, b.Transactions)
		notify.BUS.Publish(notify.BlockAddSucc, &notify.BlockMessage{Block: *b,})
		GroupChainImpl.RemoveDismissGroupFromCache(b.Header.Height)

		headerMsg := network.Message{Code: network.NewBlockHeaderMsg, Body: headerJson}
		network.GetNetInstance().Relay(headerMsg, 1)
		network.Logger.Debugf("After add on chain,spread block %d-%d header to neighbor,header size %d,hash:%v", b.Header.Height, b.Header.ProveValue, len(headerJson), b.Header.Hash)
	}
	return status

}

func (chain *FullBlockChain) updateLastBlock(state *core.AccountDB, header *types.BlockHeader, headerJson []byte) int8 {
	err := chain.blockHeight.Put([]byte(BLOCK_STATUS_KEY), headerJson)
	if err != nil {
		fmt.Printf("[block]fail to put current, error:%s \n", err)
		return -1
	}
	chain.latestStateDB = state
	chain.latestBlock = header
	Logger.Debugf("blockchain update latestStateDB:%s height:%d", header.StateTree.Hex(), header.Height)
	return 0
}

//根据指定哈希查询块
func (chain *FullBlockChain) QueryBlockByHash(hash common.Hash) *types.BlockHeader {
	return chain.queryBlockHeaderByHash(hash)
}

func (chain *FullBlockChain) QueryBlockBody(blockHash common.Hash) []*types.Transaction {
	block := chain.queryBlockByHash(blockHash)
	if nil == block {
		return nil
	}
	return block.Transactions
}

func (chain *FullBlockChain) queryBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	block := chain.queryBlockByHash(hash)
	if nil == block {
		return nil
	}
	return block.Header
}

func (chain *FullBlockChain) queryBlockByHash(hash common.Hash) *types.Block {
	result, err := chain.blocks.Get(hash.Bytes())

	if result != nil {
		var block *types.Block
		block, err = types.UnMarshalBlock(result)
		if err != nil || &block == nil {
			return nil
		}
		return block
	} else {
		return nil
	}
}

//进行HASH校验，如果请求结点和当前结点在同一条链上面 返回height到本地高度之间所有的块
//否则返回本地链从height向前开始一定长度的非空块hash 用于查找公公祖先
func (chain *FullBlockChain) QueryBlockInfo(height uint64, hash common.Hash, verifyHash bool) *BlockInfo {
	chain.lock.RLock("GetBlockInfo")
	defer chain.lock.RUnlock("GetBlockInfo")
	localHeight := chain.latestBlock.Height

	bh := chain.queryBlockHeaderByHeight(height, true)
	if bh == nil {
		Logger.Debugf("[QueryBlockInfo]height:%d,bh is nil", height)
	} else {
		Logger.Debugf("[QueryBlockInfo]height:%d,bh hash:%x,hash:%x,verifyHash:%t", height, bh.Hash, hash, verifyHash)
	}
	if (bh != nil && bh.Hash == hash) || !verifyHash {
		//当前结点和请求结点在同一条链上
		Logger.Debugf("[BlockChain]Self is on the same branch with request node!")
		var b *types.Block
		for i := height + 1; i <= chain.Height(); i++ {
			bh := chain.queryBlockHeaderByHeight(i, true)
			if nil == bh {
				continue
			}
			b = chain.queryBlockByHash(bh.Hash)
			if nil == b {
				continue
			}
			break
		}
		if b == nil {
			return nil
		}

		var isTopBlock bool
		if b.Header.Height == chain.Height() {
			isTopBlock = true
		} else {
			isTopBlock = false
		}
		return &BlockInfo{Block: b, IsTopBlock: isTopBlock}
	} else {
		//当前结点和请求结点不在同一条链
		Logger.Debugf("[BlockChain]GetBlockMessage:Self is not on the same branch with request node!")
		var bhs []*BlockHash
		if height >= localHeight {
			bhs = chain.getBlockHashesFromLocalChain(localHeight-1, CHAIN_BLOCK_HASH_INIT_LENGTH)
		} else {
			bhs = chain.getBlockHashesFromLocalChain(height, CHAIN_BLOCK_HASH_INIT_LENGTH)
		}
		return &BlockInfo{ChainPiece: bhs}
	}

}

// 保存block到ldb
// todo:错误回滚
//result code:
// -1 保存失败
// 0 保存成功
func (chain *FullBlockChain) saveBlock(b *types.Block) (int8, []byte) {
	// 根据hash存block
	blockJson, err := types.MarshalBlock(b)
	if err != nil {
		log.Printf("[block]fail to json Marshal, error:%s \n", err)
		return -1, nil
	}
	err = chain.blocks.Put(b.Header.Hash.Bytes(), blockJson)
	if err != nil {
		log.Printf("[block]fail to put key:hash value:block, error:%s \n", err)
		return -1, nil
	}

	// 根据height存blockheader
	headerJson, err := types.MarshalBlockHeader(b.Header)
	if err != nil {

		log.Printf("[block]fail to json Marshal header, error:%s \n", err)
		return -1, nil
	}

	err = chain.blockHeight.Put(generateHeightKey(b.Header.Height), headerJson)
	if err != nil {
		log.Printf("[block]fail to put key:height value:headerjson, error:%s \n", err)
		return -1, nil
	}

	// 持久化保存最新块信息

	chain.topBlocks.Add(b.Header.Height, b.Header)

	return 0, headerJson
}

// 删除块
func (chain *FullBlockChain) Remove(header *types.BlockHeader) {
	hash := header.Hash
	block := chain.queryBlockByHash(hash)
	chain.blocks.Delete(hash.Bytes())
	chain.blockHeight.Delete(generateHeightKey(header.Height))

	// 删除块的交易，返回transactionpool
	if nil == block {
		return
	}
	txs := block.Transactions
	if 0 == len(txs) {
		return
	}
	chain.transactionPool.UnMarkExecuted(txs)
}

//清除链所有数据
func (chain *FullBlockChain) Clear() error {
	chain.lock.Lock("Clear")
	defer chain.lock.Unlock("Clear")

	chain.init = false
	chain.latestBlock = nil
	chain.topBlocks, _ = lru.New(1000)

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

	chain.stateCache = core.NewDatabase(chain.statedb)
	chain.executor = NewTVMExecutor(chain)

	// 创始块
	state, err := core.NewAccountDB(common.Hash{}, chain.stateCache)
	if nil == err {
		chain.latestStateDB = state
		block := GenesisBlock(state, chain.stateCache.TrieDB(), chain.genesisInfo)

		_, headerJson := chain.saveBlock(block)
		chain.updateLastBlock(state, block.Header, headerJson)
	}

	chain.init = true

	chain.transactionPool.Clear()
	return err
}

func (chain *FullBlockChain) CompareChainPiece(bhs []*BlockHash, sourceId string) {
	if bhs == nil || len(bhs) == 0 {
		return
	}
	chain.lock.Lock("CompareChainPiece")
	defer chain.lock.Unlock("CompareChainPiece")
	//Logger.Debugf("[BlockChain] CompareChainPiece get block hashes,length:%d,lowest height:%d", len(bhs), bhs[len(bhs)-1].Height)
	blockHash, hasCommonAncestor, _ := FindCommonAncestor(bhs, 0, len(bhs)-1)
	if hasCommonAncestor {
		Logger.Debugf("[BlockChain]Got common ancestor! Height:%d,localHeight:%d", blockHash.Height, chain.Height())
		//删除自身链的结点
		for height := blockHash.Height + 1; height <= chain.latestBlock.Height; height++ {
			header := chain.queryBlockHeaderByHeight(height, true)
			if header == nil {
				continue
			}
			chain.remove(header)
			chain.topBlocks.Remove(header.Height)
		}
		for h := blockHash.Height; h >= 0; h-- {
			header := chain.queryBlockHeaderByHeight(h, true)
			if header != nil {
				chain.latestBlock = header
				break
			}
		}
		RequestBlockInfoByHeight(sourceId, blockHash.Height, blockHash.Hash, true)
	} else {
		chain.SetLastBlockHash(bhs[0])
		cbhr := BlockHashesReq{Height: bhs[len(bhs)-1].Height, Length: uint64(len(bhs) * 10)}
		Logger.Debugf("[BlockChain]Do not find common ancestor!Request hashes form node:%s,base height:%d,length:%d", sourceId, cbhr.Height, cbhr.Length)
		RequestBlockHashes(sourceId, cbhr)
	}

}

// 删除块
func (chain *FullBlockChain) remove(header *types.BlockHeader) {
	hash := header.Hash
	block := chain.queryBlockByHash(hash)
	chain.blocks.Delete(hash.Bytes())
	chain.blockHeight.Delete(generateHeightKey(header.Height))

	// 删除块的交易，返回transactionpool
	if nil == block {
		return
	}
	txs := block.Transactions
	if 0 == len(txs) {
		return
	}
	chain.transactionPool.UnMarkExecuted(txs)
}

func (chain *FullBlockChain) GetTrieNodesByExecuteTransactions(header *types.BlockHeader, transactions []*types.Transaction, isInit bool) *[]types.StateNode {
	Logger.Debugf("GetTrieNodesByExecuteTransactions height:%d,stateTree:%v", header.Height, header.StateTree)
	var nodesOnBranch = make(map[string]*[]byte)
	state, err := core.NewAccountDBWithMap(header.StateTree, chain.stateCache, nodesOnBranch)
	if err != nil {
		Logger.Infof("GetTrieNodesByExecuteTransactions error,height=%d,hash=%v \n", header.Height, header.StateTree)
		return nil
	}
	chain.executor.GetBranches(state, transactions, nodesOnBranch)

	data := []types.StateNode{}
	for key, value := range nodesOnBranch {
		data = append(data, types.StateNode{Key: ([]byte)(key), Value: *value})
	}
	return &data
}

func (chain *FullBlockChain) InsertStateNode(nodes *[]types.StateNode) {
	panic("Not support!")
}

func (chain *FullBlockChain) GetCastingBlock(hash common.Hash) *types.Block {
	v, ok := chain.castedBlock.Get(hash)
	if !ok {
		return nil
	}
	return v.(*types.Block)
}

func FindCommonAncestor(bhs []*BlockHash, l int, r int) (*BlockHash, bool, int) {

	if l > r || r < 0 || l >= len(bhs) {
		return nil, false, -1
	}
	m := (l + r) / 2
	result := isCommonAncestor(bhs, m)
	if result == 0 {
		return bhs[m], true, m
	}

	if result == 1 {
		return FindCommonAncestor(bhs, l, m-1)
	}

	if result == -1 {
		return FindCommonAncestor(bhs, m+1, r)
	}
	return nil, false, -1
}

//bhs 中没有空值
//返回值
// 0  当前HASH相等，后面一块HASH不相等 是共同祖先
//1   当前HASH相等，后面一块HASH相等
//-1  当前HASH不相等
//-100 参数不合法
func isCommonAncestor(bhs []*BlockHash, index int) int {
	if index < 0 || index >= len(bhs) {
		return -100
	}
	he := bhs[index]
	bh := BlockChainImpl.(*FullBlockChain).queryBlockHeaderByHeight(he.Height, true)
	if bh == nil {
		Logger.Debugf("[BlockChain]isCommonAncestor:Height:%d,local hash:%s,coming hash:%x\n", he.Height, "null", he.Hash)
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
	afterHe := bhs[index-1]
	afterbh := BlockChainImpl.(*FullBlockChain).queryBlockHeaderByHeight(afterHe.Height, true)
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

func Clear() {
	path := datasource.DEFAULT_FILE
	if nil != common.GlobalConf {
		path = common.GlobalConf.GetString(CONFIG_SEC, "database", datasource.DEFAULT_FILE)
	}
	os.RemoveAll(path)

}

func (chain *FullBlockChain) SetVoteProcessor(processor VoteProcessor) {
	chain.lock.Lock("SetVoteProcessor")
	defer chain.lock.Unlock("SetVoteProcessor")

	chain.voteProcessor = processor
}


