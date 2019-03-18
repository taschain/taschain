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
	"storage/account"
	"storage/vm"

	"math/big"
	"middleware/types"
)

//主链接口
type BlockChain interface {
	vm.ChainReader
	AccountRepository

	IsLightMiner() bool

	//构建一个铸块（组内当前铸块人同步操作）
	CastBlock(height uint64, proveValue *big.Int, proveRoot common.Hash, qn uint64, castor []byte, groupid []byte) *types.Block

	//根据BlockHeader构建block
	GenerateBlock(bh types.BlockHeader) *types.Block

	//验证一个铸块（如本地缺少交易，则异步网络请求该交易）
	//返回:=0, 验证通过；=-1，验证失败；=1，缺少交易，已异步向网络模块请求
	//返回缺失交易列表
	VerifyBlock(bh types.BlockHeader) ([]common.Hash, int8)

	//铸块成功，上链
	//返回值: 0,上链成功
	//       -1，验证失败
	//        1, 丢弃该块(链上已存在该块）
	//        2,丢弃该块（链上存在QN值更大的相同高度块)
	//        3,分叉调整
	AddBlockOnChain(source string, b *types.Block) types.AddBlockResult

	TotalQN() uint64


	LatestStateDB() *account.AccountDB

	//query block with body by hash
	QueryBlockByHash(hash common.Hash) *types.Block

	//query first block whose height >= height
	QueryBlockCeil(height uint64) *types.Block

	//query first block whose height <= height
	QueryBlockFloor(height uint64) *types.Block

	//根据哈希取得某个交易
	// 如果本地有，则立即返回。否则需要调用p2p远程获取
	GetTransactionByHash(h common.Hash) (*types.Transaction, error)

	// 返回等待入块的交易池
	GetTransactionPool() TransactionPool

	IsAdujsting() bool

	Remove(block *types.Block) bool

	//清除链所有数据
	Clear() error

	Close()

	AddBonusTrasanction(transaction *types.Transaction)

	GetBonusManager() *BonusManager

	GetAccountDBByHash(hash common.Hash) (vm.AccountDB, error)

	GetAccountDBByHeight(height uint64) (vm.AccountDB, error)

	GetConsensusHelper() types.ConsensusHelper

	GetTransactions(blockHash common.Hash, txHashList []common.Hash) ([]*types.Transaction, []common.Hash, error)
}

type ExecutedTransaction struct {
	Receipt     *types.Receipt
	Transaction *types.Transaction
}

type ReceiptStore struct {
	Receipt     *types.Receipt
	//Transaction *types.Transaction
	BlockHash  common.Hash
}

type TransactionPool interface {
	PackForCast() []*types.Transaction

	//add new transaction to the transaction pool
	AddTransaction(tx *types.Transaction) (bool, error)

	//rcv transactions broadcast from other nodes
	AddBroadcastTransactions(txs []*types.Transaction)

	//add  local miss transactions while verifying blocks to the transaction pool
	AddMissTransactions(txs []*types.Transaction)

	GetTransaction(hash common.Hash) (*types.Transaction, error)

	GetTransactionStatus(hash common.Hash) (uint, error)

	GetReceipt(hash common.Hash) *types.Receipt

	GetExecuted(hash common.Hash) *ExecutedTransaction

	GetReceived() []*types.Transaction

	TxNum() uint64

	MarkExecuted(blockHash common.Hash, receipts types.Receipts, txs []*types.Transaction, evictedTxs []common.Hash) error

	UnMarkExecuted(blockHash common.Hash, txs []*types.Transaction) error

	Clear()

	GetTransactionsByBlockHash(hash common.Hash) []*types.Transaction
}

//组管理接口
type GroupInfoI interface {
}

// VM执行器
type VMExecutor interface {
	Execute(statedb *account.AccountDB, block *types.Block) (types.Receipts, *common.Hash, uint64, error)
}

// 账户查询接口
type AccountRepository interface {
	GetBalance(address common.Address) *big.Int

	GetNonce(address common.Address) uint64
}
