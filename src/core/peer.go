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

type EntityTotalQNMessage struct {
	TotalQN  uint64
	SourceId string
}

type EntityHeightMessage struct {
	Height   uint64
	SourceId string
}

type EntityRequestMessage struct {
	SourceHeight      uint64
	SourceCurrentHash common.Hash
	SourceId          string
}

type BlockMessage struct {
	Blocks      []*types.Block
	BlockHashes []*ChainBlockHash
}

type BlockArrivedMessage struct {
	BlockEntity BlockMessage
	SourceId    string
}

type GroupMessage struct {
	Groups []*types.Group
	Height uint64      //起始高度
	Hash   common.Hash //起始HASH
}

type GroupArrivedMessage struct {
	GroupEntity GroupMessage
	SourceId    string
}

//验证节点 交易集缺失，索要、特定交易 全网广播
func BroadcastTransactionRequest(m TransactionRequestMessage) {
	if m.SourceId == "" {
		m.SourceId = p2p.Server.SelfNetInfo.Id
	}

	body, e := marshalTransactionRequestMessage(&m)
	if e != nil {
		network.Logger.Errorf("[peer]Discard MarshalTransactionRequestMessage because of marshal error:%s!", e.Error())
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
		network.Logger.Errorf("[peer]Discard MarshalTransactions because of marshal error:%s!", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.TRANSACTION_GOT_MSG, Body: body}
	p2p.Server.SendMessage(message, sourceId)
}

//收到交易 全网扩散
func BroadcastTransactions(txs []*types.Transaction) {

	defer func() {
		if r := recover(); r != nil {
			network.Logger.Errorf("[peer]Runtime error caught: %v", r)
		}
	}()

	body, e := types.MarshalTransactions(txs)
	if e != nil {
		network.Logger.Errorf("[peer]Discard MarshalTransactions because of marshal error:%s", e.Error())
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

//向某一节点请求Block
func RequestBlockByHeight(id string, localHeight uint64, currentHash common.Hash) {
	m := EntityRequestMessage{SourceHeight: localHeight, SourceCurrentHash: currentHash, SourceId: ""}
	body, e := MarshalEntityRequestMessage(&m)
	if e != nil {
		network.Logger.Errorf("[peer]requestBlockByHeight marshal EntityRequestMessage error:%s", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.REQ_BLOCK_MSG, Body: body}
	p2p.Server.SendMessage(message, id)
}

type ChainBlockHashesReq struct {
	Height uint64 //起始高度
	Length uint64 //从起始高度开始的非空长度
}

type ChainBlockHash struct {
	Height uint64 //所在链高度
	Hash   common.Hash
}

//向目标结点索要 block hash
func RequestBlockChainHashes(targetNode string, cbhr ChainBlockHashesReq) {
	body, e := marshalChainBlockHashesReq(&cbhr)
	if e != nil {
		network.Logger.Errorf("[peer]Discard RequestBlockChainHashes because of marshal error:%s!", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.BLOCK_CHAIN_HASHES_REQ, Body: body}
	p2p.Server.SendMessage(message, targetNode)
}

//向目标结点发送 block hash
func SendChainBlockHashes(targetNode string, cbh []*ChainBlockHash) {
	body, e := marshalChainBlockHashes(cbh)
	if e != nil {
		network.Logger.Errorf("[peer]Discard sendChainBlockHashes because of marshal error:%s!", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.BLOCK_CHAIN_HASHES, Body: body}
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

func marshalChainBlockHashesReq(req *ChainBlockHashesReq) ([]byte, error) {
	if req == nil {
		return nil, nil
	}
	cbhr := chainBlockHashesReqToPb(req)
	return proto.Marshal(cbhr)
}

func chainBlockHashesReqToPb(req *ChainBlockHashesReq) *tas_middleware_pb.ChainBlockHashesReq {
	if req == nil {
		return nil
	}
	cbhr := tas_middleware_pb.ChainBlockHashesReq{Height: &req.Height, Length: &req.Length}
	return &cbhr
}
func marshalChainBlockHashes(cbh []*ChainBlockHash) ([]byte, error) {
	if cbh == nil {
		return nil, nil
	}

	chainBlockHashes := make([]*tas_middleware_pb.ChainBlockHash, 0)
	for _, c := range cbh {
		chainBlockHashes = append(chainBlockHashes, ChainBlockHashToPb(c))
	}
	r := tas_middleware_pb.ChainBlockHashSlice{ChainBlockHashes: chainBlockHashes}
	return proto.Marshal(&r)
}

func ChainBlockHashToPb(cbh *ChainBlockHash) *tas_middleware_pb.ChainBlockHash {
	if cbh == nil {
		return nil
	}

	r := tas_middleware_pb.ChainBlockHash{Hash: cbh.Hash.Bytes(), Height: &cbh.Height}
	return &r
}

func MarshalEntityRequestMessage(e *EntityRequestMessage) ([]byte, error) {
	sourceHeight := e.SourceHeight
	currentHash := e.SourceCurrentHash.Bytes()
	sourceId := []byte(e.SourceId)

	m := tas_middleware_pb.EntityRequestMessage{SourceHeight: &sourceHeight, SourceCurrentHash: currentHash, SourceId: sourceId}
	return proto.Marshal(&m)
}




