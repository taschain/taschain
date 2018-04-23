package p2p

import (
	"time"
	"core"
	"taslog"
	"sync"
	"common"
)



//-----------------------------------------------------回调函数定义-----------------------------------------------------

//其他节点获取本地块链高度
type getBlockChainHeightFn func() (int,error)

//自身请求
type getLocalBlockChainHeightFn func() (uint64, common.Hash, error)

//根据高度获取对应的block todo 返回结构体 code  []*Block  []hash []ratio
type queryBlocksByHeightFn func(localHeight uint64,currentHash []common.Hash) (map[int]core.Block, error)

//todo 入参 换成上面，结构体
type addBlocksToChainFn func(hbm map[int]core.Block)


//---------------------------------------------------------------------------------------------------------------------

const (
	BLOCK_HEIGHT_RECEIVE_INTERVAL = 30 * time.Second

	BLOCK_SYNC_INTERVAL = 60 * time.Second
)

var BlockSyncer blockSyncer

type BlockHeightRequest struct {
	Sig      []byte
	SourceId string
}

type BlockHeight struct {
	Height   uint64
	SourceId string
	Sig      []byte
}

type BlockRequest struct {
	sourceHeight uint64
	sourceCurrentHash   common.Hash
	SourceId    string
	Sig         []byte
}

type BlockArrived struct {
	BlockMap map[int]core.Block
	Sig      []byte
}

type blockSyncer struct {
	neighborMaxHeight uint64        //邻居节点的最大高度
	bestNodeId        string     //最佳同步节点
	maxHeightLock     sync.Mutex //同步锁

	HeightRequestCh chan BlockHeightRequest
	HeightCh        chan BlockHeight
	BlockRequestCh  chan BlockRequest
	BlockArrivedCh  chan BlockArrived

	getHeight      getBlockChainHeightFn
	getLocalHeight getLocalBlockChainHeightFn
	queryBlock     queryBlocksByHeightFn
	addBlocks      addBlocksToChainFn
}

func InitBlockSyncer(getHeight getBlockChainHeightFn, getLocalHeight getLocalBlockChainHeightFn, queryBlock queryBlocksByHeightFn,
	addBlocks addBlocksToChainFn) {

	BlockSyncer = blockSyncer{HeightRequestCh: make(chan BlockHeightRequest), HeightCh: make(chan BlockHeight), BlockRequestCh: make(chan BlockRequest),
		BlockArrivedCh: make(chan BlockArrived), getHeight: getHeight, getLocalHeight: getLocalHeight, queryBlock: queryBlock, addBlocks: addBlocks,
	}
	BlockSyncer.start()
}

func (bs *blockSyncer) start() {
	bs.syncBlock()
	t := time.NewTicker(BLOCK_SYNC_INTERVAL)
	for {
		select {
		case hr := <-bs.HeightRequestCh:
			//收到块高度请求
			height, e := bs.getHeight()
			if e != nil {
				taslog.P2pLogger.Errorf("%s get block height error:%s\n", hr.SourceId, e.Error())
				return
			}
			sendBlockHeight(hr.SourceId, height)
		case h := <-bs.HeightCh:
			//收到来自其他节点的块链高度
			bs.maxHeightLock.Lock()
			if h.Height > bs.neighborMaxHeight {
				bs.neighborMaxHeight = h.Height
				bs.bestNodeId = h.SourceId
			}
			bs.maxHeightLock.Unlock()
		case br := <-bs.BlockRequestCh:
			//收到块请求
			//blocks, e := bs.queryBlock(br.HeightSlice, br.Sig)
			//if e != nil {
			//	taslog.P2pLogger.Errorf("%s query block error:%s\n", br.SourceId, e.Error())
			//	return
			//}
			var blocks []*core.Block
			sendBlocks(br.SourceId, blocks)
		case bm := <-bs.BlockArrivedCh:
			//收到块信息
			bs.addBlocks(bm.BlockMap)
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
	localHeight,currentHash, e := bs.getLocalHeight()
	if e != nil {
		taslog.P2pLogger.Errorf("Self get block height error:%s\n", e.Error())
		return
	}
	bs.maxHeightLock.Lock()
	maxHeight := bs.neighborMaxHeight
	bestNodeId := bs.bestNodeId
	bs.maxHeightLock.Unlock()
	if maxHeight <= localHeight {
		taslog.P2pLogger.Info("Neightbor max block height %d is less than self block height %d don't sync!\n", maxHeight, localHeight)
		return
	} else {
		taslog.P2pLogger.Info("Neightbor max block height %d is greater than self block height %d.Sync from %s!\n", maxHeight, localHeight, bestNodeId)
		requestBlockByHeight(bestNodeId, localHeight,currentHash)
	}

}

//广播索要链高度
func requestBlockChainHeight() {
}

func sendBlockHeight(targetId string, localHeight int) {}

func sendBlocks(targetId string, blocks []*core.Block) {}

//向某一节点请求Block
//param: target peer id
//       block height slice
//       sign data
func requestBlockByHeight(id string, localHeight uint64, currentHash common.Hash) {}
