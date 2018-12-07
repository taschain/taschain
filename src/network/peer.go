//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

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

type PeerSource int32

const (
	PeerSourceUnkown PeerSource = 0
	PeerSourceKad    PeerSource = 1
	PeerSourceGroup  PeerSource = 2
)

type SendPriorityType uint32

const (
	SendPriorityHigh SendPriorityType = 0
	SendPriorityMedium  SendPriorityType = 1
	SendPriorityLow   SendPriorityType = 2
)
const MaxSendPriority = 3
const MaxPendingSend = 5
const MaxSendListSize = 256

type SendListItem struct {
	priority int
	list     *list.List
	quota    int
	curQuota int
}

func newSendListItem(priority int, quota int) *SendListItem {

	item := &SendListItem{priority: priority, quota: quota, list: list.New()}

	return item
}

type SendList struct {
	list          [MaxSendPriority]*SendListItem
	priorityTable map[uint32]SendPriorityType
	pendingSend   int
	totalQuota    int
	curQuota      int
}

func newSendList() *SendList {
	PriorityQuota := [MaxSendPriority]int{5, 3, 2}

	sl := &SendList{}

	for i := 0; i < MaxSendPriority; i++ {
		sl.list[i] = newSendListItem(i, PriorityQuota[i])
		sl.totalQuota += PriorityQuota[i]
	}

	sl.priorityTable = map[uint32]SendPriorityType{
		BlockInfoNotifyMsg: SendPriorityHigh,
		ReqBlock: SendPriorityHigh,
		BlockMsg:SendPriorityHigh,
		GroupChainCountMsg: SendPriorityHigh,
		ReqGroupMsg: SendPriorityHigh,
		GroupMsg: SendPriorityHigh,
		ChainPieceReq: SendPriorityHigh,
		ChainPiece: SendPriorityHigh,

		CastVerifyMsg: SendPriorityMedium,
		VerifiedCastMsg: SendPriorityMedium,
		ReqTransactionMsg: SendPriorityMedium,
		TransactionGotMsg: SendPriorityMedium,
		TransactionMsg: SendPriorityMedium,
		NewBlockMsg: SendPriorityMedium,
		CastRewardSignReq: SendPriorityMedium,
		CastRewardSignGot: SendPriorityMedium,

	}

	return sl
}

func (sendList *SendList) send(peer *Peer, packet *bytes.Buffer, code int) {

	if peer == nil || packet == nil {
		return
	}

	priority, isExist := sendList.priorityTable[uint32(code)]
	if !isExist {
		priority = MaxSendPriority - 1
	}
	Logger.Debugf("SendList.send  net id:%v session:%v code:%v size %v priority:%v", peer.Id.GetHexString(), peer.seesionId,code, len(packet.Bytes()),priority)

	sendListItem := sendList.list[priority]
	if sendListItem.list.Len() > MaxSendListSize {
		Logger.Debugf("SendList.send  net id:%v session:%v send list is full  drop this message!", peer.Id.GetHexString(), peer.seesionId)
		return
	}
	sendListItem.list.PushBack(packet)


	netCore.flowMeter.send(int64(code), int64(len(packet.Bytes())))
	sendList.autoSend(peer)
}

func (sendList *SendList) isSendAvailable() bool {
	return sendList.pendingSend < MaxPendingSend
}

