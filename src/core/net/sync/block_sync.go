package sync

import (
	"time"
	"core"
	"sync"
	"utility"
	"network/p2p"
	"taslog"
	"common"
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
}

func InitBlockSyncer() {
	if logger == nil {
		logger = taslog.GetLoggerByName("sync" + common.GlobalConf.GetString("client", "index", ""))
	}
	BlockSyncer = blockSyncer{neighborMaxTotalQn: 0, TotalQnRequestCh: make(chan string), TotalQnCh: make(chan TotalQnInfo),}
	go BlockSyncer.start()
}

func (bs *blockSyncer) start() {
	bs.syncBlock()
	t := time.NewTicker(BLOCK_SYNC_INTERVAL)
	for {
		select {
		case sourceId := <-bs.TotalQnRequestCh:
			//收到块高度请求
			//logger.Debugf("[BlockSyncer] TotalQnRequestCh get message from:%s", sourceId)
			if nil == core.BlockChainImpl {
				return
			}
			sendBlockTotalQn(sourceId, core.BlockChainImpl.TotalQN())
		case h := <-bs.TotalQnCh:
			//收到来自其他节点的块链高度
			//logger.Debugf("[BlockSyncer] TotalQnCh get message from:%s,it's totalQN is:%d", h.SourceId, h.TotalQn)
			bs.maxTotalQnLock.Lock()
			if h.TotalQn > bs.neighborMaxTotalQn {
				bs.neighborMaxTotalQn = h.TotalQn
				bs.bestNodeId = h.SourceId
			}
			bs.maxTotalQnLock.Unlock()
		case <-t.C:
			//logger.Debugf("[BlockSyncer]sync time up, start to block sync!")
			bs.syncBlock()
		}
	}
}

func (bs *blockSyncer) syncBlock() {
	go requestBlockChainTotalQn()
	t := time.NewTimer(BLOCK_TOTAL_QN_RECEIVE_INTERVAL)

	<-t.C
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
		//logger.Debugf("[BlockSyncer]Neighbor chain's max totalQN: %d,is less than self chain's totalQN: %d.\nDon't sync!", maxTotalQN, localTotalQN)
		if !core.BlockChainImpl.IsBlockSyncInit(){
			core.BlockChainImpl.SetBlockSyncInit(true)
		}
		return
	} else {
		//logger.Debugf("[BlockSyncer]Neighbor chain's max totalQN: %d is greater than self chain's totalQN: %d.\nSync from %s!", maxTotalQN, localTotalQN, bestNodeId)
		if core.BlockChainImpl.IsAdujsting() {
			logger.Debugf("[BlockSyncer]Local chain is adujsting, don't sync")
			return
		}
		core.RequestBlockInfoByHeight(bestNodeId, localHeight, currentHash)
	}

}

//广播索要链的QN值
func requestBlockChainTotalQn() {
	message := p2p.Message{Code: p2p.REQ_BLOCK_CHAIN_TOTAL_QN_MSG}
	conns := p2p.Server.Host.Network().Conns()
	for _, conn := range conns {
		id := conn.RemotePeer()
		if id != "" {
			p2p.Server.SendMessage(message, p2p.ConvertToID(id))
		}
	}
}

//返回自身链QN值
func sendBlockTotalQn(targetId string, localTotalQN uint64) {
	body := utility.UInt64ToByte(localTotalQN)
	message := p2p.Message{Code: p2p.BLOCK_CHAIN_TOTAL_QN_MSG, Body: body}
	p2p.Server.SendMessage(message, targetId)
}
