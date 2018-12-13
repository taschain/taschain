package core

import (
	"time"
	"utility"
	"network"
	"middleware/notify"
	"middleware/types"
	"math/big"
	"middleware/pb"
	"github.com/gogo/protobuf/proto"
	"common"
	"taslog"
)

const forkTimeOut = 3 * time.Second

type forkHandler struct {
	candidite string
	reqTimer  *time.Timer
	logger    taslog.Logger
}

func initForkHandler() *forkHandler {
	fh := forkHandler{}
	fh.logger = taslog.GetLoggerByIndex(taslog.ForkLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	notify.BUS.Subscribe(notify.ChainPieceInfoReq, fh.chainPieceInfoReqHandler)
	notify.BUS.Subscribe(notify.ChainPieceInfo, fh.chainPieceHandler)
	return &fh
}

func (fh *forkHandler) requestChainPieceInfo(targetNode string, height uint64) {
	if targetNode == "" {
		return
	}
	fh.candidite = targetNode
	fh.reqTimer.Reset(forkTimeOut)
	fh.logger.Debugf("Req chain piece info to:%s,local height:%d", targetNode, height)
	body := utility.UInt64ToByte(height)
	message := network.Message{Code: network.ChainPieceInfoReq, Body: body}
	network.GetNetInstance().Send(targetNode, message)
}

func (fh *forkHandler) chainPieceInfoReqHandler(msg notify.Message) {
	chainPieceReqMessage, ok := msg.GetData().(*notify.ChainPieceReqMessage)
	if !ok {
		return
	}
	reqHeight := utility.ByteToUInt64(chainPieceReqMessage.HeightByte)
	id := chainPieceReqMessage.Peer

	chainPiece := BlockChainImpl.GetChainPiece(reqHeight)
	fh.sendChainPieceInfo(id, ChainPieceInfo{ChainPiece: chainPiece, TopHeader: BlockChainImpl.QueryTopBlock()})
}

func (fh *forkHandler) sendChainPieceInfo(targetNode string, chainPieceInfo ChainPieceInfo) {
	chainPiece := chainPieceInfo.ChainPiece
	if len(chainPiece) == 0 {
		return
	}
	fh.logger.Debugf("Send chain piece %d-%d to:%s", chainPiece[len(chainPiece)-1].Height, chainPiece[0].Height, targetNode)
	body, e := marshalChainPieceInfo(chainPieceInfo)
	if e != nil {
		fh.logger.Errorf("Discard marshalChainPiece because of marshal error:%s!", e.Error())
		return
	}
	message := network.Message{Code: network.ChainPieceInfo, Body: body}
	network.GetNetInstance().Send(targetNode, message)
}

func (fh *forkHandler) chainPieceHandler(msg notify.Message) {
	chainPieceMessage, ok := msg.GetData().(*notify.ChainPieceMessage)
	if !ok {
		return
	}
	chainPieceInfo, err := fh.unMarshalChainPieceInfo(chainPieceMessage.ChainPieceInfoByte)
	if err != nil {
		fh.logger.Errorf("unMarshalChainPiece error:%s", err.Error())
		return
	}
	fh.processChainPieceInfo(chainPieceMessage.Peer, chainPieceInfo.ChainPiece, chainPieceInfo.TopHeader)
}



func (fh *forkHandler) RequestChainPieceBlock(id string, height uint64) {
	fh.logger.Debugf("Req chain piece block to:%s,height:%d", id, height)
	body := utility.UInt64ToByte(height)
	message := network.Message{Code: network.ReqChainPieceBlock, Body: body}
	go network.GetNetInstance().Send(id, message)
}

func (fh *forkHandler) newBlockHandler(msg notify.Message) {
	m, ok := msg.GetData().(types.Block)
	if !ok {
		return
	}
	fh.logger.Debugf("Rcv new block hash:%v,height:%d,totalQn:%d,tx len:%d", m.Header.Hash.Hex(), m.Header.Height, m.Header.TotalQN, len(m.Transactions))
	//todo 收齐上链
}

func (fh *forkHandler)unMarshalChainPieceInfo(b []byte) (*ChainPieceInfo, error) {
	message := new(tas_middleware_pb.ChainPieceInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		fh.logger.Errorf("unMarshalChainPieceInfo error:%s", e.Error())
		return nil, e
	}

	chainPiece := make([]*types.BlockHeader, 0)
	for _, header := range message.BlockHeaders {
		h := types.PbToBlockHeader(header)
		chainPiece = append(chainPiece, h)
	}
	topHeader := types.PbToBlockHeader(message.TopHeader)
	chainPieceInfo := ChainPieceInfo{ChainPiece: chainPiece, TopHeader: topHeader}
	return &chainPieceInfo, nil
}
