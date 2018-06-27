package handler

import (
	"time"
	"network/p2p"
	"github.com/gogo/protobuf/proto"
	"common"
	"core"
	"core/net/sync"
	"utility"
	"fmt"
	"network"
	"middleware/types"
	"middleware/pb"
)

const MAX_TRANSACTION_REQUEST_INTERVAL = 20 * time.Second

type ChainHandler struct{}

func (c *ChainHandler) HandlerMessage(code uint32, body []byte, sourceId string) ([]byte, error) {
	switch code {
	case p2p.REQ_TRANSACTION_MSG:
		m, e := unMarshalTransactionRequestMessage(body)
		if e != nil {
			core.Logger.Errorf("[handler]Discard TransactionRequestMessage because of unmarshal error:%s", e.Error())
			return nil, nil
		}
		OnTransactionRequest(m, sourceId)
	case p2p.TRANSACTION_GOT_MSG, p2p.TRANSACTION_MSG:
		m, e := types.UnMarshalTransactions(body)
		if e != nil {
			core.Logger.Errorf("[handler]Discard TRANSACTION_MSG because of unmarshal error:%s", e.Error())
			return nil, nil
		}
		if code == p2p.TRANSACTION_GOT_MSG {
			core.Logger.Debugf("receive TRANSACTION_GOT_MSG from %s,tx_len:%d,time at:%v", sourceId, len(m), time.Now())
		}
		err := onMessageTransaction(m)
		return nil, err
	case p2p.NEW_BLOCK_MSG:
		block, e := types.UnMarshalBlock(body)
		if e != nil {
			core.Logger.Errorf("[handler]Discard NEW_BLOCK_MSG because of unmarshal error:%s", e.Error())
			return nil, nil
		}
		onMessageNewBlock(block)

	case p2p.REQ_GROUP_CHAIN_HEIGHT_MSG:
		sync.GroupSyncer.HeightRequestCh <- sourceId
	case p2p.GROUP_CHAIN_HEIGHT_MSG:
		height := utility.ByteToUInt64(body)
		ghi := sync.GroupHeightInfo{Height: height, SourceId: sourceId}
		sync.GroupSyncer.HeightCh <- ghi
	case p2p.REQ_GROUP_MSG:
		baseHeight := utility.ByteToUInt64(body)
		gri := sync.GroupRequestInfo{BaseHeight: baseHeight, SourceId: sourceId}
		sync.GroupSyncer.GroupRequestCh <- gri
	case p2p.GROUP_MSG:
		m, e := unMarshalGroups(body)
		if e != nil {
			core.Logger.Errorf("[handler]Discard GROUP_MSG because of unmarshal error:%s", e.Error())
			return nil, e
		}
		sync.GroupSyncer.GroupCh <- m

	case p2p.REQ_BLOCK_CHAIN_TOTAL_QN_MSG:
		sync.BlockSyncer.TotalQnRequestCh <- sourceId
	case p2p.BLOCK_CHAIN_TOTAL_QN_MSG:
		totalQn := utility.ByteToUInt64(body)
		s := sync.TotalQnInfo{TotalQn: totalQn, SourceId: sourceId}
		sync.BlockSyncer.TotalQnCh <- s
	case p2p.REQ_BLOCK_INFO:
		m, e := unMarshalBlockRequestInfo(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard REQ_BLOCK_MSG_WITH_PRE because of unmarshal error:%s", e.Error())
			return nil, e
		}
		onBlockInfoReq(*m, sourceId)
	case p2p.BLOCK_INFO:
		m, e := unMarshalBlockInfo(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard BLOCK_MSG because of unmarshal error:%s", e.Error())
			return nil, e
		}
		onBlockInfo(*m, sourceId)
	case p2p.BLOCK_HASHES_REQ:
		cbhr, e := unMarshalBlockHashesReq(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard BLOCK_CHAIN_HASHES_REQ because of unmarshal error:%s", e.Error())
			return nil, e
		}
		onBlockHashesReq(cbhr, sourceId)
	case p2p.BLOCK_HASHES:
		cbh, e := unMarshalBlockHashes(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard BLOCK_CHAIN_HASHES because of unmarshal error:%s", e.Error())
			return nil, e
		}
		onBlockHashes(cbh, sourceId)
	}
	return nil, nil
}

//-----------------------------------------------铸币-------------------------------------------------------------------

