package network

import (
	"bytes"
	nnet "net"
	"time"
	"sync"
)

// Group 组对象
type Group struct {
	id      string
	members []NodeID
	nodes   map[NodeID]*Node
	mutex sync.Mutex
}

func newGroup(id string, members []NodeID) *Group {

	g := &Group{id: id, members: members, nodes: make(map[NodeID]*Node)}

	return g
}

func (g *Group) addNode(node *Node) {
	g.nodes[node.Id] = node
}

func (g *Group) doRefresh() {
	if len(g.nodes) ==  len(g.members) {
		return
	}
	Logger.Debugf("Group doRefresh id： %v", g.id)

	for i := 0; i < len(g.members); i++ {
		id := g.members[i]
		if id == net.netCore.id {
			continue
		}
		node, ok := g.nodes[id]
		if node != nil && ok {
			continue
		}
		node = net.netCore.kad.resolve(id)
		if node != nil {
			g.nodes[id] = node
			Logger.Debugf("Group Resolve id：%v ip: %v  port:%v", id, node.Ip, node.Port)
			go net.netCore.ping(node.Id,&nnet.UDPAddr{IP: node.Ip, Port: int(node.Port)})
		} else {
			Logger.Debugf("Group Resolve id：%v  nothing!!!", id)
		}
	}
}

func (g *Group) send( packet *bytes.Buffer) {

	for _, node := range g.nodes {
		if node != nil {

			Logger.Debugf("SendGroup node ip:%v port:%v", node.Ip, node.Port)

			go net.netCore.peerManager.write(node.Id, &nnet.UDPAddr{IP: node.Ip, Port: int(node.Port)}, packet)
		}
	}
	return
}

//GroupManager 组管理
type GroupManager struct {
	groups map[string]*Group
	mutex sync.RWMutex
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

	Logger.Debugf("AddGroup node id:%v len:%v", ID,len(members))

	g := newGroup(ID, members)
	gm.groups[ID] = g
	go gm.doRefresh()
	return g
}

//RemoveGroup 移除组
func (gm *GroupManager) removeGroup(ID string) {
	//todo
}


func (gm *GroupManager) loop() {

	const refreshInterval = 1 * time.Second

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
		group.doRefresh()
	}
}

//SendGroup 向所有已经连接的组内节点发送自定义数据包
func (gm *GroupManager) sendGroup(id string, packet *bytes.Buffer) {
	gm.mutex.RLock()
	defer gm.mutex.RUnlock()

	Logger.Debugf("SendGroup  id:%v", id)
	g := gm.groups[id]
	if g == nil {
		Logger.Debugf("SendGroup not find group")
		return
	}
	g.send(packet)

	return
}
