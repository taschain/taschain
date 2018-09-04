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
	"bytes"
	"fmt"
)

//todo  消息传输是否需要签名？ 异常代码处理
const (
	GROUP_HEIGHT_RECEIVE_INTERVAL = 1 * time.Second

	GROUP_SYNC_INTERVAL = 3 * time.Second
)

var GroupSyncer groupSyncer
var logger taslog.Logger

type GroupHeightInfo struct {
	Height   uint64
	SourceId string
}

type GroupRequestInfo struct {
	Height	uint64
	SourceId string
}

type GroupInfo struct {
	Groups      []*types.Group
	IsTopGroup bool
	SourceId   string
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
		//t.Reset(INIT_INTERVAL)
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
	gs.replyCount = 0
	gs.lock.Unlock()

	if maxHeight <= localHeight {
		logger.Debugf("[GroupSyncer]Neightbor max group height %d is less than self group height %d\n don't sync!\n", maxHeight, localHeight)
		if !gs.init {
			gs.init = true
		}
	} else {
		logger.Debugf("[GroupSyncer]Neightbor max group height %d is greater than self group height %d.\nSync from %s!\n", maxHeight, localHeight, bestNode)
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
			logger.Debugf("[GroupSyncer]Rcv group height req from:%s", sourceId)
			sendGroupHeight(sourceId, core.GroupChainImpl.Count())
		case h := <-gs.HeightCh:
			//收到来自其他节点的组链高度
			logger.Debugf("[GroupSyncer]Rcv group height from:%s,height:%d", h.SourceId, h.Height)
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
			logger.Debugf("[GroupSyncer]Rcv group from:%s,height:%d\n", gri.SourceId, gri.Height)
			groups := core.GroupChainImpl.GetSyncGroupsByHeight(gri.Height, 5)
			l := len(groups)
			if l == 0 {
				logger.Errorf("[GroupSyncer]Get nil group by id:%s", gri.SourceId)
				continue
			} else {
				var isTop bool
				if bytes.Equal(groups[l-1].Id, core.GroupChainImpl.LastGroup().Id) {
					isTop = true
				}
				sendGroups(gri.SourceId, groups, isTop)
				logger.Debugf("[GroupSyncer]ReqGroupCh sendGroups:%s,lastId:%d\n", gri.SourceId, groups[l-1].Id)
			}
		case groupInfos := <-gs.GroupCh:
			//收到组信息
			logger.Debugf("[GroupSyncer]Rcv groups len :%d,from:%d", len(groupInfos.Groups),groupInfos.SourceId)

			for _,group := range groupInfos.Groups {
				e := core.GroupChainImpl.AddGroup(group, nil, nil)
				if e != nil {
					logger.Errorf("[GroupSyncer]add group on chain error:%s", e.Error())
					//TODO  上链失败 异常处理
					return
				}
			}

			if !groupInfos.IsTopGroup {
				localHeight := core.GroupChainImpl.Count()
				requestGroupByHeight(groupInfos.SourceId, localHeight+1)
			} else {
				if !gs.init {
					fmt.Printf("group sync init finish,local group height:%d\n", core.GroupChainImpl.Count())
					gs.init = true
					continue
				}
			}

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

//向某一节点请求Group
func requestGroupByHeight(id string, groupHeight uint64) {
	logger.Debugf("[GroupSyncer]Req group for %s,height:%d!",id,groupHeight)
	body := utility.UInt64ToByte(groupHeight)
	message := network.Message{Code: network.ReqGroupMsg, Body: body}
	network.GetNetInstance().Send(id, message)
}

//本地查询之后将结果返回
func sendGroups(targetId string, groups []*types.Group, isTop bool) {
	logger.Debugf("[GroupSyncer]Send group to %s,group len:%d",targetId,len(groups))
	body, e := marshalGroupInfo(groups, isTop)
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

func marshalGroupInfo(e []*types.Group,isTop bool) ([]byte, error) {
	var groups []*tas_middleware_pb.Group
	for _,g := range e{
		groups = append(groups, types.GroupToPb(g))
	}

	groupInfo := tas_middleware_pb.GroupInfo{Groups: groups, IsTopGroup: &isTop}
	return proto.Marshal(&groupInfo)
}
