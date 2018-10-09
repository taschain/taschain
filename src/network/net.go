package network

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	nnet "net"
	"time"

	"github.com/gogo/protobuf/proto"
	"middleware/statistics"
)

//Version 版本号
const Version = 1

const (
	PacketTypeSize           = 4
	PacketLenSize            = 4
	PacketHeadSize           = PacketTypeSize + PacketLenSize
	MaxUnhandledMessageCount = 256
)

// Errors
var (
	errPacketTooSmall   = errors.New("too small")
	errDataNotEnough   	= errors.New("data not enough")
	errBadPacket        = errors.New("bad Packet")
	errExpired          = errors.New("expired")
	errUnsolicitedReply = errors.New("unsolicited reply")
	errUnknownNode      = errors.New("unknown node")
	errTimeout          = errors.New("RPC timeout")
	errClockWarp        = errors.New("reply deadline too far in the future")
	errClosed           = errors.New("socket closed")
)

const NatServerPort = 70

var NatServerIp = "47.104.83.120"

// Timeouts
const (
	respTimeout              = 500 * time.Millisecond
	clearMessageCacheTimeout = time.Minute
	expiration               = 30 * time.Second
	connectTimeout           = 3 * time.Second
)

//NetCore p2p网络传输类
type NetCore struct {
	ourEndPoint      RpcEndPoint
	id               NodeID
	nid              uint64
	addpending       chan *pending
	gotreply         chan reply
	unhandled        chan *Peer
	unhandledDataMsg int
	closing          chan struct{}

	kad                *Kad
	peerManager        *PeerManager  //节点连接管理器
	groupManager       *GroupManager //组管理器
	messageManager     *MessageManager
	natTraversalEnable bool
}

type pending struct {
	from  NodeID
	ptype MessageType

	deadline time.Time

	callback func(resp interface{}) (done bool)

	errc chan<- error
}

type reply struct {
	from    NodeID
	ptype   MessageType
	data    interface{}
	matched chan<- bool
}

//netCoreNodeID NodeID 转网络id
func netCoreNodeID(id NodeID) uint64 {
	h := fnv.New64a()
	h.Write(id[:])
	return uint64(h.Sum64())
}

//nodeFromRPC rpc节点转换
func (nc *NetCore) nodeFromRPC(sender *nnet.UDPAddr, rn RpcNode) (*Node, error) {
	if rn.Port <= 1024 {
		return nil, errors.New("low port")
	}


	n := NewNode(common.HexStringToAddress(rn.Id), nnet.ParseIP(rn.Ip), int(rn.Port))

	err := n.validateComplete()
	return n, err
}

var netCore *NetCore

type NetCoreConfig struct {
	ListenAddr         *nnet.UDPAddr
	Id                 NodeID
	Seeds              []*Node
	NatTraversalEnable bool
}

//MakeEndPoint 创建节点描述对象
func MakeEndPoint(addr *nnet.UDPAddr, tcpPort int32) RpcEndPoint {
	ip := addr.IP.To4()
	if ip == nil {
		ip = addr.IP.To16()
	}
	return RpcEndPoint{Ip: ip.String(), Port: int32(addr.Port)}
}

func nodeToRPC(n *Node) RpcNode {
	return RpcNode{Id: n.Id.GetHexString(), Ip: n.Ip.String(), Port: int32(n.Port)}
}

