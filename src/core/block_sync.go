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
	"middleware/pb"
	"github.com/gogo/protobuf/proto"
)

const (
	BLOCK_TOTAL_QN_RECEIVE_INTERVAL = 1 * time.Second

	BLOCK_SYNC_INTERVAL = 3 * time.Second

	INIT_INTERVAL = 3 * time.Second
)

var BlockSyncer blockSyncer

var lightMiner bool

type TotalQnInfo struct {
	TotalQn  uint64
	Height   uint64
	SourceId string
}

type blockSyncer struct {
	TotalQnCh    chan TotalQnInfo

	maxTotalQn     uint64
	bestNode       string
	bestNodeHeight uint64
	lock           sync.Mutex

	init             bool
	syncedFirstBlock bool
	hasNeighbor      bool
}

func InitBlockSyncer(isLightMiner bool) {
	lightMiner = isLightMiner
	if logger == nil {
		logger = taslog.GetLoggerByName("sync" + common.GlobalConf.GetString("instance", "index", ""))
	}
	BlockSyncer = blockSyncer{maxTotalQn: 0,  TotalQnCh: make(chan TotalQnInfo), hasNeighbor: false, init: false, syncedFirstBlock: false}
	go BlockSyncer.start()
}

func (bs *blockSyncer) IsInit() bool {
	return bs.init
}

func (bs *blockSyncer) SetInit(init bool) {
	bs.init = init
}

func (bs *blockSyncer) IsSyncedFirstBlock() bool {
	return bs.syncedFirstBlock
}

func (bs *blockSyncer) SetSyncedFirstBlock(syncedFirstBlock bool) {
	bs.syncedFirstBlock = syncedFirstBlock
}


func (bs *blockSyncer) start() {
	go bs.loop()
	logger.Debug("[BlockSyncer]Wait for connecting...")
	detectConnTicker := time.NewTicker(INIT_INTERVAL)
	for {
		<-detectConnTicker.C
		if bs.hasNeighbor {
			logger.Debug("[BlockSyncer]Detect node and start sync block...")
			break
		}
	}
	bs.sync()
}

func (bs *blockSyncer) sync() {
	if nil == BlockChainImpl {
		panic("BlockChainImpl should not be nil!")
		return
	}
	localTotalQN, localHeight := BlockChainImpl.TotalQN(), BlockChainImpl.Height()
	bs.lock.Lock()
	maxTotalQN := bs.maxTotalQn
	bestNodeId := bs.bestNode
	bestNodeHeight := bs.bestNodeHeight

	bs.maxTotalQn = 0
	bs.bestNode = ""
	bs.bestNodeHeight = 0
	bs.lock.Unlock()

	if maxTotalQN-localTotalQN <= 0 {
		logger.Debugf("[BlockSyncer]Neighbor chain's max totalQN: %d,is less than self chain's totalQN: %d.\nDon't sync!", maxTotalQN, localTotalQN)
		if !bs.init {
			bs.init = true
		}
		return
	} else {
		logger.Debugf("[BlockSyncer]Neighbor chain's max totalQN: %d is greater than self chain's totalQN: %d.\nSync from %s!", maxTotalQN, localTotalQN, bestNodeId)
		if BlockChainImpl.IsAdujsting() {
			logger.Debugf("[BlockSyncer]Local chain is adujsting, don't sync")
			return
		}
		if lightMiner && BlockChainImpl.Height() == 0 {
			var height uint64 = 0
			if bestNodeHeight > 11 {
				height = bestNodeHeight - 10
			}
			RequestBlock(bestNodeId, height)
			return
		}
		RequestBlock(bestNodeId, localHeight+1)
	}

}

func (bs *blockSyncer) loop() {
	t := time.NewTicker(BLOCK_SYNC_INTERVAL)
	for {
		select {
		case h := <-bs.TotalQnCh:
			logger.Debugf("[BlockSyncer] Rcv total qn from:%s,totalQN:%d", h.SourceId, h.TotalQn)
			if !bs.hasNeighbor {
				bs.hasNeighbor = true
			}
			bs.lock.Lock()
			if h.TotalQn-bs.maxTotalQn > 0 {
				bs.maxTotalQn = h.TotalQn
				bs.bestNode = h.SourceId
				bs.bestNodeHeight = h.Height
			}
			bs.lock.Unlock()
		case <-t.C:
			if BlockChainImpl.TotalQN() > 0 {
				sendBlockTotalQnToNeighbor(BlockChainImpl.TotalQN(), BlockChainImpl.Height())
			}
			if !bs.init {
				continue
			}
			logger.Debugf("[BlockSyncer]sync time up, start to block sync!")
			go bs.sync()
		}
	}
}

func sendBlockTotalQnToNeighbor(localTotalQN uint64, height uint64) {
	logger.Debugf("[BlockSyncer]Send local total qn %d to neighbor!", localTotalQN)
	body, e := marshalTotalQnInfo(TotalQnInfo{TotalQn: localTotalQN, Height: height})
	if e != nil {
		logger.Errorf("[BlockSyncer]marshal TotalQnInfo error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.BlockChainTotalQnMsg, Body: body}
	network.GetNetInstance().TransmitToNeighbor(message)
}

func marshalTotalQnInfo(totalQnInfo TotalQnInfo) ([]byte, error) {
	t := tas_middleware_pb.TotalQnInfo{TotalQn: &totalQnInfo.TotalQn, Height: &totalQnInfo.Height}
	return proto.Marshal(&t)
}
