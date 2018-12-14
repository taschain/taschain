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
	"github.com/ethereum/go-ethereum/common/math"
)

const (
	blockSyncInterval          = 3 * time.Second
	blockSyncCandidatePoolSize = 3
	blockSyncReqTimeout        = 3 * time.Second
)

type blockSyncer struct {
	syncing       bool
	candidate     string
	candidatePool map[string]TopBlockInfo
	lock          sync.Mutex

	init                 bool
	reqTimeoutTimer      *time.Timer
	syncTimer            *time.Timer
	blockInfoNotifyTimer *time.Timer
	logger               taslog.Logger
}

type TopBlockInfo struct {
	TotalQn uint64
	Hash    common.Hash
	Height  uint64
	PreHash common.Hash
}

func InitBlockSyncer() *blockSyncer {
	syncer := &blockSyncer{, init: false, lightMiner: isLightMiner}
	syncer.logger = taslog.GetLoggerByIndex(taslog.BlockSyncLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	syncer.syncTimer = time.NewTimer(blockSyncInterval)
	syncer.blockInfoNotifyTimer = time.NewTimer(blockSyncInterval)
	notify.BUS.Subscribe(notify.BlockInfoNotify, syncer.blockInfoHandler)
	go syncer.loop()
	return syncer
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
				topBlockInfo := TopBlockInfo{Hash: topBlock.Hash, TotalQn: topBlock.TotalQN, Height: topBlock.Height, PreHash: topBlock.PreHash}
				go bs.sendTopBlockInfoToNeighbor(topBlockInfo)
			}
		case <-bs.syncTimer.C:
			bs.logger.Debugf("block sync time up! sync")
			go bs.trySync()
		case <-bs.reqTimeoutTimer.C:
			bs.lock.Lock()
			bs.syncing = false
			bs.candidate = ""
			//todo 加入黑名单
			bs.lock.Unlock()
		}
	}
}

func (bs *blockSyncer) sendTopBlockInfoToNeighbor(bi TopBlockInfo) {
	bs.lock.Lock()
	bs.blockInfoNotifyTimer.Reset(blockSyncInterval)
	bs.lock.Unlock()
	if bi.Height == 0 {
		return
	}
	bs.logger.Debugf("Send local total qn %d to neighbor!", bi.TotalQn)
	body, e := marshalBlockInfo(bi)
	if e != nil {
		bs.logger.Errorf("marshal blockInfo error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.BlockInfoNotifyMsg, Body: body}
	network.GetNetInstance().TransmitToNeighbor(message)
}

func (bs *blockSyncer) blockInfoHandler(msg notify.Message) {
	bnm, ok := msg.GetData().(*notify.BlockInfoNotifyMessage)
	if !ok {
		Logger.Errorf("BlockInfoNotifyMessage GetData assert not ok!")
		return
	}
	blockInfo, e := bs.unMarshalTopBlockInfo(bnm.BlockInfo)
	if e != nil {
		Logger.Errorf("Discard BlockInfoNotifyMessage because of unmarshal error:%s", e.Error())
		return
	}

	bs.logger.Debugf("Rcv total qn from:%s,totalQN:%d,height:%d", bnm.Peer, blockInfo.TotalQn, blockInfo.Height)

	source := bnm.Peer
	topBlock := BlockChainImpl.QueryTopBlock()
	localTotalQn, localTopHash := topBlock.TotalQN, topBlock.Hash
	if !bs.isUsefulCandidate(localTotalQn, localTopHash, blockInfo.TotalQn, blockInfo.Hash) {
		return
	}
	bs.addCandidatePool(source, *blockInfo)
}

func (bs *blockSyncer) trySync() {
	bs.lock.Lock()
	bs.syncTimer.Reset(blockSyncInterval)
	if bs.syncing {
		return
	}
	id, height := bs.getCandidateForSync()
	if id == "" {
		return
	}
	bs.syncing = true
	bs.candidate = id
	bs.reqTimeoutTimer.Reset(blockSyncReqTimeout)
	bs.lock.Unlock()
	RequestBlock(id, height)
}

func (bs *blockSyncer) getCandidateForSync() (string, uint64) {
	topBlock := BlockChainImpl.QueryTopBlock()
	localTotalQN, localTopHash, localHeight := topBlock.TotalQN, topBlock.Hash, topBlock.Height
	uselessCandidate := make([]string, 0, blockSyncCandidatePoolSize)
	for id, topBlockInfo := range bs.candidatePool {
		if !bs.isUsefulCandidate(localTotalQN, localTopHash, topBlockInfo.TotalQn, topBlockInfo.Hash) {
			uselessCandidate = append(uselessCandidate, id)
		}
	}
	if len(uselessCandidate) != 0 {
		for _, id := range uselessCandidate {
			delete(bs.candidatePool, id)
		}
	}
	candidateId := ""
	var candidateMaxTotalQn uint64 = 0
	var candidateHeight uint64 = 0
	for id, topBlockInfo := range bs.candidatePool {
		if topBlockInfo.TotalQn > candidateMaxTotalQn {
			candidateId = id
			candidateMaxTotalQn = topBlockInfo.TotalQn
			candidateHeight = topBlockInfo.Height
		}
	}
	if localHeight > candidateHeight {
		return candidateId, candidateHeight
	}
	return candidateId, localHeight
}

func (bs *blockSyncer) addCandidatePool(id string, topBlockInfo TopBlockInfo) {
	bs.lock.Lock()
	defer bs.lock.Unlock()
	if len(bs.candidatePool) < blockSyncCandidatePoolSize {
		bs.candidatePool[id] = topBlockInfo
		return
	}
	totalQnMinId := ""
	var minTotalQn uint64 = math.MaxUint64
	for id, tbi := range bs.candidatePool {
		if tbi.TotalQn <= minTotalQn {
			totalQnMinId = id
			minTotalQn = tbi.TotalQn
		}
	}
	if topBlockInfo.TotalQn > minTotalQn {
		delete(bs.candidatePool, totalQnMinId)
		bs.candidatePool[id] = topBlockInfo
		if !bs.syncing {
			go bs.trySync()
		}
	}
}

func (bs *blockSyncer) isUsefulCandidate(localTotalQn uint64, localTopHash common.Hash, candidateToltalQn uint64, candidateTopHash common.Hash) bool {
	if candidateToltalQn < localTotalQn || (localTotalQn == candidateToltalQn && localTopHash == candidateTopHash) {
		return false
	}
	return true
}

func marshalBlockInfo(bi TopBlockInfo) ([]byte, error) {
	blockInfo := tas_middleware_pb.BlockInfo{Hash: bi.Hash.Bytes(), TotalQn: &bi.TotalQn, Height: &bi.Height, PreHash: bi.PreHash.Bytes()}
	return proto.Marshal(&blockInfo)
}

func (bs *blockSyncer) unMarshalTopBlockInfo(b []byte) (*TopBlockInfo, error) {
	message := new(tas_middleware_pb.BlockInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		bs.logger.Errorf("unMarshalBlockInfo error:%s", e.Error())
		return nil, e
	}
	blockInfo := TopBlockInfo{Hash: common.BytesToHash(message.Hash), TotalQn: *message.TotalQn, Height: *message.Height, PreHash: common.BytesToHash(message.PreHash)}
	return &blockInfo, nil
}
