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
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/middleware/notify"
	"github.com/taschain/taschain/middleware/pb"
	"github.com/taschain/taschain/middleware/types"
	"github.com/taschain/taschain/network"
	"github.com/taschain/taschain/taslog"
	"sync"

	"github.com/gogo/protobuf/proto"
)

const (
	reqPieceTimeout  = 60
	chainPieceLength = 10
)

const tickerReqPieceBlock = "req_chain_piece_block"

type forkSyncContext struct {
	target       string
	targetTop    *TopBlockInfo
	lastReqPiece *ChainPieceReq
	localTop     *TopBlockInfo
}

func (fctx *forkSyncContext) getLastHash() common.Hash {
	size := len(fctx.lastReqPiece.ChainPiece)
	return fctx.lastReqPiece.ChainPiece[size-1]
}

type forkProcessor struct {
	chain *FullBlockChain

	syncCtx *forkSyncContext

	lock   sync.RWMutex
	logger taslog.Logger
}

type ChainPieceBlockMsg struct {
	Blocks       []*types.Block
	TopHeader    *types.BlockHeader
	FindAncestor bool
}

type ChainPieceReq struct {
	ChainPiece []common.Hash
	ReqCnt     int32
}

func initForkProcessor(chain *FullBlockChain) *forkProcessor {
	fh := forkProcessor{
		chain: chain,
	}
	fh.logger = taslog.GetLoggerByIndex(taslog.ForkLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	notify.BUS.Subscribe(notify.ChainPieceBlockReq, fh.chainPieceBlockReqHandler)
	notify.BUS.Subscribe(notify.ChainPieceBlock, fh.chainPieceBlockHandler)

	return &fh
}

func (fp *forkProcessor) targetTop(id string, bh *types.BlockHeader) *TopBlockInfo {
	targetTop := BlockSyncer.getPeerTopBlock(id)
	tb := newTopBlockInfo(bh)
	if targetTop != nil && targetTop.MoreWeight(&tb.BlockWeight) {
		return targetTop
	}
	return tb
}

func (fp *forkProcessor) updateContext(id string, bh *types.BlockHeader) bool {
	ctx := fp.syncCtx
	targetTop := fp.targetTop(id, bh)

	if ctx == nil || targetTop.MoreWeight(&ctx.targetTop.BlockWeight) {
		newCtx := &forkSyncContext{
			target:    id,
			targetTop: targetTop,
			localTop:  newTopBlockInfo(fp.chain.QueryTopBlock()),
		}
		fp.syncCtx = newCtx
		return true
	}
	fp.logger.Warnf("old target %v %v %v, new target %v %v %v, won't process", fp.syncCtx.target, fp.syncCtx.targetTop.Height, fp.syncCtx.targetTop.TotalQN, id, targetTop.Height, targetTop.TotalQN)

	return false
}

func (fp *forkProcessor) getLocalPieceInfo(topHash common.Hash) []common.Hash {
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
	return pieces
}

func (fp *forkProcessor) tryToProcessFork(targetNode string, b *types.Block) {
	if BlockSyncer == nil {
		return
	}
	if targetNode == "" {
		return
	}

	fp.lock.Lock()
	defer fp.lock.Unlock()

	bh := b.Header
	if fp.chain.HasBlock(bh.PreHash) {
		fp.chain.AddBlockOnChain(targetNode, b)
		return
	}

	if !fp.updateContext(targetNode, bh) {
		return
	}

	fp.requestPieceBlock(fp.chain.QueryTopBlock().Hash)
}

func (fp *forkProcessor) reqPieceTimeout(id string) {
	fp.logger.Debugf("req piece from %v timeout", id)
	if fp.syncCtx == nil {
		return
	}
	fp.lock.Lock()
	defer fp.lock.Unlock()

	if fp.syncCtx.target != id {
		return
	}
	PeerManager.timeoutPeer(fp.syncCtx.target)
	PeerManager.updateReqBlockCnt(fp.syncCtx.target, false)
	fp.reset()
}

func (fp *forkProcessor) reset() {
	fp.syncCtx = nil
}

func (fp *forkProcessor) requestPieceBlock(topHash common.Hash) {

	chainPieceInfo := fp.getLocalPieceInfo(topHash)
	if len(chainPieceInfo) == 0 {
		fp.reset()
		return
	}

	reqCnt := PeerManager.getPeerReqBlockCount(fp.syncCtx.target)

	pieceReq := &ChainPieceReq{
		ChainPiece: chainPieceInfo,
		ReqCnt:     int32(reqCnt),
	}

	body, e := marshalChainPieceInfo(pieceReq)
	if e != nil {
		fp.logger.Errorf("Marshal chain piece info error:%s!", e.Error())
		fp.reset()
		return
	}
	fp.logger.Debugf("req piece from %v, reqCnt %v", fp.syncCtx.target, reqCnt)

	message := network.Message{Code: network.ReqChainPieceBlock, Body: body}
	network.GetNetInstance().Send(fp.syncCtx.target, message)

	fp.syncCtx.lastReqPiece = pieceReq

	// Start ticker
	fp.chain.ticker.RegisterOneTimeRoutine(tickerReqPieceBlock, func() bool {
		fp.reqPieceTimeout(fp.syncCtx.target)
		return true
	}, reqPieceTimeout)
}

func (fp *forkProcessor) findCommonAncestor(piece []common.Hash) *common.Hash {
	for _, h := range piece {
		if fp.chain.HasBlock(h) {
			return &h
		}
	}
	return nil
}

func (fp *forkProcessor) chainPieceBlockReqHandler(msg notify.Message) {
	m := notify.AsDefault(msg)

	source := m.Source()
	pieceReq, err := unMarshalChainPieceInfo(m.Body())
	if err != nil {
		fp.logger.Errorf("unMarshalChainPieceInfo err %v", err)
		return
	}

	fp.logger.Debugf("Rcv chain piece block req from:%s, pieceSize %v, reqCnt %v", source, len(pieceReq.ChainPiece), pieceReq.ReqCnt)
	if pieceReq.ReqCnt > maxReqBlockCount {
		pieceReq.ReqCnt = maxReqBlockCount
	}

	blocks := make([]*types.Block, 0)
	ancestor := fp.findCommonAncestor(pieceReq.ChainPiece)

	response := &ChainPieceBlockMsg{
		TopHeader:    fp.chain.QueryTopBlock(),
		FindAncestor: ancestor != nil,
		Blocks:       blocks,
	}

	if ancestor != nil { // Find a common ancestor
		ancestorBH := fp.chain.queryBlockHeaderByHash(*ancestor)
		// Maybe the ancestor were killed due to forks
		if ancestorBH != nil {
			blocks = fp.chain.BatchGetBlocksAfterHeight(ancestorBH.Height, int(pieceReq.ReqCnt))
			response.Blocks = blocks
		}
	}
	fp.sendChainPieceBlock(source, response)
}

func (fp *forkProcessor) sendChainPieceBlock(targetID string, msg *ChainPieceBlockMsg) {
	fp.logger.Debugf("Send chain piece blocks to:%s, findAncestor=%v, blockSize=%v", targetID, msg.FindAncestor, len(msg.Blocks))
	body, e := marshalChainPieceBlockMsg(msg)
	if e != nil {
		fp.logger.Errorf("Marshal chain piece block msg error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.ChainPieceBlock, Body: body}
	network.GetNetInstance().Send(targetID, message)
}

func (fp *forkProcessor) reqFinished(id string, reset bool) {
	if fp.syncCtx == nil || fp.syncCtx.target != id {
		return
	}
	PeerManager.heardFromPeer(id)
	fp.chain.ticker.RemoveRoutine(tickerReqPieceBlock)
	PeerManager.updateReqBlockCnt(id, true)
	if reset {
		fp.syncCtx = nil
	}
	return
}

func (fp *forkProcessor) getNextSyncHash() *common.Hash {
	last := fp.syncCtx.getLastHash()
	bh := fp.chain.QueryBlockHeaderByHash(last)
	if bh != nil {
		h := bh.PreHash
		return &h
	}
	return nil
}

func (fp *forkProcessor) chainPieceBlockHandler(msg notify.Message) {
	m := notify.AsDefault(msg)

	fp.lock.Lock()
	defer fp.lock.Unlock()

	source := m.Source()

	ctx := fp.syncCtx
	if ctx == nil {
		fp.logger.Debugf("ctx is nil: source=%v", source)
		return
	}
	chainPieceBlockMsg, e := unmarshalChainPieceBlockMsg(m.Body())
	if e != nil {
		fp.logger.Debugf("Unmarshal chain piece block msg error:%d", e.Error())
		return
	}

	blocks := chainPieceBlockMsg.Blocks
	topHeader := chainPieceBlockMsg.TopHeader

	// Target changed
	if source != ctx.target {
		// If the received block contains a branch of the ctx request, you can continue the add on chain process.
		sameFork := false
		if topHeader != nil && topHeader.Hash == ctx.targetTop.Hash {
			sameFork = true
		} else if len(blocks) > 0 {
			for _, b := range blocks {
				if b.Header.Hash == ctx.targetTop.Hash {
					sameFork = true
					break
				}
			}
		}
		if !sameFork {
			fp.logger.Debugf("Unexpected chain piece block from %s, expect from %s, blocksize %v", source, ctx.target, len(blocks))
			return
		}
		fp.logger.Debugf("upexpected target blocks, buf same fork!target=%v, expect=%v, blocksize %v", source, ctx.target, len(blocks))
	}
	var reset = true
	defer func() {
		fp.reqFinished(source, reset)
	}()

	if ctx.lastReqPiece == nil {
		return
	}

	if topHeader == nil || len(blocks) == 0 {
		return
	}

	// Giving a piece to go is not enough to find a common ancestor, continue to request a piece
	if !chainPieceBlockMsg.FindAncestor {
		fp.logger.Debugf("cannot find common ancestor from %v, keep finding", source)
		nextSync := fp.getNextSyncHash()
		if nextSync != nil {
			fp.requestPieceBlock(*nextSync)
			reset = false
		}
	} else {
		ancestorBH := blocks[0].Header
		if !fp.chain.HasBlock(ancestorBH.Hash) {
			fp.logger.Errorf("local ancestor block not exist, hash=%v, height=%v", ancestorBH.Hash.String(), ancestorBH.Height)
		} else if len(blocks) > 1 {
			fp.chain.batchAddBlockOnChain(source, "fork", blocks, func(b *types.Block, ret types.AddBlockResult) bool {
				fp.logger.Debugf("sync fork block from %v, hash=%v,height=%v,addResult=%v", source, b.Header.Hash.String(), b.Header.Height, ret)
				return ret == types.AddBlockSucc || ret == types.BlockExisted
			})
			// Start synchronization if the local weight is still below the weight of the other party
			if fp.chain.compareChainWeight(topHeader) < 0 {
				go BlockSyncer.trySyncRoutine()
			}
		}
	}
}

func unMarshalChainPieceInfo(b []byte) (*ChainPieceReq, error) {
	message := new(tas_middleware_pb.ChainPieceReq)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}

	chainPiece := make([]common.Hash, 0)
	for _, hashBytes := range message.Pieces {
		h := common.BytesToHash(hashBytes)
		chainPiece = append(chainPiece, h)
	}
	cnt := int32(maxReqBlockCount)
	if message.ReqCnt != nil {
		cnt = *message.ReqCnt
	}
	chainPieceInfo := &ChainPieceReq{ChainPiece: chainPiece, ReqCnt: cnt}
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

func marshalChainPieceInfo(chainPieceInfo *ChainPieceReq) ([]byte, error) {
	pieces := make([][]byte, 0)
	for _, hash := range chainPieceInfo.ChainPiece {
		pieces = append(pieces, hash.Bytes())
	}
	message := tas_middleware_pb.ChainPieceReq{Pieces: pieces, ReqCnt: &chainPieceInfo.ReqCnt}
	return proto.Marshal(&message)
}
