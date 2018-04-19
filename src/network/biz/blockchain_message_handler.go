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

//获取本地组链高度
type getLocalGroupChainHeightFn func() (int, error)

//根据高度获取对应的组信息
type queryGroupInfoByHeightFn func(hs []int) (map[int]logical.StaticGroupInfo, error)

//同步多组到链上
type syncGroupInfoToChainFn func(hbm map[int]logical.StaticGroupInfo, sd logical.SignData)

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

	getGroupChainHeight getLocalGroupChainHeightFn
	queryGroup          queryGroupInfoByHeightFn
	syncGroup           syncGroupInfoToChainFn

	addTxToPool addTransactionToPoolFn
}

func NewBlockChainMessageHandler(queryTx queryTracsactionFn, txGotNofifyB transactionArrivedNotifyBlockChainFn, txGotNofifyC transactionArrivedNotifyConsensusFn,
	addNewBlock addNewBlockToChainFn, getBlockChainHeight, getGroupChainHeight getLocalGroupChainHeightFn, queryGroup queryGroupInfoByHeightFn,
	syncGroup syncGroupInfoToChainFn, addTxToPool addTransactionToPoolFn) BlockChainMessageHandler {

	return BlockChainMessageHandler{
		queryTx:             queryTx,
		txGotNofifyC:        txGotNofifyC,
		txGotNofifyB:        txGotNofifyB,
		addNewBlock:         addNewBlock,
		getGroupChainHeight: getGroupChainHeight,
		queryGroup:          queryGroup,
		syncGroup:           syncGroup,
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

////////////////////////////////////////////////////////组同步//////////////////////////////////////////////////////////

//接收到组链高度信息请求，返回自身组链高度
//param: signData
func (h BlockChainMessageHandler) onGroupChainHeightRequest(sd logical.SignData) int {
	//todo 这里是否要签名入参
	height, e := h.getGroupChainHeight()
	if e != nil {
		return height
	}
	return -1
}

//接收到组链高度信息，对比自身组链高度，判定是否发起同步
//param: group chain height
//       signData
func (h BlockChainMessageHandler) onMessageGroupChainHeight(height int, sd logical.SignData) {

	//TODO 时间窗口内找出最高块的高度
	bestHeight := 0
	//todo 这里是否要签名入参
	height, e := h.getGroupChainHeight()
	if e != nil {
		if bestHeight > height {
			//todo 发起组同步
			//requestGroupInfoByHeight
		}
	}
}

//节点接收索要组信息请求
//param: group height slice
//       signData
func (h BlockChainMessageHandler) onGroupInfoRequest(gs []int, sd logical.SignData) map[int]logical.StaticGroupInfo {

	hgm, e := h.queryGroup(gs)
	if e != nil {
		return nil
	}
	return hgm
}

//节点收到组信息
//param: heigth group map
//       signData
func (h BlockChainMessageHandler) onMessageGroupInfo(hgm map[int]logical.StaticGroupInfo, sd logical.SignData) {
	h.syncGroup(hgm, sd)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

//接收来自客户端的交易
func (h BlockChainMessageHandler) onNewTransaction(t core.Transaction, sd logical.SignData) {}
