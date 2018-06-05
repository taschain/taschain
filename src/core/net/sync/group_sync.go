package sync

import (
	"core"
	"sync"
	"time"
	"common"
	"network/p2p"
	"utility"
	"pb"
	"github.com/gogo/protobuf/proto"
	"log"
)

const (
	GROUP_HEIGHT_RECEIVE_INTERVAL = 3 * time.Second

	GROUP_SYNC_INTERVAL = 5 * time.Second
)

var GroupSyncer groupSyncer

type groupSyncer struct {
	neighborMaxHeight uint64     //邻居节点的最大高度
	bestNodeId        string     //最佳同步节点
	maxHeightLock     sync.Mutex //同步锁

	HeightRequestCh chan string
	HeightCh        chan core.EntityHeightMessage
	GroupRequestCh  chan core.EntityRequestMessage
	GroupArrivedCh  chan core.GroupArrivedMessage
}

func InitGroupSyncer() {
	GroupSyncer = groupSyncer{HeightRequestCh: make(chan string), HeightCh: make(chan core.EntityHeightMessage),
		GroupRequestCh: make(chan core.EntityRequestMessage), GroupArrivedCh: make(chan core.GroupArrivedMessage),}
	go GroupSyncer.start()
}

func (gs *groupSyncer) start() {
	gs.syncGroup()
	t := time.NewTicker(GROUP_SYNC_INTERVAL)
	for {
		select {
		case sourceId := <-gs.HeightRequestCh:
			log.Printf("[GroupSyncer]GroupSyncer HeightRequestCh get message from:%s", sourceId)
			//收到组高度请求
			if nil == core.GroupChainImpl {
				return
			}
			sendGroupHeight(sourceId, core.GroupChainImpl.Count())
		case h := <-gs.HeightCh:
			log.Printf("[GroupSyncer]GroupSyncer HeightCh get message from:%s,it's height is:%d", h.SourceId, h.Height)
			//收到来自其他节点的组链高度
			gs.maxHeightLock.Lock()
			if h.Height > gs.neighborMaxHeight {
				gs.neighborMaxHeight = h.Height
				gs.bestNodeId = h.SourceId
			}
			gs.maxHeightLock.Unlock()
		case br := <-gs.GroupRequestCh:
			log.Printf("[GroupSyncer]GroupRequestCh get message from:%s\n,current height:%d,current hash:%s", br.SourceId, br.SourceHeight, br.SourceCurrentHash.String())
			//收到组请求
			if nil == core.GroupChainImpl {
				return
			}
			groups, e := core.GroupChainImpl.GetGroupsByHeight(br.SourceHeight, br.SourceCurrentHash)
			if e != nil {
				logger.Errorf("[GroupSyncer]%s query group error:%s", br.SourceId, e.Error())
				return
			}
			entity := core.GroupMessage{Groups: groups, Height: br.SourceHeight, Hash: br.SourceCurrentHash}
			sendGroups(br.SourceId, &entity)
		case bm := <-gs.GroupArrivedCh:
			log.Printf("[GroupSyncer]GroupArrivedCh get message from:%s,block length:%v", bm.SourceId, len(bm.GroupEntity.Groups))
			//收到组信息
			if nil == core.GroupChainImpl {
				return
			}
			groups := bm.GroupEntity.Groups
			if nil != groups && 0 != len(groups) {
				for _, group := range groups {
					core.GroupChainImpl.AddGroup(group, nil, nil)
				}
			}
		case <-t.C:
			log.Printf("[GroupSyncer]sync time up, start to group sync!")
			gs.syncGroup()
		}
	}
}

func (gs *groupSyncer) syncGroup() {
	go requestGroupChainHeight()
	t := time.NewTimer(GROUP_HEIGHT_RECEIVE_INTERVAL)

	<-t.C
	log.Printf("[GroupSyncer]group height request  time up!")
	if nil == core.GroupChainImpl {
		return
	}
	localHeight, currentHash := core.GroupChainImpl.Count(), common.BytesToHash([]byte{})
	gs.maxHeightLock.Lock()
	maxHeight := gs.neighborMaxHeight
	bestNodeId := gs.bestNodeId
	gs.maxHeightLock.Unlock()
	if maxHeight <= localHeight {
		log.Printf("[GroupSyncer]Neightbor max group height %d is less than self group height %d don't sync!\n", maxHeight, localHeight)
		return
	} else {
		log.Printf("[GroupSyncer]Neightbor max group height %d is greater than self group height %d.Sync from %s!\n", maxHeight, localHeight, bestNodeId)
		requestGroupByHeight(bestNodeId, localHeight, currentHash)
	}

}

//广播索要链高度
func requestGroupChainHeight() {
	message := p2p.Message{Code: p2p.REQ_GROUP_CHAIN_HEIGHT_MSG}
	conns := p2p.Server.Host.Network().Conns()
	for _, conn := range conns {
		id := conn.RemotePeer()
		if id != "" {
			p2p.Server.SendMessage(message, p2p.ConvertToID(id))
		}
	}
}

//返回自身组链高度
func sendGroupHeight(targetId string, localHeight uint64) {
	body := utility.UInt64ToByte(localHeight)
	message := p2p.Message{Code: p2p.GROUP_CHAIN_HEIGHT_MSG, Body: body}
	p2p.Server.SendMessage(message, targetId)
}

//向某一节点请求Block
func requestGroupByHeight(id string, localHeight uint64, currentHash common.Hash) {
	m := core.EntityRequestMessage{SourceHeight: localHeight, SourceCurrentHash: currentHash}
	body, e := core.MarshalEntityRequestMessage(&m)
	if e != nil {
		log.Printf("[GroupSyncer]requestGroupByHeight marshal EntityRequestMessage error:%s", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.REQ_GROUP_MSG, Body: body}
	p2p.Server.SendMessage(message, id)
}

//本地查询之后将结果返回
func sendGroups(targetId string, groupEntity *core.GroupMessage) {
	body, e := marshalGroupMessage(groupEntity)
	if e != nil {
		log.Printf("[GroupSyncer]" +
			"sendGroups marshal groupEntity error:%s", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.GROUP_MSG, Body: body}
	p2p.Server.SendMessage(message, targetId)
}

//----------------------------------------------组同步------------------------------------------------------------------
func marshalGroupMessage(e *core.GroupMessage) ([]byte, error) {
	groups := make([]*tas_pb.Group, 0)

	if e.Groups != nil {
		for _, g := range e.Groups {
			groups = append(groups, core.GroupToPb(g))
		}
	}
	groupSlice := tas_pb.GroupSlice{Groups: groups}

	height := e.Height
	message := tas_pb.GroupMessage{Groups: &groupSlice, Height: &height, Hash: e.Hash.Bytes()}
	return proto.Marshal(&message)
}
