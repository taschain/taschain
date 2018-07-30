package network

import (
	"bytes"
	nnet "net"
	"time"
	"sync"
)

// Group 组对象
type Group struct {
	ID      string
	members []NodeID
	nodes   map[NodeID]*Node
	mutex sync.Mutex
}

func newGroup(ID string, members []NodeID) *Group {

	g := &Group{ID: ID, members: members, nodes: make(map[NodeID]*Node)}

	return g
}

func (g *Group) addGroup(node *Node) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.nodes[node.Id] = node
}

func (g *Group) doRefresh() {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	if len(g.nodes) ==  len(g.members) {
		return
	}
	Logger.Debugf("Group doRefresh id： %v\n", g.ID)

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
			Logger.Debugf("Group Resolve id：%v ip: %v  port:%v\n", id, node.Ip, node.Port)
			go net.netCore.ping(node.Id,&nnet.UDPAddr{IP: node.Ip, Port: int(node.Port)})
		} else {
			Logger.Debugf("Group Resolve id：%v  nothing!!!\n", id)
		}
	}
}

func (g *Group) Send( packet *bytes.Buffer) {
	g.mutex.Lock()
	defer g.mutex.Unlock()


	for _, node := range g.nodes {
		if node != nil {

			Logger.Debugf("SendGroup node ip:%v port:%v\n", node.Ip, node.Port)

			go net.netCore.peerManager.write(node.Id, &nnet.UDPAddr{IP: node.Ip, Port: int(node.Port)}, packet)
		}
	}
	return
}

//GroupManager 组管理
type GroupManager struct {
	groups map[string]*Group
	mutex sync.Mutex
}

func newGroupManager() *GroupManager {

	gm := &GroupManager{
		groups: make(map[string]*Group),
	}
	go gm.loop()
	return gm
}

//AddGroup 添加组
func (gm *GroupManager) AddGroup(ID string, members []NodeID) *Group {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	Logger.Debugf("AddGroup node id:%s len:%v\n", ID,len(members))

	g := newGroup(ID, members)
	gm.groups[ID] = g
	go gm.doRefresh()
	return g
}

//RemoveGroup 移除组
func (gm *GroupManager) RemoveGroup(ID string) {
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
	//fmt.Printf("groupManager doRefresh \n")
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	for _, group := range gm.groups {
		group.doRefresh()
	}
}

//SendGroup 向所有已经连接的组内节点发送自定义数据包
func (gm *GroupManager) SendGroup(id string, packet *bytes.Buffer) {
	gm.mutex.Lock()
	defer gm.mutex.Unlock()

	Logger.Debugf("SendGroup  id:%v\n", id)
	g := gm.groups[id]
	if g == nil {
		Logger.Debugf("SendGroup not find group\n")
		return
	}
	g.Send(packet)

	return
}
