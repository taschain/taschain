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

	init                 bool
	//reqTimeoutTimer      *time.Timer
	//syncTimer            *time.Timer
	//blockInfoNotifyTimer *time.Timer

	lock   sync.RWMutex
	logger taslog.Logger
}

type TopBlockInfo struct {
	TotalQn uint64
	Hash    common.Hash
	Height  uint64
	PreHash common.Hash
}

type forkInfo map[common.Hash]uint64

func (fi forkInfo) blockInFork(tb *TopBlockInfo) bool {
	if _, ok := fi[tb.Hash]; ok {
		return true
	}
	if _, ok := fi[tb.PreHash]; ok {
		return true
	}
	return false
}

func (fi forkInfo) addBlock(tb *TopBlockInfo)  {
	fi[tb.PreHash] = tb.Height-1
	fi[tb.Hash] = tb.Height
}

func (fi forkInfo) size() int {
	return len(fi)
}


func InitBlockSyncer() {
	bs := &blockSyncer{
		candidatePool: make(map[string]*TopBlockInfo),
		init: false,
		chain: BlockChainImpl.(*FullBlockChain),
		syncingPeers: make(map[string]byte),
	}
	BlockSyncer.logger = taslog.GetLoggerByIndex(taslog.BlockSyncLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	//BlockSyncer.reqTimeoutTimer = time.NewTimer(blockSyncReqTimeout)
	//BlockSyncer.syncTimer = time.NewTimer(blockSyncInterval)
	//BlockSyncer.blockInfoNotifyTimer = time.NewTimer(sendTopBlockInfoInterval)
	bs.chain.ticker.RegisterPeriodicRoutine(tickerSendLocalTop, bs.notifyLocalTopBlockRoutine, sendLocalTopInterval)
	bs.chain.ticker.StartTickerRoutine(tickerSendLocalTop, false)

	bs.chain.ticker.RegisterPeriodicRoutine(tickerSyncNeighbor, bs.trySyncRoutine, syncNeightborsInterval)
	bs.chain.ticker.StartTickerRoutine(tickerSyncNeighbor, false)

	notify.BUS.Subscribe(notify.BlockInfoNotify, BlockSyncer.topBlockInfoNotifyHandler)
	notify.BUS.Subscribe(notify.BlockReq, BlockSyncer.blockReqHandler)
	notify.BUS.Subscribe(notify.BlockResponse, BlockSyncer.blockResponseMsgHandler)

	BlockSyncer = bs

}

func (bs *blockSyncer) IsInit() bool {
	return bs.init
}

func (bs *blockSyncer) getBestCandidate() (string, uint64) {
	topBH := bs.chain.QueryTopBlock()
	localTopBlock := &TopBlockInfo{
		TotalQn: topBH.TotalQN,
		Hash: topBH.Hash,
		Height: topBH.Height,
		PreHash: topBH.PreHash,
	}
	bs.logger.Debugf("Local totalQn:%d,height:%d,topHash:%s", localTopBlock.TotalQn, localTopBlock.Height, localTopBlock.Hash.String())

	bs.candidatePoolDump()

	for id, _ := range bs.candidatePool {
		if PeerManager.isEvil(id) {
			delete(bs.candidatePool, id)
		}
	}

	forks := make([]*forkInfo, 0)

	for _, top := range bs.candidatePool {
		hasFork := false
		for _, fm := range forks {
			if fm.blockInFork(top) {
				fm.addBlock(top)
				hasFork = true
				break
			}
		}
		if !hasFork {
			fi := &forkInfo{}
			fi.addBlock(top)
			forks = append(forks, fi)
		}
	}
	maxCand := 0
	maxCandFork := &forkInfo{}
	for _, fi := range forks {
		if fi.size() > maxCand {
			maxCand = fi.size()
			maxCandFork = fi
		}
	}

	candidateId := ""
	var candidateMaxTotalQn uint64 = 0
	var candidateHeight uint64

	for id, topBlockInfo := range bs.candidatePool {
		if maxCandFork.blockInFork(topBlockInfo) && topBlockInfo.TotalQn > candidateMaxTotalQn {
			candidateId = id
			candidateMaxTotalQn = topBlockInfo.TotalQn
			candidateHeight = topBlockInfo.Height
		}
	}

	if maxCandFork.blockInFork(localTopBlock) && candidateMaxTotalQn <= localTopBlock.TotalQn {
		candidateId = ""
	}
	return candidateId, candidateHeight
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
	bs.lock.Lock()
	defer bs.lock.Unlock()

	candidate, _ := bs.getBestCandidate()
	if candidate == "" {
		bs.logger.Debugf("Get no candidate for sync!")
		if !bs.init {
			bs.init = true
		}
		return false
	}
	beginHeight := bs.chain.Height()+1
	bs.requestBlock(candidate, beginHeight)
	return true
}

func (bs *blockSyncer) requestBlock(id string, height uint64) {
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
	topBlockInfo := &TopBlockInfo{Hash: top.Hash, TotalQn: top.TotalQN, Height: top.Height, PreHash: top.PreHash}

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

	source := bnm.Peer
	PeerManager.heardFromPeer(source)

	//topBlock := bs.chain.QueryTopBlock()
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
		bs.logger.Warnf("sync from %v timeout", id)
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
	defer func() {
		bs.syncComplete(source, false)
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

		bs.chain.batchAddBlockOnChain(source, blocks, func(b *types.Block, ret types.AddBlockResult) bool {
			bs.logger.Debugf("sync block from %v, hash=%v,height=%v,addResult=%v", source, b.Header.Hash.String(), b.Header.Height, ret)
			return ret == types.AddBlockSucc || ret == types.BlockExisted
		})

		//权重还是比较低，继续同步
		if peerTop != nil && bs.chain.TotalQN() < peerTop.TotalQn {
			go bs.trySyncRoutine()
		}
	}
}

func (bs *blockSyncer) addCandidatePool(id string, topBlockInfo *TopBlockInfo) {
	//if PeerManager.isEvil(id) {
	//	bs.logger.Debugf("Top block info notify id:%s is marked evil.Drop it!", id)
	//	return
	//}
	bs.lock.Lock()
	defer bs.lock.Unlock()

	if len(bs.candidatePool) < blockSyncCandidatePoolSize {
		bs.candidatePool[id] = topBlockInfo
		return
	}
	totalQnMinId := ""
	var minTotalQn uint64 = common.MaxUint64
	for id, tbi := range bs.candidatePool {
		if tbi.TotalQn <= minTotalQn {
			totalQnMinId = id
			minTotalQn = tbi.TotalQn
		}
	}
	if topBlockInfo.TotalQn > minTotalQn {
		delete(bs.candidatePool, totalQnMinId)
		bs.candidatePool[id] = topBlockInfo
	}
}


func (bs blockSyncer) blockReqHandler(msg notify.Message) {
	m, ok := msg.(*notify.BlockReqMessage)
	if !ok {
		Logger.Debugf("blockReqHandler:Message assert not ok!")
		return
	}
	reqHeight := utility.ByteToUInt64(m.ReqBody)
	localHeight := bs.chain.Height()

	Logger.Debugf("Rcv block request:reqHeight:%d,localHeight:%d", reqHeight, localHeight)
	blocks := bs.chain.batchGetBlocksAfterHeight(reqHeight, blockResponseSize)
	responseBlocks(m.Peer, blocks)
}

func responseBlocks(targetId string, blocks []*types.Block) {
	body, e := marshalBlockMsgResponse(&BlockResponseMessage{Blocks: blocks})
	if e != nil {
		Logger.Errorf("Marshal block msg response error:%s", e.Error())
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
		bs.logger.Debugf("Candidate id:%s,totalQn:%d,height:%d,topHash:%s", id, topBlockInfo.TotalQn, topBlockInfo.Height, topBlockInfo.Hash.String())
	}
}

func marshalTopBlockInfo(bi *TopBlockInfo) ([]byte, error) {
	blockInfo := tas_middleware_pb.TopBlockInfo{Hash: bi.Hash.Bytes(), TotalQn: &bi.TotalQn, Height: &bi.Height, PreHash: bi.PreHash.Bytes()}
	return proto.Marshal(&blockInfo)
}

func (bs *blockSyncer) unMarshalTopBlockInfo(b []byte) (*TopBlockInfo, error) {
	message := new(tas_middleware_pb.TopBlockInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		bs.logger.Errorf("unMarshalBlockInfo error:%s", e.Error())
		return nil, e
	}
	blockInfo := TopBlockInfo{Hash: common.BytesToHash(message.Hash), TotalQn: *message.TotalQn, Height: *message.Height, PreHash: common.BytesToHash(message.PreHash)}
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
