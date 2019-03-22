package core

import (
	"storage/account"
	"middleware/notify"
	"middleware/types"
	"common"
	"errors"
	"math/big"
	"time"
	"bytes"
	"fmt"
	"consensus/groupsig"
)

/*
**  Creator: pxf
**  Date: 2019/3/11 下午3:15
**  Description: 
*/

type batchAddBlockCallback func(b *types.Block, ret types.AddBlockResult) bool


//构建一个铸块（组内当前铸块人同步操作）
func (chain *FullBlockChain) CastBlock(height uint64, proveValue *big.Int, proveRoot common.Hash, qn uint64, castor []byte, groupid []byte) *types.Block {
	latestBlock := chain.QueryTopBlock()
	if latestBlock != nil && height <= latestBlock.Height {
		Logger.Info("[BlockChain] fail to cast block: height problem. height:%d, latest:%d", height, latestBlock.Height)
		return nil
	}

	begin := time.Now()
	block := new(types.Block)

	defer func() {
		Logger.Debugf("cast block, height=%v, hash=%v, cost %v", block.Header.Height, block.Header.Hash.String(), time.Since(begin).String())
	}()

	block.Transactions = chain.transactionPool.PackForCast()
	block.Header = &types.BlockHeader{
		CurTime:    time.Now(), //todo:时区问题
		Height:     height,
		ProveValue: proveValue,
		Castor:     castor,
		GroupId:    groupid,
		TotalQN:    latestBlock.TotalQN + qn, //todo:latestBlock != nil?
		StateTree:  common.BytesToHash(latestBlock.StateTree.Bytes()),
		ProveRoot:  proveRoot,
	}

	if latestBlock != nil {
		block.Header.PreHash = latestBlock.Hash
		block.Header.PreTime = latestBlock.CurTime
	}

	preRoot := common.BytesToHash(latestBlock.StateTree.Bytes())

	state, err := account.NewAccountDB(preRoot, chain.stateCache)
	if err != nil {
		var buffer bytes.Buffer
		buffer.WriteString("fail to new statedb, lateset height: ")
		buffer.WriteString(fmt.Sprintf("%d", latestBlock.Height))
		buffer.WriteString(", block height: ")
		buffer.WriteString(fmt.Sprintf("%d error:", block.Header.Height))
		buffer.WriteString(fmt.Sprint(err))
		panic(buffer.String())

	}

	statehash, evictedTxs, transactions, receipts, err, _ := chain.executor.Execute(state, block, height, "casting")
	transactionHashes := make([]common.Hash, len(transactions))

	block.Transactions = transactions
	for i, transaction := range transactions {
		transactionHashes[i] = transaction.Hash
	}
	block.Header.Transactions = transactionHashes
	block.Header.TxTree = calcTxTree(block.Transactions)
	block.Header.EvictedTxs = evictedTxs

	block.Header.StateTree = common.BytesToHash(statehash.Bytes())
	block.Header.ReceiptTree = calcReceiptsTree(receipts)
	block.Header.Hash = block.Header.GenHash()
	defer Logger.Infof("casting block %d,hash:%v,qn:%d,tx:%d,TxTree:%v,proValue:%v,stateTree:%s,prestatetree:%s",
		height, block.Header.Hash.String(), block.Header.TotalQN, len(block.Transactions), block.Header.TxTree.Hex(),
		chain.consensusHelper.VRFProve2Value(block.Header.ProveValue), block.Header.StateTree.String(), preRoot.String())
	//自己铸的块 自己不需要验证
	chain.verifiedBlocks.Add(block.Header.Hash, &castingBlock{
		state:    state,
		receipts: receipts,
	})
	return block
}


func (chain *FullBlockChain) GenerateBlock(bh types.BlockHeader) *types.Block {
	block := &types.Block{
		Header: &bh,
	}

	txs, missTxs := chain.GetTransactions(bh.Hash, bh.Transactions)

	if len(missTxs) != 0 {
		Logger.Debugf("GenerateBlock can not get all txs,return nil block!")
		return nil
	}
	block.Transactions = txs
	return block
}

//验证一个铸块（如本地缺少交易，则异步网络请求该交易）
//返回值:
// 0, 验证通过
// -1，验证失败
// 1 无法验证（缺少交易，已异步向网络模块请求）
// 2 无法验证（前一块在链上不存存在）
func (chain *FullBlockChain) VerifyBlock(bh types.BlockHeader) ([]common.Hash, int8) {
	_, _, txs, ret := chain.verifyBlock(bh, nil)
	return txs, ret
}