//Init 初始化
func (nc *NetCore) InitNetCore(cfg NetCoreConfig) (*NetCore, error) {

	nc.id = cfg.Id
	nc.closing = make(chan struct{})
	nc.gotreply = make(chan reply)
	nc.addpending = make(chan *pending)
	nc.unhandled = make(chan *Peer,64)
	nc.natTraversalEnable = cfg.NatTraversalEnable
	nc.nid = netCoreNodeID(cfg.Id)
	nc.peerManager = newPeerManager()
	nc.peerManager.natTraversalEnable = cfg.NatTraversalEnable
	nc.groupManager = newGroupManager()
	nc.messageManager = newMessageManager(nc.id)
	realaddr := cfg.ListenAddr

	Logger.Debugf("kad id: %v ", nc.id.GetHexString())
	Logger.Debugf("P2PConfig: %v ", nc.nid)
	P2PConfig(nc.nid)

	if nc.natTraversalEnable {
		Logger.Debugf("P2PProxy: %v %v", NatServerIp, uint16(NatServerPort))
		P2PProxy(NatServerIp, uint16(NatServerPort))
	} else {
		Logger.Debugf("P2PListen: %v %v", realaddr.IP.String(), uint16(realaddr.Port))
		P2PListen(realaddr.IP.String(), uint16(realaddr.Port))
	}

	nc.ourEndPoint = MakeEndPoint(realaddr, int32(realaddr.Port))
	kad, err := newKad(nc, cfg.Id, realaddr, cfg.Seeds)
	if err != nil {
		return nil, err
	}
	nc.kad = kad
	netCore = nc
	go nc.loop()
	go nc.decodeLoop()

	return nc, nil
}

// 关闭
func (nc *NetCore) close() {
	P2PClose()
	close(nc.closing)
}

func (nc *NetCore) AddGroup(id string, members []NodeID) *Group {
	return nc.groupManager.addGroup(id, members)
}

func (nc *NetCore) ping(toid NodeID, toaddr *nnet.UDPAddr) error {

	to := MakeEndPoint(&nnet.UDPAddr{}, 0)
	if toaddr != nil {
		to = MakeEndPoint(toaddr, 0)
	}
	req := &MsgPing{
		Version:    Version,
		From:       &nc.ourEndPoint,
		To:         &to,
		NodeId:     nc.id[:],
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	}
	Logger.Info("ping node, id:%v, ip:%v, port:%v", toid.GetHexString(), to.Ip, to.Port)

	packet, _, err := nc.encodePacket(MessageType_MessagePing, req)
	if err != nil {
		fmt.Printf("net encodePacket err  %v", err)
		return err
	}

	nc.peerManager.write(toid, toaddr, packet)

	return nil
}

func (nc *NetCore) findNode(toid NodeID, toaddr *nnet.UDPAddr, target NodeID) ([]*Node, error) {
	nodes := make([]*Node, 0, bucketSize)
	nreceived := 0
	errc := nc.pending(toid, MessageType_MessageNeighbors, func(r interface{}) bool {
		reply := r.(*MsgNeighbors)
		for _, rn := range reply.Nodes {
			nreceived++
			n, err := nc.nodeFromRPC(toaddr, *rn)
			if err != nil {
				continue
			}
			//fmt.Printf("find node:%v, %v, %v",n.ID.GetHexString(),n.IP,n.Port)
			nodes = append(nodes, n)
		}
		return nreceived >= bucketSize
	})
	nc.SendMessage(toid, toaddr, MessageType_MessageFindnode, &MsgFindNode{
		Target:     target[:],
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	})
	err := <-errc
	return nodes, err
}

func (nc *NetCore) pending(id NodeID, ptype MessageType, callback func(interface{}) bool) <-chan error {
	ch := make(chan error, 1)
	p := &pending{from: id, ptype: ptype, callback: callback, errc: ch}
	select {
	case nc.addpending <- p:
	case <-nc.closing:
		ch <- errClosed
	}
	return ch
}

func (nc *NetCore) handleReply(from NodeID, ptype MessageType, req interface{}) bool {
	matched := make(chan bool, 1)
	select {
	case nc.gotreply <- reply{from, ptype, req, matched}:
		// loop will handle it
		return <-matched
	case <-nc.closing:
		return false
	}
}

func (nc *NetCore) decodeLoop() {

	for {
		select {
		case peer := <-nc.unhandled:
			for {
				err := nc.handleMessage(peer)
				if err != nil || peer.isEmpty() {
					break
				}
			}
		}
	}
}

