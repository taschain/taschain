package p2p

import (
	"time"
	"core"
	"sync"
	"common"
	"utility"
)

//-----------------------------------------------------回调函数定义-----------------------------------------------------

//其他节点获取本地块链高度
type getBlockChainHeightFn func() (uint64, error)

//自身请求
type getLocalBlockChainHeightFn func() (uint64, common.Hash, error)

//根据高度获取对应的block
type queryBlocksByHeightFn func(localHeight uint64, currentHash common.Hash) (*BlockEntity, error)

type addBlocksToChainFn func(blockEntity *BlockEntity,targetId string)

//---------------------------------------------------------------------------------------------------------------------

const (
	BLOCK_HEIGHT_RECEIVE_INTERVAL = 30 * time.Second

	BLOCK_SYNC_INTERVAL = 60 * time.Second
)

var BlockSyncer blockSyncer

type BlockOrGroupRequestEntity struct {
	SourceHeight      uint64
	SourceCurrentHash common.Hash
}

type BlockEntity struct {
	blocks      []*core.Block

	height      uint64//起始高度，如果返回blocks，那就是BLOCKS的起始高度，如果返回blockHashes那就是HASH的起始高度
	blockHashes []common.Hash
	blockRatios []float64
}

type blockHeight struct {
	height   uint64
	sourceId string
}

type blockRequest struct {
	bre      BlockOrGroupRequestEntity
	sourceId string
}


type blockArrived struct {
	blockEntity BlockEntity
	sourceId string
}
type blockSyncer struct {
	neighborMaxHeight uint64     //邻居节点的最大高度
	bestNodeId        string     //最佳同步节点
	maxHeightLock     sync.Mutex //同步锁

	HeightRequestCh chan string
	HeightCh        chan blockHeight
	BlockRequestCh  chan blockRequest
	BlockArrivedCh  chan blockArrived

	getHeight      getBlockChainHeightFn
	getLocalHeight getLocalBlockChainHeightFn
	queryBlock     queryBlocksByHeightFn
	addBlocks      addBlocksToChainFn
}

func InitBlockSyncer(getHeight getBlockChainHeightFn, getLocalHeight getLocalBlockChainHeightFn, queryBlock queryBlocksByHeightFn,
	addBlocks addBlocksToChainFn) {

	BlockSyncer = blockSyncer{HeightRequestCh: make(chan string), HeightCh: make(chan blockHeight), BlockRequestCh: make(chan blockRequest),
		BlockArrivedCh: make(chan blockArrived), getHeight: getHeight, getLocalHeight: getLocalHeight, queryBlock: queryBlock, addBlocks: addBlocks,
	}
	BlockSyncer.start()
}

func (bs *blockSyncer) start() {
	bs.syncBlock()
	t := time.NewTicker(BLOCK_SYNC_INTERVAL)
	for {
		select {
		case sourceId := <-bs.HeightRequestCh:
			//收到块高度请求
			height, e := bs.getHeight()
			if e != nil {
				logger.Errorf("Get block height rquest from %s error:%s\n", sourceId, e.Error())
				return
			}
			sendBlockHeight(sourceId, height)
		case h := <-bs.HeightCh:
			//收到来自其他节点的块链高度
			bs.maxHeightLock.Lock()
			if h.height > bs.neighborMaxHeight {
				bs.neighborMaxHeight = h.height
				bs.bestNodeId = h.sourceId
			}
			bs.maxHeightLock.Unlock()
		case br := <-bs.BlockRequestCh:
			//收到块请求
			blockEntity, e := bs.queryBlock(br.bre.SourceHeight, br.bre.SourceCurrentHash)
			if e != nil {
				logger.Errorf("query block request from %s error:%s\n", br.sourceId, e.Error())
				return
			}
			sendBlocks(br.sourceId, blockEntity)
		case bm := <-bs.BlockArrivedCh:
			//收到块信息
			bs.addBlocks(&bm.blockEntity,bm.sourceId)
		case <-t.C:
			bs.syncBlock()
		}
	}
}

func (bs *blockSyncer) syncBlock() {
	bs.maxHeightLock.Lock()
	bs.neighborMaxHeight = 0
	bs.bestNodeId = ""
	bs.maxHeightLock.Unlock()

	go requestBlockChainHeight()
	t := time.NewTimer(BLOCK_HEIGHT_RECEIVE_INTERVAL)

	<-t.C
	localHeight, currentHash, e := bs.getLocalHeight()
	if e != nil {
		logger.Errorf("Self get block height error:%s\n", e.Error())
		return
	}
	bs.maxHeightLock.Lock()
	maxHeight := bs.neighborMaxHeight
	bestNodeId := bs.bestNodeId
	bs.maxHeightLock.Unlock()
	if maxHeight <= localHeight {
		logger.Info("Neightbor max block height %d is less than self block height %d don't sync!\n", maxHeight, localHeight)
		return
	} else {
		logger.Info("Neightbor max block height %d is greater than self block height %d.Sync from %s!\n", maxHeight, localHeight, bestNodeId)
		Peer.RequestBlockByHeight(bestNodeId, localHeight, currentHash)
	}

}

//广播索要链高度
func requestBlockChainHeight() {
	message := Message{Code: REQ_BLOCK_CHAIN_HEIGHT_MSG}
	conns := Server.host.Network().Conns()
	for _, conn := range conns {
		id := conn.RemotePeer()
		if id != "" {
			Server.SendMessage(message, string(id))
		}
	}
}

//返回自身链高度
func sendBlockHeight(targetId string, localHeight uint64) {
	body := utility.UInt64ToByte(localHeight)
	message := Message{Code: BLOCK_CHAIN_HEIGHT_MSG, Body: body}
	Server.SendMessage(message, targetId)
}

//本地查询之后将结果返回
func sendBlocks(targetId string, blockEntity *BlockEntity) {
	body,e := MarshalBlockEntity(blockEntity)
	if e!=nil{
		logger.Errorf("sendBlocks marshal BlockEntity error:%s\n",e.Error())
		return
	}
	message := Message{Code: BLOCK_MSG, Body: body}
	Server.SendMessage(message, targetId)
}