func (chain *FullBlockChain) verifyBlock(bh types.BlockHeader, txs []*types.Transaction) (*account.AccountDB, types.Receipts, []common.Hash, int8) {
	begin := time.Now()
	var err error
	defer func() {
		Logger.Infof("verifyBlock hash:%v,height:%d,totalQn:%d,preHash:%v,len header tx:%d,len tx:%d, cost:%v, err=%v", bh.Hash.String(), bh.Height, bh.TotalQN, bh.PreHash.String(), len(bh.Transactions), len(txs), time.Since(begin).String(), err)
	}()

	if bh.Hash != bh.GenHash() {
		Logger.Debugf("Validate block hash error!")
		err = fmt.Errorf("hash diff")
		return nil, nil, nil, -1
	}

	if !chain.hasBlock(bh.PreHash) {
		err = ErrPreNotExist
		return  nil, nil, nil, 2
	}

	transactions := txs

	if txs == nil {
		gotTxs, missing := chain.GetTransactions(bh.Hash, bh.Transactions)
		if 0 != len(missing) {
			var castorId groupsig.ID
			error := castorId.Deserialize(bh.Castor)
			if error != nil {
				panic("Groupsig id deserialize error:" + error.Error())
			}
			//向CASTOR索取交易
			m := &transactionRequestMessage{TransactionHashes: missing, CurrentBlockHash: bh.Hash}
			go requestTransaction(m, castorId.String())
			err = fmt.Errorf("miss transaction size %v", len(missing))
			return  nil, nil, missing, 1
		}
		transactions = gotTxs
	}

	if !chain.validateTxRoot(bh.TxTree, transactions) {
		err = fmt.Errorf("validate tx root fail")
		return  nil, nil, nil, -1
	}

	block := types.Block{Header: &bh, Transactions: transactions}
	executeTxResult, state, receiptes := chain.executeTransaction(&block)
	if !executeTxResult {
		err = fmt.Errorf("execute transaction fail")
		return  nil, nil, nil, -1
	}

	return state, receiptes, nil, 0
}

//铸块成功，上链
//返回值: 0,上链成功
//       -1，验证失败
//        1, 丢弃该块(链上已存在该块）
//        2,丢弃该块（链上存在QN值更大的相同高度块)
//        3,分叉调整
func (chain *FullBlockChain) AddBlockOnChain(source string, b *types.Block) types.AddBlockResult {
	ret, _ := chain.addBlockOnChain(source, b)
	return ret
}

func (chain *FullBlockChain) consensusVerifyBlock(bh *types.BlockHeader) (bool, error) {
	if chain.Height() == 0 {
		return true, nil
	}
	pre := chain.queryBlockByHash(bh.PreHash)
	if pre == nil {
		return false, errors.New("has no pre")
	}
	result, err := chain.GetConsensusHelper().VerifyNewBlock(bh, pre.Header)
	if err != nil {
		Logger.Errorf("consensusVerifyBlock error:%s", err.Error())
		return false, err
	}
	return result, err
}

func (chain *FullBlockChain) processFutureBlock(b *types.Block, source string)  {
	chain.futureBlocks.Add(b.Header.PreHash, b)
	if source == "" {
		return
	}
	eh := chain.GetConsensusHelper().EstimatePreHeight(b.Header)
	if eh <= chain.Height() {	//pre高度小于当前高度，则判断为产生分叉
		bh := b.Header
		top := chain.latestBlock
		Logger.Warnf("detect fork. hash=%v, height=%v, preHash=%v, topHash=%v, topHeight=%v, topPreHash=%v", bh.Hash.String(), bh.Height, bh.PreHash.String(), top.Hash.String(), top.Height, top.PreHash.String())
		chain.forkProcessor.tryToProcessFork(source, b.Header)
	}
}

func (chain *FullBlockChain) validateBlock(source string, b *types.Block) (bool, error) {
	if b == nil {
		return false, fmt.Errorf("block is nil")
	}

	if !chain.hasBlock(b.Header.PreHash) {
		chain.processFutureBlock(b, source)
		return false, ErrPreNotExist
	}

	if chain.compareChainWeight(b.Header) > 0 {
		return false, ErrLocalMoreWeight
	}
	if check, err := chain.GetConsensusHelper().CheckProveRoot(b.Header); !check {
		return false, fmt.Errorf("check prove root fail, err=%v", err.Error())
	}

	groupValidateResult, err := chain.consensusVerifyBlock(b.Header)
	if !groupValidateResult {
		if err == common.ErrSelectGroupNil || err == common.ErrSelectGroupInequal {
			Logger.Infof("Add block on chain failed: depend on group!")
		} else {
			Logger.Errorf("Fail to validate group sig!Err:%s", err.Error())
		}
		return false, fmt.Errorf("consensus verify fail, err=%v", err.Error())
	}
	return true, nil
}

