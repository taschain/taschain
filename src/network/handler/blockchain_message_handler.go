package handler

import (
	"common"
	"core"
	"consensus/logical"
	"consensus/groupsig"
)

//-----------------------------------------------------回调函数定义-----------------------------------------------------

//本地查询transaction
type queryTracsactionFn func(hs []common.Hash) ([]core.Transaction, error)

//监听到交易到达 //todo  班德 鸠兹 各自实现一个
type transactionArrivedNotifyFn func(ts []core.Transaction)

//接收到新的块 本地上链
type addNewBlockToChainFn func(b core.Block)

//获取本地链高度
type getLocalBlockChainHeightFn func() (int, error)

//根据高度获取对应的block
type queryBlocksByHeightFn func(hs []int) (map[int]core.Block, error)

//同步多个区块到链上
type syncBlocksToChainFn func(hbm map[int]core.Block)

//获取本地组链高度
type getLocalGroupChainHeightFn func() (int, error)

//根据高度获取对应的组信息
type queryGroupInfoByHeightFn func(hs []int) (map[int]logical.StaticGroupInfo, error)

//同步多组到链上
type syncGroupInfoToChainFn func(hbm map[int]logical.StaticGroupInfo)

//将接收到的交易加入交易池
type addTransactionToPoolFn func(t core.Transaction)

//根据hash获取对应的block todo 暂不使用
type queryBlocksByHashFn func(h common.Hash) (core.Block, error)

//---------------------------------------------------------------------------------------------------------------------
type BlockChainMessageHandler struct {

	sv *signValidator
	queryTx     queryTracsactionFn
	txGot       transactionArrivedNotifyFn
	addNewBlock addNewBlockToChainFn

	getBlockChainHeight getLocalBlockChainHeightFn
	queryBlock          queryBlocksByHeightFn
	syncBlocks          syncBlocksToChainFn

	getGroupChainHeight getLocalGroupChainHeightFn
	queryGroup          queryGroupInfoByHeightFn
	syncGroup           syncGroupInfoToChainFn

	addTxToPool addTransactionToPoolFn
}

func newBlockChainMessageHandler(queryTx queryTracsactionFn, txGot transactionArrivedNotifyFn,
	addNewBlock addNewBlockToChainFn, getBlockChainHeight getLocalBlockChainHeightFn, queryBlock queryBlocksByHeightFn,
	syncBlocks syncBlocksToChainFn, getGroupChainHeight getLocalGroupChainHeightFn, queryGroup queryGroupInfoByHeightFn,
	syncGroup syncGroupInfoToChainFn, addTxToPool addTransactionToPoolFn) *BlockChainMessageHandler {

	return &BlockChainMessageHandler{
		sv:                  GetSignValidatorInstance(),
		queryTx:             queryTx,
		txGot:               txGot,
		addNewBlock:         addNewBlock,
		getBlockChainHeight: getBlockChainHeight,
		queryBlock:          queryBlock,
		syncBlocks:          syncBlocks,
		getGroupChainHeight: getGroupChainHeight,
		queryGroup:          queryGroup,
		syncGroup:           syncGroup,
		addTxToPool:         addTxToPool,
	}
}

//-----------------------------------------------铸币-------------------------------------------------------------------
//验证节点 交易集缺失，索要、特定交易 全网广播
//param:hash slice of transaction slice
//      signData
func (h *BlockChainMessageHandler) requestTransactionByHash(hs []common.Hash, sd logical.SignData) {}

//接收索要交易请求 查询自身是否有该交易 有的话返回
//param: hash list of transaction slice
//       signData
func (h *BlockChainMessageHandler) onTransactionRequest(hs []common.Hash, sd logical.SignData) []core.Transaction {
	signResult := h.sv.isNodeSignCorrect(sd)
	if signResult {
		transactions, e := h.queryTx(hs)
		if e != nil {
			return transactions
		}
	}
	return nil
}

//验证节点接收交易 判定是否是待验证blockheader的交易集 是的话累加，全部交易集都拿到之后 开始验证
//param: transaction slice
//       signData
func (h *BlockChainMessageHandler) onMessageTransaction(ts []core.Transaction, sd logical.SignData) {
	signResult := h.sv.isNodeSignCorrect(sd)
	if signResult {
		h.txGot(ts)
	}
}

//对外广播经过组签名的block 全网广播
//param: block
//       member signature
//       signData
func (h *BlockChainMessageHandler) broadcastNewBlock(b core.Block, sd logical.SignData) {}

