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
)

//主链接口
type BlockChainI interface {
	//主要方法族

	//根据哈希取得某个交易
	// 如果本地有，则立即返回。否则需要调用p2p远程获取
	GetTransactionByHash(h common.Hash) (*types.Transaction, error)

	//构建一个铸块（组内当前铸块人同步操作）
	CastingBlock(height uint64, nonce uint64, proveValue *big.Int, castor []byte, groupid []byte) *types.Block

	//根据BlockHeader构建block
	GenerateBlock(bh types.BlockHeader) *types.Block

	//验证一个铸块（如本地缺少交易，则异步网络请求该交易）
	//返回:=0, 验证通过；=-1，验证失败；=1，缺少交易，已异步向网络模块请求
	//返回缺失交易列表
	VerifyCastingBlock(bh types.BlockHeader) ([]common.Hash, int8, *core.AccountDB, vtypes.Receipts)

	//铸块成功，上链
	//返回:=0,上链成功；=-1，验证失败；=1,上链成功，上链过程中发现分叉并进行了权重链调整
	AddBlockOnChain(b *types.Block) int8

	//辅助方法族
	//查询最高块
	QueryTopBlock() *types.BlockHeader

	//根据指定哈希查询块
	QueryBlockByHash(hash common.Hash) *types.BlockHeader

	QueryBlockBody(blockHash common.Hash) []*types.Transaction

	//根据指定高度查询块
	QueryBlockByHeight(height uint64) *types.BlockHeader

	// 返回等待入块的交易池
	GetTransactionPool() *TransactionPool

	//清除链所有数据
	Clear() error
	//是否正在调整分叉
	IsAdujsting() bool

    GenerateBonus(targetIds []int, blockHash common.Hash, groupId []byte, totalValue uint64) (*types.Bonus,*types.Transaction)

	AddBonusTrasanction(transaction *types.Transaction)
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
