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

	resolvingNodes map[NodeID]time.Time
	curIndex int
}


func (g Group) Len() int {
	return len(g.members)
}


func (g Group) Less(i, j int) bool {
	return g.members[i].GetHexString() < g.members[j].GetHexString()
}


func (g Group) Swap(i, j int) {
	g.members[i], g.members[j] = g.members[j], g.members[i]
}

func newGroup(id string, members []NodeID) *Group {

	g := &Group{id: id, members: members, needConnectNodes:make([]NodeID,0), resolvingNodes: make(map[NodeID]time.Time)}
	Logger.Debugf("new group id：%v", id)
	for i:= 0;i<len(g.members);i++ {
		Logger.Debugf("before id：%v", g.members[i].GetHexString())
	}
	sort.Sort(g)
	for i:= 0;i<len(g.members);i++ {
		Logger.Debugf("after id：%v", g.members[i].GetHexString())
	}

	g.curIndex =0
	for i:= 0;i<len(g.members);i++ {
		if g.members[i] == net.netCore.id {
			g.curIndex = i
			break
		}
	}
	Logger.Debugf("curIndex：%v", g.curIndex)

	connectCount := GroupBaseConnectNodeCount;
	if connectCount >  len(g.members) -1 {
		connectCount = len(g.members) -1
	}

	nextIndex := g.getNextIndex(g.curIndex)
	g.needConnectNodes = append(g.needConnectNodes,g.members[nextIndex])
	nextIndex = g.getNextIndex(nextIndex)
	g.needConnectNodes = append(g.needConnectNodes,g.members[nextIndex])

	peerSize := len(g.members)
	maxCount := int(math.Sqrt(float64(peerSize)));
	maxCount -=  len(g.needConnectNodes)
	step :=  len(g.members)/ maxCount
	for i:=0;i<maxCount ;i++ {
		nextIndex += step
		if nextIndex >= len(g.members) {
			nextIndex %= len(g.members)
		}
		g.needConnectNodes = append(g.needConnectNodes,g.members[nextIndex])
	}


	//for i:= 0;i<len(g.members);i++ {
	//	g.needConnectNodes = append(g.needConnectNodes,g.members[i])
	//}

	for i:= 0;i<len(g.needConnectNodes);i++ {
		Logger.Debugf("needConnectNodes  id：%v", g.needConnectNodes[i].GetHexString())
	}
	return g
}

func (g Group) getNextIndex(index int) int {
	index = index +1
	if index >= len(g.members) {
		index=0
	}
	return index
}

func (g *Group) doRefresh() {
	memberSize := len(g.needConnectNodes)

	Logger.Debugf("Group doRefresh  id： %v", g.id)

	for i := 0; i < memberSize; i++ {
		id := g.needConnectNodes[i]
		if id == net.netCore.id {
			continue
		}

		p := net.netCore.peerManager.peerByID(id)
		if p != nil {
			continue
		}
		node := net.netCore.kad.find(id)
		if node != nil && node.Ip != nil && node.Port > 0 {
			Logger.Debugf("Group doRefresh node found in KAD id：%v ip: %v  port:%v", id.GetHexString(), node.Ip, node.Port)
			go net.netCore.ping(node.Id, &nnet.UDPAddr{IP: node.Ip, Port: int(node.Port)})
		} else {
			go net.netCore.ping(id, nil)

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
	go net.netCore.kad.resolve(id)
}

func (g *Group) send(packet *bytes.Buffer) {

	connected := 0
	kad := 0
	other := 0
	for i := 0; i < len(g.needConnectNodes); i++ {
		id := g.needConnectNodes[i]
		if id == net.netCore.id {
			continue
		}
		p := net.netCore.peerManager.peerByID(id)
		if p != nil {
			connected +=1
			go net.netCore.peerManager.write(id, &nnet.UDPAddr{IP: p.Ip, Port: int(p.Port)}, packet)
		} else {
			node := net.netCore.kad.find(id)
			if node != nil && node.Ip != nil && node.Port > 0 {
				Logger.Debugf("sendGroup node not connected ,but found in KAD : id：%v ip: %v  port:%v", id.GetHexString(), node.Ip, node.Port)
				kad +=1
				go net.netCore.peerManager.write(node.Id, &nnet.UDPAddr{IP: node.Ip, Port: int(node.Port)}, packet)
			} else {
				Logger.Debugf("sendGroup node not connected and not found in KAD : id：%v", id.GetHexString())

				other+=1
				go net.netCore.peerManager.write(id, nil, packet)
			}
		}
	}
	Logger.Debugf("SendGroup total :%v connected:%v kad:%v other:%v", len(g.members),connected,kad,other)

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
	go gm.loop()
	return gm
}

//AddGroup 添加组
func (gm *GroupManager) addGroup(ID string, members []NodeID) *Group {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	Logger.Debugf("AddGroup node id:%v len:%v", ID, len(members))

	g := newGroup(ID, members)
	gm.groups[ID] = g
	go gm.doRefresh()
	return g
}

//RemoveGroup 移除组
func (gm *GroupManager) removeGroup(id string) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()


	Logger.Debugf("removeGroup :%v.",id)

	//g := gm.groups[id]
	//if g == nil {
	//	Logger.Debugf("removeGroup not found group.")
	//	return
	//}
	//memberSize := len(g.members)
	//
	//for i := 0; i < memberSize; i++ {
	//	id := g.members[i]
	//	if id == net.netCore.id {
	//		continue
	//	}
	//
	//	node := net.netCore.kad.find(id)
	//	if node == nil {
	//		net.netCore.peerManager.disconnect(id)
	//	}
	//}
	delete(gm.groups, id)
}

func (gm *GroupManager) loop() {

	const refreshInterval = 5 * time.Second

	var (
		refresh = time.NewTicker(refreshInterval)
	)
	defer refresh.Stop()
	for {
		select {
		case <-refresh.C:
			go gm.doRefresh()
		}
	}

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
func (gm *GroupManager) sendGroup(id string, packet *bytes.Buffer) {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	Logger.Debugf("SendGroup  id:%v", id)
	g := gm.groups[id]
	if g == nil {
		Logger.Debugf("SendGroup not found group.")
		return
	}
	g.send(packet)

	return
}