func (chain *FullBlockChain) addBlockOnChain(source string, b *types.Block) (ret types.AddBlockResult, err error) {
	begin := time.Now()
	defer func() {
		Logger.Debugf("addBlockOnchain hash=%v, height=%v, err=%v, cost=%v", b.Header.Hash.String(), b.Header.Height, err, time.Since(begin).String())
	}()
	topBlock := chain.getLatestBlock()
	bh := b.Header
	Logger.Debugf("coming block:hash=%v, preH=%v, height=%v,totalQn:%d", b.Header.Hash.Hex(), b.Header.PreHash.Hex(), b.Header.Height, b.Header.TotalQN)
	Logger.Debugf("Local tophash=%v, topPreHash=%v, height=%v,totalQn:%d", topBlock.Hash.Hex(), topBlock.PreHash.Hex(), topBlock.Height, topBlock.TotalQN)

	if chain.hasBlock(bh.Hash) {
		return types.BlockExisted, ErrBlockExist
	}
	if ok, e := chain.validateBlock(source, b); !ok {
		ret = types.AddBlockFailed
		err = e
		return
	}

	state, receipts, _, verifyResult := chain.verifyBlock(*bh, b.Transactions)
	if verifyResult != 0 {
		Logger.Errorf("Fail to VerifyCastingBlock, reason code:%d \n", verifyResult)
		//if verifyResult == 2 {
		//	Logger.Debugf("coming block  has no pre on local gchain.Forking...", )
		//	go gchain.forkProcessor.tryToProcessFork(source, gchain.latestBlock.Height)
		//}
		ret = types.AddBlockFailed
		err = fmt.Errorf("verify block fail")
		return
	}

	chain.mu.Lock()
	defer chain.mu.Unlock()

	defer func() {
		if ret == types.AddBlockSucc {
			chain.addTopBlock(b)
			chain.successOnChainCallBack(b)
		}
	}()

	topBlock = chain.getLatestBlock()

	if chain.hasBlock(bh.Hash) {
		ret = types.BlockExisted
		err = ErrBlockExist
		return
	}

	if !chain.hasBlock(bh.PreHash) {
		chain.processFutureBlock(b, source)
		ret = types.AddBlockFailed
		err = ErrPreNotExist
		return
	}

	//直接链上
	if bh.PreHash == topBlock.Hash {
		ok, e := chain.commitBlock(b, state, receipts)
		if ok {
			ret = types.AddBlockSucc
			return
		} else {
			Logger.Warnf("insert block fail, hash=%v, height=%v, err=%v", bh.Hash.String(), bh.Height, e)
			ret =  types.AddBlockFailed
			err = ErrCommitBlockFail
			return
		}
	}

	cmpWeight := chain.compareChainWeight(bh)
	if cmpWeight > 0 {	//本地权重更大，丢弃
		ret = types.BlockTotalQnLessThanLocal
		err = ErrLocalMoreWeight
		return
	} else if cmpWeight == 0 {
		ret = types.BlockExisted
		err = ErrBlockExist
		return
	} else {//分叉
		newTop := chain.queryBlockHeaderByHash(bh.PreHash)
		old := chain.latestBlock
		Logger.Debugf("simple fork reset top: old %v %v %v %v, coming %v %v %v %v", old.Hash.ShortS(), old.Height, old.PreHash.ShortS(), old.TotalQN, bh.Hash.ShortS(), bh.Height, bh.PreHash.ShortS(), bh.TotalQN)
		if e := chain.resetTop(newTop); e != nil {
			Logger.Warnf("reset top err, currTop %v, setTop %v, setHeight %v", topBlock.Hash.String(), newTop.Hash.String(), newTop.Height)
			ret = types.AddBlockFailed
			err = fmt.Errorf("reset top err:%v", e)
			return
		}

		if chain.getLatestBlock().Hash != bh.PreHash {
			panic("reset top error")
		}
		ok, e := chain.commitBlock(b, state, receipts)
		if ok {
			ret = types.AddBlockSucc
			return
		} else {
			Logger.Warnf("insert block fail, hash=%v, height=%v, err=%v", bh.Hash.String(), bh.Height, e)
			ret =  types.AddBlockFailed
			err = ErrCommitBlockFail
			return
		}
	}
}

