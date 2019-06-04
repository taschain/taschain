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
	"bytes"
	"errors"
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/middleware/notify"
	"github.com/taschain/taschain/middleware/types"
	"github.com/taschain/taschain/storage/account"
	"github.com/taschain/taschain/taslog"
	"time"
)

/*
**  Creator: pxf
**  Date: 2019/3/11 下午3:15
**  Description:
 */

type batchAddBlockCallback func(b *types.Block, ret types.AddBlockResult) bool

type executePostState struct {
	state     *account.AccountDB
	receipts  types.Receipts
	evitedTxs []common.Hash
	txs       []*types.Transaction
}

//构建一个铸块（组内当前铸块人同步操作）
func (chain *FullBlockChain) CastBlock(height uint64, proveValue []byte, qn uint64, castor []byte, groupid []byte) *types.Block {
	chain.mu.Lock()
	defer chain.mu.Unlock()

	latestBlock := chain.QueryTopBlock()
	if latestBlock != nil && height <= latestBlock.Height {
		Logger.Info("[BlockChain] fail to cast block: height problem. height:%d, latest:%d", height, latestBlock.Height)
		return nil
	}

	begin := time.Now()
	block := new(types.Block)

	defer func() {
		Logger.Debugf("cast block, height=%v, hash=%v, cost %v", block.Header.Height, block.Header.Hash.Hex(), time.Since(begin).String())
	}()

	block.Header = &types.BlockHeader{
		CurTime:    chain.ts.Now(),
		Height:     height,
		ProveValue: proveValue,
		Castor:     castor,
		GroupID:    groupid,
		TotalQN:    latestBlock.TotalQN + qn, //todo:latestBlock != nil?
		StateTree:  common.BytesToHash(latestBlock.StateTree.Bytes()),
		//ProveRoot:  proveRoot,
		PreHash: latestBlock.Hash,
		Nonce:   common.ChainDataVersion,
	}
	block.Header.Elapsed = int32(block.Header.CurTime.Since(latestBlock.CurTime))

	if block.Header.Elapsed < 0 {
		Logger.Error("cur time is before pre time:height=%v, curtime=%v, pretime=%v", height, block.Header.CurTime, latestBlock.CurTime)
		return nil
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

	txs := chain.transactionPool.PackForCast()

	statehash, evitTxs, transactions, receipts, err := chain.executor.Execute(state, block.Header, txs, true)

	block.Transactions = transactions
	//block.Header.Transactions = transactionHashes
	block.Header.TxTree = calcTxTree(block.Transactions)

	block.Header.StateTree = common.BytesToHash(statehash.Bytes())
	block.Header.ReceiptTree = calcReceiptsTree(receipts)

	block.Header.Hash = block.Header.GenHash()

	//Logger.Errorf("receiptes cast bh %v, %+v", block.Header.Hash.String(), receipts)

	defer Logger.Infof("casting block %d,hash:%v,qn:%d,tx:%d,TxTree:%v,proValue:%v,stateTree:%s,prestatetree:%s",
		height, block.Header.Hash.Hex(), block.Header.TotalQN, len(block.Transactions), block.Header.TxTree.Hex(),
		chain.consensusHelper.VRFProve2Value(block.Header.ProveValue), block.Header.StateTree.Hex(), preRoot.Hex())
	//自己铸的块 自己不需要验证
	chain.verifiedBlocks.Add(block.Header.Hash, &executePostState{
		state:     state,
		receipts:  receipts,
		evitedTxs: evitTxs,
		txs:       block.Transactions,
	})
	return block
}

func (chain *FullBlockChain) verifyTxs(bh *types.BlockHeader, txs []*types.Transaction) (ps *executePostState, ret int8) {
	begin := time.Now()
	slog := taslog.NewSlowLog("verifyTxs", 0.8)
	var err error
	defer func() {
		Logger.Infof("verifyTxs hash:%v,height:%d,totalQn:%d,preHash:%v,len tx:%d, cost:%v, err=%v", bh.Hash.Hex(), bh.Height, bh.TotalQN, bh.PreHash.Hex(), len(txs), time.Since(begin).String(), err)
		slog.Log("hash=%v, height=%v, err=%v", bh.Hash.Hex(), bh.Height, err)
	}()

	size := 0
	if txs != nil {
		size = len(txs)
	}
	slog.AddStage(fmt.Sprintf("validateTxs%v", size))
	if !chain.validateTxs(txs) {
		slog.EndStage()
		return nil, -1
	}
	slog.EndStage()

	slog.AddStage("validateTxRoot")
	if !chain.validateTxRoot(bh.TxTree, txs) {
		slog.EndStage()
		err = fmt.Errorf("validate tx root fail")
		return nil, -1
	}
	slog.EndStage()

	slog.AddStage("exeTxs")
	block := types.Block{Header: bh, Transactions: txs}
	executeTxResult, ps := chain.executeTransaction(&block)
	slog.EndStage()
	if !executeTxResult {
		err = fmt.Errorf("execute transaction fail")
		return nil, -1
	}

	return ps, 0
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

func (chain *FullBlockChain) processFutureBlock(b *types.Block, source string) {
	chain.futureBlocks.Add(b.Header.PreHash, b)
	if source == "" {
		return
	}
	eh := chain.GetConsensusHelper().EstimatePreHeight(b.Header)
	if eh <= chain.Height() { //pre高度小于当前高度，则判断为产生分叉
		bh := b.Header
		top := chain.latestBlock
		Logger.Warnf("detect fork. hash=%v, height=%v, preHash=%v, topHash=%v, topHeight=%v, topPreHash=%v", bh.Hash.Hex(), bh.Height, bh.PreHash.Hex(), top.Hash.Hex(), top.Height, top.PreHash.Hex())
		go chain.forkProcessor.tryToProcessFork(source, b)
	} else { //
		go BlockSyncer.syncFrom(source)
	}
}

func (chain *FullBlockChain) validateBlock(source string, b *types.Block) (bool, error) {
	if b == nil {
		return false, fmt.Errorf("block is nil")
	}

	if !chain.HasBlock(b.Header.PreHash) {
		chain.processFutureBlock(b, source)
		return false, ErrPreNotExist
	}

	if chain.compareChainWeight(b.Header) > 0 {
		return false, ErrLocalMoreWeight
	}

	//if check, err := chain.GetConsensusHelper().CheckProveRoot(b.Header); !check {
	//	return false, fmt.Errorf("check prove root fail, err=%v", err.Error())
	//}

	groupValidateResult, err := chain.consensusVerifyBlock(b.Header)
	if !groupValidateResult {
		if err == common.ErrSelectGroupNil || err == common.ErrSelectGroupInequal {
			Logger.Infof("Add block on chain failed: depend on group! trigger group sync")
			if GroupSyncer != nil {
				go GroupSyncer.trySyncRoutine()
			}
		} else {
			Logger.Errorf("Fail to validate group sig!Err:%s", err.Error())
		}
		return false, fmt.Errorf("consensus verify fail, err=%v", err.Error())
	}
	return true, nil
}

func (chain *FullBlockChain) addBlockOnChain(source string, b *types.Block) (ret types.AddBlockResult, err error) {
	begin := time.Now()
	slog := taslog.NewSlowLog("addBlockOnChain", 0.8)
	defer func() {
		Logger.Debugf("addBlockOnchain hash=%v, height=%v, err=%v, cost=%v", b.Header.Hash.Hex(), b.Header.Height, err, time.Since(begin).String())
		slog.Log("hash=%v, height=%v, err=%v", b.Header.Hash.Hex(), b.Header.Height, err)
	}()

	if b == nil {
		return types.AddBlockFailed, fmt.Errorf("nil block")
	}
	bh := b.Header

	if bh.Hash != bh.GenHash() {
		Logger.Debugf("Validate block hash error!")
		err = fmt.Errorf("hash diff")
		return types.AddBlockFailed, err
	}

	topBlock := chain.getLatestBlock()
	Logger.Debugf("coming block:hash=%v, preH=%v, height=%v,totalQn:%d, Local tophash=%v, topPreHash=%v, height=%v,totalQn:%d", b.Header.Hash.ShortS(), b.Header.PreHash.ShortS(), b.Header.Height, b.Header.TotalQN, topBlock.Hash.ShortS(), topBlock.PreHash.ShortS(), topBlock.Height, topBlock.TotalQN)

	if chain.HasBlock(bh.Hash) {
		return types.BlockExisted, ErrBlockExist
	}
	slog.AddStage("validateBlock")
	if ok, e := chain.validateBlock(source, b); !ok {
		slog.EndStage()
		ret = types.AddBlockFailed
		err = e
		return
	}
	slog.EndStage()

	slog.AddStage("verify")
	ps, verifyResult := chain.verifyTxs(bh, b.Transactions)
	if verifyResult != 0 {
		slog.EndStage()
		Logger.Errorf("Fail to VerifyCastingBlock, reason code:%d \n", verifyResult)
		ret = types.AddBlockFailed
		err = fmt.Errorf("verify block fail")
		return
	}
	slog.EndStage()

	slog.AddStage("getMuLock")
	chain.mu.Lock()
	defer chain.mu.Unlock()
	slog.EndStage()

	defer func() {
		if ret == types.AddBlockSucc {
			chain.addTopBlock(b)
			chain.successOnChainCallBack(b)
		}
	}()

	topBlock = chain.getLatestBlock()

	slog.AddStage("HasBlock")
	if chain.HasBlock(bh.Hash) {
		slog.EndStage()
		ret = types.BlockExisted
		err = ErrBlockExist
		return
	}
	slog.EndStage()

	slog.AddStage("hasPre")
	if !chain.HasBlock(bh.PreHash) {
		slog.EndStage()
		chain.processFutureBlock(b, source)
		ret = types.AddBlockFailed
		err = ErrPreNotExist
		return
	}
	slog.EndStage()

	//直接链上
	if bh.PreHash == topBlock.Hash {
		slog.AddStage("commitBlock")
		ok, e := chain.commitBlock(b, ps)
		slog.EndStage()
		if ok {
			ret = types.AddBlockSucc
			return
		} else {
			Logger.Warnf("insert block fail, hash=%v, height=%v, err=%v", bh.Hash.Hex(), bh.Height, e)
			ret = types.AddBlockFailed
			err = ErrCommitBlockFail
			return
		}
	}

	cmpWeight := chain.compareChainWeight(bh)
	if cmpWeight > 0 { //本地权重更大，丢弃
		ret = types.BlockTotalQnLessThanLocal
		err = ErrLocalMoreWeight
		return
	} else if cmpWeight == 0 {
		ret = types.BlockExisted
		err = ErrBlockExist
		return
	} else { //分叉
		newTop := chain.queryBlockHeaderByHash(bh.PreHash)
		old := chain.latestBlock
		slog.AddStage("resetTop")
		Logger.Debugf("simple fork reset top: old %v %v %v %v, coming %v %v %v %v", old.Hash.ShortS(), old.Height, old.PreHash.ShortS(), old.TotalQN, bh.Hash.ShortS(), bh.Height, bh.PreHash.ShortS(), bh.TotalQN)
		if e := chain.resetTop(newTop); e != nil {
			slog.EndStage()
			Logger.Warnf("reset top err, currTop %v, setTop %v, setHeight %v", topBlock.Hash.Hex(), newTop.Hash.Hex(), newTop.Height)
			ret = types.AddBlockFailed
			err = fmt.Errorf("reset top err:%v", e)
			return
		}
		slog.EndStage()

		if chain.getLatestBlock().Hash != bh.PreHash {
			panic("reset top error")
		}

		slog.AddStage("commitBlock2")
		ok, e := chain.commitBlock(b, ps)
		slog.EndStage()
		if ok {
			ret = types.AddBlockSucc
			return
		} else {
			Logger.Warnf("insert block fail, hash=%v, height=%v, err=%v", bh.Hash.Hex(), bh.Height, e)
			ret = types.AddBlockFailed
			err = ErrCommitBlockFail
			return
		}
	}
}

//check tx sign and recover source
func (chain *FullBlockChain) validateTxs(txs []*types.Transaction) bool {
	if txs == nil || len(txs) == 0 {
		return true
	}
	recoverCnt := 0
	for _, tx := range txs {
		if tx.Source != nil {
			continue
		}
		poolTx := chain.transactionPool.GetTransaction(tx.Type == types.TransactionTypeBonus, tx.Hash)
		if poolTx != nil {
			if tx.Hash != tx.GenHash() {
				Logger.Debugf("fail to validate txs: hash diff at %v, expect hash %v", tx.Hash.Hex(), tx.GenHash().Hex())
				return false
			}
			if !bytes.Equal(tx.Sign, poolTx.Sign) {
				Logger.Debugf("fail to validate txs: sign diff at %v, [%v %v]", tx.Hash.Hex(), tx.Sign, poolTx.Sign)
				return false
			}
			tx.Source = poolTx.Source
		} else {
			recoverCnt++
			TxSyncer.add(tx)
			if err := chain.transactionPool.RecoverAndValidateTx(tx); err != nil {
				Logger.Debugf("fail to validate txs RecoverAndValidateTx err:%v at %v", err, tx.Hash.Hex())
				return false
			}
		}
	}
	Logger.Debugf("validate txs size %v, recover cnt %v", len(txs), recoverCnt)
	return true
}

func (chain *FullBlockChain) validateTxRoot(txMerkleTreeRoot common.Hash, txs []*types.Transaction) bool {
	txTree := calcTxTree(txs)

	if txTree != txMerkleTreeRoot {
		Logger.Errorf("Fail to verify txTree, hash1:%s hash2:%s", txTree.Hex(), txMerkleTreeRoot.Hex())
		return false
	}
	return true
}

func (chain *FullBlockChain) executeTransaction(block *types.Block) (bool, *executePostState) {
	cached, _ := chain.verifiedBlocks.Get(block.Header.Hash)
	if cached != nil {
		cb := cached.(*executePostState)
		return true, cb
	}
	preBlock := chain.queryBlockHeaderByHash(block.Header.PreHash)
	if preBlock == nil {
		return false, nil
	}

	preRoot := preBlock.StateTree
	if len(block.Transactions) > 0 {
		//Logger.Debugf("NewAccountDB height:%d StateTree:%s preHash:%s preRoot:%s", block.Header.Height, block.Header.StateTree.Hex(), preBlock.Hash.Hex(), preRoot.Hex())
	}
	state, err := account.NewAccountDB(preRoot, chain.stateCache)
	if err != nil {
		Logger.Errorf("Fail to new statedb, error:%s", err)
		return false, nil
	}

	statehash, evitTxs, _, receipts, err := chain.executor.Execute(state, block.Header, block.Transactions, false)
	if statehash != block.Header.StateTree {
		Logger.Errorf("Fail to verify statetrexecute transaction failee, hash1:%s hash2:%s", statehash.Hex(), block.Header.StateTree.Hex())
		return false, nil
	}
	receiptsTree := calcReceiptsTree(receipts)
	if receiptsTree != block.Header.ReceiptTree {
		Logger.Errorf("fail to verify receipt, hash1:%s hash2:%s", receiptsTree.Hex(), block.Header.ReceiptTree.Hex())
		//Logger.Errorf("receiptes verify bh %v, %+v", block.Header.Hash.String(), receipts)
		return false, nil
	}

	eps := &executePostState{state: state, receipts: receipts, evitedTxs: evitTxs, txs: block.Transactions}
	chain.verifiedBlocks.Add(block.Header.Hash, eps)
	return true, eps
}

func (chain *FullBlockChain) successOnChainCallBack(remoteBlock *types.Block) {
	notify.BUS.Publish(notify.BlockAddSucc, &notify.BlockOnChainSuccMessage{Block: remoteBlock})

	//GroupChainImpl.RemoveDismissGroupFromCache(b.Header.Height)
	//if BlockSyncer != nil {
	//	topBlockInfo := TopBlockInfo{Hash: gchain.latestBlock.Hash, TotalQn: gchain.latestBlock.TotalQN, Height: gchain.latestBlock.Height, PreHash: gchain.latestBlock.PreHash}
	//	go BlockSyncer.sendTopBlockInfoToNeighbor(topBlockInfo)
	//}
}

func (chain *FullBlockChain) onBlockAddSuccess(message notify.Message) {
	b := message.GetData().(*types.Block)
	if value, _ := chain.futureBlocks.Get(b.Header.Hash); value != nil {
		block := value.(*types.Block)
		Logger.Debugf("Get block from future blocks,hash:%s,height:%d", block.Header.Hash.Hex(), block.Header.Height)
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

func (chain *FullBlockChain) batchAddBlockOnChain(source string, module string, blocks []*types.Block, callback batchAddBlockCallback) {
	if !chain.ensureBlocksChained(blocks) {
		Logger.Errorf("%v blocks not chained! size %v", module, len(blocks))
		return
	}
	//先预处理恢复交易source
	for _, b := range blocks {
		if b.Transactions != nil && len(b.Transactions) > 0 {
			go chain.transactionPool.AsyncAddTxs(b.Transactions)
		}
	}

	chain.batchMu.Lock()
	defer chain.batchMu.Unlock()

	localTop := chain.latestBlock

	var addBlocks []*types.Block
	for i, b := range blocks {
		if !chain.hasBlock(b.Header.Hash) {
			addBlocks = blocks[i:]
			break
		}
	}
	if addBlocks == nil || len(addBlocks) == 0 {
		return
	}
	firstBH := addBlocks[0]
	if firstBH.Header.PreHash != localTop.Hash {
		pre := chain.QueryBlockHeaderByHash(firstBH.Header.PreHash)
		if pre != nil {
			last := addBlocks[len(addBlocks)-1].Header
			Logger.Debugf("%v batchAdd reset top:old %v %v %v, new %v %v %v, last %v %v %v", module, localTop.Hash.ShortS(), localTop.Height, localTop.TotalQN, pre.Hash.ShortS(), pre.Height, pre.TotalQN, last.Hash.ShortS(), last.Height, last.TotalQN)
			chain.ResetTop(pre)
		} else {
			//大分叉了
			Logger.Debugf("%v batchAdd detect fork from %v: local %v %v, peer %v %v", module, source, localTop.Hash.ShortS(), localTop.Height, firstBH.Header.Hash.ShortS(), firstBH.Header.Height)
			go chain.forkProcessor.tryToProcessFork(source, firstBH)
			return
		}
	}
	chain.isAdujsting = true
	defer func() {
		chain.isAdujsting = false
	}()

	for _, b := range addBlocks {
		ret := chain.AddBlockOnChain(source, b)
		if !callback(b, ret) {
			break
		}
	}
}
