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
	//blockResponseSize = 15
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
	syncingPeers 	map[string]uint64

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
		syncingPeers: make(map[string]uint64),
	}
	bs.ticker = bs.chain.ticker
	bs.logger = taslog.GetLoggerByIndex(taslog.BlockSyncLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	bs.ticker.RegisterPeriodicRoutine(tickerSendLocalTop, bs.notifyLocalTopBlockRoutine, sendLocalTopInterval)
	bs.ticker.StartTickerRoutine(tickerSendLocalTop, false)

	bs.ticker.RegisterPeriodicRoutine(tickerSyncNeighbor, bs.trySyncRoutine, syncNeightborsInterval)
	bs.ticker.StartTickerRoutine(tickerSyncNeighbor, false)

	notify.BUS.Subscribe(notify.BlockInfoNotify, bs.topBlockInfoNotifyHandler)
	notify.BUS.Subscribe(notify.BlockReq, bs.blockReqHandler)
	notify.BUS.Subscribe(notify.BlockResponse, bs.blockResponseMsgHandler)

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

func (bs *blockSyncer) isSyncing() bool {
	localHeight := bs.chain.Height()
	bs.lock.RLock()
	defer bs.lock.RUnlock()

	_, candTop := bs.getBestCandidate("")
	if candTop == nil {
		return false
	}
	return candTop.Height > localHeight+50
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

func (bs *blockSyncer) getBestCandidate(candidateId string) (string, *TopBlockInfo) {
	if candidateId == "" {
		for id, _ := range bs.candidatePool {
			if PeerManager.isEvil(id) {
				bs.logger.Debugf("peer meter evil id:%+v", PeerManager.getOrAddPeer(id))
				delete(bs.candidatePool, id)
			}
		}
		//bs.candidatePoolDump()
		if len(bs.candidatePool) == 0 {
			return "", nil
		}
		var maxWeight float64 = 0

		for id, top := range bs.candidatePool {
			w := float64(top.TotalQN)*float64(common.MaxUint64) + float64(top.ShrinkPV)
			if w > maxWeight {
				maxWeight = w
				candidateId = id
			}
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
	return bs.syncFrom("")
}

func (bs *blockSyncer) syncFrom(from string) bool {
	topBH := bs.chain.QueryTopBlock()
	localTopBlock := bs.newTopBlockInfo(topBH)

	if bs.chain.IsAdujsting() {
		bs.logger.Debugf("chain is adjusting, won't sync")
		return false
	}
	bs.logger.Debugf("Local totalQn:%d,PV:%v, height:%d,topHash:%s", localTopBlock.TotalQN, localTopBlock.ShrinkPV, localTopBlock.Height, localTopBlock.Hash.String())

	bs.lock.Lock()
	defer bs.lock.Unlock()

	//bs.candidatePoolDump()
	candidate, candidateTop := bs.getBestCandidate(from)
	if candidate == "" {
		bs.logger.Debugf("Get no candidate for sync!")
		return false
	}
	bs.logger.Debugf("candidate info: id %v, top %v %v %v", candidate, candidateTop.Hash.String(), candidateTop.Height, candidateTop.TotalQN)

	if localTopBlock.moreWeight(candidateTop) {
		bs.logger.Debugf("local top more weight: local:%v %v %v, candidate: %v %v %v", localTopBlock.Height, localTopBlock.Hash.String(), localTopBlock.ShrinkPV, candidateTop.Height, candidateTop.Hash.String(), candidateTop.ShrinkPV)
		return false
	}
	if bs.chain.HasBlock(candidateTop.Hash) {
		bs.logger.Debugf("local has block %v, won't sync", candidateTop.Hash.String())
		return false
	}
	beginHeight := uint64(0)
	localHeight := bs.chain.Height()
	if candidateTop.Height <= localHeight {
		beginHeight = candidateTop.Height
	} else {
		beginHeight = localHeight+1
	}

	bs.logger.Debugf("beginHeight %v, candidateHeight %v", beginHeight, candidateTop.Height)
	if beginHeight > candidateTop.Height {
		return false
	}

	for syncId, h := range bs.syncingPeers {
		if h == beginHeight {
			bs.logger.Debugf("height %v in syncing from %v", beginHeight, syncId)
			return false
		}
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

	br := &SyncRequest{
		ReqHeight: height,
		ReqSize:  int32(PeerManager.getPeerReqBlockCount(id)),
	}

	body, err := MarshalSyncRequest(br)
	if err != nil {
		bs.logger.Errorf("marshalSyncRequest error %v", err)
		return
	}

	message := network.Message{Code: network.ReqBlock, Body: body}
	network.GetNetInstance().Send(id, message)

	bs.syncingPeers[id] = ci.ReqHeight

	bs.chain.ticker.RegisterOneTimeRoutine(bs.syncTimeoutRoutineName(id), func() bool {
		return bs.syncComplete(id,true)
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

	//if blockInfo.Height > bs.chain.Height()+2 {
	//	bs.logger.Debugf("Rcv topBlock Notify from %v, topHash %v, height %v， localHeight %v", bnm.Peer, blockInfo.Hash.String(), blockInfo.Height, bs.chain.Height())
	//}

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
	PeerManager.updateReqBlockCnt(id, !timeout)
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
		bs.logger.Debugf("blockResponseMsgHandler rcv from %s! [%v-%v]", source, blocks[0].Header.Height, blocks[len(blocks)-1].Header.Height)
		peerTop := bs.getPeerTopBlock(source)
		localTop := bs.newTopBlockInfo(bs.chain.QueryTopBlock())

		//先比较权重
		if peerTop != nil && localTop.moreWeight(peerTop) {
			bs.logger.Debugf("sync block from %v, local top hash %v, height %v, totalQN %v, peerTop hash %v, height %v, totalQN %v", localTop.Hash.String(), localTop.Height, localTop.TotalQN, peerTop.Hash.String(), peerTop.Height, peerTop.TotalQN)
			return
		}

		allSuccess := true

		bs.chain.batchAddBlockOnChain(source, "sync", blocks, func(b *types.Block, ret types.AddBlockResult) bool {
			bs.logger.Debugf("sync block from %v, hash=%v,height=%v,addResult=%v", source, b.Header.Hash.String(), b.Header.Height, ret)
			if ret == types.AddBlockSucc || ret == types.BlockExisted {
				return true
			}
			allSuccess = false
			return false
		})

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

	br, err := UnmarshalSyncRequest(m.ReqBody)
	if err != nil {
		bs.logger.Errorf("unmarshalSyncRequest error %v", err)
		return
	}
	localHeight := bs.chain.Height()

	bs.logger.Debugf("Rcv block request:reqHeight:%d, reqSize:%v, localHeight:%d", br.ReqHeight, br.ReqSize, localHeight)
	blocks := bs.chain.BatchGetBlocksAfterHeight(br.ReqHeight, int(br.ReqSize))
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
