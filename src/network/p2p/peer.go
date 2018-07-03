package p2p

import (
	"bytes"
	fmt "fmt"
	"net"
	"time"
)

//Peer 节点连接对象
type Peer struct {
	ID         NodeID
	seesionID  uint32
	IP     	net.IP
	Port    int
	sendList   []*bytes.Buffer
	recvBuffer *bytes.Buffer
	expiration uint64
}

func newPeer(ID NodeID, seesionID uint32) *Peer {

	p := &Peer{ID: ID, seesionID: seesionID, sendList: make([]*bytes.Buffer, 0)}

	return p
}

//PeerManager 节点连接管理
type PeerManager struct {
	peers map[uint64]*Peer //key为网络ID
}

func newPeerManager() *PeerManager {

	pm := &PeerManager{
		peers: make(map[uint64]*Peer),
	}

	return pm
}

func (pm *PeerManager) write(toid NodeID, toaddr *net.UDPAddr, packet *bytes.Buffer) error {
	netID := NetCoreNodeID(toid).Sum64()

	//fmt.Printf("write data ID：%v  len: %v\n ", netID, packet.Len())
	//if toaddr != nil {
	//	fmt.Printf("ip:%v port:%v\n ", toaddr.IP.String(), uint16(toaddr.Port))
	//}

	p, ok := pm.peers[netID]
	if !ok && toaddr != nil {
		//P2PConnect(netID, "47.96.186.139", 70)
		P2PConnect(netID, toaddr.IP.String(), uint16(toaddr.Port))
		p = &Peer{ID: toid, seesionID: 0, sendList: make([]*bytes.Buffer, 0)}
		p.sendList = append(p.sendList, packet)
		p.expiration = uint64(time.Now().Add(connectTimeout).Unix())
		p.IP = toaddr.IP
		p.Port = toaddr.Port
		pm.peers[netID] = p

	} else if ok && p.seesionID == 0 && expired(p.expiration) && toaddr != nil {
		p.expiration = uint64(time.Now().Add(connectTimeout).Unix())
		//P2PConnect(netID, "47.96.186.139", 70)
		P2PConnect(netID, toaddr.IP.String(), uint16(toaddr.Port))

	} else if ok && p.seesionID > 0 {
		fmt.Printf("P2PSend %v len: %v\n ", p.seesionID, packet.Len())
		P2PSend(p.seesionID, packet.Bytes())
	} else {
		fmt.Printf("error : write data ID：%v  len: %v\n ", netID, packet.Len())
	}
	return nil
}

//OnConnected 处理连接成功的回调
func (pm *PeerManager) OnConnected(id uint64, session uint32, p2pType uint32) {

	p, ok := pm.peers[id]
	if !ok {
		p = &Peer{ID: NodeID{}, seesionID: session, sendList: make([]*bytes.Buffer, 0)}
		p.expiration = uint64(time.Now().Add(connectTimeout).Unix())
		pm.peers[id] = p
	} else {
		p.seesionID = session
	}
	if p != nil {
		for i := 0; i < len(p.sendList); i++ {
			P2PSend(p.seesionID, p.sendList[i].Bytes())
		}
		p.sendList = make([]*bytes.Buffer, 0)
	}
}

//OnDisconnected 处理连接断开的回调
func (pm *PeerManager) OnDisconnected(id uint64, session uint32, p2pCode uint32) {

	p, ok := pm.peers[id]
	if !ok {
		//fmt.Printf("OnDisconnected : no peer id:%v mynid:%v\n ", id, nc.nid)
	} else {
		p.seesionID = 0
	}
}

//OnChecked 网络类型检查
func (pm *PeerManager) OnChecked(p2pType uint32, privateIP string, publicIP string) {
	//nc.ourEndPoint = MakeEndPoint(&net.UDPAddr{IP: net.ParseIP(publicIP), Port: 8686}, 8686)
}

//SendDataToAll 向所有已经连接的节点发送自定义数据包
func (pm *PeerManager) SendDataToAll(packet *bytes.Buffer) {
	//fmt.Printf("SendDataToAll  peer size:%v\n", len(pm.peers))

	for _, p := range pm.peers {
		if p.seesionID > 0 {
			pm.write(p.ID, nil, packet)
		}
	}
	return
}


func (pm *PeerManager) peerByID(id NodeID) *Peer {
	netID := NetCoreNodeID(id).Sum64()

	p, _ := pm.peers[netID]
	return p
}


func (pm *PeerManager) peerByNetID(netID uint64) *Peer {

	p, _ := pm.peers[netID]
	return p
}

func (pm *PeerManager) addPeer(netID uint64, peer *Peer) {

	pm.peers[netID] = peer
}
