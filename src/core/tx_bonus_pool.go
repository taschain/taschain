package core

import (
	"github.com/hashicorp/golang-lru"
	"middleware/types"
	"common"
)

/*
**  Creator: pxf
**  Date: 2019/3/25 上午9:46
**  Description: 
*/

type bonusPool struct {
	bm *BonusManager
	pool *lru.Cache				//txHash -> *Transaction
	blockHashIndex *lru.Cache	//blockHash -> []*Transaction
}

func newBonusPool(pm *BonusManager, size int) *bonusPool {
	pc, _ := lru.New(size)
	idx, _ := lru.New(size)
	return &bonusPool{
		pool:pc,
		blockHashIndex:idx,
		bm: pm,
	}
}

func (bp *bonusPool) add(tx *types.Transaction) bool {
	if bp.pool.Contains(tx.Hash) {
		return false
	}
    bp.pool.Add(tx.Hash, tx)
    blockHash := bp.bm.parseBonusBlockHash(tx)

    var txs []*types.Transaction
	if v, ok := bp.blockHashIndex.Get(blockHash); ok {
		txs = v.([]*types.Transaction)
	} else {
		txs = make([]*types.Transaction, 0)
	}
	txs = append(txs, tx)
	bp.blockHashIndex.Add(blockHash, txs)
	return true
}

func (bp *bonusPool) remove(txHash common.Hash)  {
    tx, _ := bp.pool.Get(txHash)
	if tx != nil {
		bp.pool.Remove(txHash)
		bhash := bp.bm.parseBonusBlockHash(tx.(*types.Transaction))
		txs, _ := bp.blockHashIndex.Get(bhash)
		if txs != nil {
			//Logger.Debugf("remove from bonus pool size %v, block %v", len(txs.([]*types.Transaction)), bhash.String())
			for _, trans := range txs.([]*types.Transaction) {
				if trans.Hash != txHash {
					bp.pool.Remove(trans.Hash)
				}
			}
			bp.blockHashIndex.Remove(bhash)
		}
	}
}

func (bp *bonusPool) get(hash common.Hash) *types.Transaction {
    if v, ok :=  bp.pool.Get(hash); ok {
    	return v.(*types.Transaction)
	}
	return nil
}

func (bp *bonusPool) len() int {
    return bp.pool.Len()
}

func (bp *bonusPool) contains(hash common.Hash) bool {
    return bp.pool.Contains(hash)
}

func (bp *bonusPool) forEach(f func(tx *types.Transaction) bool)  {
	for _, k := range bp.pool.Keys() {
		v := bp.get(k.(common.Hash))
		if v != nil {
			if !f(v) {
				break
			}
		}
	}
}