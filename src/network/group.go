package network

import (
	"bytes"
	nnet "net"
	"sync"
	"time"
)

// Group 组对象
type Group struct {
	id             string
	members        []NodeID
	resolvingNodes map[NodeID]time.Time
}

func newGroup(id string, members []NodeID) *Group {

	g := &Group{id: id, members: members,resolvingNodes: make(map[NodeID]time.Time)}

	return g
}

func (g *Group) doRefresh() {
	memberSize := len(g.members)

	Logger.Debugf("Group doRefresh  id： %v", g.id)

	for i := 0; i < memberSize; i++ {
		id := g.members[i]
		if id == net.netCore.id {
			continue
		}

		p := net.netCore.peerManager.peerByID(id)
		if p != nil {
			continue
		}
		node := net.netCore.kad.find(id)
		if node != nil && node.Ip != nil && node.Port > 0{
			Logger.Debugf("Group doRefresh node found in KAD id：%v ip: %v  port:%v", id.GetHexString(), node.Ip, node.Port)
			go net.netCore.ping(node.Id, &nnet.UDPAddr{IP: node.Ip, Port: int(node.Port)})
		} else {
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

	for i := 0; i < len(g.members); i++ {
		id := g.members[i]
		if id == net.netCore.id {
			continue
		}
		p := net.netCore.peerManager.peerByID(id)
		if p != nil && p.Ip != nil && p.Port > 0 {
			Logger.Debugf("sendGroup node is connected : id：%v ip: %v  port:%v", id.GetHexString(), p.Ip, p.Port)
			go net.netCore.peerManager.write(id, &nnet.UDPAddr{IP: p.Ip, Port: int(p.Port)}, packet)
		} else {
			node := net.netCore.kad.find(id)
			if node != nil {
				Logger.Debugf("sendGroup node not connected ,but find in KAD : id：%v ip: %v  port:%v", id.GetHexString(), node.Ip, node.Port)
				go net.netCore.peerManager.write(node.Id, &nnet.UDPAddr{IP: node.Ip, Port: int(node.Port)}, packet)
			} else {
				go net.netCore.peerManager.write(id,nil, packet)
				Logger.Debugf("sendGroup node not connected  & can not find in KAD , resolve ....  id：%v ", id.GetHexString())
				g.resolve(id)
			}
		}
	}
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
	g := gm.groups[id]
	if g == nil {
		Logger.Debugf("removeGroup not found group.")
		return
	}
	memberSize := len(g.members)

	for i := 0; i < memberSize; i++ {
		id := g.members[i]
		if id == net.netCore.id {
			continue
		}

		node := net.netCore.kad.find(id)
		if node == nil {
			net.netCore.peerManager.disconnect(id)
		}
	}
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