func (nc *NetCore) loop() {
	var (
		plist             = list.New()
		timeout           = time.NewTimer(0)
		clearMessageCache = time.NewTicker(clearMessageCacheTimeout)
		nextTimeout       *pending
		contTimeouts      = 0
	)
	<-timeout.C // ignore first timeout
	defer timeout.Stop()
	defer clearMessageCache.Stop()

	resetTimeout := func() {
		if plist.Front() == nil || nextTimeout == plist.Front().Value {
			return
		}
		now := time.Now()
		for el := plist.Front(); el != nil; el = el.Next() {
			nextTimeout = el.Value.(*pending)
			if dist := nextTimeout.deadline.Sub(now); dist < 2*respTimeout {
				timeout.Reset(dist)
				return
			}

			nextTimeout.errc <- errClockWarp
			plist.Remove(el)
		}
		nextTimeout = nil
		timeout.Stop()
	}

	for {
		resetTimeout()

		select {
		case <-nc.closing:
			for el := plist.Front(); el != nil; el = el.Next() {
				el.Value.(*pending).errc <- errClosed
			}
			return
		case p := <-nc.addpending:
			p.deadline = time.Now().Add(respTimeout)
			plist.PushBack(p)

		case r := <-nc.gotreply:
			var matched bool
			for el := plist.Front(); el != nil; el = el.Next() {
				p := el.Value.(*pending)
				if p.from == r.from && p.ptype == r.ptype {
					matched = true
					if p.callback(r.data) {
						p.errc <- nil
						plist.Remove(el)
					}
					contTimeouts = 0
				}
			}
			r.matched <- matched

		case now := <-timeout.C:
			nextTimeout = nil
			for el := plist.Front(); el != nil; el = el.Next() {
				p := el.Value.(*pending)
				if now.After(p.deadline) || now.Equal(p.deadline) {
					p.errc <- errTimeout
					plist.Remove(el)
					contTimeouts++
				}
			}

		case <-clearMessageCache.C:
			nc.messageManager.clear()

		}
	}
}

func init() {

}

//Send 发送包
func (nc *NetCore) SendMessage(toid NodeID, toaddr *nnet.UDPAddr, ptype MessageType, req proto.Message) ([]byte, error) {
	packet, hash, err := nc.encodePacket(ptype, req)
	if err != nil {
		return hash, err
	}
	return hash, nc.peerManager.write(toid, toaddr, packet)
}

//SendAll 向所有已经连接的节点发送自定义数据包
func (nc *NetCore) SendAll(data []byte, broadcast bool, msgDigest MsgDigest, relayCount int32) {
	Logger.Infof("SendAll len: %v", len(data))
	dataType := DataType_DataNormal
	if broadcast {
		dataType = DataType_DataGlobal
	}
	packet, _, err := nc.encodeDataPacket(data, dataType, "", nil, msgDigest, relayCount)
	if err != nil {
		return
	}
	nc.peerManager.SendAll(packet)
	return
}

//BroadcastRandom 随机发送广播数据包
func (nc *NetCore) BroadcastRandom(data []byte, relayCount int32) {
	dataType := DataType_DataGlobalRandom

	packet, _, err := nc.encodeDataPacket(data, dataType, "", nil, nil, relayCount)
	if err != nil {
		return
	}
	nc.peerManager.BroadcastRandom(packet)
	return
}

//SendGroup 向所有已经连接的组内节点发送自定义数据包
func (nc *NetCore) SendGroup(id string, data []byte, broadcast bool) {
	dataType := DataType_DataNormal
	if broadcast {
		dataType = DataType_DataGroup
	}
	packet, _, err := nc.encodeDataPacket(data, dataType, id, nil, nil, -1)
	if err != nil {
		return
	}
	nc.groupManager.sendGroup(id, packet)
	return
}