func (sendList *SendList) autoSend(peer *Peer) {

//	Logger.Debugf("SendList.autoSend start,pendingSend:%v",sendList.pendingSend)

	if peer.seesionId == 0 || !sendList.isSendAvailable() {
		return
	}

	remain :=0
	for i := 0; i < MaxSendPriority && sendList.isSendAvailable() ; i++ {
		item := sendList.list[i]
		//Logger.Debugf("SendList.autoSend item priority:%v list len:%v  item.curQuota :%v item.quota:%v ", i,item.list.Len(),item.curQuota ,item.quota)

		for item.list.Len() > 0 && sendList.isSendAvailable() {
			e := item.list.Front()
			if e != nil &&  e.Value ==nil{
				item.list.Remove(e)
				break
			}
			buf := e.Value.(*bytes.Buffer)
			P2PSend(peer.seesionId, buf.Bytes())

			netCore.bufferPool.FreeBuffer(buf)

			item.list.Remove(e)
			sendList.pendingSend += 1

			item.curQuota += 1
			sendList.curQuota += 1

//			Logger.Debugf("SendList.autoSend SendData pendingSend:%v curQuota：%v,sendList.curQuota：%v", sendList.pendingSend,item.curQuota,sendList.curQuota)


			if item.curQuota >= item.quota {
				//Logger.Debugf("SendList.autoSend  item.curQuota >= item.quota break")
				break
			}
		}
		remain += item.list.Len()
		if sendList.curQuota >= sendList.totalQuota {
			//Logger.Debugf("SendList.autoSend sendList.curQuota >= sendList.totalQuota reset quota ")
			sendList.resetQuota()
		}

	}
	if remain > 0 && sendList.isSendAvailable() {
		//Logger.Debugf("SendList.autoSend remain > 0 && sendList.pendingSend < MaxPendingSend  reset quota and auto send")

		sendList.resetQuota()
		sendList.autoSend(peer)
	}
//	Logger.Debugf("SendList.autoSend end sendList.curQuota：%v ",sendList.curQuota)

}


func (sendList *SendList) resetQuota() {

	sendList.curQuota = 0

	for i := 0; i < MaxSendPriority; i++ {
		item := sendList.list[i]
		item.curQuota = 0
	}

}


func (sendList *SendList) getDataSize() int {
	size := 0
	for i := 0; i < MaxSendPriority ; i++ {
		item := sendList.list[i]

		for e := item.list.Front();e != nil; e = e.Next() {
			buf := e.Value.(*bytes.Buffer)
			size += buf.Len()
		}
	}
	return size
}


//Peer 节点连接对象
type Peer struct {
	Id         NodeID
	seesionId  uint32
	Ip         nnet.IP
	Port       int
	sendList   *SendList
	recvList   *list.List
	expiration uint64
	mutex      sync.RWMutex
	connecting bool
	source     PeerSource
}

func newPeer(Id NodeID, seesionId uint32) *Peer {

	p := &Peer{Id: Id, seesionId: seesionId, sendList: newSendList(), recvList: list.New(), source: PeerSourceUnkown}

	return p
}

func (p *Peer) addData(data []byte) {

	p.mutex.Lock()
	defer p.mutex.Unlock()
	b := netCore.bufferPool.GetBuffer(len(data))
	//b := &bytes.Buffer{}
	b.Write(data)
	p.recvList.PushBack(b)
	Logger.Debugf("session : %v addData size %v", p.seesionId,len(data))
}

