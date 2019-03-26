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
	"taslog"
	"common"
	"network"
	"middleware/notify"
	"middleware/pb"
	"utility"
	"middleware/types"
	"github.com/gogo/protobuf/proto"
	"sync"
	"middleware/ticker"
	"math/big"
	"fmt"
)

const (
	sendLocalTopInterval       = 3
	syncNeightborsInterval       = 3
	syncNeightborTimeout       = 5
	blockSyncCandidatePoolSize = 100
	blockResponseSize = 10
)

const (
	tickerSendLocalTop = "send_local_top"
	tickerSyncNeighbor = "sync_neightbors"
	tickerSyncTimeout = "sync_timeout"
)

var BlockSyncer *blockSyncer

type blockSyncer struct {
	chain 			*FullBlockChain
	//syncing       int32

	candidatePool map[string]*TopBlockInfo
	syncingPeers 	map[string]byte

	//reqTimeoutTimer      *time.Timer
	//syncTimer            *time.Timer
	//blockInfoNotifyTimer *time.Timer
	ticker 		*ticker.GlobalTicker

	lock   sync.RWMutex
	logger taslog.Logger
}

type TopBlockInfo struct {
	ShrinkPV uint64
	TotalQN uint64
	Hash    common.Hash
	Height  uint64
	PreHash common.Hash
}
var maxInt194, _ = new(big.Int).SetString("2ffffffffffffffffffffffffffffffffffffffffffffffff", 16)
var float194 = new(big.Float).SetInt(maxInt194)


func (tb *TopBlockInfo) moreWeight(tb1 *TopBlockInfo) bool {
	if tb.TotalQN > tb1.TotalQN {
		return true
	}
	if tb.TotalQN < tb1.TotalQN {
		return false
	}
	return tb.ShrinkPV > tb1.ShrinkPV
}


