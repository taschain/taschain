package handler

import (
	"common"
	"core"
	"consensus/logical"
)

type BlockChainMessageHandler struct{

}

//验证节点 交易集缺失，索要、特定交易 全网广播
//param:hash slice of transaction slice
//      signData
func (h *BlockChainMessageHandler)requestTransactionByHash(hs []common.Hash,sd logical.SignData){}

//接收索要交易请求 查询自身是否有该交易 有的话返回
//param: hash list of transaction slice
//       signData
func (h *BlockChainMessageHandler)onTransactionRequest(hs []common.Hash,sd logical.SignData)core.Transaction{
	var a core.Transaction
	return a
}

//验证节点接收交易 判定是否是待验证blockheader的交易集 是的话累加，全部交易集都拿到之后 开始验证
//param: transaction slice
//       signData
func (h *BlockChainMessageHandler)onMessageTransaction(ts []core.Transaction,sd logical.SignData){}



//对外广播经过组签名的block 全网广播
//param: block
//       member signature
//       signData
func (h *BlockChainMessageHandler)broadcastNewBlock(b core.Block,sd logical.SignData){}

//全网其他节点 接收block 进行验证
//param: block
//       member signature
//       signData
func (h *BlockChainMessageHandler)onMessageNewBlock(b core.Block,sd logical.SignData){}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////