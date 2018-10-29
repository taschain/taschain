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

package core

import (
	"sync"
	"time"
	"common"
	"utility"
	"github.com/gogo/protobuf/proto"
	"taslog"
	"middleware/pb"
	"middleware/types"
	"network"
	"bytes"
)

//todo  消息传输是否需要签名？ 异常代码处理
const (
	GROUP_SYNC_INTERVAL = 3 * time.Second
)

var GroupSyncer groupSyncer
var logger taslog.Logger

type GroupHeightInfo struct {
	Height   uint64
	SourceId string
}

type GroupRequestInfo struct {
	//Height	uint64
	GroupId  []byte
	SourceId string
}

type GroupInfo struct {
	Groups     []*types.Group
	IsTopGroup bool
	SourceId   string
}

type groupSyncer struct {
	HeightCh   chan GroupHeightInfo
	ReqGroupCh chan GroupRequestInfo
	GroupCh    chan GroupInfo

	maxHeight uint64
	bestNode  string
	lock      sync.Mutex

	init        bool
	hasNeighbor bool
}

func InitGroupSyncer() {
	logger = taslog.GetLoggerByName("sync" + common.GlobalConf.GetString("instance", "index", ""))
	GroupSyncer = groupSyncer{HeightCh: make(chan GroupHeightInfo, 100),
		ReqGroupCh: make(chan GroupRequestInfo, 100), GroupCh: make(chan GroupInfo, 100), maxHeight: 0, init: false, hasNeighbor: false}
	go GroupSyncer.start()
}

func (gs *groupSyncer) IsInit() bool {
	return gs.init
}

func (gs *groupSyncer) start() {
	logger.Debug("[GroupSyncer]Wait for connecting...")
	go gs.loop()

	detectConnTicker := time.NewTicker(INIT_INTERVAL)
	for {
		<-detectConnTicker.C
		if gs.hasNeighbor {
			logger.Debug("[GroupSyncer]Detect node and start sync block...")
			break
		}
	}
	gs.sync()
}

func (gs *groupSyncer) sync() {
	localHeight := GroupChainImpl.Count()
	gs.lock.Lock()
	maxHeight := gs.maxHeight
	bestNode := gs.bestNode
	gs.lock.Unlock()

	if maxHeight <= localHeight {
		logger.Debugf("[GroupSyncer]Neighbor max group height %d is less than self group height %d\n don't sync!\n", maxHeight, localHeight)
		if !gs.init {
			logger.Info("Group init sync finished!")
			gs.init = true
		}
	} else {
		logger.Debugf("[GroupSyncer]Neighbor max group height %d is greater than self group height %d.\nSync from %s!\n", maxHeight, localHeight, bestNode)
		if bestNode != "" {
			requestGroupByGroupId(bestNode, GroupChainImpl.LastGroup().Id)
		}
	}
}

func (gs *groupSyncer) loop() {
	t := time.NewTicker(GROUP_SYNC_INTERVAL)
	for {
		select {
		case h := <-gs.HeightCh:
			//收到来自其他节点的组链高度
			logger.Debugf("[GroupSyncer]Rcv group height from:%s,height:%d", h.SourceId, h.Height)

			if !gs.hasNeighbor {
				gs.hasNeighbor = true
			}
			gs.lock.Lock()
			if h.Height > gs.maxHeight {
				gs.maxHeight = h.Height
				gs.bestNode = h.SourceId
			}
			gs.lock.Unlock()
		case gri := <-gs.ReqGroupCh:
			//收到组请求
			logger.Debugf("[GroupSyncer]Rcv group from:%s,id:%v\n", gri.SourceId, gri.GroupId)
			groups := GroupChainImpl.GetSyncGroupsById(gri.GroupId)
			l := len(groups)
			if l == 0 {
				logger.Errorf("[GroupSyncer]Get nil group by id:%s", gri.SourceId)
				continue
			} else {
				var isTop bool
				if bytes.Equal(groups[l-1].Id, GroupChainImpl.LastGroup().Id) {
					isTop = true
				}
				sendGroups(gri.SourceId, groups, isTop)
				logger.Debugf("[GroupSyncer]ReqGroupCh sendGroups:%s,lastId:%d\n", gri.SourceId, groups[l-1].Id)
			}
		case groupInfos := <-gs.GroupCh:
			//收到组信息
			logger.Debugf("[GroupSyncer]Rcv groups len:%d,from:%s", len(groupInfos.Groups), groupInfos.SourceId)
			for _, group := range groupInfos.Groups {
				e := GroupChainImpl.AddGroup(group, nil, nil)
				logger.Debugf("[GroupSyncer] AddGroup Height:%d Id:%s Err:%v", GroupChainImpl.Count()-1,
					common.BytesToAddress(group.Id).GetHexString(), e)
				if e != nil {
					logger.Errorf("[GroupSyncer]add group on chain error:%s", e.Error())
					//TODO  上链失败 异常处理
					continue
				}
			}

			if !groupInfos.IsTopGroup {
				requestGroupByGroupId(groupInfos.SourceId, GroupChainImpl.LastGroup().Id)
			} else {
				if !gs.init {
					logger.Info("Group init sync finished!")
					gs.init = true
					continue
				}
			}
		case <-t.C:
			sendGroupHeightToNeighbor(GroupChainImpl.Count())
			if !gs.init {
				continue
			}
			//logger.Debugf("[GroupSyncer]sync time up, start to group sync!")
			gs.sync()
		}
	}
}

//返回自身组链高度
func sendGroupHeightToNeighbor(localCount uint64) {
	logger.Debugf("[GroupSyncer]Send local group height %d to neighbor!", localCount)
	body := utility.UInt64ToByte(localCount)
	message := network.Message{Code: network.GroupChainCountMsg, Body: body}
	network.GetNetInstance().TransmitToNeighbor(message)
}

//向某一节点请求Group
func requestGroupByGroupId(id string, groupId []byte) {
	//logger.Debugf("[GroupSyncer]Req group for %s,id:%s!",id,groupId)
	message := network.Message{Code: network.ReqGroupMsg, Body: groupId}
	network.GetNetInstance().Send(id, message)
}

//本地查询之后将结果返回
func sendGroups(targetId string, groups []*types.Group, isTop bool) {
	logger.Debugf("[GroupSyncer]Send group to %s,group len:%d", targetId, len(groups))
	body, e := marshalGroupInfo(groups, isTop)

	if e != nil {
		logger.Errorf("[GroupSyncer]"+"sendGroup marshal group error:%s", e.Error())
		return
	}
	message := network.Message{Code: network.GroupMsg, Body: body}
	network.GetNetInstance().Send(targetId, message)
}

func marshalGroupInfo(e []*types.Group, isTop bool) ([]byte, error) {
	var groups []*tas_middleware_pb.Group
	for _, g := range e {
		groups = append(groups, types.GroupToPb(g))
	}

	groupInfo := tas_middleware_pb.GroupInfo{Groups: groups, IsTopGroup: &isTop}
	return proto.Marshal(&groupInfo)
}

//func marshalGroup(e *types.Group) ([]byte, error) {
//	group := types.GroupToPb(e)
//	return proto.Marshal(group)
//}
