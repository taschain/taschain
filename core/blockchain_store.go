package core

import (
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/middleware/types"
	"github.com/taschain/taschain/storage/account"
	"github.com/taschain/taschain/utility"
)

/*
**  Creator: pxf
**  Date: 2019/3/11 下午1:30
**  Description:
 */

func (chain *FullBlockChain) saveBlockState(b *types.Block, state *account.AccountDB) error {
	root, err := state.Commit(true)
	if err != nil {
		return fmt.Errorf("state commit error:%s", err.Error())
	}

	triedb := chain.stateCache.TrieDB()
	err = triedb.Commit(root, false)
	if err != nil {
		return fmt.Errorf("trie commit error:%s", err.Error())
	}
	return nil
}

func (chain *FullBlockChain) saveCurrentBlock(hash common.Hash) error {
	err := chain.blocks.AddKv(chain.batch, []byte(blockStatusKey), hash.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (chain *FullBlockChain) updateLatestBlock(state *account.AccountDB, header *types.BlockHeader) {
	chain.latestStateDB = state
	chain.latestBlock = header
}

func (chain *FullBlockChain) saveBlockHeader(hash common.Hash, dataBytes []byte) error {
	return chain.blocks.AddKv(chain.batch, hash.Bytes(), dataBytes)
}

func (chain *FullBlockChain) saveBlockHeight(height uint64, dataBytes []byte) error {
	return chain.blockHeight.AddKv(chain.batch, utility.UInt64ToByte(height), dataBytes)
}

func (chain *FullBlockChain) saveBlockTxs(blockHash common.Hash, dataBytes []byte) error {
	return chain.txdb.AddKv(chain.batch, blockHash.Bytes(), dataBytes)
}

// commitBlock persist a block in a batch
func (chain *FullBlockChain) commitBlock(block *types.Block, ps *executePostState) (ok bool, err error) {

	bh := block.Header
	headerBytes, err := types.MarshalBlockHeader(bh)
	if err != nil {
		Logger.Errorf("Fail to json Marshal, error:%s", err.Error())
		return
	}

	bodyBytes, err := encodeBlockTransactions(block)
	if err != nil {
		Logger.Errorf("encode block transaction error:%v", err)
		return
	}

	chain.rwLock.Lock()
	defer chain.rwLock.Unlock()

	defer chain.batch.Reset()

	// Commit state
	if err = chain.saveBlockState(block, ps.state); err != nil {
		return
	}
	// Save hash to block header key value pair
	if err = chain.saveBlockHeader(bh.Hash, headerBytes); err != nil {
		return
	}
	// Save height to block hash key value pair
	if err = chain.saveBlockHeight(bh.Height, bh.Hash.Bytes()); err != nil {
		return
	}
	// Save hash to transactions key value pair
	if err = chain.saveBlockTxs(bh.Hash, bodyBytes); err != nil {
		return
	}
	// Save hash to receipt key value pair
	if err = chain.transactionPool.saveReceipts(bh.Hash, ps.receipts); err != nil {
		return
	}
	// Save current block
	if err = chain.saveCurrentBlock(bh.Hash); err != nil {
		return
	}
	// Batch write
	if err = chain.batch.Write(); err != nil {
		return
	}
	chain.updateLatestBlock(ps.state, bh)

	// If the block is successfully submitted, the transaction
	// corresponding to the transaction pool should be deleted
	if block.Transactions != nil {
		chain.transactionPool.RemoveFromPool(block.GetTransactionHashs())
	}
	// Remove eviction transactions from the transaction pool
	if ps.evitedTxs != nil {
		chain.transactionPool.RemoveFromPool(ps.evitedTxs)
	}

	ok = true
	return
}

func (chain *FullBlockChain) resetTop(block *types.BlockHeader) error {
	if !chain.isAdujsting {
		chain.isAdujsting = true
		defer func() {
			chain.isAdujsting = false
		}()
	}

	// Add read and write locks, block reading at this time
	chain.rwLock.Lock()
	defer chain.rwLock.Unlock()

	if nil == block {
		return fmt.Errorf("block is nil")
	}
	if block.Hash == chain.latestBlock.Hash {
		return nil
	}
	Logger.Debugf("reset top hash:%s height:%d ", block.Hash.Hex(), block.Height)

	var err error

	defer chain.batch.Reset()

	curr := chain.getLatestBlock()
	recoverTxs := make([]*types.Transaction, 0)
	delRecepites := make([]common.Hash, 0)
	for curr.Hash != block.Hash {
		// Delete the old block header
		if err = chain.saveBlockHeader(curr.Hash, nil); err != nil {
			return err
		}
		// Delete the old block height
		if err = chain.saveBlockHeight(curr.Height, nil); err != nil {
			return err
		}
		// Delete the old block's transactions
		if err = chain.saveBlockTxs(curr.Hash, nil); err != nil {
			return err
		}
		txs := chain.queryBlockTransactionsAll(curr.Hash)
		if txs != nil {
			recoverTxs = append(recoverTxs, txs...)
			for _, tx := range txs {
				delRecepites = append(delRecepites, tx.Hash)
			}
		}

		chain.removeTopBlock(curr.Hash)
		if curr.PreHash == block.Hash {
			break
		}
		curr = chain.queryBlockHeaderByHash(curr.PreHash)
	}
	// Delete receipts corresponding to the transactions in the discard block
	if err = chain.transactionPool.deleteReceipts(delRecepites); err != nil {
		return err
	}
	// Reset the current block
	if err = chain.saveCurrentBlock(block.Hash); err != nil {
		return err
	}
	state, err := account.NewAccountDB(block.StateTree, chain.stateCache)
	if err != nil {
		return err
	}
	if err = chain.batch.Write(); err != nil {
		return err
	}
	chain.updateLatestBlock(state, block)

	chain.transactionPool.BackToPool(recoverTxs)

	return nil
}

// removeOrphan remove the orphan block
func (chain *FullBlockChain) removeOrphan(block *types.Block) error {

	// Add read and write locks, block reading at this time
	chain.rwLock.Lock()
	defer chain.rwLock.Unlock()

	if nil == block {
		return nil
	}
	hash := block.Header.Hash
	height := block.Header.Height
	Logger.Debugf("remove hash:%s height:%d ", hash.Hex(), height)

	var err error
	defer chain.batch.Reset()

	if err = chain.saveBlockHeader(hash, nil); err != nil {
		return err
	}
	if err = chain.saveBlockHeight(height, nil); err != nil {
		return err
	}
	if err = chain.saveBlockTxs(hash, nil); err != nil {
		return err
	}
	txs := chain.queryBlockTransactionsAll(hash)
	if txs != nil {
		txHashs := make([]common.Hash, len(txs))
		for i, tx := range txs {
			txHashs[i] = tx.Hash
		}
		if err = chain.transactionPool.deleteReceipts(txHashs); err != nil {
			return err
		}
	}

	if err = chain.batch.Write(); err != nil {
		return err
	}
	chain.removeTopBlock(hash)
	return nil
}

func (chain *FullBlockChain) loadCurrentBlock() *types.BlockHeader {
	bs, err := chain.blocks.Get([]byte(blockStatusKey))
	if err != nil {
		return nil
	}
	hash := common.BytesToHash(bs)
	return chain.queryBlockHeaderByHash(hash)
}

func (chain *FullBlockChain) hasBlock(hash common.Hash) bool {
	if ok, _ := chain.blocks.Has(hash.Bytes()); ok {
		return ok
	}
	return false
	//pre := gchain.queryBlockHeaderByHash(bh.PreHash)
	//return pre != nil
}

func (chain *FullBlockChain) hasHeight(h uint64) bool {
	if ok, _ := chain.blockHeight.Has(utility.UInt64ToByte(h)); ok {
		return ok
	}
	return false
}

func (chain *FullBlockChain) queryBlockHash(height uint64) *common.Hash {
	result, _ := chain.blockHeight.Get(utility.UInt64ToByte(height))
	if result != nil {
		hash := common.BytesToHash(result)
		return &hash
	}
	return nil
}

func (chain *FullBlockChain) queryBlockHashCeil(height uint64) *common.Hash {
	iter := chain.blockHeight.NewIterator()
	defer iter.Release()
	if iter.Seek(utility.UInt64ToByte(height)) {
		hash := common.BytesToHash(iter.Value())
		return &hash
	}
	return nil
}

func (chain *FullBlockChain) queryBlockHeaderBytesFloor(height uint64) (common.Hash, []byte) {
	iter := chain.blockHeight.NewIterator()
	defer iter.Release()
	if iter.Seek(utility.UInt64ToByte(height)) {
		realHeight := utility.ByteToUInt64(iter.Key())
		if realHeight == height {
			hash := common.BytesToHash(iter.Value())
			return hash, chain.queryBlockHeaderBytes(hash)
		}
	}
	if iter.Prev() {
		hash := common.BytesToHash(iter.Value())
		return hash, chain.queryBlockHeaderBytes(hash)
	}
	return common.Hash{}, nil
}

func (chain *FullBlockChain) queryBlockHeaderByHeightFloor(height uint64) *types.BlockHeader {
	iter := chain.blockHeight.NewIterator()
	defer iter.Release()
	if iter.Seek(utility.UInt64ToByte(height)) {
		realHeight := utility.ByteToUInt64(iter.Key())
		if realHeight == height {
			hash := common.BytesToHash(iter.Value())
			bh := chain.queryBlockHeaderByHash(hash)
			if bh == nil {
				panic(fmt.Sprintf("data error:height %v, hash %v", height, hash.String()))
			}
			if bh.Height != height {
				panic(fmt.Sprintf("key height not equal to value height:keyHeight=%v, valueHeight=%v", realHeight, bh.Height))
			}
			return bh
		}
	}
	if iter.Prev() {
		hash := common.BytesToHash(iter.Value())
		return chain.queryBlockHeaderByHash(hash)
	}
	return nil
}

func (chain *FullBlockChain) queryBlockBodyBytes(hash common.Hash) []byte {
	bs, err := chain.txdb.Get(hash.Bytes())
	if err != nil {
		Logger.Errorf("get txdb err:%v, key:%v", err.Error(), hash.String())
		return nil
	}
	return bs
}

func (chain *FullBlockChain) queryBlockTransactionsAll(hash common.Hash) []*types.Transaction {
	bs := chain.queryBlockBodyBytes(hash)
	if bs == nil {
		return nil
	}
	txs, err := decodeBlockTransactions(bs)
	if err != nil {
		Logger.Errorf("decode transactions err:%v, key:%v", err.Error(), hash.String())
		return nil
	}
	return txs
}

func (chain *FullBlockChain) batchGetBlocksAfterHeight(h uint64, limit int) []*types.Block {
	blocks := make([]*types.Block, 0)
	iter := chain.blockHeight.NewIterator()
	defer iter.Release()

	// No higher block after the specified block height
	if !iter.Seek(utility.UInt64ToByte(h)) {
		return blocks
	}
	cnt := 0
	for cnt < limit {
		hash := common.BytesToHash(iter.Value())
		b := chain.queryBlockByHash(hash)
		if b == nil {
			break
		}
		blocks = append(blocks, b)
		if !iter.Next() {
			break
		}
		cnt++
	}
	return blocks
}

func (chain *FullBlockChain) queryBlockHeaderByHeight(height uint64) *types.BlockHeader {
	hash := chain.queryBlockHash(height)
	if hash != nil {
		return chain.queryBlockHeaderByHash(*hash)
	}
	return nil
}

func (chain *FullBlockChain) queryBlockByHash(hash common.Hash) *types.Block {
	bh := chain.queryBlockHeaderByHash(hash)
	if bh == nil {
		return nil
	}

	txs := chain.queryBlockTransactionsAll(hash)
	b := &types.Block{
		Header:       bh,
		Transactions: txs,
	}
	return b
}

func (chain *FullBlockChain) queryBlockHeaderBytes(hash common.Hash) []byte {
	result, _ := chain.blocks.Get(hash.Bytes())
	return result
}

func (chain *FullBlockChain) queryBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	bs := chain.queryBlockHeaderBytes(hash)
	if bs != nil {
		block, err := types.UnMarshalBlockHeader(bs)
		if err != nil {
			fmt.Println(err)
			return nil
		}
		return block
	}
	return nil
}

func (chain *FullBlockChain) addTopBlock(b *types.Block) {
	chain.topBlocks.Add(b.Header.Hash, b)
}

func (chain *FullBlockChain) removeTopBlock(hash common.Hash) {
	chain.topBlocks.Remove(hash)
}

func (chain *FullBlockChain) getTopBlockByHash(hash common.Hash) *types.Block {
	if v, ok := chain.topBlocks.Get(hash); ok {
		return v.(*types.Block)
	}
	return nil
}

func (chain *FullBlockChain) getTopBlockByHeight(height uint64) *types.Block {
	if chain.topBlocks.Len() == 0 {
		return nil
	}
	for _, k := range chain.topBlocks.Keys() {
		b := chain.getTopBlockByHash(k.(common.Hash))
		if b != nil && b.Header.Height == height {
			return b
		}
	}
	return nil
}

func (chain *FullBlockChain) queryBlockTransactionsOptional(txIdx int, height uint64, txHash common.Hash) *types.Transaction {

	bh := chain.queryBlockHeaderByHeight(height)
	if bh == nil {
		return nil
	}
	bs, err := chain.txdb.Get(bh.Hash.Bytes())
	if err != nil {
		Logger.Errorf("queryBlockTransactionsOptional get txdb err:%v, key:%v", err.Error(), bh.Hash.String())
		return nil
	}
	tx, err := decodeTransaction(txIdx, txHash, bs)
	if tx != nil {
		return tx
	}
	Logger.Errorf("queryBlockTransactionsOptional decode tx error: hash=%v, err=%v", txHash.String(), err.Error())
	return nil
}
