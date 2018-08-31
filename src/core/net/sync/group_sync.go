//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

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
	"network"
)

//todo  消息传输是否需要签名？ 异常代码处理
const (
	GROUP_HEIGHT_RECEIVE_INTERVAL = 1 * time.Second

	GROUP_SYNC_INTERVAL = 3 * time.Second
)

var GroupSyncer groupSyncer
var logger taslog.Logger

type GroupCountInfo struct {
	Count    uint64
	SourceId string
}

type GroupRequestInfo struct {
	CurrentTopGroupId   []byte
	ExistGroupIds		[][]byte
	SourceId string
}

type GroupInfo struct {
	Group      *types.Group
	IsTopGroup bool
	SourceId   string
}

type GroupInfos struct {
	Groups      []*types.Group
	SourceId   string
}

type groupSyncer struct {
	ReqHeightCh chan string
	CountCh     chan GroupCountInfo
	ReqGroupCh  chan GroupRequestInfo
	GroupCh     chan GroupInfos

	maxCount uint64
	bestNode string
	lock     sync.Mutex

	init       bool
	replyCount int
}

func InitGroupSyncer() {
	logger = taslog.GetLoggerByName("sync" + common.GlobalConf.GetString("instance", "index", ""))
	GroupSyncer = groupSyncer{ReqHeightCh: make(chan string), CountCh: make(chan GroupCountInfo),
		ReqGroupCh: make(chan GroupRequestInfo), GroupCh: make(chan GroupInfos), maxCount: 0, init: false, replyCount: 0}
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
	maxHeight := gs.maxCount
	bestNode := gs.bestNode
	gs.maxCount = 0
	gs.bestNode = ""
	gs.lock.Unlock()

	if maxHeight <= localHeight {
		logger.Debugf("[GroupSyncer]Neightbor max group height %d is less than self group height %d\n don't sync!\n", maxHeight, localHeight)
		if !gs.init {
			gs.init = true
		}
		return
	} else {
		logger.Debugf("[GroupSyncer]Neightbor max group height %d is greater than self group height %d.\nSync from %s!\n", maxHeight, localHeight, bestNode)
		if bestNode != "" {
			for _,groupId := range core.GroupChainImpl.MissingGroupIds(){
				requestGroupByGroupId(bestNode, groupId)
			}
		}
	}
}

func (gs *groupSyncer) loop() {
	t := time.NewTicker(GROUP_SYNC_INTERVAL)
	for {
		select {
		case sourceId := <-gs.ReqHeightCh:
			//收到组高度请求
			logger.Debugf("[GroupSyncer]Rcv group height req from:%s", sourceId)
			sendGroupHeight(sourceId, core.GroupChainImpl.Count())
		case h := <-gs.CountCh:
			//收到来自其他节点的组链高度
			logger.Debugf("[GroupSyncer]Rcv group height from:%s,height:%d", h.SourceId, h.Count)
			if !gs.init {
				gs.replyCount++
			}
			gs.lock.Lock()
			if h.Count > gs.maxCount {
				gs.maxCount = h.Count
				gs.bestNode = h.SourceId
			}
			gs.lock.Unlock()
		case gri := <-gs.ReqGroupCh:
			//收到组请求
			logger.Debugf("[GroupSyncer]Rcv group from:%s\n,id:%s", gri.SourceId, gri.CurrentTopGroupId)
			groups := core.GroupChainImpl.GetOtherLackGroups(gri.CurrentTopGroupId,gri.ExistGroupIds)
			if len(groups) == 0 {
				logger.Errorf("[GroupSyncer]Get nil group by id:%s", gri.SourceId)
				continue
			}

			sendGroups(gri.SourceId, groups)
		case groupInfos := <-gs.GroupCh:
			//收到组信息
			logger.Debugf("[GroupSyncer]Rcv groups len :%d,from:%d", len(groupInfos.Groups),groupInfos.SourceId)
			for _,group := range groupInfos.Groups {
				e := core.GroupChainImpl.AddGroup(group, nil, nil)
				if e != nil {
					logger.Errorf("[GroupSyncer]add group on chain error:%s", e.Error())
					//TODO  上链失败 异常处理
				}
			}
			//else {
			//	if !groupInfo.IsTopGroup {
			//		localHeight := core.GroupChainImpl.Count()
			//		requestGroupByGroupId(groupInfo.SourceId, localHeight+1)
			//	} else {
			//		if !gs.init {
			//			fmt.Printf("group sync init finish,local group height:%d\n", core.GroupChainImpl.Count())
			//			gs.init = true
			//			continue
			//		}
			//	}
			//}
		case <-t.C:
			if !gs.init {
				continue
			}
			logger.Debugf("[GroupSyncer]sync time up, start to group sync!")
			gs.sync()
		}
	}
}

//广播索要组链高度
func requestGroupChainHeight() {
	logger.Debugf("[GroupSyncer]Req group height for neighbor!")
	message := network.Message{Code: network.ReqGroupChainCountMsg}
	network.GetNetInstance().TransmitToNeighbor(message)
}

//返回自身组链高度
func sendGroupHeight(targetId string, localCount uint64) {
	logger.Debugf("[GroupSyncer]Send local group height %d to %s!", localCount,targetId)
	body := utility.UInt64ToByte(localCount)
	message := network.Message{Code: network.GroupChainCountMsg, Body: body}
	network.GetNetInstance().Send(targetId, message)
}

//向某一节点请求Group
func requestGroupByGroupId(id string, groupId []byte) {
	logger.Debugf("[GroupSyncer]Req group for %s,id:%s!",id,groupId)
	message := network.Message{Code: network.ReqGroupMsg, Body: groupId}
	network.GetNetInstance().Send(id, message)
}

//本地查询之后将结果返回
func sendGroup(targetId string, group *types.Group, isTopGroup bool) {
	logger.Debugf("[GroupSyncer]Send group to %s,group id:%s",targetId,group.Id)
	groupInfo := GroupInfo{Group: group, IsTopGroup: isTopGroup}
	body, e := marshalGroupInfo(groupInfo)
	if e != nil {
		logger.Errorf("[GroupSyncer]"+"sendGroup marshal group error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.GroupMsg, Body: body}
	network.GetNetInstance().Send(targetId, message)
}

func sendGroups(targetId string, groups []*types.Group) {
	logger.Debugf("[GroupSyncer]Send group to %s,group len:%d",targetId,len(groups))
	body, e := marshalGroupInfos(groups)
	if e != nil {
		logger.Errorf("[GroupSyncer]"+"sendGroup marshal group error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.GroupMsg, Body: body}
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

func marshalGroupInfos(e []*types.Group) ([]byte, error) {
	var groups []*tas_middleware_pb.Group
	for _,g := range e{
		groups = append(groups, types.GroupToPb(g))
	}

	groupInfo := tas_middleware_pb.GroupSlice{Groups: groups}
	return proto.Marshal(&groupInfo)
}
