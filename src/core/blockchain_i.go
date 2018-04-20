package core

import (
	"common"
	"vm/core/state"
	"vm/core/types"
	"math/big"
)

//主链接口
type BlockChainI interface {
	//主要方法族

	//根据哈希取得某个交易
	// 如果本地有，则立即返回。否则需要调用p2p远程获取
	GetTransactionByHash(h common.Hash) (*Transaction, error)

	//构建一个铸块（组内当前铸块人同步操作）
	CastingBlock() *Block

	//验证一个铸块（如本地缺少交易，则异步网络请求该交易）
	//返回:=0, 验证通过；=-1，验证失败；=1，缺少交易，已异步向网络模块请求
	VerifyCastingBlock(bh BlockHeader) int8

	//铸块成功，上链
	//返回:=0,上链成功；=-1，验证失败；=1,上链成功，上链过程中发现分叉并进行了权重链调整
	AddBlockOnChain(b *Block) int8

	//辅助方法族
	//查询最高块
	QueryTopBlock() *BlockHeader

	//根据指定哈希查询块
	QueryBlockByHash(hash common.Hash) *BlockHeader

	//根据指定高度查询块
	QueryBlockByHeight(height uint64) *BlockHeader

	// 返回等待入块的交易池
	GetTransactionPool() *TransactionPool

	//清除链所有数据
	Clear() error
}

//组管理接口
type GroupInfoI interface {
}

// VM执行器
type VMExecutor interface {
	//Execute(statedb *state.StateDB, block *Block) (types.Receipts, *common.Hash, uint64, error)
	Execute(statedb *state.StateDB, block *Block) (types.Receipts, *common.Hash, uint64, error)
}

// 账户查询接口
type AccountRepository interface {
	GetBalance(address common.Address) *big.Int

	GetNonce(address common.Address) uint64

}

// transactionPool 接口
type TransactionPoolI interface {

}
