package core

import (
	"time"
	"utility"
	"network"
	"middleware/notify"
	"middleware/types"
	"middleware/pb"
	"github.com/gogo/protobuf/proto"
	"common"
	"taslog"
	"sync"
)

const forkTimeOut = 3 * time.Second

type forkProcessor struct {
	candidite string
	reqTimer  *time.Timer
	lock      sync.Mutex
	logger    taslog.Logger
}

func initForkHandler() *forkProcessor {
	fh := forkProcessor{lock: sync.Mutex{}, reqTimer: time.NewTimer(forkTimeOut)}
	fh.logger = taslog.GetLoggerByIndex(taslog.ForkLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	notify.BUS.Subscribe(notify.ChainPieceInfoReq, fh.chainPieceInfoReqHandler)
	notify.BUS.Subscribe(notify.ChainPieceInfo, fh.chainPieceInfoHandler)
	return &fh
}

func (fh *forkProcessor) requestChainPieceInfo(targetNode string, height uint64) {
	if targetNode == "" {
		return
	}
	if fh.candidite != "" {
		fh.logger.Debug("Processing fork! Do not req chain piece info anymore")
		return
	}
	fh.lock.Lock()
	fh.candidite = targetNode
	fh.reqTimer.Reset(forkTimeOut)
	fh.lock.Unlock()
	fh.logger.Debugf("Req chain piece info to:%s,local height:%d", targetNode, height)
	body := utility.UInt64ToByte(height)
	message := network.Message{Code: network.ChainPieceInfoReq, Body: body}
	network.GetNetInstance().Send(targetNode, message)
}

func (fh *forkProcessor) chainPieceInfoReqHandler(msg notify.Message) {
	chainPieceReqMessage, ok := msg.GetData().(*notify.ChainPieceInfoReqMessage)
	if !ok {
		return
	}
	reqHeight := utility.ByteToUInt64(chainPieceReqMessage.HeightByte)
	id := chainPieceReqMessage.Peer

	chainPiece := BlockChainImpl.GetChainPiece(reqHeight)
	fh.sendChainPieceInfo(id, ChainPieceInfo{ChainPiece: chainPiece, TopHeader: BlockChainImpl.QueryTopBlock()})
}

func (fh *forkProcessor) sendChainPieceInfo(targetNode string, chainPieceInfo ChainPieceInfo) {
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

func (fh *forkProcessor) chainPieceInfoHandler(msg notify.Message) {
	chainPieceInfoMessage, ok := msg.GetData().(*notify.ChainPieceInfoMessage)
	if !ok {
		return
	}
	chainPieceInfo, err := fh.unMarshalChainPieceInfo(chainPieceInfoMessage.ChainPieceInfoByte)
	if err != nil {
		fh.logger.Errorf("unMarshalChainPiece error:%s", err.Error())
		return
	}
	source := chainPieceInfoMessage.Peer
	if source != fh.candidite {
		fh.logger.Debugf("Unexpected chain piece info from %s, expect from %s!", chainPieceInfoMessage.Peer, source)
		return
	}
	if !fh.verifyChainPiece(chainPieceInfo.ChainPiece, chainPieceInfo.TopHeader) {
		//todo 加入黑名单
		return
	}
	status, reqHeight := BlockChainImpl.ProcessChainPieceInfo(chainPieceInfo.ChainPiece, chainPieceInfo.TopHeader)
	if status == 0 {
		fh.candidite = ""
		fh.reqTimer.Stop()
		return
	}
	if status == 1 {
		fh.requestChainPieceBlock(source, reqHeight)
		return
	}

	if status == 2 {
		fh.requestChainPieceInfo(source, reqHeight)
		return
	}
}

func (fh *forkProcessor) requestChainPieceBlock(id string, height uint64) {
	fh.logger.Debugf("Req chain piece block to:%s,height:%d", id, height)
	body := utility.UInt64ToByte(height)
	message := network.Message{Code: network.ReqChainPieceBlock, Body: body}
	go network.GetNetInstance().Send(id, message)
}

func (fh *forkProcessor) newBlockHandler(msg notify.Message) {
	m, ok := msg.GetData().(types.Block)
	if !ok {
		return
	}
	fh.logger.Debugf("Rcv new block hash:%v,height:%d,totalQn:%d,tx len:%d", m.Header.Hash.Hex(), m.Header.Height, m.Header.TotalQN, len(m.Transactions))
	//todo 收齐上链
	//if source != fh.candidite {
	//	fh.logger.Debugf("Unexpected chain piece info from %s, expect from %s!", chainPieceInfoMessage.Peer, source)
	//}
	//收齐之后
	fh.lock.Lock()
	fh.candidite = ""
	fh.reqTimer.Stop()
	fh.lock.Unlock()
}

func (fh *forkProcessor) verifyChainPiece(chainPiece []*types.BlockHeader, topHeader *types.BlockHeader) bool {
	if len(chainPiece) == 0 {
		return false
	}
	if topHeader.Hash != topHeader.GenHash() {
		Logger.Infof("invalid topHeader!Hash:%s", topHeader.Hash.String())
		return false
	}

	for i := 0; i < len(chainPiece)-1; i++ {
		bh := chainPiece[i]
		if bh.Hash != bh.GenHash() {
			Logger.Infof("invalid chainPiece element,hash:%s", bh.Hash.String())
			return false
		}
		if bh.PreHash != chainPiece[i+1].Hash {
			Logger.Infof("invalid preHash,expect prehash:%s,real hash:%s", bh.PreHash.String(), chainPiece[i+1].Hash.String())
			return false
		}
	}
	return true
}

func (fh *forkProcessor) unMarshalChainPieceInfo(b []byte) (*ChainPieceInfo, error) {
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

func (fh *forkProcessor) loop() {
	for {
		select {
		case fh.reqTimer.C:
			fh.lock.Lock()
			if fh.candidite != "" {
				fh.logger.Debugf("Fork req time out to %s", fh.candidite)
				//todo 超时标记
				fh.candidite = ""
			}
			fh.lock.Unlock()
		}
	}
}