//接收索要交易请求 查询自身是否有该交易 有的话返回, 没有的话自己广播该请求
func OnTransactionRequest(m *core.TransactionRequestMessage, sourceId string) error {
	//core.Logger.Debugf("receive REQ_TRANSACTION_MSG from %s,%d-%D,tx_len", sourceId, m.BlockHeight, m.BlockQn,len(m.TransactionHashes))
	//本地查询transaction
	if nil == core.BlockChainImpl {
		return nil
	}
	transactions, need, e := core.BlockChainImpl.GetTransactionPool().GetTransactions(m.CurrentBlockHash, m.TransactionHashes)
	if e == core.ErrNil {
		m.TransactionHashes = need
	}

	if nil != transactions && 0 != len(transactions) {
		core.SendTransactions(transactions, sourceId, m.BlockHeight, m.BlockQn)
	}

	return nil
}

//验证节点接收交易 或者接收来自客户端广播的交易
func onMessageTransaction(txs []*types.Transaction) error {
	//验证节点接收交易 加入交易池
	if nil == core.BlockChainImpl {
		return nil
	}
	e := core.BlockChainImpl.GetTransactionPool().AddTransactions(txs)
	if e != nil {
		return e
	}
	return nil
}

//全网其他节点 接收block 进行验证
func onMessageNewBlock(b *types.Block) error {
	//接收到新的块 本地上链
	if nil == core.BlockChainImpl {
		return nil
	}
	if core.BlockChainImpl.AddBlockOnChain(b) == -1 {
		core.Logger.Errorf("[handler]Add new block to chain error!")
		return fmt.Errorf("fail to add block")
	}
	return nil
}

//----------------------------------------------链调整-------------------------------------------------------------

func onBlockHashesReq(cbhr *core.BlockHashesReq, sourceId string) {
	if cbhr == nil {
		return
	}
	cbh := core.BlockChainImpl.GetBlockHashesFromLocalChain(cbhr.Height, cbhr.Length)
	core.SendBlockHashes(sourceId, cbh)
}

func onBlockHashes(bhs []*core.BlockHash, sourceId string) {
	//core.Logger.Debugf("Get OnChainBlockHashes from:%s", sourceId)
	core.BlockChainImpl.CompareChainPiece(bhs, sourceId)
}

func onBlockInfoReq(erm core.BlockRequestInfo, sourceId string) {
	//收到块请求
	//core.Logger.Debugf("[handler]onBlockInfoReq get message from:%s", sourceId)
	if nil == core.BlockChainImpl {
		return
	}
	blockInfo := core.BlockChainImpl.GetBlockInfo(erm.SourceHeight, erm.SourceCurrentHash)
	core.SendBlockInfo(sourceId, blockInfo)
}

func onBlockInfo(blockInfo core.BlockInfo, sourceId string) {
	//收到块信息
	//core.Logger.Debugf("[handler] onBlockInfo get message from:%s", sourceId)
	if nil == core.BlockChainImpl {
		return
	}
	blocks := blockInfo.Blocks
	if blocks != nil && len(blocks) != 0 {
		//core.Logger.Debugf("[handler] onBlockInfo receive blocks,length:%d", len(blocks))
		for i := 0; i < len(blocks); i++ {
			block := blocks[i]
			code := core.BlockChainImpl.AddBlockOnChain(block)
			if code < 0 {
				core.BlockChainImpl.SetAdujsting(false)
				core.Logger.Errorf("fail to add block to block chain,code:%d", code)
				return
			}
			if code == 2 {
				return
			}
		}
		//todo 如果将来改为发送多次 此处需要修改
		core.BlockChainImpl.SetAdujsting(false)
		if !core.BlockChainImpl.IsBlockSyncInit() {
			core.BlockChainImpl.SetBlockSyncInit(true)
		}
	} else {
		//core.Logger.Debugf("[handler] onBlockInfo receive chainPiece,length:%d", len(blockInfo.ChainPiece))
		chainPiece := blockInfo.ChainPiece
		core.BlockChainImpl.CompareChainPiece(chainPiece, sourceId)
	}
}

func unMarshalTransactionRequestMessage(b []byte) (*core.TransactionRequestMessage, error) {
	m := new(tas_middleware_pb.TransactionRequestMessage)
	e := proto.Unmarshal(b, m)
	if e != nil {
		network.Logger.Errorf("[handler]UnMarshal TransactionRequestMessage error:%s", e.Error())
		return nil, e
	}

	txHashes := make([]common.Hash, 0)
	for _, txHash := range m.TransactionHashes {
		txHashes = append(txHashes, common.BytesToHash(txHash))
	}

	currentBlockHash := common.BytesToHash(m.CurrentBlockHash)
	message := core.TransactionRequestMessage{TransactionHashes: txHashes, CurrentBlockHash: currentBlockHash, BlockHeight: *m.BlockHeight, BlockQn: *m.BlockQn}
	return &message, nil
}

