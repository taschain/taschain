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
	"middleware/notify"
)

//todo  消息传输是否需要签名？ 异常代码处理
const (
	GROUP_SYNC_INTERVAL = 3 * time.Second
)

var GroupSyncer groupSyncer
var groupSyncLogger taslog.Logger

type GroupInfo struct {
	Groups     []*types.Group
	IsTopGroup bool
}

type groupSyncer struct {
	maxHeight uint64
	bestNode  string
	lock      sync.Mutex

	init bool
}

func InitGroupSyncer() {
	groupSyncLogger = taslog.GetLoggerByName("group_sync" + common.GlobalConf.GetString("instance", "index", ""))
	GroupSyncer = groupSyncer{maxHeight: 0, init: false,}
	notify.BUS.Subscribe(notify.GroupHeight, GroupSyncer.groupHeightHandler)
	notify.BUS.Subscribe(notify.GroupReq, GroupSyncer.groupReqHandler)
	notify.BUS.Subscribe(notify.Group, GroupSyncer.groupHandler)

	go GroupSyncer.loop()
}

func (gs *groupSyncer) IsInit() bool {
	return gs.init
}

func (gs *groupSyncer) sync() {
	localHeight := GroupChainImpl.Count()
	gs.lock.Lock()
	maxHeight := gs.maxHeight
	bestNode := gs.bestNode
	gs.lock.Unlock()

	if maxHeight <= localHeight {
		groupSyncLogger.Debugf("[GroupSyncer]Neighbor max group height %d is less than self group height %d\n don't sync!\n", maxHeight, localHeight)
		if !gs.init {
			groupSyncLogger.Info("Group init sync finished!")
			gs.init = true
		}
	} else {
		groupSyncLogger.Debugf("[GroupSyncer]Neighbor max group height %d is greater than self group height %d.\nSync from %s!\n", maxHeight, localHeight, bestNode)
		if bestNode != "" {
			requestGroupByGroupId(bestNode, GroupChainImpl.LastGroup().Id)
		}
	}
}

func (gs *groupSyncer) groupHeightHandler(msg notify.Message) {
	groupHeightMsg, ok := msg.GetData().(*notify.GroupHeightMessage)
	if !ok {
		Logger.Debugf("groupHeightHandler GetData assert not ok!")
		return
	}

	height := utility.ByteToUInt64(groupHeightMsg.HeightByte)
	groupSyncLogger.Debugf("[GroupSyncer]Rcv group height from:%s,height:%d", groupHeightMsg.Peer, height)

	gs.lock.Lock()
	if height > gs.maxHeight {
		gs.maxHeight = height
		gs.bestNode = groupHeightMsg.Peer
	}
	gs.lock.Unlock()
}

func (gs *groupSyncer) groupReqHandler(msg notify.Message) {
	groupReqMsg, ok := msg.GetData().(*notify.GroupReqMessage)
	if !ok {
		Logger.Debugf("groupReqHandler GetData assert not ok!")
		return
	}

	sourceId := groupReqMsg.Peer
	groupId := groupReqMsg.GroupIdByte
	groupSyncLogger.Debugf("[GroupSyncer]Rcv group req from:%s,id:%v\n", sourceId, groupId)
	groups := GroupChainImpl.GetSyncGroupsById(groupId)
	l := len(groups)
	if l == 0 {
		groupSyncLogger.Errorf("[GroupSyncer]Get nil group by id:%v", groupId)
		return
	} else {
		var isTop bool
		if bytes.Equal(groups[l-1].Id, GroupChainImpl.LastGroup().Id) {
			isTop = true
		}
		sendGroups(sourceId, groups, isTop)
		groupSyncLogger.Debugf("SendGroups:%s,lastId:%d\n", sourceId, groups[l-1].Id)
	}
}

func (gs *groupSyncer) groupHandler(msg notify.Message) {
	groupInfoMsg, ok := msg.GetData().(*notify.GroupInfoMessage)
	if !ok {
		Logger.Debugf("groupHandler GetData assert not ok!")
		return
	}

	groupInfo, e := unMarshalGroupInfo(groupInfoMsg.GroupInfoByte)
	if e != nil {
		groupSyncLogger.Errorf("[handler]Discard GROUP_MSG because of unmarshal error:%s", e.Error())
		return
	}

	sourceId := groupInfoMsg.Peer
	groups := groupInfo.Groups
	groupSyncLogger.Debugf("[GroupSyncer]Rcv groups ,from:%s,groups len %d", sourceId, len(groups))
	for _, group := range groupInfo.Groups {
		groupSyncLogger.Debugf("[GroupSyncer] AddGroup Id:%s,pre id:%s", common.BytesToAddress(group.Id).GetHexString(), common.BytesToAddress(group.Parent).GetHexString())
		groupSyncLogger.Debugf("[GroupSyncer] Local height:%d,local top group id:%s", GroupChainImpl.Count(), common.BytesToAddress(GroupChainImpl.LastGroup().Id).GetHexString(), )
		e := GroupChainImpl.AddGroup(group, nil, nil)
		if e != nil {
			groupSyncLogger.Errorf("[GroupSyncer]add group on chain error:%s", e.Error())
			//TODO  上链失败 异常处理
			continue
		}
	}

	if !groupInfo.IsTopGroup {
		requestGroupByGroupId(sourceId, GroupChainImpl.LastGroup().Id)
	} else {
		if !gs.init {
			groupSyncLogger.Info("Group init sync finished!")
			gs.init = true
		}
	}
}

func (gs *groupSyncer) loop() {
	t := time.NewTicker(GROUP_SYNC_INTERVAL)
	var tick time.Time
	for tick = range t.C {
		_ = tick
		go sendGroupHeightToNeighbor(GroupChainImpl.Count())
		go gs.sync()
	}
}

//返回自身组链高度
func sendGroupHeightToNeighbor(localCount uint64) {
	groupSyncLogger.Debugf("[GroupSyncer]Send local group height %d to neighbor!", localCount)
	body := utility.UInt64ToByte(localCount)
	message := network.Message{Code: network.GroupChainCountMsg, Body: body}
	network.GetNetInstance().TransmitToNeighbor(message)
}

//向某一节点请求Group
func requestGroupByGroupId(id string, groupId []byte) {
	groupSyncLogger.Debugf("[GroupSyncer]Req group for %s,id:%v!", id, groupId)
	message := network.Message{Code: network.ReqGroupMsg, Body: groupId}
	network.GetNetInstance().Send(id, message)
}

//本地查询之后将结果返回
func sendGroups(targetId string, groups []*types.Group, isTop bool) {
	groupSyncLogger.Debugf("[GroupSyncer]Send group to %s,groups:%d-%d,isTop:%t", targetId, groups[0].GroupHeight, groups[len(groups)-1].GroupHeight, isTop)
	body, e := marshalGroupInfo(groups, isTop)
	if e != nil {
		groupSyncLogger.Errorf("[GroupSyncer]sendGroup marshal group error:%s", e.Error())
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

func unMarshalGroupInfo(b []byte) (*GroupInfo, error) {
	message := new(tas_middleware_pb.GroupInfo)
	e := proto.Unmarshal(b, message)
	if e != nil {
		groupSyncLogger.Errorf("[handler]unMarshalGroupInfo error:%s", e.Error())
		return nil, e
	}
	groups := make([]*types.Group, len(message.Groups))
	for i, g := range message.Groups {
		groups[i] = types.PbToGroup(g)
	}
	groupInfo := GroupInfo{Groups: groups, IsTopGroup: *message.IsTopGroup}
	return &groupInfo, nil
}