func (p *Peer) addDataToHead(data *bytes.Buffer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.recvList.PushFront(data)
	Logger.Debugf("session : %v addDataToHead size %v", p.seesionId,data.Len())
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

func (p *Peer) onSendWaited() {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.sendList.pendingSend = 0
	Logger.Debugf("onSendWaited p.sendList.pendingSend: %v", p.sendList.pendingSend)
	p.sendList.autoSend(p)
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

func (p *Peer) write(packet *bytes.Buffer, code uint32) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	b := netCore.bufferPool.GetBuffer(packet.Len())
	//b := &bytes.Buffer{}
	b.Write(packet.Bytes())
	p.sendList.send(p, b, int(code))
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

func (pm *PeerManager) write(toid NodeID, toaddr *nnet.UDPAddr, packet *bytes.Buffer, code uint32) error {

	netId := netCoreNodeID(toid)
	p := pm.peerByNetID(netId)
	if p == nil {
		p = newPeer(toid, 0)
		p.expiration = 0
		p.connecting = false
		pm.addPeer(netId, p)
	}

	Logger.Debugf("write Id:%v net id:%v session:%v size %v", toid.GetHexString(), netId, p.seesionId, len(packet.Bytes()))

	p.write(packet, code)

	if p.seesionId != 0 {
		return nil
	}
	if ((toaddr != nil && toaddr.IP != nil && toaddr.Port > 0) || pm.natTraversalEnable) && !p.connecting {
		p.expiration = uint64(time.Now().Add(connectTimeout).Unix())
		p.connecting = true

		if toaddr != nil {
			p.Ip = toaddr.IP
			p.Port = toaddr.Port
		}

		if pm.natTraversalEnable {
			P2PConnect(netId, NatServerIp, NatServerPort)
			Logger.Debugf("P2PConnect[nat]: %v ", toid.GetHexString())
		} else {
			P2PConnect(netId, toaddr.IP.String(), uint16(toaddr.Port))
			Logger.Debugf("P2PConnect[direct]: id: %v ip: %v port:%v ", toid.GetHexString(), toaddr.IP.String(), uint16(toaddr.Port))
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
		p.recvList = list.New()
		p.seesionId = session
	}
	p.connecting = false

	netCore.ping(p.Id, nil)
	p.sendList.pendingSend = 0
	p.sendList.autoSend(p)
	Logger.Debugf("newConnection node id:%v  netid :%v session:%v isAccepted:%v ", p.Id.GetHexString(), id, session, isAccepted)
}

//OnSendWaited  发送队列空闲
func (pm *PeerManager) OnSendWaited(id uint64, session uint32) {
	p := pm.peerByNetID(id)
	if p != nil {
		p.onSendWaited()
	}
}


//OnDisconnected 处理连接断开的回调
func (pm *PeerManager) OnDisconnected(id uint64, session uint32, p2pCode uint32) {
	p := pm.peerByNetID(id)
	if p != nil {

		Logger.Debugf("OnDisconnected id：%v  session:%v ip:%v port:%v ", p.Id.GetHexString(), session, p.Ip, p.Port)

		p.connecting = false
		if p.seesionId == session {
			p.seesionId = 0
		}
	} else {
		Logger.Debugf("OnDisconnected net id：%v session:%v port:%v code:%v", id, session, p2pCode)
	}
}

func (pm *PeerManager) disconnect(id NodeID) {
	netID := netCoreNodeID(id)

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	p, _ := pm.peers[netID]
	if p != nil {

		Logger.Debugf("disconnect ip:%v port:%v ", p.Ip, p.Port)

		p.connecting = false
		delete(pm.peers, netID)
	}
}

//OnChecked 网络类型检查
func (pm *PeerManager) OnChecked(p2pType uint32, privateIp string, publicIp string) {

}

//SendDataToAll 向所有已经连接的节点发送自定义数据包
func (pm *PeerManager) SendAll(packet *bytes.Buffer, code uint32) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	//pm.checkPeerSource()
	Logger.Debugf("SendAll total peer size:%v", len(pm.peers))

	for _, p := range pm.peers {
		//if p.seesionId > 0 && p.source == PeerSourceKad {
		//	p.write(packet)
		//}
		if p.seesionId > 0 {
			p.write(packet, code)
		}
	}

	return
}

func (pm *PeerManager) checkPeerSource() {
	for _, p := range pm.peers {
		if p.seesionId > 0 && p.source == PeerSourceUnkown {
			node := netCore.kad.find(p.Id)
			if node != nil {
				p.source = PeerSourceKad
			} else {
				p.source = PeerSourceGroup
			}
		}
	}
}

//BroadcastRandom
func (pm *PeerManager) BroadcastRandom(packet *bytes.Buffer, code uint32) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	Logger.Debugf("BroadcastRandom total peer size:%v", len(pm.peers))

	pm.checkPeerSource()
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
			Logger.Debugf("BroadcastRandom send node id:%v", p.Id.GetHexString())
			p.write(packet, code)
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
			Logger.Debugf("BroadcastRandom send node id:%v", p.Id.GetHexString())
			p.write(packet, code)
		}
	}

	return
}

func (pm *PeerManager) print() {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()
	totalRecvBufferSize := 0
	totalSendBufferSize := 0
	for _, p := range pm.peers {
		totalRecvBufferSize += p.getDataSize()

		totalSendBufferSize += p.sendList.getDataSize()

		Logger.Debugf("PeerManager Print: peer id: %v, SendBufferSize:%v, RecvBufferSize:%v", p.Id.GetHexString(), p.sendList.getDataSize() ,p.getDataSize())

	}
	Logger.Debugf("PeerManager Print: peer size:%v ,SendBufferSize:%v, totolRecvBufferSize:%v", len(pm.peers), totalSendBufferSize, totalRecvBufferSize)

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
