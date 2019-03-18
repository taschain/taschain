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
	"math"
	"middleware/types"
	"storage/account"
	"consensus/groupsig"
	"math/big"
	"fmt"
	"storage/vm"
)


func (chain *FullBlockChain) IsLightMiner() bool {
	return chain.isLightMiner
}


func (chain *FullBlockChain) Height() uint64 {
	if nil == chain.latestBlock {
		return math.MaxUint64
	}
	return chain.QueryTopBlock().Height
}

func (chain *FullBlockChain) TotalQN() uint64 {
	if nil == chain.latestBlock {
		return 0
	}
	return chain.QueryTopBlock().TotalQN
}

func (chain *FullBlockChain) GetTransactionByHash(h common.Hash) (*types.Transaction, error) {
	return chain.transactionPool.GetTransaction(h)
}

func (chain *FullBlockChain) GetTransactionPool() TransactionPool {
	return chain.transactionPool
}

func (chain *FullBlockChain) IsAdujsting() bool {
	return chain.isAdujsting
}

func (chain *FullBlockChain) LatestStateDB() *account.AccountDB {
	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()
	return chain.latestStateDB
}

func (chain *FullBlockChain) missTransaction(bh types.BlockHeader, txs []*types.Transaction) (bool, []common.Hash, []*types.Transaction) {
	var missing []common.Hash
	var transactions []*types.Transaction
	if nil == txs {
		transactions, missing, _ = chain.GetTransactions(bh.Hash, bh.Transactions)
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
		m := &transactionRequestMessage{TransactionHashes: missing, CurrentBlockHash: bh.Hash, BlockHeight: bh.Height, BlockPv: bh.ProveValue,}
		go requestTransaction(*m, castorId.String())
		return true, missing, transactions
	}
	return false, missing, transactions
}

func (chain *FullBlockChain) GetTransactions(blockHash common.Hash, txHashList []common.Hash) ([]*types.Transaction, []common.Hash, error) {
	if nil == txHashList || 0 == len(txHashList) {
		return nil, nil, ErrNil
	}

	verifiedBody, _ := chain.verifiedBodyCache.Get(blockHash)
	var verifiedTxs []*types.Transaction
	if nil != verifiedBody {
		verifiedTxs = verifiedBody.([]*types.Transaction)
	}

	txs := make([]*types.Transaction, 0)
	need := make([]common.Hash, 0)
	var err error
	for _, hash := range txHashList {
		var tx *types.Transaction
		if verifiedTxs != nil {
			for _, verifiedTx := range verifiedTxs {
				if verifiedTx.Hash == hash {
					tx = verifiedTx
					break
				}
			}
		}

		if tx == nil {
			tx, err = chain.transactionPool.GetTransaction(hash)
		}

		if tx != nil {
			txs = append(txs, tx)
		} else {
			need = append(need, hash)
		}
	}
	return txs, need, err
}

//查询最高块
func (chain *FullBlockChain) QueryTopBlock() *types.BlockHeader {
	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()

	return chain.getLatestBlock()
}

func (chain *FullBlockChain) HasBlock(hash common.Hash) bool {
    return chain.hasBlock(hash)
}

func (chain *FullBlockChain) HasHeight(height uint64) bool {
	return chain.hasHeight(height)
}

// 根据指定高度查询块
// 带有缓存
func (chain *FullBlockChain) QueryBlockByHeight(height uint64) *types.BlockHeader {
	b := chain.getTopBlockByHeight(height)
	if b != nil {
		return b.Header
	}

	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()

	return chain.queryBlockHeaderByHeight(height)
}


//根据指定哈希查询块
func (chain *FullBlockChain) QueryBlockHeaderByHash(hash common.Hash) *types.BlockHeader {
	if b := chain.getTopBlockByHash(hash); b != nil {
		return b.Header
	}
	return chain.queryBlockHeaderByHash(hash)
}

func (chain *FullBlockChain) QueryBlockByHash(hash common.Hash) *types.Block {
	if b := chain.getTopBlockByHash(hash); b != nil {
		return b
	}

	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()

	return chain.queryBlockByHash(hash)
}

func (chain *FullBlockChain) QueryBlockCeil(height uint64) *types.Block {
	if b := chain.getTopBlockByHeight(height); b != nil {
		return b
	}

	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()

	hash := chain.queryBlockHashCeil(height)
	if hash == nil {
		return nil
	}
	return chain.queryBlockByHash(*hash)
}

func (chain *FullBlockChain) QueryBlockFloor(height uint64) *types.Block {
	if b := chain.getTopBlockByHeight(height); b != nil {
		return b
	}

	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()

	header := chain.queryBlockHeaderByHeightFloor(height)
	if header == nil {
		return nil
	}
	txs := chain.transactionPool.GetTransactionsByBlockHash(header.Hash)
	b := &types.Block{
		Header: header,
		Transactions: txs,
	}
	return b
}

func (chain *FullBlockChain) GetBalance(address common.Address) *big.Int {
	if nil == chain.latestStateDB {
		return nil
	}

	return chain.latestStateDB.GetBalance(common.BytesToAddress(address.Bytes()))
}

func (chain *FullBlockChain) GetNonce(address common.Address) uint64 {
	if nil == chain.latestStateDB {
		return 0
	}

	return chain.latestStateDB.GetNonce(common.BytesToAddress(address.Bytes()))
}


func (chain *FullBlockChain) GetAccountDBByHash(hash common.Hash) (vm.AccountDB, error) {
	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()

	header := chain.queryBlockHeaderByHash(hash)
	return account.NewAccountDB(header.StateTree, chain.stateCache)
}

func (chain *FullBlockChain) GetAccountDBByHeight(height uint64) (vm.AccountDB, error) {
	chain.rwLock.RLock()
	defer chain.rwLock.RUnlock()

	h := height
	header := chain.queryBlockHeaderByHeightFloor(height)
	if header == nil {
		return nil, fmt.Errorf("no data at height %v-%v", h, height)
	}
	return account.NewAccountDB(header.StateTree, chain.stateCache)
}