//全网其他节点 接收block 进行验证
//param: block
//       member signature
//       signData
func (h *BlockChainMessageHandler) onMessageNewBlock(b core.Block, sd logical.SignData) {
	signResult := h.sv.isGroupSignCorrect(sd)
	if signResult {
		h.addNewBlock(b)
	}
}

/////////////////////////////////////////////////////链同步/////////////////////////////////////////////////////////////

//广播索要链高度
//param: signData
func (h *BlockChainMessageHandler) requestBlockChainHeight(sd logical.SignData) {}

//接收到链高度信息请求，返回自身链高度
//param: signData
func (h *BlockChainMessageHandler) onBlockChainHeightRequest(sd logical.SignData) int {
	signResult := h.sv.isNodeSignCorrect(sd)
	if signResult {
		h, e := h.getBlockChainHeight()
		if e != nil {
			return h
		}
	}
	return -1
}

//接收到链高度信息，对比自身高度，判定是否发起同步
//param: block chain height
//       signData
func (h *BlockChainMessageHandler) onMessageBlockChainHeight(height int, sd logical.SignData) {
	signResult := h.sv.isNodeSignCorrect(sd)
	if signResult {
		//TODO 时间窗口内找出最高块的高度
		bestHeight := 0

		h, e := h.getBlockChainHeight()
		if e != nil {
			if bestHeight > h {
				//todo 发起块同步
				//requestBlockByHeight
			}
		}
	}
}

//向某一节点请求Block
//param: target peer id
//       block height slice
//       sign data
func (h *BlockChainMessageHandler) requestBlockByHeight(id groupsig.ID, hs []int, sd logical.SignData) {}

//节点接收索要块请求
//param: block height slice
//       signData
func (h *BlockChainMessageHandler) onBlockRequest(hs []int, sd logical.SignData) map[int]core.Block {
	signResult := h.sv.isNodeSignCorrect(sd)
	if signResult {
		hbm, e := h.queryBlock(hs)
		if e != nil {
			return hbm
		}
	}
	return nil
}

//节点收到块信息
//param: heigth block map
//       signData
func (h *BlockChainMessageHandler) onMessageBlock(hbm map[int]core.Block, sd logical.SignData) {
	signResult := h.sv.isNodeSignCorrect(sd)
	if signResult {
		h.syncBlocks(hbm)
	}
}

////////////////////////////////////////////////////////组同步//////////////////////////////////////////////////////////
//广播索要组链高度
//param: signData
func (h *BlockChainMessageHandler) requestGroupChainHeight(sd logical.SignData) {}

//接收到组链高度信息请求，返回自身组链高度
//param: signData
func (h *BlockChainMessageHandler) onGroupChainHeightRequest(sd logical.SignData) int {
	signResult := h.sv.isNodeSignCorrect(sd)
	if signResult {
		h, e := h.getGroupChainHeight()
		if e != nil {
			return h
		}
	}
	return -1
}

//接收到组链高度信息，对比自身组链高度，判定是否发起同步
//param: group chain height
//       signData
func (h *BlockChainMessageHandler) onMessageGroupChainHeight(height int, sd logical.SignData) {
	signResult := h.sv.isNodeSignCorrect(sd)
	if signResult {
		//TODO 时间窗口内找出最高块的高度
		bestHeight := 0

		h, e := h.getBlockChainHeight()
		if e != nil {
			if bestHeight > h {
				//todo 发起组同步
				//requestGroupInfoByHeight
			}
		}
	}
}

//向某一节点请求GroupInfo
//param: target peer id
//       group height slice
//       sign data
func (h *BlockChainMessageHandler) requestGroupInfoByHeight(id groupsig.ID, gs []int, sd logical.SignData) {}

//节点接收索要组信息请求
//param: group height slice
//       signData
func (h *BlockChainMessageHandler) onGroupInfoRequest(gs []int, sd logical.SignData) map[int]logical.StaticGroupInfo {
	signResult := h.sv.isNodeSignCorrect(sd)
	if signResult {
		hgm, e := h.queryGroup(gs)
		if e != nil {
			return hgm
		}
	}
	return nil
}

//节点收到组信息
//param: heigth group map
//       signData
func (h *BlockChainMessageHandler) onMessageGroupInfo(hgm map[int]logical.StaticGroupInfo, sd logical.SignData) {
	signResult := h.sv.isNodeSignCorrect(sd)
	if signResult {
		h.syncGroup(hgm)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

//接收来自客户端的交易
func (h *BlockChainMessageHandler) onNewTransaction(t core.Transaction, sd logical.SignData) {}


