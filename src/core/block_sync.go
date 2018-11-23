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
	"time"
	"sync"
	"taslog"
	"common"
	"network"
	"middleware/notify"
	"middleware/pb"
	"github.com/gogo/protobuf/proto"
)

const (
	blockSyncInterval = 3 * time.Second
)

var BlockSyncer *blockSyncer
var blockSyncLogger taslog.Logger

type blockSyncer struct {
	candidate blockSyncCandidate
	lock      sync.Mutex

	init                 bool
	hasNeighbor          bool
	lightMiner           bool
	syncTimer            *time.Timer
	blockInfoNotifyTimer *time.Timer
}

type blockSyncCandidate struct {
	id      string
	totalQn uint64
	hash    common.Hash
	preHash common.Hash
	height  uint64
}

type BlockInfo struct {
	TotalQn uint64
	Hash    common.Hash
	Height  uint64
	PreHash common.Hash
}

func InitBlockSyncer(isLightMiner bool) {
	blockSyncLogger = taslog.GetLoggerByName("block_sync" + common.GlobalConf.GetString("instance", "index", ""))
	BlockSyncer = &blockSyncer{hasNeighbor: false, init: false, lightMiner: isLightMiner}
	BlockSyncer.syncTimer = time.NewTimer(blockSyncInterval)
	BlockSyncer.blockInfoNotifyTimer = time.NewTimer(blockSyncInterval)
	notify.BUS.Subscribe(notify.BlockInfoNotify, BlockSyncer.blockInfoHandler)
	go BlockSyncer.loop()
}

func (bs *blockSyncer) IsInit() bool {
	return bs.init
}

func (bs *blockSyncer) loop() {
	for {
		select {
		case <-bs.blockInfoNotifyTimer.C:
			if !BlockChainImpl.IsLightMiner() {
				topBlock := BlockChainImpl.QueryTopBlock()
				topBlockInfo := BlockInfo{Hash: topBlock.Hash, TotalQn: topBlock.TotalQN, Height: topBlock.Height, PreHash: topBlock.PreHash}
				go bs.SendTopBlockInfoToNeighbor(topBlockInfo)
			}
		case <-bs.syncTimer.C:
			go bs.sync(nil)
		}
	}
}

func (bs *blockSyncer) SendTopBlockInfoToNeighbor(bi BlockInfo) {
	bs.blockInfoNotifyTimer.Reset(blockSyncInterval)
	blockSyncLogger.Debugf("[BlockSyncer]Send local total qn %d to neighbor!", bi.TotalQn)
	if bi.Height == 0 {
		return
	}
	body, e := marshalBlockInfo(bi)
	if e != nil {
		blockSyncLogger.Errorf("marshal blockInfo error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.BlockInfoNotifyMsg, Body: body}
	network.GetNetInstance().TransmitToNeighbor(message)
}

func (bs *blockSyncer) blockInfoHandler(msg notify.Message) {
	bnm, ok := msg.GetData().(*notify.BlockInfoNotifyMessage)
	if !ok {
		Logger.Debugf("[ChainHandler]BlockInfoNotifyMessage GetData assert not ok!")
		return
	}
	blockInfo, e := unMarshalBlockInfo(bnm.BlockInfo)
	if e != nil {
		Logger.Errorf("[handler]Discard BlockInfoNotifyMessage because of unmarshal error:%s", e.Error())
		return
	}

	blockSyncLogger.Debugf("[BlockSyncer] Rcv total qn from:%s,totalQN:%d,height:%d", bnm.Peer, blockInfo.TotalQn, blockInfo.Height)
	if !bs.hasNeighbor {
		bs.hasNeighbor = true
	}
	candidate := blockSyncCandidate{id: bnm.Peer, totalQn: blockInfo.TotalQn, hash: blockInfo.Hash, preHash: blockInfo.PreHash, height: blockInfo.Height}
	if candidate.totalQn < BlockChainImpl.TotalQN() {
		return
	}
	if candidate.height <= BlockChainImpl.Height()+3 {
		go bs.sync(&candidate)
	}

	if blockInfo.TotalQn > bs.candidate.totalQn {
		bs.lock.Lock()
		bs.candidate = candidate
		bs.lock.Unlock()
		return
	}
	if blockInfo.TotalQn == bs.candidate.totalQn && blockInfo.Hash != bs.candidate.hash && BlockChainImpl.QueryBlockByHash(candidate.hash) == nil {
		bs.lock.Lock()
		bs.candidate = candidate
		bs.lock.Unlock()
	}

}

func (bs *blockSyncer) sync(candidate *blockSyncCandidate) {
	if candidate == nil {
		bs.lock.Lock()
		candidate = &bs.candidate
		bs.lock.Unlock()
	}
	if candidate.id == "" {
		return
	}
	blockSyncLogger.Debugf("Start sync!")
	topBlock := BlockChainImpl.QueryTopBlock()
	localTotalQN, localHash, localPreHash, localHeight := topBlock.TotalQN, topBlock.Hash, topBlock.PreHash, topBlock.Height
	bs.lock.Lock()
	bs.syncTimer.Reset(blockSyncInterval)
	candidateQN, candidateId, candidateHash, candidatePreHash, candidateHeight := bs.candidate.totalQn, bs.candidate.id, bs.candidate.hash, bs.candidate.preHash, bs.candidate.height
	bs.lock.Unlock()

	if candidateQN < localTotalQN || candidateHash == localHash {
		blockSyncLogger.Debugf("[BlockSyncer]Neighbor chain's max totalQN: %d,is less than self chain's totalQN: %d.\nDon't sync!", candidateQN, localTotalQN)
		if !bs.init {
			blockSyncLogger.Info("Block first sync finished!")
			bs.init = true
		}

		if BlockChainImpl.IsAdujsting() {
			BlockChainImpl.SetAdujsting(false)
		}
		return
	}

	blockSyncLogger.Debugf("[Sync]Neighbor Top hash:%v,height:%d,totalQn:%d,pre hash:%v,!", candidateHash.Hex(), candidateHeight, candidateQN, candidatePreHash.Hex())
	blockSyncLogger.Debugf("[Sync]Local Top hash:%v,height:%d,totalQn:%d,pre hash:%v,!", localHash.Hex(), localHeight, localTotalQN, localPreHash.Hex())
	if candidatePreHash == localHash || (candidatePreHash == localPreHash && candidateQN > localTotalQN) {
		RequestBlock(candidateId, candidateHeight)
		return
	}

	if BlockChainImpl.Height() == 0 {
		RequestBlock(candidateId, 1)
		return
	}
	RequestChainPiece(candidateId, localHeight)
}

func marshalBlockInfo(bi BlockInfo) ([]byte, error) {
	blockInfo := tas_middleware_pb.BlockInfo{Hash: bi.Hash.Bytes(), TotalQn: &bi.TotalQn, Height: &bi.Height, PreHash: bi.PreHash.Bytes()}
	return proto.Marshal(&blockInfo)
}

func unMarshalBlockInfo(b []byte) (*BlockInfo, error) {
	message := new(tas_middleware_pb.BlockInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		blockSyncLogger.Errorf("unMarshalBlockInfo error:%s", e.Error())
		return nil, e
	}
	blockInfo := BlockInfo{Hash: common.BytesToHash(message.Hash), TotalQn: *message.TotalQn, Height: *message.Height, PreHash: common.BytesToHash(message.PreHash)}
	return &blockInfo, nil
}