//--------------------------------------------------Block---------------------------------------------------------------

//func unMarshalBlocks(b []byte) ([]*core.Block, error) {
//	blockSlice := new(tas_pb.BlockSlice)
//	error := proto.Unmarshal(b, blockSlice)
//	if error != nil {
//		logger.Errorf("Unmarshal Blocks error:%s\n", error.Error())
//		return nil, error
//	}
//	blocks := blockSlice.Blocks
//	result := make([]*core.Block, 0)
//
//	for _, b := range blocks {
//		block := pbToBlock(b)
//		result = append(result, block)
//	}
//	return result, nil
//}

func unMarshalBlockHashesReq(byte []byte) (*core.BlockHashesReq, error) {
	b := new(tas_middleware_pb.BlockHashesReq)

	error := proto.Unmarshal(byte, b)
	if error != nil {
		network.Logger.Errorf("[handler]unMarshalChainBlockHashesReq error:%s", error.Error())
		return nil, error
	}
	r := pbToBlockHashesReq(b)
	return r, nil
}

func pbToBlockHashesReq(cbhr *tas_middleware_pb.BlockHashesReq) *core.BlockHashesReq {
	if cbhr == nil {
		return nil
	}
	r := core.BlockHashesReq{Height: *cbhr.Height, Length: *cbhr.Length}
	return &r
}

func unMarshalBlockHashes(b []byte) ([]*core.BlockHash, error) {
	blockHashSlice := new(tas_middleware_pb.BlockHashSlice)
	error := proto.Unmarshal(b, blockHashSlice)
	if error != nil {
		network.Logger.Errorf("[handler]unMarshalChainBlockHashes error:%s\n", error.Error())
		return nil, error
	}
	chainBlockHashes := blockHashSlice.BlockHashes
	result := make([]*core.BlockHash, 0)

	for _, b := range chainBlockHashes {
		h := pbToBlockHash(b)
		result = append(result, h)
	}
	return result, nil
}

func pbToBlockHash(cbh *tas_middleware_pb.BlockHash) *core.BlockHash {
	if cbh == nil {
		return nil
	}
	r := core.BlockHash{Height: *cbh.Height, Hash: common.BytesToHash(cbh.Hash)}
	return &r
}

//----------------------------------------------块同步------------------------------------------------------------------

func unMarshalBlockRequestInfo(b []byte) (*core.BlockRequestInfo, error) {
	m := new(tas_middleware_pb.BlockRequestInfo)

	e := proto.Unmarshal(b, m)
	if e != nil {
		network.Logger.Errorf("[handler]Unmarshal EntityRequestMessage error:%s", e.Error())
		return nil, e
	}

	sourceHeight := m.SourceHeight
	sourceCurrentHash := common.BytesToHash(m.SourceCurrentHash)
	message := core.BlockRequestInfo{SourceHeight: *sourceHeight, SourceCurrentHash: sourceCurrentHash}
	return &message, nil
}

func unMarshalBlockInfo(b []byte) (*core.BlockInfo, error) {
	message := new(tas_middleware_pb.BlockInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		network.Logger.Errorf("[handler]Unmarshal BlockMessage error:%s", e.Error())
		return nil, e
	}

	blocks := make([]*types.Block, 0)
	if message.Blocks != nil && message.Blocks.Blocks != nil {
		for _, b := range message.Blocks.Blocks {
			blocks = append(blocks, types.PbToBlock(b))
		}
	}

	cbh := make([]*core.BlockHash, 0)
	if message.BlockHashes != nil && message.BlockHashes.BlockHashes != nil {
		for _, b := range message.BlockHashes.BlockHashes {
			cbh = append(cbh, pbToBlockHash(b))
		}
	}

	m := core.BlockInfo{Blocks: blocks, ChainPiece: cbh}
	return &m, nil
}

func unMarshalGroups(b []byte) ([]*types.Group, error) {
	message := new(tas_middleware_pb.GroupSlice)
	e := proto.Unmarshal(b, message)
	if e != nil {
		core.Logger.Errorf("[handler]Unmarshal Groups error:%s", e.Error())
		return nil, e
	}

	groups := make([]*types.Group, 0)
	if message.Groups != nil {
		for _, g := range message.Groups {
			groups = append(groups, types.PbToGroup(g))
		}
	}
	return groups, nil
}
