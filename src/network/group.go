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

package network

import (
	"bytes"
	nnet "net"
	"sync"
	"time"
	"sort"
	"math"
)

const GroupBaseConnectNodeCount = 2
// Group 组对象
type Group struct {
	id             		string
	members        		[]NodeID
	needConnectNodes    []NodeID
	mutex  sync.Mutex
	resolvingNodes map[NodeID]time.Time
	curIndex int
}


func (g *Group) Len() int {
	return len(g.members)
}


func (g *Group) Less(i, j int) bool {
	return g.members[i].GetHexString() < g.members[j].GetHexString()
}


func (g *Group) Swap(i, j int) {
	g.members[i], g.members[j] = g.members[j], g.members[i]
}

func newGroup(id string, members []NodeID) *Group {

	g := &Group{id: id, members: members, needConnectNodes:make([]NodeID,0), resolvingNodes: make(map[NodeID]time.Time)}

	Logger.Debugf("new group id：%v", id)
	g.genConnectNodes()
	return g
}


func (g* Group) rebuildGroup( members []NodeID) {

	Logger.Debugf("rebuild group id：%v", g.id)
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.members = members
	g.genConnectNodes()

	go g.doRefresh()
}

func (g* Group) genConnectNodes() {

	sort.Sort(g)
	peerSize := len(g.members)
	g.curIndex =0
	for i:= 0;i<len(g.members);i++ {
		if g.members[i] == netCore.id {
			g.curIndex = i
			break
		}
	}

	Logger.Debugf("[genConnectNodes] curIndex:%v",g.curIndex)
	for i:= 0;i<len(g.members);i++ {
		Logger.Debugf("[genConnectNodes] members id：%v",g.members[i].GetHexString())
	}

	connectCount := GroupBaseConnectNodeCount
	if connectCount >  len(g.members) -1 {
		connectCount = len(g.members) -1
	}

	nextIndex := g.getNextIndex(g.curIndex)
	g.needConnectNodes = append(g.needConnectNodes,g.members[nextIndex])
	if peerSize >=5 {

		nextIndex = g.getNextIndex(nextIndex)
		g.needConnectNodes = append(g.needConnectNodes,g.members[nextIndex])

		maxCount := int(math.Sqrt(float64(peerSize))/2);
		maxCount -=  len(g.needConnectNodes)
		step := 1

		if maxCount > 0 {
			step = len(g.members)/ maxCount
		}

		for i:=0;i<maxCount ;i++ {
			nextIndex += step
			if nextIndex >= len(g.members) {
				nextIndex %= len(g.members)
			}
			g.needConnectNodes = append(g.needConnectNodes,g.members[nextIndex])
		}
	}

	for i:= 0;i<len(g.needConnectNodes);i++ {
		Logger.Debugf("[genConnectNodes] needConnectNodes id：%v", g.needConnectNodes[i].GetHexString())
	}

}


func (g *Group) getNextIndex(index int) int {
	index = index +1
	if index >= len(g.members) {
		index=0
	}
	return index
}

func (g *Group) doRefresh() {

	g.mutex.Lock()
	defer g.mutex.Unlock()

	memberSize := len(g.needConnectNodes)

	for i := 0; i < memberSize; i++ {
		id := g.needConnectNodes[i]
		if id == netCore.id {
			continue
		}

		p := netCore.peerManager.peerByID(id)
		if p != nil {
			continue
		}
		node := netCore.kad.find(id)
		if node != nil && node.Ip != nil && node.Port > 0 {
			Logger.Debugf("Group doRefresh node found in KAD id：%v ip: %v  port:%v", id.GetHexString(), node.Ip, node.Port)
			go netCore.ping(node.Id, &nnet.UDPAddr{IP: node.Ip, Port: int(node.Port)})
		} else {
			go netCore.ping(id, nil)

			Logger.Debugf("Group doRefresh node can not find in KAD ,resolve ....  id：%v ", id.GetHexString())
			g.resolve(id)
		}
	}
}

func (g *Group) resolve(id NodeID) {
	resolveTimeout := 3 * time.Minute
	t, ok := g.resolvingNodes[id]
	if ok && time.Since(t) < resolveTimeout {
		return
	}
	g.resolvingNodes[id] = time.Now()
	go netCore.kad.resolve(id)
}

func (g *Group) send(packet *bytes.Buffer,code uint32) {

	for i := 0; i < len(g.needConnectNodes); i++ {
		id := g.needConnectNodes[i]
		if id == netCore.id {
			continue
		}
		p := netCore.peerManager.peerByID(id)
		if p != nil {
			netCore.peerManager.write(id, &nnet.UDPAddr{IP: p.Ip, Port: int(p.Port)}, packet, code)
		} else {
			node := netCore.kad.find(id)
			if node != nil && node.Ip != nil && node.Port > 0 {
				Logger.Debugf("SendGroup node not connected ,but in KAD : id：%v ip: %v  port:%v", id.GetHexString(), node.Ip, node.Port)
				netCore.peerManager.write(node.Id, &nnet.UDPAddr{IP: node.Ip, Port: int(node.Port)}, packet, code)
			} else {
				Logger.Debugf("SendGroup node not connected and not in KAD : id：%v", id.GetHexString())
				netCore.peerManager.write(id, nil, packet, code)
			}
		}
	}
	netCore.bufferPool.FreeBuffer(packet)
	return
}

//GroupManager 组管理
type GroupManager struct {
	groups map[string]*Group
	mutex  sync.RWMutex
}

func newGroupManager() *GroupManager {

	gm := &GroupManager{
		groups: make(map[string]*Group),
	}
	return gm
}

//buildGroup 创建组，如果组已经存在，则重建组网络
func (gm *GroupManager) buildGroup(ID string, members []NodeID) *Group {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	Logger.Debugf("build group, id:%v, count:%v", ID, len(members))

	g,isExist := gm.groups[ID]
	if !isExist {
		g = newGroup(ID, members)
		gm.groups[ID] = g
	} else {
		g.rebuildGroup(members)
	}
	go g.doRefresh()
	return g
}

//RemoveGroup 移除组
func (gm *GroupManager) removeGroup(id string) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()


	Logger.Debugf("remove group, id:%v.",id)


	delete(gm.groups, id)
}

func (gm *GroupManager) doRefresh() {
	//fmt.Printf("groupManager doRefresh ")
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	for _, group := range gm.groups {
		go group.doRefresh()
	}
}

//SendGroup 向所有已经连接的组内节点发送自定义数据包
func (gm *GroupManager) sendGroup(id string, packet *bytes.Buffer ,code uint32) {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	Logger.Debugf("send group, id:%v", id)
	g := gm.groups[id]
	if g == nil {
		Logger.Debugf("group not found.")
		return
	}
	buf := netCore.bufferPool.GetBuffer(packet.Len())
	buf.Write(packet.Bytes())

	go g.send(buf, code)

	return
}
