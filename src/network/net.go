package network

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"net"
	"sync"
	"time"

	"github.com/gogo/protobuf/proto"
	"common"
)

//Version 版本号
const Version = 1

// Errors
var (
	errPacketTooSmall   = errors.New("too small")
	errBadHash          = errors.New("bad hash")
	errExpired          = errors.New("expired")
	errUnsolicitedReply = errors.New("unsolicited reply")
	errUnknownNode      = errors.New("unknown node")
	errTimeout          = errors.New("RPC timeout")
	errClockWarp        = errors.New("reply deadline too far in the future")
	errClosed           = errors.New("socket closed")
)

// Timeouts
const (
	respTimeout    = 500 * time.Millisecond
	expiration     = 20 * time.Second
	connectTimeout = 3 * time.Second

	ntpFailureThreshold = 32               // Continuous timeouts after which to check NTP
	ntpWarningCooldown  = 10 * time.Minute // Minimum amount of time to pass before repeating NTP warning
	driftThreshold      = 1 * time.Second  // Allowed clock drift before warning user
)

//NetCore p2p网络传输类
type NetCore struct {
	//netrestrict *netutil.Netlist
	priv        *common.PrivateKey
	ourEndPoint RpcEndPoint
	id          NodeID
	nid         uint64
	addpending  chan *pending
	gotreply    chan reply
	unhandle   chan *Peer

	closing chan struct{}

	kad 			*Kad
	peerManager  	*PeerManager  //节点连接管理器
	groupManager  	*GroupManager //组管理器
	messageManager 	*MessageManager
	natTraversalEnable bool;
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


//NetCoreNodeID NodeID 转网络id
func NetCoreNodeID(id NodeID) uint64 {
	h := fnv.New64a()
	h.Write(id[:])
	return uint64(h.Sum64())
}

//NodeFromRPC rpc节点转换
func (nc *NetCore) NodeFromRPC(sender *net.UDPAddr, rn RpcNode) (*Node, error) {
	if rn.Port <= 1024 {
		return nil, errors.New("low port")
	}
	// if err := netutil.CheckRelayIP(sender.IP, rn.IP); err != nil {
	// 	return nil, err
	// }
	// if t.netrestrict != nil && !t.netrestrict.Contains(rn.IP) {
	// 	return nil, errors.New("not contained in netrestrict whitelist")
	// }
	n := NewNode(common.HexStringToAddress(rn.Id), net.ParseIP(rn.Ip), int(rn.Port))
	err := n.validateComplete()
	return n, err
}

var netCore *NetCore
var lock = &sync.Mutex{}


// Config holds Table-related settings.
type Config struct {
	PrivateKey *common.PrivateKey
	ListenAddr *net.UDPAddr // local address announced in the DHT
	NodeDBPath string       // if set, the node database is stored at this filesystem location
	Id        NodeID
	Bootnodes []*Node           // list of bootstrap nodes
	NatTraversalEnable 	bool
}

//MakeEndPoint 创建节点描述对象
func MakeEndPoint(addr *net.UDPAddr, tcpPort int32) RpcEndPoint {
	ip := addr.IP.To4()
	if ip == nil {
		ip = addr.IP.To16()
	}
	return RpcEndPoint{Ip: ip.String(), Port: int32(addr.Port)}
}

func nodeToRPC(n *Node) RpcNode {
	return RpcNode{Id:n.Id.GetHexString(), Ip: n.Ip.String(), Port: int32(n.Port)}
}

//Init 初始化
func (nc *NetCore) InitNetCore(cfg Config) (*NetCore, error) {

	nc.priv = cfg.PrivateKey
	nc.id = cfg.Id
	nc.closing = make(chan struct{})
	nc.gotreply = make(chan reply)
	nc.addpending = make(chan *pending)
	nc.unhandle = make(chan *Peer)
	nc.natTraversalEnable = cfg.NatTraversalEnable
	nc.nid = NetCoreNodeID(cfg.Id)
	nc.peerManager = newPeerManager()
	nc.peerManager.natTraversalEnable = cfg.NatTraversalEnable
	nc.groupManager = newGroupManager()
	nc.messageManager = newMessageManager(nc.id)
	realaddr := cfg.ListenAddr

	fmt.Printf("KAD ID %v \n", nc.id.GetHexString())
	fmt.Printf("P2PConfig %v \n", nc.nid)
	fmt.Printf("P2PListen %v %v\n", realaddr.IP.String(), uint16(realaddr.Port))
	P2PConfig(nc.nid)


	if nc.natTraversalEnable {
		P2PProxy("47.98.212.107", uint16(70))
	}else {
		P2PListen(realaddr.IP.String(), uint16(realaddr.Port))
	}

	nc.ourEndPoint = MakeEndPoint(realaddr, int32(realaddr.Port))
	kad, err := newKad(nc, cfg.Id, realaddr, cfg.NodeDBPath, cfg.Bootnodes)
	if err != nil {
		return nil, err
	}
	nc.kad = kad
	go nc.loop()
	go nc.decodeLoop()

	return nc, nil
}

// 关闭
func (nc *NetCore) close() {
	P2PClose()
	close(nc.closing)
}

func (nc *NetCore)  print() {
	//nc.peerManager.print()
}

// ping sends a ping message to the given node and waits for a reply.
func (nc *NetCore) ping(toid NodeID, toaddr *net.UDPAddr) error {

	to := MakeEndPoint(toaddr, 0)
	req := &MsgPing{
		Version:    Version,
		From:       &nc.ourEndPoint,
		To:         &to,
		NodeId:nc.id[:],
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	}
	//fmt.Printf("ping node:%v, %v, %v\n",toid.GetHexString(),to.IP,to.Port)

	packet, hash, err := nc.encodePacket(MessageType_MessagePing, req)
	if err != nil {
		fmt.Printf("net encodePacket err  %v\n", err)
		return err
	}
	errc := nc.pending(toid, MessageType_MessagePong, func(p interface{}) bool {
		return bytes.Equal(p.(*MsgPong).ReplyToken, hash)
	})
	nc.peerManager.write(toid, toaddr, packet)

	return <-errc
}


func (nc *NetCore) waitping(from NodeID) error {
	return <-nc.pending(from, MessageType_MessagePing, func(interface{}) bool { return true })
}

// findnode sends a findnode request to the given node and waits until
// the node has sent up to k neighbors.
func (nc *NetCore) findnode(toid NodeID, toaddr *net.UDPAddr, target NodeID) ([]*Node, error) {
	nodes := make([]*Node, 0, bucketSize)
	nreceived := 0
	errc := nc.pending(toid, MessageType_MessageNeighbors, func(r interface{}) bool {
		reply := r.(*MsgNeighbors)
		for _, rn := range reply.Nodes {
			nreceived++
			n, err := nc.NodeFromRPC(toaddr, *rn)
			if err != nil {
				//log.Trace("Invalid neighbor node received", "ip", rn.IP, "addr", toaddr, "err", err)
				continue
			}
			//fmt.Printf("find node:%v, %v, %v\n",n.ID.GetHexString(),n.IP,n.Port)
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

// pending adds a reply callback to the pending reply queue.
// see the documentation of type pending for a detailed explanation.
func (nc *NetCore) pending(id NodeID, ptype MessageType, callback func(interface{}) bool) <-chan error {
	ch := make(chan error, 1)
	p := &pending{from: id, ptype: ptype, callback: callback, errc: ch}
	select {
	case nc.addpending <- p:
		// loop will handle it
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
		case peer:=<-nc.unhandle:
			for {
				//fmt.Printf("handleMessage : id:%v \n ", peer.ID.B58String())
				err := nc.handleMessage(peer)
				if err != nil || peer.isEmpty() {
					break
				}
			}
		}
	}
}


// loop runs in its own goroutine. it keeps track of
// the refresh timer and the pending reply queue.
func (nc *NetCore) loop() {
	var (
		plist        = list.New()
		timeout      = time.NewTimer(0)
		nextTimeout  *pending // head of plist when timeout was last reset
		contTimeouts = 0      // number of continuous timeouts to do NTP checks
		ntpWarnTime  = time.Unix(0, 0)
	)
	<-timeout.C // ignore first timeout
	defer timeout.Stop()

	resetTimeout := func() {
		if plist.Front() == nil || nextTimeout == plist.Front().Value {
			return
		}
		// Start the timer so it fires when the next pending reply has expired.
		now := time.Now()
		for el := plist.Front(); el != nil; el = el.Next() {
			nextTimeout = el.Value.(*pending)
			if dist := nextTimeout.deadline.Sub(now); dist < 2*respTimeout {
				timeout.Reset(dist)
				return
			}
			// Remove pending replies whose deadline is too far in the
			// future. These can occur if the system clock jumped
			// backwards after the deadline was assigned.
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
					// Remove the matcher if its callback indicates
					// that all replies have been received. This is
					// required for packet types that expect multiple
					// reply packets.
					if p.callback(r.data) {
						p.errc <- nil
						plist.Remove(el)
					}
					// Reset the continuous timeout counter (time drift detection)
					contTimeouts = 0
				}
			}
			r.matched <- matched

		case now := <-timeout.C:
			nextTimeout = nil
			// Notify and remove callbacks whose deadline is in the past.
			for el := plist.Front(); el != nil; el = el.Next() {
				p := el.Value.(*pending)
				if now.After(p.deadline) || now.Equal(p.deadline) {
					p.errc <- errTimeout
					plist.Remove(el)
					contTimeouts++
				}
			}
			// If we've accumulated too many timeouts, do an NTP time sync check
			if contTimeouts > ntpFailureThreshold {
				if time.Since(ntpWarnTime) >= ntpWarningCooldown {
					ntpWarnTime = time.Now()
					//go checkClockDrift()
				}
				contTimeouts = 0
			}
		}
	}
}

const (
	typeSize = 4
	lenSize  = 4
	headSize = typeSize + lenSize
)

var (
	headSpace = make([]byte, headSize)

	maxNeighbors int
)

func init() {
	p := MsgNeighbors{Expiration: ^uint64(0)}
	maxSizeNode := RpcNode{Ip: make(net.IP, 16).String(), Port: ^int32(0)}
	for n := 0; ; n++ {
		p.Nodes = append(p.Nodes, &maxSizeNode)
		pdata, err := proto.Marshal(&p)

		if err != nil {
			panic("cannot encode: " + err.Error())
		}
		if headSize+len(pdata)+1 >= 1280 {
			maxNeighbors = n
			break
		}
	}
}

//Send 发送包
func (nc *NetCore) SendMessage(toid NodeID, toaddr *net.UDPAddr, ptype MessageType, req proto.Message) ([]byte, error) {
	packet, hash, err := nc.encodePacket(ptype, req)
	if err != nil {
		return hash, err
	}
	return hash, nc.peerManager.write(toid, toaddr, packet)
}

//SendAll 向所有已经连接的节点发送自定义数据包
func (nc *NetCore) SendAll(data []byte) {

	packet, _, err := nc.encodeDataPacket(data,DataType_DataGlobal,"")
	if err != nil {
		return
	}
	nc.peerManager.SendAll(packet)
	return
}

//SendGroup 向所有已经连接的组内节点发送自定义数据包
func (nc *NetCore) SendGroup(id string, data []byte) {

	packet, _, err := nc.encodeDataPacket(data,DataType_DataGroup,id)
	if err != nil {
		return
	}
	nc.groupManager.SendGroup(id, packet)
	return
}

//SendData 发送自定义数据包C
func (nc *NetCore) Send(toid NodeID, toaddr *net.UDPAddr, data []byte) ([]byte, error) {
	packet, hash, err := nc.encodeDataPacket(data,DataType_DataNormal,"")
	if err != nil {
		return hash, err
	}
	return hash, nc.peerManager.write(toid, toaddr, packet)
}

//OnConnected 处理连接成功的回调
func (nc *NetCore) OnConnected(id uint64, session uint32, p2pType uint32) {

	nc.peerManager.OnConnected(id, session, p2pType)
	p := nc.peerManager.peerByNetID(id)

	go nc.ping(p.Id,&net.UDPAddr{IP:p.Ip, Port: p.Port})
}

//OnConnected 处理接受连接的回调
func (nc *NetCore) OnAccepted(id uint64, session uint32, p2pType uint32) {

	nc.peerManager.OnConnected(id, session, p2pType)
}

//OnDisconnected 处理连接断开的回调
func (nc *NetCore) OnDisconnected(id uint64, session uint32, p2pCode uint32) {

	nc.peerManager.OnDisconnected(id, session, p2pCode)
}

//OnChecked 网络类型检查
func (nc *NetCore) OnChecked(p2pType uint32, privateIP string, publicIP string) {
	nc.ourEndPoint = MakeEndPoint(&net.UDPAddr{IP: net.ParseIP(publicIP), Port: 8686}, 8686)
	nc.peerManager.OnChecked(p2pType, privateIP, publicIP)
}

//OnRecved 数据回调
func (nc *NetCore) OnRecved(netID uint64, session uint32, data []byte) {
	nc.recvData(netID,session,data)
}

func (nc *NetCore) recvData(netId uint64, session uint32, data []byte) {

	start := time.Now()

	p := nc.peerManager.peerByNetID(netId)
	if p == nil {
		p = newPeer(NodeID{}, 0)
		nc.peerManager.addPeer(netId, p)
	}

	p.addData(data)

	nc.unhandle<-p

	diff := time.Now().Sub(start)
	if diff > 500 *time.Millisecond {
		fmt.Printf("Recv timeout:%v\n", diff)
	}

}


func (nc *NetCore)encodeDataPacket( data []byte,dataType DataType, groupId string) (msg *bytes.Buffer, hash []byte, err error) {
	msgData := &MsgData{
		Data:  data,
		DataType: dataType,
		GroupId:groupId,
		MessageId:nc.messageManager.genMessageId(),
		Expiration: uint64(time.Now().Add(expiration).Unix())}

	return nc.encodePacket(MessageType_MessageData,msgData)
}

func  (nc *NetCore)encodePacket(ptype MessageType, req proto.Message) (msg *bytes.Buffer, hash []byte, err error) {
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
	buf := p.getData()
	if buf == nil || buf.Len() == 0 {
		return nil
	}
	msgType, packetSize, msg, err := decodePacket(buf)

	fromId := p.Id
	if err != nil {
		if err == errPacketTooSmall {
			p.addDataToHead(buf.Bytes())
		}

		//log.Debug("Bad discv4 packet", "addr", from, "err", err)
		return err
	}

	if int(packetSize) < buf.Len() {
		p.addDataToHead(buf.Bytes()[packetSize:])
	}
	//fmt.Printf("handleMessage : msgType: %v \n", msgType)

	switch msgType {
	case MessageType_MessagePing:
		fromId =  MustBytesID(msg.(*MsgPing).NodeId)
		if fromId != p.Id {
			p.Id = fromId
		}
		nc.handlePing(msg.(*MsgPing),fromId)
	case MessageType_MessagePong:
		nc.handlePong(msg.(*MsgPong), fromId)
	case MessageType_MessageFindnode:
		nc.handleFindNode(msg.(*MsgFindNode), fromId)
	case MessageType_MessageNeighbors:
		nc.handleNeighbors(msg.(*MsgNeighbors), fromId)
	case MessageType_MessageData:
		go nc.handleData(msg.(*MsgData), buf.Bytes()[0:packetSize], fromId)
	default:
		return fmt.Errorf("unknown type: %d", msgType)
	}
	return nil
}

func decodePacket(buffer *bytes.Buffer) (MessageType, int, proto.Message, error) {
	buf := buffer.Bytes()
	msgType := MessageType(binary.BigEndian.Uint32(buf[:typeSize]))
	msgLen := binary.BigEndian.Uint32(buf[typeSize : typeSize+lenSize])
	packetSize := int(msgLen + headSize)

	//fmt.Printf("decodePacket id %v \n ",ID.GetHexString())
	//fmt.Printf("decodePacket :packetSize: %v  msgType: %v  msgLen:%v   bufSize:%v\n ", packetSize, msgType, msgLen, len(buf))

	if buffer.Len() < packetSize {
		return MessageType_MessageNone,0, nil, errPacketTooSmall
	}
	data := buf[headSize : headSize+msgLen]
	var req proto.Message
	switch msgType {
	case MessageType_MessagePing:
		req = new(MsgPing)
	case MessageType_MessagePong:
		req = new(MsgPong)
	case MessageType_MessageFindnode:
		req = new(MsgFindNode)
	case MessageType_MessageNeighbors:
		req = new(MsgNeighbors)
	case MessageType_MessageData:
		req = new(MsgData)
	default:
		return msgType, packetSize, nil, fmt.Errorf("unknown type: %d", msgType)
	}

	var err error
	if req != nil {
		err = proto.Unmarshal(data, req)
	}

	//fmt.Printf("decodePacket end\n ")

	return msgType, packetSize, req, err
}

func (nc *NetCore) handlePing(req *MsgPing,fromId NodeID ) error {

	//fmt.Printf("handlePing from ip:%v %v to ip:%v %v \n ", req.From.IP, req.From.Port, req.To.IP, req.To.Port)

	if expired(req.Expiration) {
		return errExpired
	}
	p := nc.peerManager.peerByID(fromId)
	if p != nil {
		//fmt.Printf("update  ip \n ")

		p.Ip = net.ParseIP(req.From.Ip)
		p.Port =  int(req.From.Port)
	}
	from := net.UDPAddr{IP: net.ParseIP(req.From.Ip), Port: int(req.From.Port)}
	to := MakeEndPoint(&from, req.From.Port)
	nc.SendMessage(fromId, &from, MessageType_MessagePong, &MsgPong{
		To:         &to,
		ReplyToken: nil,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	})
	if !nc.handleReply(fromId, MessageType_MessagePing, req) {
		//fmt.Printf("handlePing bond\n")

		// Note: we're ignoring the provided IP address right now
		go nc.kad.bond(true, fromId, &from, int(req.From.Port))
	}
	return nil
}

func (nc *NetCore) handlePong(req *MsgPong, fromId NodeID) error {
	if expired(req.Expiration) {
		return errExpired
	}
	if !nc.handleReply(fromId, MessageType_MessagePong, req) {
		return errUnsolicitedReply
	}
	return nil
}

func (nc *NetCore) handleFindNode(req *MsgFindNode, fromId NodeID) error {

	if expired(req.Expiration) {
		return errExpired
	}
	if !nc.kad.hasBond(fromId) {
		return errUnknownNode
	}
	target := req.Target
	nc.kad.mutex.Lock()
	closest := nc.kad.closest(target, bucketSize).entries
	nc.kad.mutex.Unlock()

	p := MsgNeighbors{Expiration: uint64(time.Now().Add(expiration).Unix())}

	for _, n := range closest {
		node := nodeToRPC(n)
		if len(node.Ip) >0 && node.Port >0 {
			p.Nodes = append(p.Nodes, &node)
		}

		//if len(p.Nodes) == maxNeighbors {
		//	nc.SendMessage(fromId, nil, MessageType_MessageNeighbors, &p)
		//	p.Nodes = p.Nodes[:0]
		//	sent = true
		//}
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

func (nc *NetCore) handleData(req *MsgData, packet []byte,fromId NodeID) error {
	id := fromId.GetHexString()
	//fmt.Printf("data from:%v  len:%v DataType:%v messageId:%X\n", id, len(req.Data),req.DataType,req.MessageId)
	needHandle := false
	if req.DataType == DataType_DataNormal {
		needHandle = true
	} else {
		forwarded := nc.messageManager.isForwarded(req.MessageId)
		if !forwarded {
			//fmt.Printf("forwarded message DataType:%v messageId:%X\n", req.DataType,req.MessageId)
			needHandle = true
			dataBuffer :=bytes.NewBuffer(packet)
			nc.messageManager.forward(req.MessageId)
			if req.DataType == DataType_DataGroup {
				nc.groupManager.SendGroup(req.GroupId,dataBuffer)
			} else if req.DataType == DataType_DataGlobal {
				nc.peerManager.SendAll(dataBuffer)
			}
		}
	}
	if needHandle {
		Network.handleMessage(req.Data,id)
	}
	return nil
}

func expired(ts uint64) bool {
	return time.Unix(int64(ts), 0).Before(time.Now())
}
