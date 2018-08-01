package network

import (
	"bytes"
	nnet "net"
	"time"
	"sync"
)

//Peer 节点连接对象
type Peer struct {
	Id         NodeID
	seesionId  uint32
	Ip     	nnet.IP
	Port    int
	sendList   []*bytes.Buffer
	dataBuffer *bytes.Buffer
	expiration uint64
	mutex sync.Mutex
	connecting bool
}

func newPeer(Id NodeID, seesionId uint32) *Peer {

	p := &Peer{Id: Id, seesionId: seesionId, sendList: make([]*bytes.Buffer, 0)}

	return p
}

func (p*Peer ) addData(data []byte) {


	p.mutex.Lock()
	if p.dataBuffer == nil {
		p.dataBuffer = bytes.NewBuffer(nil)
		p.dataBuffer.Write(data)
	} else {
		p.dataBuffer.Write(data)
	}

	p.mutex.Unlock()
}

func (p *Peer ) addDataToHead(data []byte) {
	p.mutex.Lock()
	if p.dataBuffer == nil {
		p.dataBuffer = bytes.NewBuffer(nil)
		p.dataBuffer.Write(data)
	} else {
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


func (pm *PeerManager) write(toid NodeID, toaddr *nnet.UDPAddr, packet *bytes.Buffer) error {
	netId := netCoreNodeID(toid)
	p := pm.peerByNetID(netId)
	if p == nil {
		p = &Peer{Id: toid, seesionId: 0, sendList: make([]*bytes.Buffer, 0)}
		p.sendList = append(p.sendList, packet)
		p.expiration = 0
		p.connecting = false
		pm.addPeer(netId,p)
	}
	if  p.seesionId > 0 {
		Logger.Infof("P2PSend %v len: %v ", p.seesionId, packet.Len())
		P2PSend(p.seesionId, packet.Bytes())
	} else {

		if toaddr != nil && toaddr.IP != nil && toaddr.Port>0  && !p.connecting {
			p.expiration = uint64(time.Now().Add(connectTimeout).Unix())
			p.connecting = true
			p.Ip = toaddr.IP
			p.Port = toaddr.Port
			p.sendList = append(p.sendList, packet)

			if pm.natTraversalEnable {
				P2PConnect(netId, "47.98.212.107", 70)
			} else {
				P2PConnect(netId, toaddr.IP.String(), uint16(toaddr.Port))
			}
			Logger.Infof("[P2P]Connect：ID: %v IP: %v Port:%v ",toid.GetHexString(), toaddr.IP.String(), uint16(toaddr.Port))
		}
	}

	return nil
}

//OnConnected 处理连接成功的回调
func (pm *PeerManager) OnConnected(id uint64, session uint32, p2pType uint32) {

	Logger.Infof("OnConnected netid :%v session:%v ", id,session)

	p := pm.peerByNetID(id)
	if p == nil {
		p = &Peer{Id: NodeID{}, seesionId: session, sendList: make([]*bytes.Buffer, 0)}
		p.expiration = uint64(time.Now().Add(connectTimeout).Unix())
		pm.addPeer(id,p)
	} else if session >0{
		p.seesionId = session
	}
	p.connecting = false

	if p != nil {
		for i := 0; i < len(p.sendList); i++ {
			P2PSend(p.seesionId, p.sendList[i].Bytes())
		}
		p.sendList = make([]*bytes.Buffer, 0)
	}
}

//OnDisconnected 处理连接断开的回调
func (pm *PeerManager) OnDisconnected(id uint64, session uint32, p2pCode uint32) {
	p := pm.peerByNetID(id)
	if p != nil {

		Logger.Infof("OnDisconnected ip:%v port:%v ", p.Ip,p.Port)

		p.connecting = false
		if p.seesionId == session {
			p.seesionId = 0
		}
	}
}

//OnChecked 网络类型检查
func (pm *PeerManager) OnChecked(p2pType uint32, privateIp string, publicIp string) {
	//nc.ourEndPoint = MakeEndPoint(&net.UDPAddr{Ip: net.ParseIP(publicIp), Port: 8686}, 8686)
}

//SendDataToAll 向所有已经连接的节点发送自定义数据包
func (pm *PeerManager) SendAll(packet *bytes.Buffer) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	
	for _, p := range pm.peers {
		if p.seesionId > 0 {
			//pm.write(p.Id, nil, packet)
			go P2PSend(p.seesionId, packet.Bytes())

		}
	}

	return
}

func (pm *PeerManager) print() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	Logger.Infof("PeerManager Print peer size:%v", len(pm.peers))

	for _, p := range pm.peers {
		Logger.Infof("id:%v session:%v  ip:%v  port:%v", p.Id.GetHexString(),p.seesionId,p.Ip,p.Port)
	}
	return
}


func (pm *PeerManager) peerByID(id NodeID) *Peer {
	netID := netCoreNodeID(id)

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	p, _ := pm.peers[netID]
	return p
}


func (pm *PeerManager) peerByNetID(netId uint64) *Peer {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	p, _ := pm.peers[netId]
	return p
}

func (pm *PeerManager) addPeer(netId uint64, peer *Peer) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.peers[netId] = peer

}