//GroupBroadcastWithMembers 通过组成员发组广播
func (nc *NetCore) GroupBroadcastWithMembers(id string, data []byte, msgDigest MsgDigest, groupMembers []string) {
	dataType := DataType_DataGroup
	Logger.Infof("GroupBroadcastWithMembers: group id:%v", id)

	packet, _, err := nc.encodeDataPacket(data, dataType, id, nil, msgDigest, -1)
	if err != nil {
		return
	}
	const MaxSendCount = 3
	nodesHasSend := make(map[NodeID]bool)
	count := 0
	//先找已经连接的
	for i := 0; i < len(groupMembers) && count < MaxSendCount; i++ {
		id := newNodeID(groupMembers[i])
		p := nc.peerManager.peerByID(id)
		if p != nil && p.seesionId > 0 {
			count += 1
			nodesHasSend[id] = true
			nc.peerManager.write(id, nil, packet)
		}
	}

	//已经连接的不够，通过穿透服务器连接
	for i := 0; i < len(groupMembers) && count < MaxSendCount && count < len(groupMembers); i++ {
		id := newNodeID(groupMembers[i])
		if nodesHasSend[id] != true && id != nc.id {
			count += 1
			nc.peerManager.write(id, nil, packet)
		}
	}

	return
}

func (nc *NetCore) SendGroupMember(id string, data []byte, memberId NodeID) {

	Logger.Infof("SendGroupMember: group id:%v node id :%v", id, memberId.GetHexString())

	p := nc.peerManager.peerByID(memberId)
	if (p != nil && p.seesionId > 0) || nc.natTraversalEnable {
		//Logger.Infof("node id:%v connected send packet", memberId.GetHexString())
		go nc.Send(memberId, nil, data)
	} else {
		node := net.netCore.kad.find(memberId)
		if node != nil && node.Ip != nil && node.Port > 0 {
			Logger.Infof("node id:%v found in kad send packet", memberId.GetHexString())

			go nc.Send(memberId, &nnet.UDPAddr{IP: node.Ip, Port: int(node.Port)}, data)
		} else {
			Logger.Infof("node id:%v can not found ,group broadcast packet", memberId.GetHexString())

			packet, _, err := nc.encodeDataPacket(data, DataType_DataGroup, id, &memberId, nil, -1)
			if err != nil {
				return
			}

			nc.groupManager.sendGroup(id, packet)
		}
	}
	return
}

//SendData 发送自定义数据包C
func (nc *NetCore) Send(toid NodeID, toaddr *nnet.UDPAddr, data []byte) ([]byte, error) {
	packet, hash, err := nc.encodeDataPacket(data, DataType_DataNormal, "", nil, nil, -1)
	if err != nil {
		Logger.Infof("Send encodeDataPacket err :%v ", toid.GetHexString())
		return hash, err
	}
	return hash, nc.peerManager.write(toid, toaddr, packet)
}

//OnConnected 处理连接成功的回调
func (nc *NetCore) OnConnected(id uint64, session uint32, p2pType uint32) {

	nc.peerManager.newConnection(id, session, p2pType, false)

}

//OnConnected 处理接受连接的回调
func (nc *NetCore) OnAccepted(id uint64, session uint32, p2pType uint32) {

	nc.peerManager.newConnection(id, session, p2pType, true)
}

//OnDisconnected 处理连接断开的回调
func (nc *NetCore) OnDisconnected(id uint64, session uint32, p2pCode uint32) {

	nc.peerManager.OnDisconnected(id, session, p2pCode)
}

//OnChecked 网络类型检查
func (nc *NetCore) OnChecked(p2pType uint32, privateIP string, publicIP string) {
	nc.ourEndPoint = MakeEndPoint(&nnet.UDPAddr{IP: nnet.ParseIP(publicIP), Port: 8686}, 8686)
	nc.peerManager.OnChecked(p2pType, privateIP, publicIP)
}

//OnRecved 数据回调
func (nc *NetCore) OnRecved(netID uint64, session uint32, data []byte) {
	nc.recvData(netID, session, data)
}

