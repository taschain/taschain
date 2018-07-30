package sync

import (
	"core"
	"sync"
	"time"
	"common"
	"utility"
	"github.com/gogo/protobuf/proto"
	"taslog"
	"middleware/pb"
	"middleware/types"
	"fmt"
	"network"
)

//todo  消息传输是否需要签名？ 异常代码处理
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
	Height   uint64
	SourceId string
}

type GroupInfo struct {
	Group      *types.Group
	IsTopGroup bool
	SourceId string
}

type groupSyncer struct {
	ReqHeightCh chan string
	HeightCh    chan GroupHeightInfo
	ReqGroupCh  chan GroupRequestInfo
	GroupCh     chan GroupInfo

	maxHeight uint64
	bestNode  string
	lock      sync.Mutex

	init       bool
	replyCount int
}

func InitGroupSyncer() {
	logger = taslog.GetLoggerByName("sync" + common.GlobalConf.GetString("instance", "index", ""))
	GroupSyncer = groupSyncer{ReqHeightCh: make(chan string), HeightCh: make(chan GroupHeightInfo),
		ReqGroupCh: make(chan GroupRequestInfo), GroupCh: make(chan GroupInfo), maxHeight: 0, init: false, replyCount: 0}
	go GroupSyncer.start()
}

func (gs *groupSyncer) IsInit() bool {
	return gs.init
}

func (gs *groupSyncer) start() {
	logger.Debug("[GroupSyncer]Wait for connecting...")
	go gs.loop()
	for {
		requestGroupChainHeight()
		t := time.NewTimer(INIT_INTERVAL)
		<-t.C
		if gs.replyCount > 0 {
			logger.Debug("[GroupSyncer]Detect node and start sync group...")
			break
		}
	}
	gs.sync()
}

func (gs *groupSyncer) sync() {
	requestGroupChainHeight()
	t := time.NewTimer(GROUP_HEIGHT_RECEIVE_INTERVAL)

	<-t.C
	localHeight := core.GroupChainImpl.Count()
	gs.lock.Lock()
	maxHeight := gs.maxHeight
	bestNode := gs.bestNode
	gs.maxHeight = 0
	gs.bestNode = ""
	gs.lock.Unlock()

	if maxHeight <= localHeight {
		logger.Debugf("[GroupSyncer]Neightbor max group height %d is less than self group height %d don't sync!\n", maxHeight, localHeight)
		if !gs.init {
			gs.init = true
		}
		return
	} else {
		logger.Debugf("[GroupSyncer]Neightbor max group height %d is greater than self group height %d.Sync from %s!\n", maxHeight, localHeight, bestNode)
		if bestNode != "" {
			requestGroupByHeight(bestNode, localHeight)
		}
	}
}

func (gs *groupSyncer) loop() {
	t := time.NewTicker(GROUP_SYNC_INTERVAL)
	for {
		select {
		case sourceId := <-gs.ReqHeightCh:
			//收到组高度请求
			logger.Debugf("[GroupSyncer]Request group height from:%s", sourceId)
			sendGroupHeight(sourceId, core.GroupChainImpl.Count())
		case h := <-gs.HeightCh:
			//收到来自其他节点的组链高度
			logger.Debugf("[GroupSyncer]Receive group height from:%s,height:%d", h.SourceId, h.Height)
			if !gs.init {
				gs.replyCount++
			}
			gs.lock.Lock()
			if h.Height > gs.maxHeight {
				gs.maxHeight = h.Height
				gs.bestNode = h.SourceId
			}
			gs.lock.Unlock()
		case gri := <-gs.ReqGroupCh:
			//收到组请求
			logger.Debugf("[GroupSyncer]Request group from:%s\n,height:%d", gri.SourceId, gri.Height)
			group := core.GroupChainImpl.GetGroupByHeight(gri.Height)
			if group == nil {
				logger.Errorf("[GroupSyncer]Get nil group by height:%d", gri.Height)
				continue
			}
			var isTopGroup bool
			topHeight := core.GroupChainImpl.Count()
			if gri.Height == topHeight{
				isTopGroup = true
			}else {
				isTopGroup = false
			}
			sendGroup(gri.SourceId, group,isTopGroup)
		case groupInfo := <-gs.GroupCh:
			//收到组信息
			logger.Debugf("[GroupSyncer]Receive group :%d", groupInfo.Group.Id)

			e := core.GroupChainImpl.AddGroup(groupInfo.Group, nil, nil)
			if e != nil {
				logger.Errorf("[GroupSyncer]add group on chain error:%s", e.Error())
				//TODO  上链失败 异常处理
				continue
			}else {
				if !groupInfo.IsTopGroup{
					localHeight := core.GroupChainImpl.Count()
					requestGroupByHeight(groupInfo.SourceId, localHeight+1)
				} else{
					if!gs.init {
						fmt.Printf("group sync init finish,local group height:%d\n", core.GroupChainImpl.Count())
						gs.init = true
						continue
					}
				}
			}
		case <-t.C:
			if !gs.init {
				continue
			}
			logger.Debugf("[GroupSyncer]sync time up, start to group sync!")
			go gs.sync()
		}
	}
}

//广播索要组链高度
func requestGroupChainHeight() {
	message := network.Message{Code: network.REQ_GROUP_CHAIN_HEIGHT_MSG}
	network.GetNetInstance().TransmitToNeighbor(message)
}

//返回自身组链高度
func sendGroupHeight(targetId string, localHeight uint64) {
	body := utility.UInt64ToByte(localHeight)
	message := network.Message{Code: network.GROUP_CHAIN_HEIGHT_MSG, Body: body}
	network.GetNetInstance().Send(targetId, message)
}

//向某一节点请求Group
func requestGroupByHeight(id string, groupHeight uint64) {
	body := utility.UInt64ToByte(groupHeight)
	message := network.Message{Code: network.REQ_GROUP_MSG, Body: body}
	network.GetNetInstance().Send(id, message)
}

//本地查询之后将结果返回
func sendGroup(targetId string, group *types.Group, isTopGroup bool) {
	groupInfo := GroupInfo{Group:group,IsTopGroup:isTopGroup}
	body, e := marshalGroupInfo(groupInfo)
	if e != nil {
		logger.Errorf("[GroupSyncer]"+"sendGroup marshal group error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.GROUP_MSG, Body: body}
	network.GetNetInstance().Send(targetId, message)
}

func marshalGroup(e *types.Group) ([]byte, error) {
	group := types.GroupToPb(e)
	return proto.Marshal(group)
}

func marshalGroupInfo(e GroupInfo) ([]byte, error) {
	group := types.GroupToPb(e.Group)
	groupInfo := tas_middleware_pb.GroupInfo{Group: group, IsTopGroup: &e.IsTopGroup}
	return proto.Marshal(&groupInfo)
}
