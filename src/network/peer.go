package network

import (
	"bytes"
	"net"
	"time"
	"fmt"
	"sync"
)

//Peer 节点连接对象
type Peer struct {
	ID         NodeID
	seesionID  uint32
	IP     	net.IP
	Port    int
	sendList   []*bytes.Buffer
	dataBuffer *bytes.Buffer
	expiration uint64
	mutex sync.Mutex
	connecting bool
}

func newPeer(ID NodeID, seesionID uint32) *Peer {

	p := &Peer{ID: ID, seesionID: seesionID, sendList: make([]*bytes.Buffer, 0)}

	return p
}



func (p*Peer ) addData(data []byte) {


	p.mutex.Lock()
	if p.dataBuffer == nil {
		//fmt.Printf("addData ID：%v  len: %v\n ", p.ID.B58String(), len(data))
		p.dataBuffer = bytes.NewBuffer(nil)
		p.dataBuffer.Write(data)
	} else {
		//fmt.Printf("addData ID：%v  len: %v old len :%v\n ", p.ID.B58String(), len(data),p.dataBuffer.Len())
		p.dataBuffer.Write(data)
	}

	p.mutex.Unlock()
}

func (p*Peer ) addDataToHead(data []byte) {
	p.mutex.Lock()
	if p.dataBuffer == nil {
		//fmt.Printf("addDataToHead ID：%v  len: %v\n ", p.ID.B58String(), len(data))

		p.dataBuffer = bytes.NewBuffer(nil)
		p.dataBuffer.Write(data)
	} else {
		//fmt.Printf("addDataToHead ID：%v  len: %v old len :%v\n ", p.ID.B58String(), len(data),p.dataBuffer.Len())

		newBuf :=  bytes.NewBuffer(nil)
		newBuf.Write(data)
		newBuf.Write(p.dataBuffer.Bytes())
		p.dataBuffer =  newBuf
	}
	p.mutex.Unlock()
}

func (p*Peer ) getData() *bytes.Buffer{
	p.mutex.Lock()
	buf:= p.dataBuffer;
	p.dataBuffer = nil;
	p.mutex.Unlock()
	return  buf
}

func (p*Peer ) isEmpty() bool{
	empty := true
	p.mutex.Lock()
	if p.dataBuffer != nil && p.dataBuffer.Len() >0 {
		empty = false
	}
	p.mutex.Unlock()
	return  empty
}


//PeerManager 节点连接管理
type PeerManager struct {
	peers map[uint64]*Peer //key为网络ID
	mutex sync.Mutex
	natTraversalEnable bool
}

func newPeerManager() *PeerManager {

	pm := &PeerManager{
		peers: make(map[uint64]*Peer),
	}

	return pm
}

func (pm *PeerManager) write(toid NodeID, toaddr *net.UDPAddr, packet *bytes.Buffer) error {
	netID := NetCoreNodeID(toid)
	p := pm.peerByNetID(netID)
	if p == nil {
		p = &Peer{ID: toid, seesionID: 0, sendList: make([]*bytes.Buffer, 0)}
		p.sendList = append(p.sendList, packet)
		p.expiration = 0
		p.connecting = false
		pm.addPeer(netID,p)
	}
	if  p.seesionID > 0 {
		//fmt.Printf("P2PSend %v len: %v\n ", p.seesionID, packet.Len())
		P2PSend(p.seesionID, packet.Bytes())
	} else {

		if toaddr != nil && toaddr.IP != nil && toaddr.Port>0  && !p.connecting {
			p.expiration = uint64(time.Now().Add(connectTimeout).Unix())
			p.connecting = true
			p.IP = toaddr.IP
			p.Port = toaddr.Port
			p.sendList = append(p.sendList, packet)

			if pm.natTraversalEnable {
				P2PConnect(netID, "47.98.212.107", 70)
			} else {
				P2PConnect(netID, toaddr.IP.String(), uint16(toaddr.Port))
			}

		}
	}

	return nil
}

//OnConnected 处理连接成功的回调
func (pm *PeerManager) OnConnected(id uint64, session uint32, p2pType uint32) {
	p := pm.peerByNetID(id)
	if p == nil {
		p = &Peer{ID: NodeID{}, seesionID: session, sendList: make([]*bytes.Buffer, 0)}
		p.expiration = uint64(time.Now().Add(connectTimeout).Unix())
		pm.addPeer(id,p)
	} else if session >0{
		p.seesionID = session
	}
	p.connecting = false

	if p != nil {
		for i := 0; i < len(p.sendList); i++ {
			P2PSend(p.seesionID, p.sendList[i].Bytes())
		}
		p.sendList = make([]*bytes.Buffer, 0)
	}
}

//OnDisconnected 处理连接断开的回调
func (pm *PeerManager) OnDisconnected(id uint64, session uint32, p2pCode uint32) {
	p := pm.peerByNetID(id)
	if p != nil {

		//fmt.Printf("OnDisconnected ip:%v port:%v\n ", p.IP,p.Port)

		p.connecting = false
		if p.seesionID == session {
			p.seesionID = 0
		}
	}
}

//OnChecked 网络类型检查
func (pm *PeerManager) OnChecked(p2pType uint32, privateIP string, publicIP string) {
	//nc.ourEndPoint = MakeEndPoint(&net.UDPAddr{IP: net.ParseIP(publicIP), Port: 8686}, 8686)
}

//SendDataToAll 向所有已经连接的节点发送自定义数据包
func (pm *PeerManager) SendDataToAll(packet *bytes.Buffer) {
//	fmt.Printf("SendDataToAll  peer size:%v\n", len(pm.peers))


	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	for _, p := range pm.peers {
		if p.seesionID > 0 {
			//pm.write(p.ID, nil, packet)
			go P2PSend(p.seesionID, packet.Bytes())

		}
	}

	return
}

func (pm *PeerManager) print() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	fmt.Printf("PeerManager Print peer size:%v\n", len(pm.peers))

	for _, p := range pm.peers {
		fmt.Printf("id:%v session:%v  ip:%v  port:%v\n", p.ID.GetHexString(),p.seesionID,p.IP,p.Port)
	}
	return
}


func (pm *PeerManager) peerByID(id NodeID) *Peer {
	netID := NetCoreNodeID(id)

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	p, _ := pm.peers[netID]
	return p
}


func (pm *PeerManager) peerByNetID(netID uint64) *Peer {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	p, _ := pm.peers[netID]
	return p
}

func (pm *PeerManager) addPeer(netID uint64, peer *Peer) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.peers[netID] = peer

}
