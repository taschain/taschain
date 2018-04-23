package biz

import (
	"common"
	"core"
)

//-----------------------------------------------------回调函数定义-----------------------------------------------------

//本地查询transaction
type queryTransactionFn func(hs []common.Hash) ([]*core.Transaction, error)

//监听到交易到达
type transactionArrivedNotifyBlockChainFn func(ts []*core.Transaction,)error

type transactionArrivedNotifyConsensusFn func(ts []*core.Transaction)error

//接收到新的块 本地上链  先调用鸠兹  返回成功再调用班德
//porcess.go OnMessageBlock
type addNewBlockToChainFn func(b *core.Block, sig []byte)


//将接收到的交易加入交易池
type addTransactionToPoolFn func(t []*core.Transaction)

//根据hash获取对应的block  暂不使用
//type queryBlocksByHashFn func(h common.Hash) (core.Block, error)

//---------------------------------------------------------------------------------------------------------------------
type BlockChainMessageHandler struct {
	queryTx      queryTransactionFn
	txGotNofifyB transactionArrivedNotifyBlockChainFn
	txGotNofifyC transactionArrivedNotifyConsensusFn
	addNewBlock  addNewBlockToChainFn

	addTxToPool addTransactionToPoolFn
}

func NewBlockChainMessageHandler(queryTx queryTransactionFn, txGotNofifyB transactionArrivedNotifyBlockChainFn, txGotNofifyC transactionArrivedNotifyConsensusFn,
	addNewBlock addNewBlockToChainFn , addTxToPool addTransactionToPoolFn) BlockChainMessageHandler {

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
//todo 如果本地查不到需要  自己广播该请求
func (h BlockChainMessageHandler) onTransactionRequest(hs []common.Hash) []*core.Transaction {
	transactions, e := h.queryTx(hs)
	if e != nil {
		return transactions
	}
	return nil
}

//验证节点接收交易 判定是否是待验证blockheader的交易集 是的话累加，全部交易集都拿到之后 开始验证
//param: transaction slice
//       signData
func (h BlockChainMessageHandler) onMessageTransaction(ts []*core.Transaction) {
	//todo 有先后顺序  先给鸠兹  鸠兹没问题 再给班德  应该有返回值error
	h.txGotNofifyB(ts)
	h.txGotNofifyC(ts)
}

//全网其他节点 接收block 进行验证
//param: block
//       member signature
//       signData
func (h BlockChainMessageHandler) onMessageNewBlock(b *core.Block, sig []byte) {

	h.addNewBlock(b, sig)
}

//接收来自客户端的交易
func (h BlockChainMessageHandler) onNewTransaction(t core.Transaction) {}
