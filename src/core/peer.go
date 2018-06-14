package core

import (
	"github.com/gogo/protobuf/proto"
	"network/p2p"
	"time"
	"common"
	"network"
	"middleware/types"
	"middleware/pb"
)

type TransactionRequestMessage struct {
	TransactionHashes []common.Hash
	SourceId          string
	RequestTime       time.Time
}

type BlockHashesReq struct {
	Height uint64 //起始高度
	Length uint64 //从起始高度开始,向前的非空长度
}

type BlockHash struct {
	Height uint64 //所在链高度
	Hash   common.Hash
	Qn     uint64
}

type BlockRequestInfo struct {
	SourceHeight      uint64
	SourceCurrentHash common.Hash
}

type BlockInfo struct {
	Blocks     []*types.Block
	ChainPiece []*BlockHash
}

//验证节点 交易集缺失，索要、特定交易 全网广播
func BroadcastTransactionRequest(m TransactionRequestMessage) {
	if m.SourceId == "" {
		m.SourceId = p2p.Server.SelfNetInfo.Id
	}

	body, e := marshalTransactionRequestMessage(&m)
	if e != nil {
		Logger.Errorf("[peer]Discard MarshalTransactionRequestMessage because of marshal error:%s!", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.REQ_TRANSACTION_MSG, Body: body}

	conns := p2p.Server.Host.Network().Conns()
	for _, conn := range conns {
		id := conn.RemotePeer()
		if id != "" {
			p2p.Server.SendMessage(message, p2p.ConvertToID(id))
		}
	}
}

//本地查询到交易，返回请求方
func SendTransactions(txs []*types.Transaction, sourceId string) {
	body, e := types.MarshalTransactions(txs)
	if e != nil {
		Logger.Errorf("[peer]Discard MarshalTransactions because of marshal error:%s!", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.TRANSACTION_GOT_MSG, Body: body}
	p2p.Server.SendMessage(message, sourceId)
}

//收到交易 全网扩散
func BroadcastTransactions(txs []*types.Transaction) {
	defer func() {
		if r := recover(); r != nil {
			Logger.Errorf("[peer]Runtime error caught: %v", r)
		}
	}()

	body, e := types.MarshalTransactions(txs)
	if e != nil {
		Logger.Errorf("[peer]Discard MarshalTransactions because of marshal error:%s", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.TRANSACTION_MSG, Body: body}

	conns := p2p.Server.Host.Network().Conns()
	for _, conn := range conns {
		id := conn.RemotePeer()
		if id != "" {
			p2p.Server.SendMessage(message, p2p.ConvertToID(id))
		}
	}
}

//向某一节点请求Block信息
func RequestBlockInfoByHeight(id string, localHeight uint64, currentHash common.Hash) {
	m := BlockRequestInfo{SourceHeight: localHeight, SourceCurrentHash: currentHash}
	body, e := MarshalBlockRequestInfo(&m)
	if e != nil {
		Logger.Errorf("[peer]RequestBlockInfoByHeight marshal EntityRequestMessage error:%s", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.REQ_BLOCK_INFO, Body: body}
	p2p.Server.SendMessage(message, id)
}

//本地查询之后将结果返回
func SendBlockInfo(targetId string, blockInfo *BlockInfo) {
	body, e := marshalBlockInfo(blockInfo)
	if e != nil {
		Logger.Errorf("[peer]SendBlockInfo marshal BlockEntity error:%s", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.BLOCK_INFO, Body: body}
	p2p.Server.SendMessage(message, targetId)
}

//向目标结点索要 block hash
func RequestBlockHashes(targetNode string, bhr BlockHashesReq) {
	body, e := marshalBlockHashesReq(&bhr)
	if e != nil {
		Logger.Errorf("[peer]Discard RequestBlockChainHashes because of marshal error:%s!", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.BLOCK_HASHES_REQ, Body: body}
	p2p.Server.SendMessage(message, targetNode)
}

//向目标结点发送 block hash
func SendBlockHashes(targetNode string, bhs []*BlockHash) {
	body, e := marshalBlockHashes(bhs)
	if e != nil {
		Logger.Errorf("[peer]Discard sendChainBlockHashes because of marshal error:%s!", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.BLOCK_HASHES, Body: body}
	p2p.Server.SendMessage(message, targetNode)
}

//--------------------------------------------------Transaction---------------------------------------------------------------
func marshalTransactionRequestMessage(m *TransactionRequestMessage) ([]byte, error) {
	txHashes := make([][]byte, 0)
	for _, txHash := range m.TransactionHashes {
		txHashes = append(txHashes, txHash.Bytes())
	}

	sourceId := []byte(m.SourceId)

	requestTime, e := m.RequestTime.MarshalBinary()
	if e != nil {
		network.Logger.Errorf("[peer]TransactionRequestMessage request time marshal error:%s\n", e.Error())
	}
	message := tas_middleware_pb.TransactionRequestMessage{TransactionHashes: txHashes, SourceId: sourceId, RequestTime: requestTime}
	return proto.Marshal(&message)
}

//--------------------------------------------------Block---------------------------------------------------------------

func marshalBlockHashesReq(req *BlockHashesReq) ([]byte, error) {
	if req == nil {
		return nil, nil
	}
	cbhr := blockHashesReqToPb(req)
	return proto.Marshal(cbhr)
}

func blockHashesReqToPb(req *BlockHashesReq) *tas_middleware_pb.BlockHashesReq {
	if req == nil {
		return nil
	}
	cbhr := tas_middleware_pb.BlockHashesReq{Height: &req.Height, Length: &req.Length}
	return &cbhr
}
func marshalBlockHashes(cbh []*BlockHash) ([]byte, error) {
	if cbh == nil {
		return nil, nil
	}

	blockHashes := make([]*tas_middleware_pb.BlockHash, 0)
	for _, c := range cbh {
		blockHashes = append(blockHashes, blockHashToPb(c))
	}
	r := tas_middleware_pb.BlockHashSlice{BlockHashes: blockHashes}
	return proto.Marshal(&r)
}

func blockHashToPb(bh *BlockHash) *tas_middleware_pb.BlockHash {
	if bh == nil {
		return nil
	}

	r := tas_middleware_pb.BlockHash{Hash: bh.Hash.Bytes(), Height: &bh.Height}
	return &r
}

func MarshalBlockRequestInfo(e *BlockRequestInfo) ([]byte, error) {
	sourceHeight := e.SourceHeight
	currentHash := e.SourceCurrentHash.Bytes()

	m := tas_middleware_pb.BlockRequestInfo{SourceHeight: &sourceHeight, SourceCurrentHash: currentHash}
	return proto.Marshal(&m)
}

func marshalBlockInfo(e *BlockInfo) ([]byte, error) {
	if e == nil {
		return nil, nil
	}
	blocks := make([]*tas_middleware_pb.Block, 0)

	if e.Blocks != nil {
		for _, b := range e.Blocks {
			pb := types.BlockToPb(b)
			if pb == nil {
				Logger.Errorf("Block is nil while marshalBlockMessage")
			}
			blocks = append(blocks, pb)
		}
	}
	blockSlice := tas_middleware_pb.BlockSlice{Blocks: blocks}

	cbh := make([]*tas_middleware_pb.BlockHash, 0)

	if e.ChainPiece != nil {
		for _, b := range e.ChainPiece {
			pb := blockHashToPb(b)
			if pb == nil {
				Logger.Errorf("ChainBlockHash is nil while marshalBlockMessage")
			}
			cbh = append(cbh, pb)
		}
	}
	cbhs := tas_middleware_pb.BlockHashSlice{BlockHashes: cbh}

	message := tas_middleware_pb.BlockInfo{Blocks: &blockSlice, BlockHashes: &cbhs}
	return proto.Marshal(&message)
}
