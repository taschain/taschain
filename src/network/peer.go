package network

import (
	"bytes"
	nnet "net"
	"time"
	"sync"
	mrand "math/rand"
	"math"
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
	defer p.mutex.Unlock()

	if p.dataBuffer == nil {
		p.dataBuffer = bytes.NewBuffer(nil)
		p.dataBuffer.Write(data)
	} else {
		p.dataBuffer.Write(data)
	}


}

func (p *Peer ) addDataToHead(data []byte) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.dataBuffer == nil {
		p.dataBuffer = bytes.NewBuffer(nil)
		p.dataBuffer.Write(data)
	} else {
		newBuf :=  bytes.NewBuffer(nil)
		newBuf.Write(data)
		newBuf.Write(p.dataBuffer.Bytes())
		p.dataBuffer =  newBuf
	}

}

func (p*Peer ) getData() *bytes.Buffer{
	p.mutex.Lock()
	defer p.mutex.Unlock()
	buf:= p.dataBuffer
	p.dataBuffer = nil

	return  buf
}

func (p*Peer ) isEmpty() bool{
	empty := true
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.dataBuffer != nil && p.dataBuffer.Len() >0 {
		empty = false
	}

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
		Logger.Infof("P2PSend Id:%v session:%v size %v", toid.GetHexString(), p.seesionId, len(packet.Bytes()))
		P2PSend(p.seesionId, packet.Bytes())
	} else {

		if ((toaddr != nil && toaddr.IP != nil && toaddr.Port>0) || pm.natTraversalEnable)  && !p.connecting {
			p.expiration = uint64(time.Now().Add(connectTimeout).Unix())
			p.connecting = true
			if toaddr != nil {
				p.Ip = toaddr.IP
				p.Port = toaddr.Port
			}
			p.sendList = append(p.sendList, packet)

			if pm.natTraversalEnable {
				P2PConnect(netId, NatServerIp, NatServerPort)
				Logger.Infof("P2PConnect[nat]: %v ", toid.GetHexString())
			} else {
				P2PConnect(netId, toaddr.IP.String(), uint16(toaddr.Port))
				Logger.Infof("P2PConnect[direct]: id: %v ip: %v port:%v ", toid.GetHexString(), toaddr.IP.String(), uint16(toaddr.Port))
			}
			} else {
			Logger.Infof("write  error : %v ", toid.GetHexString())
		}
	}


	return nil
}

//newConnection 处理连接成功的回调
func (pm *PeerManager) newConnection(id uint64, session uint32, p2pType uint32, isAccepted bool) {


	p := pm.peerByNetID(id)
	if p == nil {
		p = &Peer{Id: NodeID{}, seesionId: session, sendList: make([]*bytes.Buffer, 0)}
		p.expiration = uint64(time.Now().Add(connectTimeout).Unix())
		pm.addPeer(id,p)
	} else if session >0 {
		if p.seesionId ==0 {
			p.dataBuffer = nil
			p.seesionId = session
		}

	}
	p.connecting = false

	if p != nil  {
		for i := 0; i < len(p.sendList); i++ {
			P2PSend(p.seesionId, p.sendList[i].Bytes())
		}
		p.sendList = make([]*bytes.Buffer, 0)
	}

	Logger.Infof("newConnection node id:%v  netid :%v session:%v isAccepted:%v ", p.Id.GetHexString(),id,session,isAccepted)

}

//OnDisconnected 处理连接断开的回调
func (pm *PeerManager) OnDisconnected(id uint64, session uint32, p2pCode uint32) {
	p := pm.peerByNetID(id)
	if p != nil {

		Logger.Infof("OnDisconnected id：%v ip:%v port:%v ",p.Id.GetHexString(), p.Ip,p.Port)

		p.connecting = false
		if p.seesionId == session {
			p.seesionId = 0
		}
	} else {
		Logger.Infof("OnDisconnected net id：%v session:%v port:%v code:%v", id,session,p2pCode)
	}
}

func (pm *PeerManager) disconnect(id NodeID) {
	netID := netCoreNodeID(id)

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	p, _ := pm.peers[netID]
	if p != nil {

		Logger.Infof("disconnect ip:%v port:%v ", p.Ip,p.Port)

		p.connecting = false
		delete(pm.peers,netID)
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
			go P2PSend(p.seesionId, packet.Bytes())
		}
	}

	return
}


//BroadcastRandom
func (pm *PeerManager) BroadcastRandom(packet *bytes.Buffer) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	Logger.Infof("BroadcastRandom total peer size:%v", len(pm.peers))


	var availablePeers[]*Peer

	for _, p := range pm.peers {
		if p.seesionId > 0 {
			availablePeers = append(availablePeers,p)
		}
	}
	peerSize :=len(availablePeers)
	maxCount := int(math.Sqrt(float64(peerSize)));
	if maxCount < 2 {
		maxCount = 2
	}

	if len(availablePeers) < maxCount {
		for _, p := range availablePeers {
			Logger.Infof("BroadcastRandom send node id:%v", p.Id.GetHexString())

			go P2PSend(p.seesionId, packet.Bytes())
		}
	} else {
		nodesHasSend := make(map[int]bool)
		rand :=mrand.New(mrand.NewSource(0))

		for i:=0;i<peerSize && len(nodesHasSend) < maxCount;i++ {
			peerIndex := rand.Intn(peerSize)
			if nodesHasSend[peerIndex] == true {
				continue
			}
			nodesHasSend[peerIndex] = true
			p:=availablePeers[peerIndex]
			Logger.Infof("BroadcastRandom send node id:%v", p.Id.GetHexString())

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
		var rtt uint32
		var pendingSendBuffer uint32

		if p.seesionId > 0 {
			rtt = P2PSessionRxrtt(p.seesionId )
			pendingSendBuffer = P2PSessionNsndbuf(p.seesionId )
		}
		Logger.Infof("id:%v session:%v  ip:%v  port:%v   rtt:%v, PendingBufferCount:%v"  , p.Id.GetHexString(),p.seesionId,p.Ip,p.Port,rtt,pendingSendBuffer)
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