func (nc *NetCore) recvData(netId uint64, session uint32, data []byte) {
	//Logger.Infof("recvData netid:%v  session:%v len:%v ", netId ,session,len(data))

	p := nc.peerManager.peerByNetID(netId)
	if p == nil {
		p = newPeer(NodeID{}, session)
		nc.peerManager.addPeer(netId, p)
	}

	p.addData(data)
	nc.unhandled <- p
}

func (nc *NetCore) encodeDataPacket(data []byte, dataType DataType, groupId string, nodeId *NodeID, msgDigest MsgDigest, relayCount int32) (msg *bytes.Buffer, hash []byte, err error) {
	nodeIdBytes := make([]byte, 0)
	if nodeId != nil {
		nodeIdBytes = nodeId.Bytes()
	}
	bizMessageIdBytes := make([]byte, 0)
	if msgDigest != nil {
		bizMessageIdBytes = msgDigest[:]
	}
	msgData := &MsgData{
		Data:         data,
		DataType:     dataType,
		GroupId:      groupId,
		MessageId:    nc.messageManager.genMessageId(),
		DestNodeId:   nodeIdBytes,
		SrcNodeId:    nc.id.Bytes(),
		BizMessageId: bizMessageIdBytes,
		RelayCount:   relayCount,
		Expiration:   uint64(time.Now().Add(expiration).Unix())}
	Logger.Infof("encodeDataPacket  DataType:%v messageId:%X ,BizMessageId:%v ,RelayCount:%v ", msgData.DataType, msgData.MessageId, msgData.BizMessageId, msgData.RelayCount)

	return nc.encodePacket(MessageType_MessageData, msgData)
}

func (nc *NetCore) encodePacket(ptype MessageType, req proto.Message) (msg *bytes.Buffer, hash []byte, err error) {
	b := new(bytes.Buffer)

	pdata, err := proto.Marshal(req)
	if err != nil {
		return nil, nil, err
	}
	length := len(pdata)
	err = binary.Write(b, binary.BigEndian, uint32(ptype))
	if err != nil {
		return nil, nil, err
	}
	err = binary.Write(b, binary.BigEndian, uint32(length))
	if err != nil {
		return nil, nil, err
	}

	b.Write(pdata)
	return b, nil, nil
}

func (nc *NetCore) handleMessage(p *Peer) error {
	if p == nil || p.isEmpty() {
		return nil
	}
	msgType, packetSize, msg, buf, err := decodePacket(p)

	if err != nil {
		return err
	}
	fromId := p.Id

	//Logger.Infof("handleMessage : msgType: %v ", msgType)

	switch msgType {
	case MessageType_MessagePing:
		//fromId = common.BytesToAddress(msg.(*MsgPing).NodeId)
		fromId.SetBytes(msg.(*MsgPing).NodeId)
		if fromId != p.Id {
			p.Id = fromId
		}
		nc.handlePing(msg.(*MsgPing), fromId)
	case MessageType_MessageFindnode:
		nc.handleFindNode(msg.(*MsgFindNode), fromId)
	case MessageType_MessageNeighbors:
		nc.handleNeighbors(msg.(*MsgNeighbors), fromId)
	case MessageType_MessageData:
		nc.handleData(msg.(*MsgData), buf.Bytes()[0:packetSize], fromId)
	default:
		return Logger.Errorf("unknown type: %d", msgType)
	}
	return nil
}