func (chain *FullBlockChain) validateTxRoot(txMerkleTreeRoot common.Hash, txs []*types.Transaction) bool {
	txTree := calcTxTree(txs)

	if !bytes.Equal(txTree.Bytes(), txMerkleTreeRoot.Bytes()) {
		Logger.Errorf("Fail to verify txTree, hash1:%s hash2:%s", txTree.Hex(), txMerkleTreeRoot.Hex())
		return false
	}
	return true
}

func (chain *FullBlockChain) executeTransaction(block *types.Block) (bool, *account.AccountDB, types.Receipts) {
	preBlock := chain.queryBlockHeaderByHash(block.Header.PreHash)
	if preBlock == nil {
		return false, nil,nil
	}

	cached, _ := chain.verifiedBlocks.Get(block.Header.Hash)
	if cached != nil {
		cb := cached.(*castingBlock)
		return true, cb.state, cb.receipts
	}

	preRoot := preBlock.StateTree
	if len(block.Transactions) > 0 {
		//Logger.Debugf("NewAccountDB height:%d StateTree:%s preHash:%s preRoot:%s", block.Header.Height, block.Header.StateTree.Hex(), preBlock.Hash.Hex(), preRoot.Hex())
	}
	state, err := account.NewAccountDB(preRoot, chain.stateCache)
	if err != nil {
		Logger.Errorf("Fail to new statedb, error:%s", err)
		return false, state, nil
	}

	statehash, _, _, receipts, err, _ := chain.executor.Execute(state, block, block.Header.Height, "fullverify")
	if statehash != block.Header.StateTree {
		Logger.Errorf("Fail to verify statetree, hash1:%s hash2:%s", statehash.String(), block.Header.StateTree.String())
		return false, state, receipts
	}
	receiptsTree := calcReceiptsTree(receipts)
	if receiptsTree != block.Header.ReceiptTree {
		Logger.Errorf("fail to verify receipt, hash1:%s hash2:%s", receiptsTree.String(), block.Header.ReceiptTree.String())
		return false, state, receipts
	}

	chain.verifiedBlocks.Add(block.Header.Hash, &castingBlock{state: state, receipts: receipts,})
	return true, state, receipts
}

func (chain *FullBlockChain) successOnChainCallBack(remoteBlock *types.Block) {
	notify.BUS.Publish(notify.BlockAddSucc, &notify.BlockOnChainSuccMessage{Block: remoteBlock,})

	//GroupChainImpl.RemoveDismissGroupFromCache(b.Header.Height)
	//if BlockSyncer != nil {
	//	topBlockInfo := TopBlockInfo{Hash: gchain.latestBlock.Hash, TotalQn: gchain.latestBlock.TotalQN, Height: gchain.latestBlock.Height, PreHash: gchain.latestBlock.PreHash}
	//	go BlockSyncer.sendTopBlockInfoToNeighbor(topBlockInfo)
	//}
}

func (chain *FullBlockChain) onBlockAddSuccess(message notify.Message)  {
	b := message.GetData().(*types.Block)
	if value, _ := chain.futureBlocks.Get(b.Header.Hash); value != nil {
		block := value.(*types.Block)
		Logger.Debugf("Get block from future blocks,hash:%s,height:%d", block.Header.Hash.String(), block.Header.Height)
		chain.addBlockOnChain("", block)
		chain.futureBlocks.Remove(b.Header.Hash)
		return
	}
}

func (chain *FullBlockChain) ensureBlocksChained(blocks []*types.Block) bool {
	if len(blocks) <= 1 {
		return true
	}
	for i := 1; i < len(blocks); i++ {
		if blocks[i].Header.PreHash != blocks[i-1].Header.Hash {
			return false
		}
	}
	return true
}

func (chain *FullBlockChain) batchAddBlockOnChain(source string, blocks []*types.Block, callback batchAddBlockCallback)  {
	if !chain.ensureBlocksChained(blocks) {
		Logger.Errorf("blocks not chained! size %v", len(blocks))
		return
	}
	for _, b := range blocks {
		ret := chain.AddBlockOnChain(source, b)
		if !callback(b, ret) {
			break
		}
	}
}
