package p2p

import (
	"time"
	"core"
	"taslog"
	"sync"
)

//todo  用于填充  将来改为班德的签名
type sig struct {
}

//-----------------------------------------------------回调函数定义-----------------------------------------------------

//其他节点获取本地链高度 todo 签名
type getBlockChainHeightFn func(sd sig) (int, error)

//自身请求
type getLocalBlockChainHeightFn func() (int, error)

//根据高度获取对应的block
type queryBlocksByHeightFn func(hs []int, sd sig) (map[int]core.Block, error)

//同步多个区块到链上
type addBlocksToChainFn func(hbm map[int]core.Block, sd sig)

//---------------------------------------------------------------------------------------------------------------------

const (
	HEIGHT_RECEIVE_INTERVAL = 30 * time.Second

	BLOCK_SYNC_INTERVAL = 60 * time.Second
)

var BlockSyncer blockSyncer

type BlockHeightRequest struct {
	Sig      sig
	SourceId string
}

type BlockHeight struct {
	Height   int
	SourceId string
	Sig      sig
}

type BlockRequest struct {
	HeightSlice []int
	SourceId    string
	Sig         sig
}

type BlockArrived struct {
	BlockMap map[int]core.Block
	Sig      sig
}

type blockSyncer struct {
	neighborMaxHeight int        //邻居节点的最大高度
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
}

func (bs *blockSyncer) Start() {
	bs.syncBlock()
	t := time.NewTicker(BLOCK_SYNC_INTERVAL)
	for {
		select {
		case hr := <-bs.HeightRequestCh:
			//收到块高度请求
			//todo  验证签名
			height, e := bs.getHeight(hr.Sig)
			if e != nil {
				taslog.P2pLogger.Errorf("%s get block height error:%s\n", hr.SourceId, e.Error())
				return
			}
			sendHeight(hr.SourceId, height)
		case h := <-bs.HeightCh:
			//收到来自其他节点的块链高度
			//todo  验证签名
			bs.maxHeightLock.Lock()
			if h.Height > bs.neighborMaxHeight {
				bs.neighborMaxHeight = h.Height
				bs.bestNodeId = h.SourceId
			}
			bs.maxHeightLock.Unlock()
		case br := <-bs.BlockRequestCh:
			//收到块请求
			blocks, e := bs.queryBlock(br.HeightSlice, br.Sig)
			if e != nil {
				taslog.P2pLogger.Errorf("%s query block error:%s\n", br.SourceId, e.Error())
				return
			}
			sendBlocks(br.SourceId, blocks)
		case bm := <-bs.BlockArrivedCh:
			//收到块信息
			bs.addBlocks(bm.BlockMap, bm.Sig)
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
	t := time.NewTimer(HEIGHT_RECEIVE_INTERVAL)

	<-t.C
	localHeight, e := bs.getLocalHeight()
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
		heightSlice := make([]int, maxHeight-localHeight)
		for i := localHeight; i <= maxHeight; i++ {
			heightSlice = append(heightSlice, i)
		}
		requestBlockByHeight(bestNodeId, heightSlice)
	}

}

//广播索要链高度
//todo param: signData
func requestBlockChainHeight() {}

//todo 签名
func sendHeight(targetId string, height int) {}

//todo 签名
func sendBlocks(targetId string, blockMap map[int]core.Block) {}

//向某一节点请求Block
//param: target peer id
//       block height slice
//       sign data
//todo 签名
func requestBlockByHeight(id string, hs []int) {}
