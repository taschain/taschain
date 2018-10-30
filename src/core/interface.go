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
	vtypes "storage/core/types"
	"math/big"
	"middleware/types"
	"storage/core"
	"storage/core/vm"
)

//主链接口
type BlockChain interface {
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
	//返回:=0,上链成功；=-1，验证失败；=1,上链成功，上链过程中发现分叉并进行了权重链调整
	AddBlockOnChain(b *types.Block) int8

	Height() uint64

	TotalQN() uint64

	//查询最高块
	QueryTopBlock() *types.BlockHeader

	//根据指定哈希查询块
	QueryBlockHeaderByHash(hash common.Hash) *types.BlockHeader

	QueryBlockByHash(hash common.Hash) *types.Block

	//根据指定高度查询块
	QueryBlockByHeight(height uint64) *types.BlockHeader

	QueryBlockBody(blockHash common.Hash) []*types.Transaction

	QueryBlock(height uint64) *types.Block

	//根据哈希取得某个交易
	// 如果本地有，则立即返回。否则需要调用p2p远程获取
	GetTransactionByHash(h common.Hash) (*types.Transaction, error)

	// 返回等待入块的交易池
	GetTransactionPool() TransactionPool

	GetBalance(address common.Address) *big.Int

	GetNonce(address common.Address) uint64

	GetSateCache() core.Database

	IsAdujsting() bool

	SetAdujsting(isAjusting bool)

	Remove(header *types.BlockHeader)

	//清除链所有数据
	Clear() error

	Close()

	GetTrieNodesByExecuteTransactions(header *types.BlockHeader, transactions []*types.Transaction, addresses []common.Address) *[]types.StateNode

	InsertStateNode(nodes *[]types.StateNode)

	AddBonusTrasanction(transaction *types.Transaction)

	GetBonusManager() *BonusManager

	GetAccountDBByHash(hash common.Hash) (vm.AccountDB, error)

	GetConsensusHelper() types.ConsensusHelper

	GetTraceHeader(hash []byte) *types.BlockHeader

	GetCheckValue(height uint64) (common.Hash, error)

	ProcessChainPiece(id string, chainPiece []*types.BlockHeader, topHeader *types.BlockHeader)
}

type TransactionPool interface {
	AddTransaction(tx *types.Transaction) (bool, error)

	AddTransactions(txs []*types.Transaction) error

	MarkExecuted(receipts vtypes.Receipts, txs []*types.Transaction)

	Remove(hash common.Hash, transactions []common.Hash, evictedTxs []common.Hash)

	UnMarkExecuted(txs []*types.Transaction)

	GetTransaction(hash common.Hash) (*types.Transaction, error)

	GetTransactions(reservedHash common.Hash, hashes []common.Hash) ([]*types.Transaction, []common.Hash, error)

	GetTransactionsForCasting() []*types.Transaction

	GetTransactionStatus(hash common.Hash) (uint, error)

	ReserveTransactions(hash common.Hash, txs []*types.Transaction)

	GetReceived() []*types.Transaction

	Clear()
}

//组管理接口
type GroupInfoI interface {
}

// VM执行器
type VMExecutor interface {
	//Execute(statedb *state.StateDB, block *Block) (types.Receipts, *common.Hash, uint64, error)
	Execute(statedb *core.AccountDB, block *types.Block) (vtypes.Receipts, *common.Hash, uint64, error)
}

// 账户查询接口
type AccountRepository interface {
	GetBalance(address common.Address) *big.Int

	GetNonce(address common.Address) uint64
}

// chain 对于投票事件接口
type VoteProcessor interface {
	BeforeExecuteTransaction(b *types.Block, db core.AccountDB, tx *types.Transaction) ([]byte, error)
	AfterAllTransactionExecuted(b *types.Block, stateDB core.AccountDB, receipts vtypes.Receipts) error
}
