package sync

import (
	"time"
	"core"
	"sync"
	"utility"
	"taslog"
	"common"
	"fmt"
)

const (
	BLOCK_TOTAL_QN_RECEIVE_INTERVAL = 1 * time.Second

	BLOCK_SYNC_INTERVAL = 3 * time.Second
)

var BlockSyncer blockSyncer

type TotalQnInfo struct {
	TotalQn  uint64
	SourceId string
}

type blockSyncer struct {
	neighborMaxTotalQn uint64     //邻居节点的最大高度
	bestNodeId         string     //最佳同步节点
	maxTotalQnLock     sync.Mutex //同步锁

	TotalQnRequestCh chan string
	TotalQnCh        chan TotalQnInfo

	rcvTotalQnCount int
}

func InitBlockSyncer() {
	if logger == nil {
		logger = taslog.GetLoggerByName("sync" + common.GlobalConf.GetString("client", "index", ""))
	}
	BlockSyncer = blockSyncer{neighborMaxTotalQn: 0, TotalQnRequestCh: make(chan string), TotalQnCh: make(chan TotalQnInfo), rcvTotalQnCount: 0}
	go BlockSyncer.syncBlock(true)
	go BlockSyncer.start()
}

func (bs *blockSyncer) start() {
	t := time.NewTicker(BLOCK_SYNC_INTERVAL)
	for {
		select {
		case sourceId := <-bs.TotalQnRequestCh:
			//收到块高度请求
			logger.Debugf("[BlockSyncer] TotalQnRequestCh get message from:%s", sourceId)
			if nil == core.BlockChainImpl {
				return
			}
			sendBlockTotalQn(sourceId, core.BlockChainImpl.TotalQN())
		case h := <-bs.TotalQnCh:
			//收到来自其他节点的块链高度
			if !core.BlockChainImpl.IsBlockSyncInit() {
				bs.rcvTotalQnCount = bs.rcvTotalQnCount + 1
			}
			logger.Debugf("[BlockSyncer] TotalQnCh get message from:%s,it's totalQN is:%d", h.SourceId, h.TotalQn)
			bs.maxTotalQnLock.Lock()
			if h.TotalQn > bs.neighborMaxTotalQn {
				bs.neighborMaxTotalQn = h.TotalQn
				bs.bestNodeId = h.SourceId
			}
			bs.maxTotalQnLock.Unlock()
		case <-t.C:
			if !core.BlockChainImpl.IsBlockSyncInit() {
				continue
			}
			logger.Debugf("[BlockSyncer]sync time up, start to block sync!")
			bs.syncBlock(false)
		}
	}
}

func (bs *blockSyncer) syncBlock(init bool) {
	if init {
		for {
			requestBlockChainTotalQn()
			t := time.NewTimer(BLOCK_TOTAL_QN_RECEIVE_INTERVAL)
			<-t.C
			if bs.rcvTotalQnCount > 0 {
				break
			}
		}
	} else {
		requestBlockChainTotalQn()
		t := time.NewTimer(BLOCK_TOTAL_QN_RECEIVE_INTERVAL)

		<-t.C
	}
	//获取本地块链高度
	if nil == core.BlockChainImpl {
		return
	}
	localTotalQN, localHeight, currentHash := core.BlockChainImpl.TotalQN(), core.BlockChainImpl.Height(), core.BlockChainImpl.QueryTopBlock().Hash
	bs.maxTotalQnLock.Lock()
	maxTotalQN := bs.neighborMaxTotalQn
	bestNodeId := bs.bestNodeId
	bs.maxTotalQnLock.Unlock()
	if maxTotalQN <= localTotalQN {
		logger.Debugf("[BlockSyncer]Neighbor chain's max totalQN: %d,is less than self chain's totalQN: %d.\nDon't sync!", maxTotalQN, localTotalQN)
		if !core.BlockChainImpl.IsBlockSyncInit() {
			fmt.Printf("maxTotalQN <= localTotalQN set block sync true,local block height:%d\n", core.BlockChainImpl.Height())
			core.BlockChainImpl.SetBlockSyncInit(true)
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

//广播索要链的QN值
func requestBlockChainTotalQn() {
	message := network.Message{Code: network.REQ_BLOCK_CHAIN_TOTAL_QN_MSG}
	network.Network.Broadcast(message)
}

//返回自身链QN值
func sendBlockTotalQn(targetId string, localTotalQN uint64) {
	body := utility.UInt64ToByte(localTotalQN)
	message := network.Message{Code: network.BLOCK_CHAIN_TOTAL_QN_MSG, Body: body}
	network.Network.Send(targetId,message)
}
