package biz

import (
	"common"
	"core"
	"time"
	"taslog"
	"consensus/logical"
)

//-----------------------------------------------------回调函数定义-----------------------------------------------------

//本地查询transaction
type queryTransactionFn func(hs []common.Hash) ([]*core.Transaction, error)

//监听到交易到达
type transactionArrivedNotifyBlockChainFn func(ts []*core.Transaction) error

type transactionArrivedNotifyConsensusFn func(ts []*core.Transaction) error

//接收到新的块 本地上链
type addNewBlockToChainFn func(b *core.Block) error

//接收到新的块 通知共识
//porcess.go OnMessageBlock
type newBlockNotifyConsensusFn func(sbm *logical.ConsensusBlockMessage)

//验证节点 交易集缺失，索要、特定交易 全网广播
//Peer.BroadcastTransactionRequest
type broadcastTransactionRequestFn func(m TransactionRequestMessage)

//本地查询到交易，返回请求方
//Peer.SendTransactions
type sendTransactionsFn func(txs []*core.Transaction, sourceId string)

//根据hash获取对应的block  暂不使用
//type queryBlocksByHashFn func(h common.Hash) (core.Block, error)

//---------------------------------------------------------------------------------------------------------------------

const MAX_TRANSACTION_REQUEST_INTERVAL = 10 * time.Second

var logger = taslog.GetLogger(taslog.P2PConfig)

type BlockChainMessageHandler struct {
	queryTx                 queryTransactionFn
	txGotNofifyB            transactionArrivedNotifyBlockChainFn
	txGotNofifyC            transactionArrivedNotifyConsensusFn
	addNewBlock             addNewBlockToChainFn
	newBlockNotifyConsensus newBlockNotifyConsensusFn

	broadcastTxReq broadcastTransactionRequestFn
	sendTxs        sendTransactionsFn
}

func NewBlockChainMessageHandler(queryTx queryTransactionFn, txGotNofifyB transactionArrivedNotifyBlockChainFn, txGotNofifyC transactionArrivedNotifyConsensusFn,
	addNewBlock addNewBlockToChainFn, broadcastTxReq broadcastTransactionRequestFn, sendTxs sendTransactionsFn, newBlockNotifyConsensus newBlockNotifyConsensusFn) BlockChainMessageHandler {

	return BlockChainMessageHandler{
		queryTx:                 queryTx,
		txGotNofifyC:            txGotNofifyC,
		txGotNofifyB:            txGotNofifyB,
		addNewBlock:             addNewBlock,
		newBlockNotifyConsensus: newBlockNotifyConsensus,

		broadcastTxReq: broadcastTxReq,
		sendTxs:        sendTxs,
	}
}

//-----------------------------------------------铸币-------------------------------------------------------------------

//接收索要交易请求 查询自身是否有该交易 有的话返回, 没有的话自己广播该请求
func (h BlockChainMessageHandler) OnTransactionRequest(m *TransactionRequestMessage) {
	requestTime := m.RequestTime
	now := time.Now()
	interval := now.Sub(requestTime)
	if interval > MAX_TRANSACTION_REQUEST_INTERVAL {
		return
	}

	transactions, e := h.queryTx(m.TransactionHashes)
	if e != nil {
		logger.Error("OnTransactionRequest get local transaction error:%s", e.Error())
		h.broadcastTxReq(*m)
		return
	}

	if len(transactions) == 0 {
		logger.Info("Local do not have transaction,broadcast this message!")
		h.broadcastTxReq(*m)
		return
	}
	h.sendTxs(transactions, m.SourceId)
}

//验证节点接收交易 判定是否是待验证blockheader的交易集 是的话累加，全部交易集都拿到之后 开始验证
func (h BlockChainMessageHandler) OnMessageTransaction(ts []*core.Transaction) {
	e := h.txGotNofifyB(ts)
	if e != nil {
		logger.Errorf("OnMessageTransaction notify block error:%s \n", e.Error())
		return
	}
	//todo  调用班德
	h.txGotNofifyC(ts)
}

//全网其他节点 接收block 进行验证
//param: block
//       member signature
//       signData
func (h BlockChainMessageHandler) OnMessageNewBlock(b *core.Block) {
	e := h.addNewBlock(b)
	if e != nil {
		logger.Errorf("Add new block to chain error:%s \n", e.Error())
		return
	}
	//todo : 调用班德 这里参数其实不对
	var cbm logical.ConsensusBlockMessage
	h.newBlockNotifyConsensus(&cbm)
}

//接收来自客户端的交易
func (h BlockChainMessageHandler) OnNewTransaction(t core.Transaction) {}
