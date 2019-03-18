package core

import (
	"common"
	"utility"
	"middleware/types"
	"storage/account"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2019/3/11 下午1:30
**  Description: 
*/

//func (chain *FullBlockChain) PutCheckValue(height uint64, hash []byte) error {
//	key := utility.UInt64ToByte(height)
//	return chain.checkdb.Put(key, hash)
//}
//
//func (chain *FullBlockChain) GetCheckValue(height uint64) (common.Hash, error) {
//	key := utility.UInt64ToByte(height)
//	raw, err := chain.checkdb.Get(key)
//	return common.BytesToHash(raw), err
//}


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
	err := chain.blocks.AddKv(chain.batch, []byte(BLOCK_STATUS_KEY), hash.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func (chain *FullBlockChain) updateLatestBlock(state *account.AccountDB, header *types.BlockHeader)  {
	chain.latestStateDB = state
	chain.latestBlock = header
	Logger.Debugf("Update latestStateDB:%s height:%d", header.StateTree.Hex(), header.Height)
}

func (chain *FullBlockChain) saveBlockHeader(hash common.Hash, dataBytes []byte) error {
    return chain.blocks.AddKv(chain.batch, hash.Bytes(), dataBytes)
}

func (chain *FullBlockChain) saveBlockHeight(height uint64, dataBytes []byte) error {
	return chain.blockHeight.AddKv(chain.batch, utility.UInt64ToByte(height), dataBytes)
}

//persist a block in a batch
func (chain *FullBlockChain) commitBlock(block *types.Block, state *account.AccountDB, receipts types.Receipts) (ok bool, err error) {
	bh := block.Header
	headerBytes, err := types.MarshalBlockHeader(bh)
	if err != nil {
		Logger.Errorf("Fail to json Marshal, error:%s", err.Error())
		return
	}

	//提交state
	if err = chain.saveBlockState(block, state); err != nil {
		return
	}
	//写hash->block
	if err = chain.saveBlockHeader(bh.Hash, headerBytes); err != nil {
		return
	}
	//写height->hash
	if err = chain.saveBlockHeight(bh.Height, bh.Hash.Bytes()); err != nil {
		return
	}
	//写交易和收据
	if err = chain.transactionPool.MarkExecuted(bh.Hash, receipts, block.Transactions, bh.EvictedTxs); err != nil {
		return
	}
	//写latest->hash
	if err = chain.saveCurrentBlock(bh.Hash); err != nil {
		return
	}
	if err = chain.batch.Write(); err != nil {
		return
	}
	chain.updateLatestBlock(state, bh)

	chain.batch.Reset()

	ok = true
	return
}

func (chain *FullBlockChain) resetTop(block *types.BlockHeader) error {
	chain.isAdujsting = true
	defer func() {
		chain.isAdujsting = false
	}()

	//加读写锁，此时阻止读
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

	curr := chain.getLatestBlock()
	for curr.Hash != block.Hash {
		if err = chain.saveBlockHeader(curr.Hash, nil); err != nil {
			return err
		}
		if err = chain.saveBlockHeight(curr.Height, nil); err != nil {
			return err
		}
		txs := chain.GetTransactionPool().GetTransactionsByBlockHash(curr.Hash)
		if err = chain.transactionPool.UnMarkExecuted(curr.Hash, txs); err != nil {
			return err
		}
		curr = chain.queryBlockHeaderByHash(curr.PreHash)
		chain.topBlocks.Remove(curr.Hash)
	}

	if err = chain.saveCurrentBlock(block.Hash); err != nil {
		return err
	}
	if err = chain.batch.Write(); err != nil {
		return err
	}

	state, err := account.NewAccountDB(block.StateTree, chain.stateCache)
	if err != nil {
		return err
	}

	chain.updateLatestBlock(state, block)

	return nil
}

// 删除孤块
func (chain *FullBlockChain) removeOrphan(block *types.Block) error {
	//加读写锁，此时阻止读
	chain.rwLock.Lock()
	defer chain.rwLock.Unlock()

	if nil == block {
		return nil
	}
	hash := block.Header.Hash
	height := block.Header.Height
	Logger.Debugf("remove hash:%s height:%d ", hash.Hex(), height)

	var err error

	if err = chain.saveBlockHeader(hash, nil); err != nil {
		return err
	}
	if err = chain.saveBlockHeight(height, nil); err != nil {
		return err
	}
	if err = chain.transactionPool.UnMarkExecuted(hash, block.Transactions); err != nil {
		return err
	}
	chain.topBlocks.Remove(hash)

	if err = chain.batch.Write(); err != nil {
		return err
	}
	return nil
}

func (chain *FullBlockChain) loadCurrentBlock() *types.BlockHeader {
	bs, err := chain.blocks.Get([]byte(BLOCK_STATUS_KEY))
	if err != nil {
		return nil
	}
	hash := common.BytesToHash(bs)
	return chain.queryBlockHeaderByHash(hash)
}


func (chain *FullBlockChain) hasBlock(hash common.Hash) bool {
	ok, err := chain.blocks.Has(hash.Bytes())
	if err != nil {
		return ok
	}
	return false
	//pre := chain.queryBlockHeaderByHash(bh.PreHash)
	//return pre != nil
}

func (chain *FullBlockChain) hasHeight(h uint64) bool {
	ok, err := chain.blockHeight.Has(utility.UInt64ToByte(h))
	if err != nil {
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
	} else {
		return nil
	}
}

func (chain *FullBlockChain) queryBlockHeaderByHeightFloor(height uint64) *types.BlockHeader {
	iter := chain.blockHeight.NewIterator()
	defer iter.Release()
	if iter.Seek(utility.UInt64ToByte(height)) {
		hash := common.BytesToHash(iter.Value())
		bh := chain.queryBlockHeaderByHash(hash)
		if bh.Height == height {
			return bh
		}
	}
	if iter.Prev() {
		hash := common.BytesToHash(iter.Value())
		return chain.queryBlockHeaderByHash(hash)
	} else {
		return nil
	}
}


func (chain *FullBlockChain) batchGetBlocksAfterHeight(h uint64, limit int) []*types.Block {
	blocks := make([]*types.Block, 0)
	iter := chain.blockHeight.NewIterator()
	defer iter.Release()

	//没有更高的块
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

	txs := chain.transactionPool.GetTransactionsByBlockHash(hash)
	b := &types.Block{
		Header: bh,
		Transactions: txs,
	}
	return b
}

func (chain *FullBlockChain) queryBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	result, err := chain.blocks.Get(hash.Bytes())
	if result != nil {
		var block *types.BlockHeader
		block, err = types.UnMarshalBlockHeader(result)
		if err != nil {
			return nil
		}
		return block
	} else {
		return nil
	}
}

func (chain *FullBlockChain) addTopBlock(b *types.Block)  {
    chain.topBlocks.Add(b.Header.Hash, b)
}

func (chain *FullBlockChain) getTopBlockByHash(hash common.Hash) *types.Block {
	if v, ok := chain.topBlocks.Get(hash); ok {
		return v.(*types.Block)
	}
	return nil
}

func (chain *FullBlockChain) getTopBlockByHeight(height uint64) *types.Block {
	for _, k := range chain.topBlocks.Keys() {
		b := chain.getTopBlockByHash(k.(common.Hash))
		if b.Header.Height == height {
			return b
		}
	}
	return nil
}