func decodePacket(p *Peer) (MessageType, int, proto.Message,*bytes.Buffer, error) {

	buffer := p.popData()
	if buffer == nil {
		Logger.Debugf("decodePacket no data.")
		return MessageType_MessageNone, 0, nil, buffer, errPacketTooSmall

	}

	for buffer.Len() < PacketHeadSize && !p.isEmpty() {
		b := p.popData()
		if b != nil && b.Len() > 0 {
			buffer.Write(b.Bytes())
		}
	}
	if buffer.Len() < PacketHeadSize {
		p.addDataToHead(buffer)
		return MessageType_MessageNone, 0, nil, buffer, errPacketTooSmall
	}

	bufBytes := buffer.Bytes()
	msgType := MessageType(binary.BigEndian.Uint32(bufBytes[:PacketTypeSize]))
	msgLen := binary.BigEndian.Uint32(bufBytes[PacketTypeSize : PacketTypeSize+PacketLenSize])
	packetSize := int(msgLen + PacketHeadSize)

	Logger.Debugf("decodePacket :packetSize: %v  msgType: %v  msgLen:%v   bufSize:%v ", packetSize, msgType, msgLen, buffer.Len())

	if  packetSize > 16 * 1024 * 1024 || packetSize <= 0 {
		Logger.Debugf("bad packet reset data!")
		p.resetData()
		return MessageType_MessageNone, 0, nil, buffer, errBadPacket
	}

	for buffer.Len() < packetSize  && !p.isEmpty() {
		b := p.popData()
		if b != nil && b.Len() > 0 {
	//		Logger.Debugf("popData size:%v!", b.Len())

			buffer.Write(b.Bytes())
		}
	}
	if  buffer.Len() < packetSize {
		p.addDataToHead(buffer)
		return MessageType_MessageNone, 0, nil, buffer, errPacketTooSmall
	}

	bufBytes = buffer.Bytes()
	//Logger.Debugf("decodePacket after :packetSize: %v  msgType: %v  msgLen:%v   bufSize:%v ", packetSize, msgType, msgLen, buffer.Len())

	if buffer.Len() < packetSize {
		p.addDataToHead(buffer)
		return MessageType_MessageNone, 0, nil,buffer,errPacketTooSmall
	}  else  if buffer.Len() > packetSize {
		p.addDataToHead(bytes.NewBuffer(bufBytes[packetSize:]))
	}
	data := bufBytes[PacketHeadSize : PacketHeadSize+msgLen]
	var req proto.Message
	switch msgType {
	case MessageType_MessagePing:
		req = new(MsgPing)
	case MessageType_MessageFindnode:
		req = new(MsgFindNode)
	case MessageType_MessageNeighbors:
		req = new(MsgNeighbors)
	case MessageType_MessageData:
		req = new(MsgData)
	default:
		return msgType, packetSize, nil, buffer,fmt.Errorf("unknown type: %d", msgType)
	}

	var err error
	if req != nil {
		err = proto.Unmarshal(data, req)
	}

	return msgType, packetSize, req,buffer, err
}

func (nc *NetCore) handlePing(req *MsgPing, fromId NodeID) error {

	//	Logger.Infof("handlePing from ip:%v %v to ip:%v %v ", req.From.Ip, req.From.Port, req.To.Ip, req.To.Port)

	if expired(req.Expiration) {
		return errExpired
	}
	p := nc.peerManager.peerByID(fromId)
	ip := nnet.ParseIP(req.From.Ip)
	port := int(req.From.Port)
	if p != nil && ip != nil && port > 0 {
		p.Ip = ip
		p.Port = port
	}
	from := nnet.UDPAddr{IP: nnet.ParseIP(req.From.Ip), Port: int(req.From.Port)}

	if !nc.handleReply(fromId, MessageType_MessagePing, req) {
		go nc.kad.onPingNode(fromId, &from)
	}
	return nil
}

func (nc *NetCore) handleFindNode(req *MsgFindNode, fromId NodeID) error {

	if expired(req.Expiration) {
		return errExpired
	}
	if !nc.kad.hasPinged(fromId) {
		return errUnknownNode
	}
	target := req.Target
	nc.kad.mutex.Lock()
	closest := nc.kad.closest(target, bucketSize).entries
	nc.kad.mutex.Unlock()

	p := MsgNeighbors{Expiration: uint64(time.Now().Add(expiration).Unix())}

	for _, n := range closest {
		node := nodeToRPC(n)
		if len(node.Ip) > 0 && node.Port > 0 {
			p.Nodes = append(p.Nodes, &node)
		}
	}
	if len(p.Nodes) > 0 {
		nc.SendMessage(fromId, nil, MessageType_MessageNeighbors, &p)
	}

	return nil
}

