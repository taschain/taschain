//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.
package core

import (
	"sync"
	"network"
	"common"
	"taslog"
	"middleware/notify"
	"middleware/types"
	"middleware/pb"

	"github.com/gogo/protobuf/proto"
)

const (
	reqPieceTimeout       = 5
	chainPieceLength      = 10
	chainPieceBlockLength = 6
)

const tickerReqPieceBlock = "req_chain_piece_block"

type forkSyncContext struct {
	target 		string
	targetTop 	*types.BlockHeader
	lastReqPiece *ChainPieceInfo
	localTop	*types.BlockHeader
}

func (fctx *forkSyncContext) getNextSyncTopHash() common.Hash {
    size := len(fctx.lastReqPiece.ChainPiece)
    return fctx.lastReqPiece.ChainPiece[size-1]
}

type forkProcessor struct {
	chain      *FullBlockChain

	syncCtx 	*forkSyncContext

	lock   sync.RWMutex
	logger taslog.Logger
}

type ChainPieceBlockMsg struct {
	Blocks    []*types.Block
	TopHeader *types.BlockHeader
	FindAncestor bool
}

type ChainPieceInfo struct {
	ChainPiece []common.Hash
}

func initForkProcessor() *forkProcessor {
	fh := forkProcessor{
		chain: BlockChainImpl.(*FullBlockChain),
	}
	fh.logger = taslog.GetLoggerByIndex(taslog.ForkLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	//notify.BUS.Subscribe(notify.ChainPieceInfoReq, fh.chainPieceInfoReqHandler)
	//notify.BUS.Subscribe(notify.ChainPieceInfo, fh.chainPieceInfoHandler)
	notify.BUS.Subscribe(notify.ChainPieceBlockReq, fh.chainPieceBlocReqHandler)
	notify.BUS.Subscribe(notify.ChainPieceBlock, fh.chainPieceBlockHandler)

	return &fh
}

func (fp *forkProcessor) updateContext(id string, bh *types.BlockHeader) bool {
	newCtx := &forkSyncContext{
		target: id,
		targetTop: bh,
	}

	ctx := fp.syncCtx
	if ctx == nil || (ctx.target != id && fp.chain.compareBlockWeight(ctx.targetTop, bh) < 0 ) {
		fp.syncCtx = newCtx
		fp.syncCtx.localTop = fp.chain.QueryTopBlock()
		return true
	}
	return false
}


func (fp *forkProcessor) getLocalPieceInfo(topHash common.Hash) *ChainPieceInfo {
	bh := fp.chain.queryBlockHeaderByHash(topHash)
	pieces := make([]common.Hash, 0)
	cnt := 0
	for cnt < chainPieceLength {
		if bh != nil {
			pieces = append(pieces, bh.Hash)
			cnt++
			bh = fp.chain.queryBlockHeaderByHash(bh.PreHash)
		} else {
			break
		}
	}
	return &ChainPieceInfo{ChainPiece: pieces}
}

func (fp *forkProcessor) tryToProcessFork(targetNode string, bh *types.BlockHeader) {
	if BlockSyncer == nil {
		return
	}
	if targetNode == "" {
		return
	}

	fp.lock.Lock()
	defer fp.lock.Unlock()

	if !fp.updateContext(targetNode, bh) {
		fp.logger.Warnf("old target %v %v, new target %v %v, wo'nt process", fp.syncCtx.target, fp.syncCtx.targetTop.TotalQN, targetNode, bh.TotalQN)
		return
	}

	fp.requestPieceBlock(fp.chain.QueryTopBlock().Hash)
}

//func (fp *forkProcessor) requestChainPieceInfo(target string, height uint64)  {
//	fp.chain.ticker.RegisterOneTimeRoutine(tickerReqPieceBlock, func() bool {
//		return 	fp.reqPieceComplete(true)
//	}, reqPieceTimeout)
//
//	fp.logger.Debugf("Req chain piece info to:%s,local height:%d", target, height)
//	body := utility.UInt64ToByte(height)
//	message := network.Message{Code: network.ChainPieceInfoReq, Body: body}
//	network.GetNetInstance().Send(target, message)
//}

func (fp *forkProcessor) reqPieceTimeout(id string) {
	fp.lock.Lock()
	defer fp.lock.Unlock()

	if fp.syncCtx.target != id {
		return
	}
	PeerManager.timeoutPeer(fp.syncCtx.target)
	fp.syncCtx = nil
}

func (fp *forkProcessor) requestPieceBlock(topHash common.Hash) {

	chainPieceInfo := fp.getLocalPieceInfo(topHash)

	chainPiece := chainPieceInfo.ChainPiece
	if len(chainPiece) == 0 {
		return
	}

	body, e := marshalChainPieceInfo(chainPieceInfo)
	if e != nil {
		fp.logger.Errorf("Marshal chain piece info error:%s!", e.Error())
		return
	}
	message := network.Message{Code: network.ReqChainPieceBlock, Body: body}
	network.GetNetInstance().Send(fp.syncCtx.target, message)

	fp.syncCtx.lastReqPiece = chainPieceInfo

	//启动定时器
	fp.chain.ticker.RegisterOneTimeRoutine(tickerReqPieceBlock, func() bool {
		fp.reqPieceTimeout(fp.syncCtx.target)
		return true
	}, reqPieceTimeout)
}

func (fp *forkProcessor) findCommonAncestor(piece []common.Hash) *common.Hash {
	for _, h := range piece {
		if fp.chain.hasBlock(h) {
			return &h
		}
	}
	return nil
}

func (fp *forkProcessor) ensurePieceChained(pieces []common.Hash) bool {
	if len(pieces) <= 1 {
		return true
	}
	for i:=1; i < len(pieces); i++ {
		if pieces[i] != pieces[i-1] {
			return false
		}
	}
	return true
}

func (fp *forkProcessor) chainPieceBlocReqHandler(msg notify.Message) {
	m, ok := msg.GetData().(*notify.ChainPieceBlockReqMessage)
	if !ok {
		return
	}
	source := m.Peer
	chainPiece, err := unMarshalChainPieceInfo(m.ReqBody)
	if err != nil {
		fp.logger.Errorf("unMarshalChainPieceInfo err %v", err)
		return
	}
	if !fp.ensurePieceChained(chainPiece.ChainPiece) {
		hashString := make([]string, 0)
		for _, piece := range chainPiece.ChainPiece {
			hashString = append(hashString, piece.String())
		}
		fp.logger.Errorf("chain piece not chained!, %v", hashString)
		return
	}
	fp.logger.Debugf("Rcv chain piece block req from:%s, pieceSize %v", source, len(chainPiece.ChainPiece))

	blocks := make([]*types.Block, 0)
	ancestor := fp.findCommonAncestor(chainPiece.ChainPiece)

	response := &ChainPieceBlockMsg{
		TopHeader: fp.chain.QueryTopBlock(),
		FindAncestor: ancestor != nil,
		Blocks: blocks,
	}

	if ancestor != nil {//找到共同祖先
		ancestorBH := fp.chain.queryBlockHeaderByHash(*ancestor)
		//可能祖先被分叉干掉了
		if ancestorBH != nil {
			blocks = fp.chain.batchGetBlocksAfterHeight(ancestorBH.Height, chainPieceBlockLength)
			response.Blocks = blocks
		}
	}
	fp.sendChainPieceBlock(source, response)
}

func (fp *forkProcessor) sendChainPieceBlock(targetId string, msg *ChainPieceBlockMsg) {
	fp.logger.Debugf("Send chain piece blocks to:%s, findAncestor=%v, blockSize=%v", targetId, msg.FindAncestor, len(msg.Blocks))
	body, e := marshalChainPieceBlockMsg(msg)
	if e != nil {
		fp.logger.Errorf("Marshal chain piece block msg error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.ChainPieceBlock, Body: body}
	network.GetNetInstance().Send(targetId, message)
}

func (fp *forkProcessor) reqFinished(id string, reset bool) {
	if fp.syncCtx.target != id {
		return
	}
	PeerManager.heardFromPeer(id)
	fp.chain.ticker.RemoveRoutine(tickerReqPieceBlock)
	if reset {
		fp.syncCtx = nil
	}
	return
}

func (fp *forkProcessor) chainPieceBlockHandler(msg notify.Message) {
	m, ok := msg.GetData().(*notify.ChainPieceBlockMessage)
	if !ok {
		return
	}

	fp.lock.Lock()
	defer fp.lock.Unlock()

	source := m.Peer

	ctx := fp.syncCtx
	if source != ctx.target {//target改变了
		fp.logger.Debugf("Unexpected chain piece block from %s, expect from %s!", source, ctx.target)
		return
	}
	var reset = true
	defer func() {
		fp.reqFinished(source, reset)
	}()

	if ctx.lastReqPiece == nil {
		return
	}

	chainPieceBlockMsg, e := unmarshalChainPieceBlockMsg(m.ChainPieceBlockMsgByte)
	if e != nil {
		fp.logger.Debugf("Unmarshal chain piece block msg error:%d", e.Error())
		return
	}

	blocks := chainPieceBlockMsg.Blocks
	topHeader := chainPieceBlockMsg.TopHeader

	if topHeader == nil || len(blocks) == 0 {
		return
	}
	//如果对方的权重已经低于本地权重，则不用后续处理
	if fp.chain.compareBlockWeight(topHeader, ctx.localTop) < 0 {
		fp.logger.Debugf("local weight is bigger than peer:%v, localTop %v %v, peerTop %v %v", source, ctx.localTop.Hash.String(), ctx.localTop.Height, topHeader.Hash.String(), topHeader.Height)
		return
	}

	//给出去的piece不足以找到共同祖先，继续请求piece
	if !chainPieceBlockMsg.FindAncestor {
		fp.logger.Debugf("cannot find common ancestor from %v, keep finding", source)
		topHash := ctx.getNextSyncTopHash()
		fp.requestPieceBlock(topHash)
		reset = false
	} else {
		ancestorBH := blocks[0].Header
		if !fp.chain.hasBlock(ancestorBH.Hash) {
			fp.logger.Errorf("local ancestor block not exist, hash=%v, height=%v", ancestorBH.Hash.String(), ancestorBH.Height)
		} else {
			fp.chain.ResetTop(ancestorBH)
			fp.chain.batchAddBlockOnChain(source, blocks, func(b *types.Block, ret types.AddBlockResult) bool {
				fp.logger.Debugf("sync fork block from %v, hash=%v,height=%v,addResult=%v", source, b.Header.Hash.String(), b.Header.Height, ret)
				return ret == types.AddBlockSucc || ret == types.BlockExisted
			})
			//如果本地权重仍低于对方权重，则启动同步
			if fp.chain.compareChainWeight(topHeader) < 0 {
				go BlockSyncer.trySyncRoutine()
			}
		}
	}
}


func unMarshalChainPieceInfo(b []byte) (*ChainPieceInfo, error) {
	message := new(tas_middleware_pb.ChainPieceInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}

	chainPiece := make([]common.Hash, 0)
	for _, hashBytes := range message.Pieces {
		h := common.BytesToHash(hashBytes)
		chainPiece = append(chainPiece, h)
	}
	chainPieceInfo := &ChainPieceInfo{ChainPiece: chainPiece}
	return chainPieceInfo, nil
}

func marshalChainPieceBlockMsg(cpb *ChainPieceBlockMsg) ([]byte, error) {
	topHeader := types.BlockHeaderToPb(cpb.TopHeader)
	blocks := make([]*tas_middleware_pb.Block, 0)
	for _, b := range cpb.Blocks {
		blocks = append(blocks, types.BlockToPb(b))
	}
	message := tas_middleware_pb.ChainPieceBlockMsg{TopHeader: topHeader, Blocks: blocks, FindAncestor: &cpb.FindAncestor}
	return proto.Marshal(&message)
}

func unmarshalChainPieceBlockMsg(b []byte) (*ChainPieceBlockMsg, error) {
	message := new(tas_middleware_pb.ChainPieceBlockMsg)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}
	topHeader := types.PbToBlockHeader(message.TopHeader)
	blocks := make([]*types.Block, 0)
	for _, b := range message.Blocks {
		blocks = append(blocks, types.PbToBlock(b))
	}
	cpb := ChainPieceBlockMsg{TopHeader: topHeader, Blocks: blocks, FindAncestor: *message.FindAncestor}
	return &cpb, nil
}

func marshalChainPieceInfo(chainPieceInfo *ChainPieceInfo) ([]byte, error) {
	pieces := make([][]byte, 0)
	for _, hash := range chainPieceInfo.ChainPiece {
		pieces = append(pieces, hash.Bytes())
	}
	message := tas_middleware_pb.ChainPieceInfo{Pieces: pieces,}
	return proto.Marshal(&message)
}
