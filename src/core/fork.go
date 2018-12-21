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

const (
	forkTimeOut         = 3 * time.Second
	blockChainPieceSzie = 10
)

type forkProcessor struct {
	candidite string
	reqTimer  *time.Timer

	lock   sync.Mutex
	logger taslog.Logger
}

type ChainPieceBlockMsg struct {
	Blocks    []*types.Block
	TopHeader *types.BlockHeader
}

func initforkProcessor() *forkProcessor {
	fh := forkProcessor{lock: sync.Mutex{}, reqTimer: time.NewTimer(forkTimeOut),}
	fh.logger = taslog.GetLoggerByIndex(taslog.ForkLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	notify.BUS.Subscribe(notify.ChainPieceInfoReq, fh.chainPieceInfoReqHandler)
	notify.BUS.Subscribe(notify.ChainPieceInfo, fh.chainPieceInfoHandler)
	notify.BUS.Subscribe(notify.ChainPieceBlockReq, fh.chainPieceBlockReqHanlder)
	notify.BUS.Subscribe(notify.ChainPieceBlock, fh.chainPieceBlockHandler)
	return &fh
}

func (fh *forkProcessor) requestChainPieceInfo(targetNode string, height uint64) {
	if targetNode == "" {
		return
	}
	if fh.candidite != "" {
		fh.logger.Debugf("Processing fork to %s! Do not req chain piece info anymore", fh.candidite)
		return
	}
	if PeerManager.isEvil(targetNode) {
		fh.logger.Debugf("Req id:%s is marked evil.Do not req!", targetNode)
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

	fh.logger.Debugf("Rcv chain piece info req from:%s,req height:%d", id, reqHeight)
	chainPiece := BlockChainImpl.GetChainPieceInfo(reqHeight)
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
		fh.logger.Debugf("Unexpected chain piece info from %s, expect from %s!", source, chainPieceInfoMessage.Peer)
		PeerManager.markEvil(source)
		return
	}
	if !fh.verifyChainPieceInfo(chainPieceInfo.ChainPiece, chainPieceInfo.TopHeader) {
		fh.logger.Debugf("Bad chain piece info from %s", source)
		PeerManager.markEvil(source)
		return
	}
	status, reqHeight := BlockChainImpl.ProcessChainPieceInfo(chainPieceInfo.ChainPiece, chainPieceInfo.TopHeader)
	if status == 0 {
		fh.reset()
		return
	}
	if status == 1 {
		fh.requestChainPieceBlock(source, reqHeight)
		return
	}

	if status == 2 {
		fh.reset()
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

func (fh *forkProcessor) chainPieceBlockReqHanlder(msg notify.Message) {
	m, ok := msg.GetData().(*notify.ChainPieceBlockReqMessage)
	if !ok {
		return
	}
	source := m.Peer
	reqHeight := utility.ByteToUInt64(m.ReqHeightByte)
	fh.logger.Debugf("Rcv chain piece block req from:%s,req height:%d", source, reqHeight)

	blocks := BlockChainImpl.GetChainPieceBlocks(reqHeight)
	topHeader := BlockChainImpl.QueryTopBlock()
	fh.sendChainPieceBlock(source, blocks, topHeader)
}

func (fh *forkProcessor) sendChainPieceBlock(targetId string, blocks []*types.Block, topHeader *types.BlockHeader) {
	fh.logger.Debugf("Send chain piece blocks %d-%d to:%s", blocks[len(blocks)-1].Header.Height, blocks[0].Header.Height, targetId)
	body, e := fh.marshalChainPieceBlockMsg(ChainPieceBlockMsg{Blocks: blocks, TopHeader: topHeader})
	if e != nil {
		fh.logger.Errorf("SendBlock marshal MarshalBlock error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.ChainPieceBlock, Body: body}
	go network.GetNetInstance().Send(targetId, message)
}

func (fh *forkProcessor) chainPieceBlockHandler(msg notify.Message) {
	m, ok := msg.GetData().(*notify.ChainPieceBlockMessage)
	if !ok {
		return
	}
	source := m.Peer
	if source != fh.candidite {
		fh.logger.Debugf("Unexpected chain piece block from %s, expect from %s!", source, fh.candidite)
		PeerManager.markEvil(source)
		return
	}

	chainPieceBlockMsg, e := fh.unmarshalChainPieceBlockMsg(m.ChainPieceBlockMsgByte)
	if e != nil {
		fh.logger.Debugf("Discard chain piece msg because unmarshalChainPieceBlockMsg error:%d", e.Error())
		return
	}

	blocks := chainPieceBlockMsg.Blocks
	topHeader := chainPieceBlockMsg.TopHeader

	if topHeader == nil {
		return
	}
	fh.logger.Debugf("Rcv chain piece block chain piece blocks %d-%d from %s", blocks[len(blocks)-1].Header.Height, blocks[0].Header.Height, source)

	if !fh.verifyChainPieceBlocks(blocks, topHeader) {
		fh.logger.Debugf("Bad chain piece blocks from %s", source)
		PeerManager.markEvil(source)
		return
	}
	BlockChainImpl.MergeFork(blocks, topHeader)
	fh.reset()

}

func (fh *forkProcessor) reset() {
	fh.lock.Lock()
	defer fh.lock.Unlock()
	fh.logger.Debugf("Fork processor reset!")
	fh.candidite = ""
	fh.reqTimer.Stop()
}

func (fh *forkProcessor) verifyChainPieceInfo(chainPiece []*types.BlockHeader, topHeader *types.BlockHeader) bool {
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

func (fh *forkProcessor) verifyChainPieceBlocks(chainPiece []*types.Block, topHeader *types.BlockHeader) bool {
	if len(chainPiece) == 0 {
		return false
	}
	if topHeader.Hash != topHeader.GenHash() {
		fh.logger.Infof("invalid topHeader!Hash:%s", topHeader.Hash.String())
		return false
	}

	for i := 0; i < len(chainPiece)-1; i++ {
		block := chainPiece[i]
		if block == nil {
			return false
		}
		if block.Header.Hash != block.Header.GenHash() {
			fh.logger.Infof("invalid chainPiece element,hash:%s", block.Header.Hash.String())
			return false
		}
		if block.Header.PreHash != chainPiece[i+1].Header.Hash {
			fh.logger.Infof("invalid preHash,expect prehash:%s,real hash:%s", block.Header.PreHash.String(), chainPiece[i+1].Header.Hash.String())
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

func (fh *forkProcessor) marshalChainPieceBlockMsg(cpb ChainPieceBlockMsg) ([]byte, error) {
	topHeader := types.BlockHeaderToPb(cpb.TopHeader)
	blocks := make([]*tas_middleware_pb.Block, 0)
	for _, b := range cpb.Blocks {
		blocks = append(blocks, types.BlockToPb(b))
	}
	message := tas_middleware_pb.ChainPieceBlockMsg{TopHeader: topHeader, Blocks: blocks}
	return proto.Marshal(&message)
}

func (fh *forkProcessor) unmarshalChainPieceBlockMsg(b []byte) (*ChainPieceBlockMsg, error) {
	message := new(tas_middleware_pb.ChainPieceBlockMsg)
	e := proto.Unmarshal(b, message)
	if e != nil {
		fh.logger.Errorf("unmarshalChainPieceBlockMsg error:%s", e.Error())
		return nil, e
	}
	topHeader := types.PbToBlockHeader(message.TopHeader)
	blocks := make([]*types.Block, 0)
	for _, b := range message.Blocks {
		blocks = append(blocks, types.PbToBlock(b))
	}
	cpb := ChainPieceBlockMsg{TopHeader: topHeader, Blocks: blocks}
	return &cpb, nil
}

func (fh *forkProcessor) loop() {
	for {
		select {
		case <-fh.reqTimer.C:
			fh.lock.Lock()
			if fh.candidite != "" {
				fh.logger.Debugf("Fork req time out to %s", fh.candidite)
				PeerManager.markEvil(fh.candidite)
				fh.reset()
			}
			fh.lock.Unlock()
		}
	}
}