func (nc *NetCore) handleNeighbors(req *MsgNeighbors, fromId NodeID) error {
	if expired(req.Expiration) {
		return errExpired
	}
	if !nc.handleReply(fromId, MessageType_MessageNeighbors, req) {
		return errUnsolicitedReply
	}
	return nil
}

func (nc *NetCore) handleData(req *MsgData, packet []byte, fromId NodeID) error {
	srcNodeId := NodeID{}
	srcNodeId.SetBytes(req.SrcNodeId)
	Logger.Infof("data from:%v  len:%v DataType:%v messageId:%X ,BizMessageId:%v ,RelayCount:%v  unhandleDataMsg:%v", srcNodeId, len(req.Data), req.DataType, req.MessageId, req.BizMessageId, req.RelayCount, nc.unhandledDataMsg)

	statistics.AddCount("net.handleData", uint32(req.DataType), uint64(len(req.Data)))
	if req.DataType == DataType_DataNormal {
		nc.onHandleDataMessage(req.Data, srcNodeId.GetHexString())
	} else {
		forwarded := false

		if req.BizMessageId != nil {
			bizId := nc.messageManager.ByteToBizId(req.BizMessageId)
			forwarded = nc.messageManager.isForwardedBiz(bizId)

		} else {
			forwarded = nc.messageManager.isForwarded(req.MessageId)
		}

		if !forwarded {
			destNodeId := NodeID{}
			destNodeId.SetBytes(req.DestNodeId)

			nc.messageManager.forward(req.MessageId)
			if req.BizMessageId != nil {
				bizId := nc.messageManager.ByteToBizId(req.BizMessageId)
				nc.messageManager.forwardBiz(bizId)
			}
			//需处理
			if len(req.DestNodeId) == 0 || destNodeId == nc.id {
				nc.onHandleDataMessage(req.Data, srcNodeId.GetHexString())
			}
			broadcast := false
			//需广播
			if len(req.DestNodeId) == 0 || destNodeId != nc.id {
				broadcast = true
			}

			if req.DataType == DataType_DataGlobalRandom && req.RelayCount == 0 {
				broadcast = false
			}
			if broadcast {
				var dataBuffer *bytes.Buffer = nil
				if req.DataType == DataType_DataGlobalRandom && req.RelayCount > 0 {
					req.RelayCount = req.RelayCount - 1
					req.Expiration = uint64(time.Now().Add(expiration).Unix())
					dataBuffer, _, _ = nc.encodePacket(MessageType_MessageData, req)
					//dataBuffer = bytes.NewBuffer(packet)

				} else {
					dataBuffer = bytes.NewBuffer(packet)

				}
				Logger.Infof("forwarded message DataType:%v messageId:%X DestNodeId：%v SrcNodeId：%v", req.DataType, req.MessageId, destNodeId.GetHexString(), srcNodeId.GetHexString())

				if req.DataType == DataType_DataGroup {
					nc.groupManager.sendGroup(req.GroupId, dataBuffer)
				} else if req.DataType == DataType_DataGlobal {
					nc.peerManager.SendAll(dataBuffer)
				} else if req.DataType == DataType_DataGlobalRandom && req.RelayCount != -1 {
					nc.peerManager.BroadcastRandom(dataBuffer)
				}
			}
		}
	}

	return nil
}

func (nc *NetCore) onHandleDataMessage(b []byte, from string) {
	if nc.unhandledDataMsg > MaxUnhandledMessageCount {
		Logger.Errorf("unhandled message too much , drop this message !")
		return
	}
	nc.unhandledDataMsg += 1
	if net != nil {
		net.handleMessage(b, from)
	}

}

func (nc *NetCore) onHandleDataMessageDone(id string) {
	nc.unhandledDataMsg -= 1
}

func expired(ts uint64) bool {
	return time.Unix(int64(ts), 0).Before(time.Now())
}
