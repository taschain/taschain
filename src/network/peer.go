package network

import (
	"bytes"
	"container/list"
	"math"
	mrand "math/rand"
	nnet "net"
	"sync"
	"time"
)

//Peer 节点连接对象
type Peer struct {
	Id         NodeID
	seesionId  uint32
	Ip         nnet.IP
	Port       int
	sendList   *list.List
	recvList   *list.List
	expiration uint64
	mutex      sync.RWMutex
	connecting bool
}

func newPeer(Id NodeID, seesionId uint32) *Peer {

	p := &Peer{Id: Id, seesionId: seesionId, sendList: list.New(), recvList: list.New()}

	return p
}

func (p *Peer) addData(data []byte) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	b := bytes.NewBuffer(nil)
	b.Write(data)
	p.recvList.PushBack(b)
}

func (p *Peer) addDataToHead(data *bytes.Buffer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.recvList.PushFront(data)
}

func (p *Peer) popData() *bytes.Buffer {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.recvList.Len() == 0 {
		return nil
	}
	buf := p.recvList.Front().Value.(*bytes.Buffer)
	p.recvList.Remove(p.recvList.Front())

	return buf
}

func (p *Peer) resetData() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.recvList = list.New()
}

func (p *Peer) isEmpty() bool {

	empty := true
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if p.recvList.Len() > 0 {
		empty = false
	}

	return empty
}

func (p *Peer) getDataSize() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	size := 0
	for e := p.recvList.Front(); e != nil; e = e.Next() {
		buf := e.Value.(*bytes.Buffer)
		size += buf.Len()
	}

	return size
}

//PeerManager 节点连接管理
type PeerManager struct {
	peers              map[uint64]*Peer //key为网络ID
	mutex              sync.RWMutex
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
		p = newPeer(toid, 0)
		p.sendList.PushBack(packet)
		p.expiration = 0
		p.connecting = false
		pm.addPeer(netId, p)
	}
	if p.seesionId > 0 {
		Logger.Infof("P2PSend Id:%v session:%v size %v", toid.GetHexString(), p.seesionId, len(packet.Bytes()))
		P2PSend(p.seesionId, packet.Bytes())
	} else {

		if ((toaddr != nil && toaddr.IP != nil && toaddr.Port > 0) || pm.natTraversalEnable) && !p.connecting {
			p.expiration = uint64(time.Now().Add(connectTimeout).Unix())
			p.connecting = true
			if toaddr != nil {
				p.Ip = toaddr.IP
				p.Port = toaddr.Port
			}
			p.sendList.PushBack(packet)

			if pm.natTraversalEnable {
				P2PConnect(netId, NatServerIp, NatServerPort)
				Logger.Infof("P2PConnect[nat]: %v ", toid.GetHexString())
			} else {
				P2PConnect(netId, toaddr.IP.String(), uint16(toaddr.Port))
				Logger.Infof("P2PConnect[direct]: id: %v ip: %v port:%v ", toid.GetHexString(), toaddr.IP.String(), uint16(toaddr.Port))
			}
		} else if p.connecting == true {
			if p.sendList.Len() > 10 {
				p.sendList = list.New()
			}
			p.sendList.PushBack(packet)
			Logger.Infof("write  error : %v ", toid.GetHexString())
		}
	}

	return nil
}

//newConnection 处理连接成功的回调
func (pm *PeerManager) newConnection(id uint64, session uint32, p2pType uint32, isAccepted bool) {

	p := pm.peerByNetID(id)
	if p == nil {
		p = newPeer(NodeID{}, session)
		p.expiration = uint64(time.Now().Add(connectTimeout).Unix())
		pm.addPeer(id, p)
	} else if session > 0 {
		if p.seesionId == 0 {
			p.recvList = list.New()
			p.seesionId = session
		}
	}
	p.connecting = false

	for e := p.sendList.Front(); e != nil; e = e.Next() {
		buf:= e.Value.(*bytes.Buffer)
		P2PSend(p.seesionId, buf.Bytes())
	}
	p.sendList = list.New()
	Logger.Infof("newConnection node id:%v  netid :%v session:%v isAccepted:%v ", p.Id.GetHexString(), id, session, isAccepted)
}

