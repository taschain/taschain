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
	"middleware/types"
	"middleware/notify"
)

const (
	BLOCK_SYNC_INTERVAL = 3 * time.Second

	INIT_INTERVAL = 3 * time.Second
)

var BlockSyncer blockSyncer

// 分叉处理只有同步才会触发
// 收到新块后，不会触发分叉处理
type blockSyncer struct {
	candidate   blockSyncCandidate
	lock        sync.Mutex
	init        bool // 是否完成第一次全量同步
	hasNeighbor bool
	lightMiner  bool
}

// 同步目标节点
type blockSyncCandidate struct {
	id      string
	totalQn uint64
	hash    common.Hash
	preHash common.Hash
	height  uint64
}

func InitBlockSyncer(isLightMiner bool) {
	if logger == nil {
		logger = taslog.GetLoggerByName("sync" + common.GlobalConf.GetString("instance", "index", ""))
	}
	BlockSyncer = blockSyncer{hasNeighbor: false, init: false, lightMiner: isLightMiner}
	notify.BUS.Subscribe(notify.BlockChainTotalQn, BlockSyncer.totalQnHandler)
	go BlockSyncer.start()
}

func (bs *blockSyncer) IsInit() bool {
	return bs.init
}

func (bs *blockSyncer) start() {
	logger.Debug("[BlockSyncer]Wait for connecting...")
	go bs.loop()

	detectConnTicker := time.NewTicker(INIT_INTERVAL)
	for {
		<-detectConnTicker.C
		if bs.hasNeighbor {
			logger.Debug("[BlockSyncer]Detect node and start sync block...")
			break
		}
	}
	bs.Sync()
}

func (bs *blockSyncer) Sync() {
	if bs.candidate.id == "" {
		return
	}

	topBlock := BlockChainImpl.QueryTopBlock()
	localTotalQN, localHash, localPreHash, localHeight := topBlock.TotalQN, topBlock.Hash, topBlock.PreHash, topBlock.Height
	bs.lock.Lock()
	candidateQN, candidateId, candidateHash, candidatePreHash, candidateHeight := bs.candidate.totalQn, bs.candidate.id, bs.candidate.hash, bs.candidate.preHash, bs.candidate.height
	bs.lock.Unlock()

	logger.Debugf("sync localHeight:%d", localHeight)

	// 本地的totalqn与候选节点的相等，我们认为无须再处理，即使这两条链不一样
	if candidateQN <= localTotalQN || candidateHash == localHash || candidateHeight == 0 {
		logger.Debugf("[BlockSyncer]Neighbor chain's max totalQN: %d,is less than self chain's totalQN: %d.\nDon't sync!", candidateQN, localTotalQN)
		if !bs.init {
			logger.Info("Block first sync finished!")
			bs.init = true
		}

		if BlockChainImpl.IsAdujsting() {
			BlockChainImpl.SetAdujsting(false)
		}
		return
	}

	logger.Debugf("[Sync]Neighbor Top hash:%v,height:%d,totalQn:%d,pre hash:%v,!", candidateHash.Hex(), candidateHeight, candidateQN, candidatePreHash.Hex())
	logger.Debugf("[Sync]Local Top hash:%v,height:%d,totalQn:%d,pre hash:%v,!", localHash.Hex(), localHeight, localTotalQN, localPreHash.Hex())
	BlockChainImpl.SetAdujsting(true)

	// candidate只比本地领先一块
	if candidatePreHash == localHash {
		RequestBlock(candidateId, candidateHeight)
		return
	}

	// 最高块分叉
	if candidatePreHash == localPreHash && candidateQN > localTotalQN {
		result := BlockChainImpl.Remove(topBlock)
		if result {
			RequestBlock(candidateId, candidateHeight)
		}
		return

	}

	if BlockChainImpl.Height() == 0 {
		RequestBlock(candidateId, 1)
		return
	}
	RequestChainPiece(candidateId, localHeight)
}

func (bs *blockSyncer) loop() {
	t := time.NewTicker(BLOCK_SYNC_INTERVAL)
	for {
		<-t.C
		if !BlockChainImpl.IsLightMiner() {
			sendBlockTotalQnToNeighbor(BlockChainImpl.QueryTopBlock())
		}

		if !bs.init {
			continue
		}
		logger.Debugf("[BlockSyncer]sync time up, start to block sync!")
		go bs.Sync()
	}
}

// 扩散本地的最高块头给邻居节点
func sendBlockTotalQnToNeighbor(topBlockHeader *types.BlockHeader) {
	logger.Debugf("[BlockSyncer]Send local total qn %d to neighbor!", topBlockHeader.TotalQN)
	body, e := types.MarshalBlockHeader(topBlockHeader)
	if e != nil {
		logger.Errorf("[BlockSyncer]marshal TotalQnInfo error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.BlockChainTotalQnMsg, Body: body}
	network.GetNetInstance().TransmitToNeighbor(message)
}

// 收到邻居的块头，比较totalqn，更新candidate状态
func (bs *blockSyncer) totalQnHandler(msg notify.Message) {
	totalQnMsg, ok := msg.GetData().(*notify.TotalQnMessage)
	if !ok {
		Logger.Debugf("[ChainHandler]totalQnHandler GetData assert not ok!")
		return
	}

	// 反序列化
	header, e := types.UnMarshalBlockHeader(totalQnMsg.BlockHeaderByte)
	if e != nil {
		Logger.Errorf("[handler]Discard totalQnHandler because of unmarshal error:%s", e.Error())
		return
	}

	// 校验块头的合法性
	if !bs.verifyTotalQnMsg(header) {
		return
	}

	logger.Debugf("[BlockSyncer] Rcv total qn from:%s,totalQN:%d,height:%d", totalQnMsg.Peer, header.TotalQN, header.Height)
	if !bs.hasNeighbor {
		bs.hasNeighbor = true
	}

	bs.lock.Lock()
	if header.TotalQN > bs.candidate.totalQn {
		bs.candidate = blockSyncCandidate{id: totalQnMsg.Peer, totalQn: header.TotalQN, hash: header.Hash, preHash: header.PreHash, height: header.Height}
	}
	bs.lock.Unlock()
}

// 校验块头
// todo: 迁移到共识相关的工具类
func (bs *blockSyncer) verifyTotalQnMsg(blockHeader *types.BlockHeader) bool {
	//if blockHeader.Hash != blockHeader.GenHash() {
	//	return false
	//}
	//result, err := BlockChainImpl.GetConsensusHelper().VerifyBlockHeader(blockHeader)
	//if err != nil {
	//	Logger.Errorf("verifyTotalQnMsg error:%s", err.Error())
	//}
	//return result
	return true
}
