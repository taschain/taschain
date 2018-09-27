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
	ReqTotalQnCh chan string
	TotalQnCh    chan TotalQnInfo

	maxTotalQn     uint64
	bestNode       string
	bestNodeHeight uint64
	lock           sync.Mutex

	init             bool
	syncedFirstBlock bool
	replyCount       int
}

func InitBlockSyncer(isLightMiner bool) {
	lightMiner = isLightMiner
	if logger == nil {
		logger = taslog.GetLoggerByName("sync" + common.GlobalConf.GetString("instance", "index", ""))
	}
	BlockSyncer = blockSyncer{maxTotalQn: 0, ReqTotalQnCh: make(chan string), TotalQnCh: make(chan TotalQnInfo), replyCount: 0, init: false, syncedFirstBlock: false}
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
	for {
		requestBlockChainTotalQn()
		t := time.NewTimer(INIT_INTERVAL)
		<-t.C
		if bs.replyCount > 0 {
			logger.Debug("[BlockSyncer]Detect node and start sync block...")
			break
		}
	}
	bs.sync()
}

func (bs *blockSyncer) sync() {
	requestBlockChainTotalQn()
	t := time.NewTimer(BLOCK_TOTAL_QN_RECEIVE_INTERVAL)

	<-t.C
	if nil == BlockChainImpl {
		return
	}
	var currentHash common.Hash
	topBlock := BlockChainImpl.QueryTopBlock()
	if topBlock != nil {
		currentHash = topBlock.Hash
	}
	localTotalQN, localHeight := BlockChainImpl.TotalQN(), BlockChainImpl.Height()
	bs.lock.Lock()
	maxTotalQN := bs.maxTotalQn
	bestNodeId := bs.bestNode
	bestNodeHeight := bs.bestNodeHeight
	bs.lock.Unlock()
	if maxTotalQN <= localTotalQN {
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
		if lightMiner && !bs.syncedFirstBlock {
			var height uint64 = 0
			if bestNodeHeight > 101 {
				height = bestNodeHeight - 100
			}
			RequestBlockInfoByHeight(bestNodeId, height, common.Hash{}, false)
			return
		}
		RequestBlockInfoByHeight(bestNodeId, localHeight, currentHash, true)
	}

}

func (bs *blockSyncer) loop() {
	t := time.NewTicker(BLOCK_SYNC_INTERVAL)
	for {
		select {
		case sourceId := <-bs.ReqTotalQnCh:
			if lightMiner {
				continue
			}
			logger.Debugf("[BlockSyncer] Rcv total qn req from:%s", sourceId)
			if nil == BlockChainImpl {
				panic("core.BlockChainImpl can not be nil!")
			}
			sendBlockTotalQn(sourceId, BlockChainImpl.TotalQN(), BlockChainImpl.Height())
		case h := <-bs.TotalQnCh:
			logger.Debugf("[BlockSyncer] Rcv total qn from:%s,totalQN:%d", h.SourceId, h.TotalQn)
			if !bs.init {
				bs.replyCount++
			}
			bs.lock.Lock()
			if h.TotalQn > bs.maxTotalQn {
				bs.maxTotalQn = h.TotalQn
				bs.bestNode = h.SourceId
				bs.bestNodeHeight = h.Height
			}
			bs.lock.Unlock()
		case <-t.C:
			if !bs.init {
				continue
			}
			logger.Debugf("[BlockSyncer]sync time up, start to block sync!")
			go bs.sync()
		}
	}
}

//广播索要链的QN值
func requestBlockChainTotalQn() {
	logger.Debugf("[BlockSyncer]Req block total qn for neighbor!")
	message := network.Message{Code: network.ReqBlockChainTotalQnMsg}
	network.GetNetInstance().TransmitToNeighbor(message)
}

//返回自身链QN值
func sendBlockTotalQn(targetId string, localTotalQN uint64, height uint64) {
	logger.Debugf("[BlockSyncer]Send local total qn %d to %s!", localTotalQN, targetId)
	body, e := marshalTotalQnInfo(TotalQnInfo{TotalQn: localTotalQN, Height: height})
	if e != nil {
		logger.Errorf("[BlockSyncer]marshal TotalQnInfo error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.BlockChainTotalQnMsg, Body: body}
	network.GetNetInstance().Send(targetId, message)
}

func marshalTotalQnInfo(totalQnInfo TotalQnInfo) ([]byte, error) {
	t := tas_middleware_pb.TotalQnInfo{TotalQn: &totalQnInfo.TotalQn, Height: &totalQnInfo.Height}
	return proto.Marshal(&t)
}
