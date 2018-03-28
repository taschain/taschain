package core

import (
	"common"
)

//主链接口
type BlockChainI interface {
	//主要方法族
	//根据哈希取得某个交易
	GetTransactionByHash(h common.Hash) Transaction
	//构建一个铸块（组内当前铸块人同步操作）
	CastingBlock() Block
	//验证一个铸块（如本地缺少交易，则异步网络请求该交易）
	//返回:=0, 验证通过；=-1，验证失败；=1，缺少交易，已异步向网络模块请求
	VerifyCastingBlock(bh BlockHeader) int8
	//铸块成功，上链
	//返回:=0,上链成功；=-1，验证失败；=1,上链成功，上链过程中发现分叉并进行了权重链调整
	AddBlockOnChain(b Block) int8
	//辅助方法族
	//查询最高块
	QueryTopBlock() BlockHeader
	//根据指定哈希查询块
	QueryBlockByHash() BlockHeader
	//根据指定高度查询块
	QueryBlockByHeight() BlockHeader
}

//组管理接口
type GroupInfoI interface {
}
