package p2p

import (
	"core"
	"sync"
	"time"
	"taslog"
)

//-----------------------------------------------------回调函数定义-----------------------------------------------------
//获取本地组链高度
type getGroupChainHeightFn func(sig []byte) (int, error)


type getLocalGroupChainHeightFn func() (int, error)

//根据高度获取对应的组信息
type queryGroupInfoByHeightFn func(hs []int,sig []byte) (map[int]core.Group, error)

//同步多组到链上
type addGroupInfoToChainFn func(hbm map[int]core.Group,sig []byte)



//---------------------------------------------------------------------------------------------------------------------

var GroupSyncer groupSyncer

type GroupHeightRequest struct {
	Sig      []byte
	SourceId string
}

type GroupHeight struct {
	Height   int
	SourceId string
	Sig      []byte
}

type GroupRequest struct {
	HeightSlice []int
	SourceId    string
	Sig         []byte
}

type GroupArrived struct {
	GroupMap map[int]core.Group
	Sig      []byte
}

type groupSyncer struct {
	neighborMaxHeight int        //邻居节点的最大高度
	bestNodeId        string     //最佳同步节点
	maxHeightLock     sync.Mutex //同步锁

	HeightRequestCh chan GroupHeightRequest
	HeightCh        chan GroupHeight
	GroupRequestCh  chan GroupRequest
	GroupArrivedCh  chan GroupArrived

	getHeight      getGroupChainHeightFn
	getLocalHeight getLocalGroupChainHeightFn
	queryGroup    queryGroupInfoByHeightFn
	addGroups      addGroupInfoToChainFn
}

func InitGroupSyncer(getHeight getGroupChainHeightFn, getLocalHeight getLocalGroupChainHeightFn, queryGroup queryGroupInfoByHeightFn,
	addGroups addGroupInfoToChainFn) {

	GroupSyncer = groupSyncer{HeightRequestCh: make(chan GroupHeightRequest), HeightCh: make(chan GroupHeight), GroupRequestCh: make(chan GroupRequest),
		GroupArrivedCh: make(chan GroupArrived), getHeight: getHeight, getLocalHeight: getLocalHeight, queryGroup: queryGroup, addGroups: addGroups,
	}
}


func (gs *groupSyncer) Start() {
	gs.syncGroup()
	t := time.NewTicker(BLOCK_SYNC_INTERVAL)
	for {
		select {
		case hr := <-gs.HeightRequestCh:
			//收到块高度请求
			//todo  验证签名
			height, e := gs.getHeight(hr.Sig)
			if e != nil {
				taslog.P2pLogger.Errorf("%s get block height error:%s\n", hr.SourceId, e.Error())
				return
			}
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
			groups, e := gs.queryGroup(br.HeightSlice, br.Sig)
			if e != nil {
				taslog.P2pLogger.Errorf("%s query block error:%s\n", br.SourceId, e.Error())
				return
			}
			sendGroups(br.SourceId, groups)
		case bm := <-gs.GroupArrivedCh:
			//收到块信息
			gs.addGroups(bm.GroupMap, bm.Sig)
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

	go requestBlockChainHeight()
	t := time.NewTimer(BLOCK_HEIGHT_RECEIVE_INTERVAL)

	<-t.C
	localHeight, e := gs.getLocalHeight()
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
		heightSlice := make([]int, maxHeight-localHeight)
		for i := localHeight; i <= maxHeight; i++ {
			heightSlice = append(heightSlice, i)
		}
		requestBlockByHeight(bestNodeId, heightSlice)
	}

}

//广播索要链高度 todo 签名
func requestGroupChainHeight() {
}

//todo 签名
func sendGroupHeight(targetId string, localHeight int) {}
//todo 签名
func sendGroups(targetId string, groupMap map[int]core.Group) {}

//向某一节点请求Block
//param: target peer id
//       block height slice
//       sign data
//todo 签名
func requestGroupByHeight(id string, hs []int) {}