func InitBlockSyncer(chain *FullBlockChain) {
	bs := &blockSyncer{
		candidatePool: make(map[string]*TopBlockInfo),
		chain: chain,
		syncingPeers: make(map[string]byte),
	}
	bs.ticker = bs.chain.ticker
	bs.logger = taslog.GetLoggerByIndex(taslog.BlockSyncLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	//BlockSyncer.reqTimeoutTimer = time.NewTimer(blockSyncReqTimeout)
	//BlockSyncer.syncTimer = time.NewTimer(blockSyncInterval)
	//BlockSyncer.blockInfoNotifyTimer = time.NewTimer(sendTopBlockInfoInterval)
	bs.ticker.RegisterPeriodicRoutine(tickerSendLocalTop, bs.notifyLocalTopBlockRoutine, sendLocalTopInterval)
	bs.ticker.StartTickerRoutine(tickerSendLocalTop, false)

	bs.ticker.RegisterPeriodicRoutine(tickerSyncNeighbor, bs.trySyncRoutine, syncNeightborsInterval)
	bs.ticker.StartTickerRoutine(tickerSyncNeighbor, false)

	notify.BUS.Subscribe(notify.BlockInfoNotify, bs.topBlockInfoNotifyHandler)
	notify.BUS.Subscribe(notify.BlockReq, bs.blockReqHandler)
	notify.BUS.Subscribe(notify.BlockResponse, bs.blockResponseMsgHandler)
	notify.BUS.Subscribe(notify.GroupAddSucc, bs.onGroupAddSuccess)

	BlockSyncer = bs

}

func (bs *blockSyncer) onGroupAddSuccess(msg notify.Message)  {
	g := msg.GetData().(*types.Group)
	beginHeight := g.Header.WorkHeight
	topHeight := bs.chain.Height()

	//当前块高已经超过生效高度了,组可能有点问题
	if beginHeight < topHeight {
		s := fmt.Sprintf("group add after can work! gid=%v, gheight=%v, beginHeight=%v, currentHeight=%v", common.Bytes2Hex(g.Id), g.GroupHeight, beginHeight, topHeight)
		panic(s)
	}
}

func (bs *blockSyncer) newTopBlockInfo(top *types.BlockHeader) *TopBlockInfo {
	f := float64(0)
	if top.Height > 0 {
		pv := bs.chain.consensusHelper.VRFProve2Value(top.ProveValue)
		pvFloat := new(big.Float).SetInt(pv)
		w := new(big.Float).Quo(pvFloat, float194)
		f, _ = w.Float64()
	}

	return &TopBlockInfo{
		ShrinkPV: uint64(f),
		TotalQN: top.TotalQN,
		Hash: top.Hash,
		Height: top.Height,
		PreHash: top.PreHash,
	}
}

func (bs *blockSyncer) getBestCandidate() (string, *TopBlockInfo) {
	for id, _ := range bs.candidatePool {
		if PeerManager.isEvil(id) {
			delete(bs.candidatePool, id)
		}
	}
	//bs.candidatePoolDump()
	if len(bs.candidatePool) == 0 {
		return "", nil
	}
	candidateId := ""
	var maxWeight float64 = 0

	for id, top := range bs.candidatePool {
		w := float64(top.TotalQN)*float64(common.MaxUint64) + float64(top.ShrinkPV)
		if w > maxWeight {
			maxWeight = w
			candidateId = id
		}
	}
	maxTop := bs.candidatePool[candidateId]
	if maxTop == nil {
		return "", nil
	}

	return candidateId, maxTop
}

func (bs *blockSyncer) getPeerTopBlock(id string) *TopBlockInfo {
	bs.lock.RLock()
	defer bs.lock.RUnlock()
	tb, ok := bs.candidatePool[id]
	if ok {
		return tb
	}
	return nil
}

func (bs *blockSyncer) trySyncRoutine() bool {
	topBH := bs.chain.QueryTopBlock()
	localTopBlock := bs.newTopBlockInfo(topBH)
	bs.logger.Debugf("Local totalQn:%d,PV:%v, height:%d,topHash:%s", localTopBlock.TotalQN, localTopBlock.ShrinkPV, localTopBlock.Height, localTopBlock.Hash.String())

	bs.lock.Lock()
	defer bs.lock.Unlock()

	candidate, candidateTop := bs.getBestCandidate()
	if candidate == "" {
		bs.logger.Debugf("Get no candidate for sync!")
		return false
	}

	if localTopBlock.moreWeight(candidateTop) {
		bs.logger.Debugf("local top more weight: local:%v %v %v, candidate: %v %v %v", localTopBlock.Height, localTopBlock.Hash.String(), localTopBlock.ShrinkPV, candidateTop.Height, candidateTop.Hash.String(), candidateTop.ShrinkPV)
		return false
	}
	if bs.chain.HasBlock(candidateTop.Hash) {
		bs.logger.Debugf("local has block %v, won't sync", candidateTop.Hash.String())
		return false
	}

	beginHeight := bs.chain.Height()+1
	if bs.chain.HasBlock(candidateTop.PreHash) {
		beginHeight = candidateTop.Height
	}

	candInfo := &SyncCandidateInfo{
		Candidate: candidate,
		CandidateHeight: candidateTop.Height,
		ReqHeight: beginHeight,
	}

	notify.BUS.Publish(notify.BlockSync, &SyncMessage{CandidateInfo:candInfo})

	bs.requestBlock(candInfo)
	return true
}

func (bs *blockSyncer) requestBlock(ci *SyncCandidateInfo) {
	id := ci.Candidate
	height := ci.ReqHeight
	if _, ok := bs.syncingPeers[id]; ok {
		return
	}
	bs.logger.Debugf("Req block to:%s,height:%d", id, height)

	body := utility.UInt64ToByte(height)

	message := network.Message{Code: network.ReqBlock, Body: body}
	network.GetNetInstance().Send(id, message)

	bs.syncingPeers[id] = 1
	bs.chain.ticker.RegisterOneTimeRoutine(bs.syncTimeoutRoutineName(id), func() bool {
		return bs.syncComplete(id, true)
	}, syncNeightborTimeout)
}

func (bs *blockSyncer) notifyLocalTopBlockRoutine() bool {
	top := bs.chain.QueryTopBlock()
	if top.Height == 0 {
		return false
	}
	topBlockInfo := bs.newTopBlockInfo(top)

	bs.logger.Debugf("Send local %d,%v to neighbor!", top.TotalQN, top.Hash.String())
	body, e := marshalTopBlockInfo(topBlockInfo)
	if e != nil {
		bs.logger.Errorf("marshal blockInfo error:%s", e.Error())
		return false
	}
	message := network.Message{Code: network.BlockInfoNotifyMsg, Body: body}
	network.GetNetInstance().TransmitToNeighbor(message)
	return true
}

func (bs *blockSyncer) topBlockInfoNotifyHandler(msg notify.Message) {
	bnm, ok := msg.GetData().(*notify.BlockInfoNotifyMessage)
	if !ok {
		bs.logger.Errorf("BlockInfoNotifyMessage GetData assert not ok!")
		return
	}
	blockInfo, e := bs.unMarshalTopBlockInfo(bnm.BlockInfo)
	if e != nil {
		bs.logger.Errorf("Discard BlockInfoNotifyMessage because of unmarshal error:%s", e.Error())
		return
	}

	if blockInfo.Height > bs.chain.Height()+2 {
		bs.logger.Debugf("Rcv topBlock Notify from %v, topHash %v, height %v， localHeight %v", bnm.Peer, blockInfo.Hash.String(), blockInfo.Height, bs.chain.Height())
	}

	source := bnm.Peer
	PeerManager.heardFromPeer(source)

	//topBlock := bs.gchain.QueryTopBlock()
	//localTotalQn, localTopHash := topBlock.TotalQN, topBlock.Hash
	//if !bs.isUsefulCandidate(localTotalQn, localTopHash, blockInfo.TotalQn, blockInfo.Hash) {
	//	return
	//}
	bs.addCandidatePool(source, blockInfo)
}

func (bs *blockSyncer) syncTimeoutRoutineName(id string) string {
    return tickerSyncTimeout + id
}


func (bs *blockSyncer) syncComplete(id string, timeout bool) bool {
	if timeout {
		PeerManager.timeoutPeer(id)
		bs.logger.Warnf("sync block from %v timeout", id)
	} else {
		PeerManager.heardFromPeer(id)
	}
	bs.chain.ticker.RemoveRoutine(bs.syncTimeoutRoutineName(id))
	bs.lock.Lock()
	defer bs.lock.Unlock()
	delete(bs.syncingPeers, id)
	return true
}

func (bs *blockSyncer) blockResponseMsgHandler(msg notify.Message) {
	m, ok := msg.(*notify.BlockResponseMessage)
	if !ok {
		return
	}
	source := m.Peer
	if bs == nil {
		panic("blockSyncer is nil!")
	}
	bs.logger.Debugf("blockResponseMsgHandler rcv from %s!", source)
	var complete = false
	defer func() {
		if !complete {
			bs.syncComplete(source, false)
		}
	}()

	blockResponse, e := bs.unMarshalBlockMsgResponse(m.BlockResponseByte)
	if e != nil {
		bs.logger.Debugf("Discard block response msg because unMarshalBlockMsgResponse error:%d", e.Error())
		return
	}

	blocks := blockResponse.Blocks

	if blocks == nil || len(blocks) == 0 {
		bs.logger.Debugf("Rcv block response nil from:%s", source)
	} else {
		peerTop := bs.getPeerTopBlock(source)

		allSuccess := true

		bs.chain.batchAddBlockOnChain(source, blocks, func(b *types.Block, ret types.AddBlockResult) bool {
			bs.logger.Debugf("sync block from %v, hash=%v,height=%v,addResult=%v", source, b.Header.Hash.String(), b.Header.Height, ret)
			if ret == types.AddBlockSucc || ret == types.BlockExisted {
				return true
			}
			allSuccess = false
			return false
		})

		localTop := bs.newTopBlockInfo(bs.chain.QueryTopBlock())
		//权重还是比较低，继续同步(必须所有上链成功，否则会造成死循环）
		if allSuccess && peerTop != nil && peerTop.moreWeight(localTop) {
			bs.syncComplete(source, false)
			complete = true
			go bs.trySyncRoutine()
		}
	}
}

func (bs *blockSyncer) addCandidatePool(source string, topBlockInfo *TopBlockInfo) {
	//if PeerManager.isEvil(id) {
	//	bs.logger.Debugf("Top block info notify id:%s is marked evil.Drop it!", id)
	//	return
	//}
	bs.lock.Lock()
	defer bs.lock.Unlock()

	if len(bs.candidatePool) < blockSyncCandidatePoolSize {
		bs.candidatePool[source] = topBlockInfo
		return
	}
	for id, tbi := range bs.candidatePool {
		if topBlockInfo.moreWeight(tbi) {
			delete(bs.candidatePool, id)
			bs.candidatePool[source] = topBlockInfo
		}
	}
}


func (bs blockSyncer) blockReqHandler(msg notify.Message) {
	m, ok := msg.(*notify.BlockReqMessage)
	if !ok {
		bs.logger.Debugf("blockReqHandler:Message assert not ok!")
		return
	}
	reqHeight := utility.ByteToUInt64(m.ReqBody)
	localHeight := bs.chain.Height()

	bs.logger.Debugf("Rcv block request:reqHeight:%d,localHeight:%d", reqHeight, localHeight)
	blocks := bs.chain.BatchGetBlocksAfterHeight(reqHeight, blockResponseSize)
	responseBlocks(m.Peer, blocks)
}

func responseBlocks(targetId string, blocks []*types.Block) {
	body, e := marshalBlockMsgResponse(&BlockResponseMessage{Blocks: blocks})
	if e != nil {
		return
	}
	message := network.Message{Code: network.BlockResponseMsg, Body: body}
	network.GetNetInstance().Send(targetId, message)
}

func marshalBlockMsgResponse(bmr *BlockResponseMessage) ([]byte, error) {
	pbblocks := make([]*tas_middleware_pb.Block, 0)
	for _, b := range bmr.Blocks {
		pb := types.BlockToPb(b)
		pbblocks = append(pbblocks, pb)
	}
	message := tas_middleware_pb.BlockResponseMsg{Blocks: pbblocks}
	return proto.Marshal(&message)
}

func (bs *blockSyncer) candidatePoolDump() {
	bs.logger.Debugf("Candidate Pool Dump:")
	for id, topBlockInfo := range bs.candidatePool {
		bs.logger.Debugf("Candidate id:%s,totalQn:%d, pv:%v, height:%d,topHash:%s", id, topBlockInfo.TotalQN, topBlockInfo.ShrinkPV, topBlockInfo.Height, topBlockInfo.Hash.String())
	}
}

func marshalTopBlockInfo(bi *TopBlockInfo) ([]byte, error) {
	blockInfo := tas_middleware_pb.TopBlockInfo{Hash: bi.Hash.Bytes(), TotalQn: &bi.TotalQN, ShrinkPV: &bi.ShrinkPV, Height: &bi.Height, PreHash: bi.PreHash.Bytes()}
	return proto.Marshal(&blockInfo)
}

func (bs *blockSyncer) unMarshalTopBlockInfo(b []byte) (*TopBlockInfo, error) {
	message := new(tas_middleware_pb.TopBlockInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		bs.logger.Errorf("unMarshalBlockInfo error:%s", e.Error())
		return nil, e
	}
	blockInfo := TopBlockInfo{Hash: common.BytesToHash(message.Hash), TotalQN: *message.TotalQn, ShrinkPV: *message.ShrinkPV, Height: *message.Height, PreHash: common.BytesToHash(message.PreHash)}
	return &blockInfo, nil
}

func (bs *blockSyncer) unMarshalBlockMsgResponse(b []byte) (*BlockResponseMessage, error) {
	message := new(tas_middleware_pb.BlockResponseMsg)
	e := proto.Unmarshal(b, message)
	if e != nil {
		bs.logger.Errorf("unMarshalBlockMsgResponse error:%s", e.Error())
		return nil, e
	}
	blocks := make([]*types.Block, 0)
	for _, pb := range message.Blocks {
		b := types.PbToBlock(pb)
		blocks = append(blocks, b)
	}
	bmr := BlockResponseMessage{Blocks: blocks}
	return &bmr, nil
}