//OnDisconnected 处理连接断开的回调
func (pm *PeerManager) OnDisconnected(id uint64, session uint32, p2pCode uint32) {
	p := pm.peerByNetID(id)
	if p != nil {

		Logger.Infof("OnDisconnected id：%v ip:%v port:%v ", p.Id.GetHexString(), p.Ip, p.Port)

		p.connecting = false
		if p.seesionId == session {
			p.seesionId = 0
		}
	} else {
		Logger.Infof("OnDisconnected net id：%v session:%v port:%v code:%v", id, session, p2pCode)
	}
}

func (pm *PeerManager) disconnect(id NodeID) {
	netID := netCoreNodeID(id)

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	p, _ := pm.peers[netID]
	if p != nil {

		Logger.Infof("disconnect ip:%v port:%v ", p.Ip, p.Port)

		p.connecting = false
		delete(pm.peers, netID)
	}
}

//OnChecked 网络类型检查
func (pm *PeerManager) OnChecked(p2pType uint32, privateIp string, publicIp string) {
	//nc.ourEndPoint = MakeEndPoint(&net.UDPAddr{Ip: net.ParseIP(publicIp), Port: 8686}, 8686)
}

//SendDataToAll 向所有已经连接的节点发送自定义数据包
func (pm *PeerManager) SendAll(packet *bytes.Buffer) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	for _, p := range pm.peers {
		if p.seesionId > 0 {
			P2PSend(p.seesionId, packet.Bytes())
		}
	}

	return
}

//BroadcastRandom
func (pm *PeerManager) BroadcastRandom(packet *bytes.Buffer) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	Logger.Infof("BroadcastRandom total peer size:%v", len(pm.peers))

	var availablePeers []*Peer

	for _, p := range pm.peers {
		if p.seesionId > 0 {
			availablePeers = append(availablePeers, p)
		}
	}
	peerSize := len(availablePeers)
	maxCount := int(math.Sqrt(float64(peerSize)))
	if maxCount < 2 {
		maxCount = 2
	}

	if len(availablePeers) < maxCount {
		for _, p := range availablePeers {
			Logger.Infof("BroadcastRandom send node id:%v", p.Id.GetHexString())

			P2PSend(p.seesionId, packet.Bytes())
		}
	} else {
		nodesHasSend := make(map[int]bool)
		rand := mrand.New(mrand.NewSource(time.Now().Unix()))

		for i := 0; i < peerSize && len(nodesHasSend) < maxCount; i++ {
			peerIndex := rand.Intn(peerSize)
			if nodesHasSend[peerIndex] == true {
				continue
			}
			nodesHasSend[peerIndex] = true
			p := availablePeers[peerIndex]
			Logger.Infof("BroadcastRandom send node id:%v", p.Id.GetHexString())

			P2PSend(p.seesionId, packet.Bytes())
		}
	}

	return
}

func (pm *PeerManager) print() {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	totolRecvBufferSize := 0
	for _, p := range pm.peers {
		var rtt uint32
		var pendingSendBuffer uint32

		if p.seesionId > 0 {
			rtt = P2PSessionRxrtt(p.seesionId)
			pendingSendBuffer = P2PSessionNsndbuf(p.seesionId)
		}
		totolRecvBufferSize += p.getDataSize()

		Logger.Infof("id:%v session:%v  ip:%v  port:%v   rtt:%v, PendingBufferCount:%v", p.Id.GetHexString(), p.seesionId, p.Ip, p.Port, rtt, pendingSendBuffer)
	}
	Logger.Infof("PeerManager Print peer size:%v totolRecvBufferSize:%v", len(pm.peers), totolRecvBufferSize)

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
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	p, _ := pm.peers[netId]
	return p
}

func (pm *PeerManager) addPeer(netId uint64, peer *Peer) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	pm.peers[netId] = peer

}
