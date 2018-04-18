package biz

import (
	"common"
	"core"
	"consensus/logical"
)

//-----------------------------------------------------回调函数定义-----------------------------------------------------

//本地查询transaction
type queryTracsactionFn func(hs []common.Hash, sd logical.SignData) ([]core.Transaction, error)

//监听到交易到达
type transactionArrivedNotifyBlockChainFn func(ts []core.Transaction, sd logical.SignData)

type transactionArrivedNotifyConsensusFn func(ts []core.Transaction, sd logical.SignData)

//接收到新的块 本地上链
type addNewBlockToChainFn func(b core.Block, sd logical.SignData)


//将接收到的交易加入交易池
type addTransactionToPoolFn func(t core.Transaction)

//根据hash获取对应的block todo 暂不使用
type queryBlocksByHashFn func(h common.Hash) (core.Block, error)

//---------------------------------------------------------------------------------------------------------------------
type BlockChainMessageHandler struct {
	queryTx      queryTracsactionFn
	txGotNofifyB transactionArrivedNotifyBlockChainFn
	txGotNofifyC transactionArrivedNotifyConsensusFn
	addNewBlock  addNewBlockToChainFn

	addTxToPool addTransactionToPoolFn
}

func NewBlockChainMessageHandler(queryTx queryTracsactionFn, txGotNofifyB transactionArrivedNotifyBlockChainFn, txGotNofifyC transactionArrivedNotifyConsensusFn,
	addNewBlock addNewBlockToChainFn, getBlockChainHeight , addTxToPool addTransactionToPoolFn) BlockChainMessageHandler {

	return BlockChainMessageHandler{
		queryTx:             queryTx,
		txGotNofifyC:        txGotNofifyC,
		txGotNofifyB:        txGotNofifyB,
		addNewBlock:         addNewBlock,
		addTxToPool:         addTxToPool,
	}
}

//-----------------------------------------------铸币-------------------------------------------------------------------

//接收索要交易请求 查询自身是否有该交易 有的话返回
//param: hash list of transaction slice
//       signData
func (h BlockChainMessageHandler) onTransactionRequest(hs []common.Hash, sd logical.SignData) []core.Transaction {
	transactions, e := h.queryTx(hs, sd)
	if e != nil {
		return transactions
	}
	return nil
}

//验证节点接收交易 判定是否是待验证blockheader的交易集 是的话累加，全部交易集都拿到之后 开始验证
//param: transaction slice
//       signData
func (h BlockChainMessageHandler) onMessageTransaction(ts []core.Transaction, sd logical.SignData) {
	//todo 有先后顺序  先给鸠兹 再给班德  应该有返回值？ 如果本地查不到需要广播该请求？
	h.txGotNofifyB(ts, sd)
	h.txGotNofifyC(ts, sd)
}

//全网其他节点 接收block 进行验证
//param: block
//       member signature
//       signData
func (h BlockChainMessageHandler) onMessageNewBlock(b core.Block, sd logical.SignData) {

	h.addNewBlock(b, sd)
}

//接收来自客户端的交易
func (h BlockChainMessageHandler) onNewTransaction(t core.Transaction, sd logical.SignData) {}
