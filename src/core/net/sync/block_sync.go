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

package sync

import (
	"time"
	"core"
	"sync"
	"taslog"
	"common"
	"network"
	"math/big"
)

const (
	BLOCK_TOTAL_QN_RECEIVE_INTERVAL = 1 * time.Second

	BLOCK_SYNC_INTERVAL = 3 * time.Second

	INIT_INTERVAL = 3 * time.Second
)

var BlockSyncer blockSyncer

var totalQnRcvTimer = time.NewTimer(0)

type TotalQnInfo struct {
	TotalQn  *big.Int
	SourceId string
}

type blockSyncer struct {
	ReqTotalQnCh chan string
	TotalQnCh        chan TotalQnInfo

	maxTotalQn *big.Int
	bestNode         string
	lock     sync.Mutex

	init       bool
	replyCount int
}

func InitBlockSyncer() {
	if logger == nil {
		logger = taslog.GetLoggerByName("sync" + common.GlobalConf.GetString("instance", "index", ""))
	}
	BlockSyncer = blockSyncer{maxTotalQn: big.NewInt(0), ReqTotalQnCh: make(chan string), TotalQnCh: make(chan TotalQnInfo), replyCount: 0}
	go BlockSyncer.loop()
	go BlockSyncer.start()
}

func (bs *blockSyncer) start() {
	logger.Debug("[BlockSyncer]Wait for connecting...")
	t := time.NewTimer(INIT_INTERVAL)
	defer t.Stop()
	for {
		requestBlockChainTotalQn()
		t.Reset(INIT_INTERVAL)
		<-t.C
		if bs.replyCount > 0 {
			logger.Debug("[BlockSyncer]Detect node and start sync block...")
			break
		}
	}
	totalQnRcvTimer.Reset(BLOCK_TOTAL_QN_RECEIVE_INTERVAL)
}

func (bs *blockSyncer) IsInit() bool {
	return bs.init
}

func (bs *blockSyncer) SetInit(init bool){
	bs.init = init
}

func (bs *blockSyncer) sync() {
	if nil == core.BlockChainImpl {
		logger.Error("core.BlockChainImpl is nil!")
		return
	}
	localTotalQN, localHeight, currentHash := core.BlockChainImpl.TotalQN(), core.BlockChainImpl.Height(), core.BlockChainImpl.QueryTopBlock().Hash
	bs.lock.Lock()
	maxTotalQN := bs.maxTotalQn
	bestNodeId := bs.bestNode
	bs.lock.Unlock()
	if maxTotalQN.Cmp(localTotalQN) <= 0 {
		logger.Debugf("[BlockSyncer]Neighbor chain's max totalQN: %d,is less than self chain's totalQN: %d.\nDon't sync!", maxTotalQN, localTotalQN)
		if !bs.init {
			bs.init = true
		}
		return
	} else {
		logger.Debugf("[BlockSyncer]Neighbor chain's max totalQN: %d is greater than self chain's totalQN: %d.\nSync from %s!", maxTotalQN, localTotalQN, bestNodeId)
		if core.BlockChainImpl.IsAdujsting() {
			logger.Debugf("[BlockSyncer]Local chain is adujsting, don't sync")
			return
		}
		core.RequestBlockInfoByHeight(bestNodeId, localHeight, currentHash)
	}

}

func (bs *blockSyncer) loop() {
	syncTicker := time.NewTicker(BLOCK_SYNC_INTERVAL)
	defer syncTicker.Stop()
	defer totalQnRcvTimer.Stop()
	for {
		select {
		case sourceId := <-bs.ReqTotalQnCh:
			logger.Debugf("[BlockSyncer] Rcv total qn req from:%s", sourceId)
			if nil == core.BlockChainImpl {
				return
			}
			sendBlockTotalQn(sourceId, core.BlockChainImpl.TotalQN())
		case h := <-bs.TotalQnCh:
			logger.Debugf("[BlockSyncer] Rcv total qn from:%s,totalQN:%d", h.SourceId, h.TotalQn)
			if !bs.init{
				bs.replyCount++
			}
			bs.lock.Lock()
			if h.TotalQn.Cmp(bs.maxTotalQn) > 0 {
				bs.maxTotalQn = h.TotalQn
				bs.bestNode = h.SourceId
			}
			bs.lock.Unlock()
		case <-syncTicker.C:
			if !bs.init{
				continue
			}
			logger.Debugf("[BlockSyncer]sync time up, start to block sync!")
			requestBlockChainTotalQn()
			totalQnRcvTimer.Reset(BLOCK_TOTAL_QN_RECEIVE_INTERVAL)
		case <-totalQnRcvTimer.C:
			if bs.replyCount <= 0 {
				continue
			}
			bs.sync()
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
func sendBlockTotalQn(targetId string, localTotalQN *big.Int) {
	logger.Debugf("[BlockSyncer]Send local total qn %d to %s!",localTotalQN,targetId)
	//body := utility.UInt64ToByte(localTotalQN)
	message := network.Message{Code: network.BlockChainTotalQnMsg, Body: localTotalQN.Bytes()}
	network.GetNetInstance().Send(targetId,message)
}
