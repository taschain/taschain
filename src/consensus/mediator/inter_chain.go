package mediator

import (
	"common"
	"core"
)

///////////////////////////////////////////////////////////////////////////////
//主链提供给共识模块的接口
//根据哈希取得某个交易
//h:交易的哈希; forced:如本地不存在是否要发送异步网络请求
//int=0，返回合法的交易；=-1，交易异常；=1，本地不存在，已发送网络请求；=2，本地不存在
type GetTransactionByHash func(h common.Hash, forced bool) (int, core.Transaction)

//构建一个铸块（组内当前铸块人同步操作）
type CastingBlock func() (core.Block, error)

//验证一个铸块（如本地缺少交易，则异步网络请求该交易）
//返回:=0, 验证通过；=-1，验证失败；=1，缺少交易，已发送网络请求
type VerifyCastingBlock func(bh core.BlockHeader) int

//铸块成功，上链
//返回:=0,上链成功；=-1，验证失败；=1,上链成功，上链过程中发现分叉并进行了权重链调整
type AddBlockOnChain func(b core.Block) int

//查询最高块
type QueryTopBlock func() core.BlockHeader

//根据指定哈希查询块，不存在则返回nil。
type QueryBlockByHash func() *core.BlockHeader

//根据指定高度查询块，不存在则返回nil。
type QueryBlockByHeight func() *core.BlockHeader
