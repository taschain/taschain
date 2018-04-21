package p2p

import (
	"core"
	"sync"
	"time"
	"taslog"
	"common"
)

//-----------------------------------------------------回调函数定义-----------------------------------------------------
//获取本地组链高度
type getGroupChainHeightFn func() (uint64, error)

type getLocalGroupChainHeightFn func() (uint64, common.Hash, error)

//根据高度获取对应的组信息
// todo 返回结构体 code  []*Block  []hash []ratio
type queryGroupInfoByHeightFn func(localHeight uint64, currentHash common.Hash) (map[int]core.Group, error)

//同步多组到链上
//todo 入参 换成上面，结构体
type addGroupInfoToChainFn func(hbm map[uint64]core.Group)

//---------------------------------------------------------------------------------------------------------------------

var GroupSyncer groupSyncer

type GroupHeightRequest struct {
	Sig      []byte
	SourceId string
}

type GroupHeight struct {
	Height   uint64
	SourceId string
	Sig      []byte
}

type GroupRequest struct {
	sourceHeight      uint64
	sourceCurrentHash common.Hash
	SourceId          string
	Sig               []byte
}

type GroupArrived struct {
	GroupMap map[uint64]core.Group
	Sig      []byte
}

type groupSyncer struct {
	neighborMaxHeight uint64     //邻居节点的最大高度
	bestNodeId        string     //最佳同步节点
	maxHeightLock     sync.Mutex //同步锁

	HeightRequestCh chan GroupHeightRequest
	HeightCh        chan GroupHeight
	GroupRequestCh  chan GroupRequest
	GroupArrivedCh  chan GroupArrived

	getHeight      getGroupChainHeightFn
	getLocalHeight getLocalGroupChainHeightFn
	queryGroup     queryGroupInfoByHeightFn
	addGroups      addGroupInfoToChainFn
}

func InitGroupSyncer(getHeight getGroupChainHeightFn, getLocalHeight getLocalGroupChainHeightFn, queryGroup queryGroupInfoByHeightFn,
	addGroups addGroupInfoToChainFn) {

	GroupSyncer = groupSyncer{HeightRequestCh: make(chan GroupHeightRequest), HeightCh: make(chan GroupHeight), GroupRequestCh: make(chan GroupRequest),
		GroupArrivedCh: make(chan GroupArrived), getHeight: getHeight, getLocalHeight: getLocalHeight, queryGroup: queryGroup, addGroups: addGroups,
	}
	GroupSyncer.start()
}

func (gs *groupSyncer) start() {
	gs.syncGroup()
	t := time.NewTicker(BLOCK_SYNC_INTERVAL)
	for {
		select {
		case hr := <-gs.HeightRequestCh:
			//收到块高度请求
			//todo  验证签名
			height, e := gs.getHeight()
			if e != nil {
				taslog.P2pLogger.Errorf("%s get block height error:%s\n", hr.SourceId, e.Error())
				return
			}
			//todo 签名
			sendGroupHeight(hr.SourceId, height)
		case h := <-gs.HeightCh:
			//收到来自其他节点的块链高度
			//todo  验证签名
			gs.maxHeightLock.Lock()
			if h.Height > gs.neighborMaxHeight {
				gs.neighborMaxHeight = h.Height
				gs.bestNodeId = h.SourceId
			}
			gs.maxHeightLock.Unlock()
		case br := <-gs.GroupRequestCh:
			//收到块请求
			//groups, e := gs.queryGroup(br.sourceHeight,br.sourceCurrentHash, br.Sig)
			//if e != nil {
			//	taslog.P2pLogger.Errorf("%s query block error:%s\n", br.SourceId, e.Error())
			//	return
			//}
			var groups []*core.Group
			//todo 签名
			sendGroups(br.SourceId, groups)
		case bm := <-gs.GroupArrivedCh:
			//收到块信息
			gs.addGroups(bm.GroupMap)
		case <-t.C:
			gs.syncGroup()
		}
	}
}

func (gs *groupSyncer) syncGroup() {
	gs.maxHeightLock.Lock()
	gs.neighborMaxHeight = 0
	gs.bestNodeId = ""
	gs.maxHeightLock.Unlock()

	//todo 签名
	go requestGroupChainHeight()
	t := time.NewTimer(BLOCK_HEIGHT_RECEIVE_INTERVAL)

	<-t.C
	localHeight, currentHash, e := gs.getLocalHeight()
	if e != nil {
		taslog.P2pLogger.Errorf("Self get group height error:%s\n", e.Error())
		return
	}
	gs.maxHeightLock.Lock()
	maxHeight := gs.neighborMaxHeight
	bestNodeId := gs.bestNodeId
	gs.maxHeightLock.Unlock()
	if maxHeight <= localHeight {
		taslog.P2pLogger.Info("Neightbor max group height %d is less than self group height %d don't sync!\n", maxHeight, localHeight)
		return
	} else {
		taslog.P2pLogger.Info("Neightbor max group height %d is greater than self group height %d.Sync from %s!\n", maxHeight, localHeight, bestNodeId)
		//todo 签名
		requestBlockByHeight(bestNodeId, localHeight, currentHash)
	}

}

//广播索要链高度
func requestGroupChainHeight() {
}

func sendGroupHeight(targetId string, localHeight uint64) {}

func sendGroups(targetId string, groups []*core.Group) {}

//向某一节点请求Block
//param: target peer id
//       block height slice
//       sign data
func requestGroupByHeight(id string, localHeight uint64, currentHash common.Hash) {}
