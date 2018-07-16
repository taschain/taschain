package network

import (
	"bytes"
	"fmt"
	"net"
	"time"
)

// Group 组对象
type Group struct {
	ID      string
	members []NodeID
	nodes   map[NodeID]*Node
}

func newGroup(ID string, members []NodeID) *Group {

	g := &Group{ID: ID, members: members, nodes: make(map[NodeID]*Node)}

	return g
}

func (g *Group) addGroup(node *Node) {
	g.nodes[node.ID] = node
}

func (g *Group) doRefresh() {

	//fmt.Printf("Group doRefresh id： %v\n", g.ID)

	for i := 0; i < len(g.members); i++ {
		id := g.members[i]
		if id == Network.netCore.id {
			continue
		}
		node, ok := g.nodes[id]
		if node != nil && ok {
			continue
		}
		node = Network.netCore.kad.Resolve(id)
		if node != nil {
			g.nodes[id] = node
		//	fmt.Printf("Group Resolve id：%v ip: %v  port:%v\n", id, node.IP, node.Port)
		} else {
		//	fmt.Printf("Group Resolve id：%v  nothing!!!\n", id)

		}
	}
}

//GroupManager 组管理
type GroupManager struct {
	groups map[string]*Group
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
	g := newGroup(ID, members)
	gm.groups[ID] = g
	//go gm.doRefresh()
	return g
}

//RemoveGroup 移除组
func (gm *GroupManager) RemoveGroup(ID string) {
	//todo
}

// loop schedules refresh.
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

	for _, group := range gm.groups {

		group.doRefresh()

		//hello := ""
		//for i := 0; i < 6; i++ {
		//	hello += "GROUP"
		//}
		//GetNetCore().SendDataToGroup(group.ID, []byte(hello))
	}
}

//SendDataToGroup 向所有已经连接的组内节点发送自定义数据包
func (gm *GroupManager) SendDataToGroup(id string, packet *bytes.Buffer) {
	fmt.Printf("SendDataToGroup  id:%v\n", id)
	g := gm.groups[id]
	if g == nil {
		fmt.Printf("SendDataToGroup not find group\n")
	}
	for _, node := range g.nodes {
		if node != nil {

			fmt.Printf("SendDataToGroup node ip:%v port:%v\n", node.IP, node.Port)

			Network.netCore.PM.write(node.ID, &net.UDPAddr{IP: node.IP, Port: int(node.Port)}, packet)
		}
	}
	return
}
