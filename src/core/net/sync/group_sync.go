package sync

import (
	"common"
	"core"
	"github.com/gogo/protobuf/proto"
	"middleware/pb"
	"middleware/types"
	"network/p2p"
	"sync"
	"taslog"
	"time"
	"utility"
)

const (
	GROUP_HEIGHT_RECEIVE_INTERVAL = 3 * time.Second

	GROUP_SYNC_INTERVAL = 5 * time.Second
)

var GroupSyncer groupSyncer
var logger taslog.Logger

type GroupHeightInfo struct {
	Height   uint64
	SourceId string
}

type GroupRequestInfo struct {
	BaseHeight uint64
	SourceId   string
}

type groupSyncer struct {
	neighborMaxHeight uint64     //邻居节点的最大高度
	bestNodeId        string     //最佳同步节点
	maxHeightLock     sync.Mutex //同步锁

	HeightRequestCh chan string
	HeightCh        chan GroupHeightInfo
	GroupRequestCh  chan GroupRequestInfo
	GroupCh         chan []*types.Group
}

func InitGroupSyncer() {
	logger = taslog.GetLoggerByName("sync" + common.GlobalConf.GetString("client", "index", ""))
	GroupSyncer = groupSyncer{HeightRequestCh: make(chan string), HeightCh: make(chan GroupHeightInfo),
		GroupRequestCh: make(chan GroupRequestInfo), GroupCh: make(chan []*types.Group)}
	go GroupSyncer.start()
}

func (gs *groupSyncer) start() {
	gs.syncGroup()
	t := time.NewTicker(GROUP_SYNC_INTERVAL)
	for {
		select {
		case sourceId := <-gs.HeightRequestCh:
			//收到组高度请求
			//logger.Debugf("[GroupSyncer]GroupSyncer HeightRequestCh get message from:%s", sourceId)
			if nil == core.GroupChainImpl {
				return
			}
			sendGroupHeight(sourceId, core.GroupChainImpl.Count())
		case h := <-gs.HeightCh:
			//收到来自其他节点的组链高度
			//logger.Debugf("[GroupSyncer]GroupSyncer HeightCh get message from:%s,it's height is:%d", h.SourceId, h.Height)
			gs.maxHeightLock.Lock()
			if h.Height > gs.neighborMaxHeight {
				gs.neighborMaxHeight = h.Height
				gs.bestNodeId = h.SourceId
			}
			gs.maxHeightLock.Unlock()
		case gri := <-gs.GroupRequestCh:
			//收到组请求
			//logger.Debugf("[GroupSyncer]GroupRequestCh get message from:%s\n,current height:%d", gri.SourceId, gri.BaseHeight)
			if nil == core.GroupChainImpl {
				return
			}
			groups, e := core.GroupChainImpl.GetGroupsByHeight(gri.BaseHeight)
			if e != nil {
				logger.Errorf("[GroupSyncer]%s query group error:%s", gri.SourceId, e.Error())
				return
			}
			sendGroups(gri.SourceId, groups)
		case groups := <-gs.GroupCh:
			//收到组信息
			//logger.Debugf("[GroupSyncer]GroupCh get message,group length:%v", len(groups))
			if nil == core.GroupChainImpl {
				return
			}
			if nil != groups && 0 != len(groups) {
				for _, group := range groups {
					e := core.GroupChainImpl.AddGroup(group, nil, nil)
					if e != nil {
						logger.Errorf("[GroupSyncer]add group on chain error:%s", e.Error())
						return
					}
				}
				if !core.GroupChainImpl.IsGroupSyncInit() {
					core.GroupChainImpl.SetGroupSyncInit(true)
				}
			}
		case <-t.C:
			//logger.Debugf("[GroupSyncer]sync time up, start to group sync!")
			gs.syncGroup()
		}
	}
}

func (gs *groupSyncer) syncGroup() {
	go requestGroupChainHeight()
	t := time.NewTimer(GROUP_HEIGHT_RECEIVE_INTERVAL)

	<-t.C
	//logger.Debugf("[GroupSyncer]group height request  time up!")
	if nil == core.GroupChainImpl {
		return
	}
	localHeight := core.GroupChainImpl.Count()
	gs.maxHeightLock.Lock()
	maxHeight := gs.neighborMaxHeight
	bestNodeId := gs.bestNodeId
	gs.maxHeightLock.Unlock()
	if maxHeight <= localHeight {
		//logger.Debugf("[GroupSyncer]Neightbor max group height %d is less than self group height %d don't sync!\n", maxHeight, localHeight)
		if !core.GroupChainImpl.IsGroupSyncInit() {
			core.GroupChainImpl.SetGroupSyncInit(true)
		}
		return
	} else {
		//logger.Debugf("[GroupSyncer]Neightbor max group height %d is greater than self group height %d.Sync from %s!\n", maxHeight, localHeight, bestNodeId)
		requestGroupByHeight(bestNodeId, localHeight)
	}

}

//广播索要组链高度
func requestGroupChainHeight() {
	message := p2p.Message{Code: p2p.REQ_GROUP_CHAIN_HEIGHT_MSG}
	// conns := p2p.Server.Host.Network().Conns()
	// for _, conn := range conns {
	// 	id := conn.RemotePeer()
	// 	if id != "" {
	// 		p2p.Server.SendMessage(message, p2p.ConvertToID(id))
	// 	}
	// }
	p2p.Server.SendMessageToAll(message)
}

//返回自身组链高度
func sendGroupHeight(targetId string, localHeight uint64) {
	body := utility.UInt64ToByte(localHeight)
	message := p2p.Message{Code: p2p.GROUP_CHAIN_HEIGHT_MSG, Body: body}
	p2p.Server.SendMessage(message, targetId)
}

//向某一节点请求Block
func requestGroupByHeight(id string, localGroupHeight uint64) {
	body := utility.UInt64ToByte(localGroupHeight)
	message := p2p.Message{Code: p2p.REQ_GROUP_MSG, Body: body}
	p2p.Server.SendMessage(message, id)
}

//本地查询之后将结果返回
func sendGroups(targetId string, groups []*types.Group) {
	body, e := marshalGroups(groups)
	if e != nil {
		logger.Errorf("[GroupSyncer]"+"sendGroups marshal groups error:%s", e.Error())
		return
	}
	message := p2p.Message{Code: p2p.GROUP_MSG, Body: body}
	p2p.Server.SendMessage(message, targetId)
}

func marshalGroups(e []*types.Group) ([]byte, error) {
	groups := make([]*tas_middleware_pb.Group, 0)

	if e != nil {
		for _, g := range e {
			groups = append(groups, types.GroupToPb(g))
		}
	}
	groupSlice := tas_middleware_pb.GroupSlice{Groups: groups}
	return proto.Marshal(&groupSlice)
}
