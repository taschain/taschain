package p2p

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

	kad *Kad
	PM  *PeerManager  //节点连接管理器
	GM  *GroupManager //组管理器

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
	h := fnv.New32()
	h.Write(id[:])
	return uint64(h.Sum32())
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
	n := NewNode(common.StringToAddress(rn.ID), net.ParseIP(rn.IP), int(rn.Port))
	err := n.validateComplete()
	return n, err
}

var netCore *NetCore
var lock = &sync.Mutex{}

// GetNetCore 网络核心类单例
func GetNetCore() *NetCore {
	lock.Lock()
	defer lock.Unlock()
	if netCore == nil {
		netCore = &NetCore{}
	}
	return netCore
}

// Config holds Table-related settings.
type Config struct {
	// These settings are required and configure the UDP listener:
	PrivateKey *common.PrivateKey
	// These settings are optional:
	ListenAddr *net.UDPAddr // local address announced in the DHT
	NodeDBPath string       // if set, the node database is stored at this filesystem location
	//NetRestrict  *netutil.Netlist  // network whitelist
	ID        NodeID
	Bootnodes []*Node           // list of bootstrap nodes
}

//MakeEndPoint 创建节点描述对象
func MakeEndPoint(addr *net.UDPAddr, tcpPort int32) RpcEndPoint {
	ip := addr.IP.To4()
	if ip == nil {
		ip = addr.IP.To16()
	}
	return RpcEndPoint{IP: ip.String(), Port: int32(addr.Port)}
}

func nodeToRPC(n *Node) RpcNode {
	return RpcNode{ID: n.ID.GetHexString(), IP: n.IP.String(), Port: int32(n.Port)}
}

//Init 初始化
func (nc *NetCore) Init(cfg Config) (*NetCore, error) {

	nc.priv = cfg.PrivateKey
	nc.id = cfg.ID
	nc.closing = make(chan struct{})
	nc.gotreply = make(chan reply)
	nc.addpending = make(chan *pending)
	nc.unhandle = make(chan *Peer)

	nc.nid = NetCoreNodeID(cfg.ID)
	nc.PM = newPeerManager()
	nc.GM = newGroupManager()
	realaddr := cfg.ListenAddr

	fmt.Printf("KAD ID %v \n", nc.id.Str())
	fmt.Printf("P2PConfig %v \n", nc.nid)
	fmt.Printf("P2PListen %v %v\n", realaddr.IP.String(), uint16(realaddr.Port))
	P2PConfig(nc.nid)


	P2PListen(realaddr.IP.String(), uint16(realaddr.Port))
	//P2PProxy("47.96.186.139", uint16(70))

	nc.ourEndPoint = MakeEndPoint(realaddr, int32(realaddr.Port))
	kad, err := newKad(nc, cfg.ID, realaddr, cfg.NodeDBPath, cfg.Bootnodes)
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
	close(nc.closing)
}

func (nc *NetCore)  print() {
	//nc.PM.print()
}

// ping sends a ping message to the given node and waits for a reply.
func (nc *NetCore) ping(toid NodeID, toaddr *net.UDPAddr) error {

	to := MakeEndPoint(toaddr, 0)
	req := &Ping{
		Version:    Version,
		From:       &nc.ourEndPoint,
		To:         &to,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	}
	//fmt.Printf("ping node:%v, %v, %v\n",toid.B58String(),to.IP,to.Port)

	packet, hash, err := encodePacket(nc.id, MessageType_MessagePing, req)
	if err != nil {
		fmt.Printf("net encodePacket err  %v\n", err)
		return err
	}
	errc := nc.pending(toid, MessageType_MessagePong, func(p interface{}) bool {
		return bytes.Equal(p.(*Pong).ReplyToken, hash)
	})
	nc.PM.write(toid, toaddr, packet)

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
		reply := r.(*Neighbors)
		for _, rn := range reply.Nodes {
			nreceived++
			n, err := nc.NodeFromRPC(toaddr, *rn)
			if err != nil {
				//log.Trace("Invalid neighbor node received", "ip", rn.IP, "addr", toaddr, "err", err)
				continue
			}
			//fmt.Printf("find node:%v, %v, %v\n",n.ID.B58String(),n.IP,n.Port)
			nodes = append(nodes, n)
		}
		return nreceived >= bucketSize
	})
	nc.Send(toid, toaddr, MessageType_MessageFindnode, &FindNode{
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
	idSize   = common.AddressLength
	headSize = typeSize + lenSize + idSize
)

var (
	headSpace = make([]byte, headSize)

	maxNeighbors int
)

func init() {
	p := Neighbors{Expiration: ^uint64(0)}
	maxSizeNode := RpcNode{IP: make(net.IP, 16).String(), Port: ^int32(0)}
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
func (nc *NetCore) Send(toid NodeID, toaddr *net.UDPAddr, ptype MessageType, req proto.Message) ([]byte, error) {
	packet, hash, err := encodePacket(nc.id, ptype, req)
	if err != nil {
		return hash, err
	}
	return hash, nc.PM.write(toid, toaddr, packet)
}

//SendDataToAll 向所有已经连接的节点发送自定义数据包
func (nc *NetCore) SendDataToAll(data []byte) {

	packet, _, err := encodeDataPacket(nc.id, data)
	if err != nil {
		return
	}
	nc.PM.SendDataToAll(packet)
	return
}

//SendDataToGroup 向所有已经连接的组内节点发送自定义数据包
func (nc *NetCore) SendDataToGroup(id string, data []byte) {

	packet, _, err := encodeDataPacket(nc.id, data)
	if err != nil {
		return
	}
	nc.GM.SendDataToGroup(id, packet)
	return
}

//SendData 发送自定义数据包C
func (nc *NetCore) SendData(toid NodeID, toaddr *net.UDPAddr, data []byte) ([]byte, error) {
	packet, hash, err := encodeDataPacket(nc.id, data)
	if err != nil {
		return hash, err
	}
	return hash, nc.PM.write(toid, toaddr, packet)
}

//OnConnected 处理连接成功的回调
func (nc *NetCore) OnConnected(id uint64, session uint32, p2pType uint32) {

	nc.PM.OnConnected(id, session, p2pType)
	p := nc.PM.peerByNetID(id)

	nc.ping(p.ID,&net.UDPAddr{IP:p.IP, Port: p.Port})
}

//OnConnected 处理接受连接的回调
func (nc *NetCore) OnAccepted(id uint64, session uint32, p2pType uint32) {

	nc.PM.OnConnected(id, session, p2pType)
}


//OnDisconnected 处理连接断开的回调
func (nc *NetCore) OnDisconnected(id uint64, session uint32, p2pCode uint32) {

	nc.PM.OnDisconnected(id, session, p2pCode)
}

//OnChecked 网络类型检查
func (nc *NetCore) OnChecked(p2pType uint32, privateIP string, publicIP string) {
	nc.ourEndPoint = MakeEndPoint(&net.UDPAddr{IP: net.ParseIP(publicIP), Port: 8686}, 8686)
	nc.PM.OnChecked(p2pType, privateIP, publicIP)
}

//OnRecved 数据回调
func (nc *NetCore) OnRecved(netID uint64, session uint32, data []byte) {

	//fmt.Printf("OnRecved : netID:%v session:%v len :%v\n ", netID, session,len(data))

	p := nc.PM.peerByNetID(netID)
	if p == nil {
		//fmt.Printf("OnConnected : no peer id:%v mynid:%v\n ", id, nc.nid)
		p = newPeer(NodeID{}, 0)
		nc.PM.addPeer(netID, p)
	}
	p.addData(data)

	nc.unhandle<-p
}


func encodeDataPacket(id NodeID, data []byte) (msg *bytes.Buffer, hash []byte, err error) {
	ptype := MessageType_MessageData
	b := new(bytes.Buffer)
	//fmt.Printf("encodeDataPacket id %v \n ", id)

	length := len(data)
	err = binary.Write(b, binary.BigEndian, uint32(ptype))
	if err != nil {
		return nil, nil, err
	}
	b.Write(id[:])
	err = binary.Write(b, binary.BigEndian, uint32(length))
	if err != nil {
		return nil, nil, err
	}

	b.Write(data)
	return b, nil, nil
}

func encodePacket(id NodeID, ptype MessageType, req proto.Message) (msg *bytes.Buffer, hash []byte, err error) {
	b := new(bytes.Buffer)

	pdata, err := proto.Marshal(req)
	//fmt.Printf("decodePacket id %v \n ", id)

	// fmt.Printf("encodePacket : msgType: %v \n ", ptype)
	if err != nil {
		// If this ever happens, it will be caught by the unit tests.
		//log.Error("Can'nc encode packet", "err", err)
		return nil, nil, err
	}
	length := len(pdata)
	err = binary.Write(b, binary.BigEndian, uint32(ptype))
	if err != nil {
		return nil, nil, err
	}
	b.Write(id[:])
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
	msgType, fromID, packetSize, msg, data, err := decodePacket(buf)


	if err != nil {
		if err == errPacketTooSmall {
			p.addDataToHead(buf.Bytes())
		}

		//log.Debug("Bad discv4 packet", "addr", from, "err", err)
		return err
	}
	if fromID != p.ID {
		p.ID = fromID
	}
	if int(packetSize) < buf.Len() {
		p.addDataToHead(buf.Bytes()[packetSize:])
	}

	//fmt.Printf("handleMessage : msgType: %v \n", msgType)

	switch msgType {
	case MessageType_MessagePing:
		nc.handlePing(msg.(*Ping), fromID)
	case MessageType_MessagePong:
		nc.handlePong(msg.(*Pong), fromID)
	case MessageType_MessageFindnode:
		nc.handleFindNode(msg.(*FindNode), fromID)
	case MessageType_MessageNeighbors:
		nc.handleNeighbors(msg.(*Neighbors), fromID)
	case MessageType_MessageData:
		nc.handleData(data, fromID)
	default:
		return fmt.Errorf("unknown type: %d", msgType)
	}
	return nil
}

func decodePacket(buffer *bytes.Buffer) (MessageType,NodeID, int, proto.Message, []byte, error) {
	buf := buffer.Bytes()
	msgType := MessageType(binary.BigEndian.Uint32(buf[:typeSize]))
	ID := MustBytesID(buf[typeSize : typeSize+idSize])
	msgLen := binary.BigEndian.Uint32(buf[typeSize+idSize : typeSize+idSize+lenSize])
	packetSize := int(msgLen + headSize)

	//fmt.Printf("decodePacket id %v \n ",ID.B58String())
	//fmt.Printf("decodePacket :packetSize: %v  msgType: %v  msgLen:%v   bufSize:%v\n ", packetSize, msgType, msgLen, len(buf))

	if buffer.Len() < packetSize {
		return MessageType_MessageNone,ID, 0, nil, nil, errPacketTooSmall
	}
	data := buf[headSize : headSize+msgLen]
	var req proto.Message
	switch msgType {
	case MessageType_MessagePing:
		req = new(Ping)
	case MessageType_MessagePong:
		req = new(Pong)
	case MessageType_MessageFindnode:
		req = new(FindNode)
	case MessageType_MessageNeighbors:
		req = new(Neighbors)
	case MessageType_MessageData:
		req = nil
	default:
		return msgType, ID,packetSize, nil, data, fmt.Errorf("unknown type: %d", msgType)
	}

	var err error
	if msgType != MessageType_MessageData {
		err = proto.Unmarshal(data, req)
	}
	//fmt.Printf("decodePacket end\n ")

	return msgType, ID, packetSize, req, data, err
}

func (nc *NetCore) handlePing(req *Ping, fromID NodeID) error {

	//fmt.Printf("handlePing from ip:%v %v to ip:%v %v \n ", req.From.IP, req.From.Port, req.To.IP, req.To.Port)

	if expired(req.Expiration) {
		return errExpired
	}
	p := nc.PM.peerByID(fromID)
	if p != nil {
		//fmt.Printf("update  ip \n ")

		p.IP = net.ParseIP(req.From.IP)
		p.Port =  int(req.From.Port)
	}
	from := net.UDPAddr{IP: net.ParseIP(req.From.IP), Port: int(req.From.Port)}
	to := MakeEndPoint(&from, req.From.Port)
	nc.Send(fromID, &from, MessageType_MessagePong, &Pong{
		To:         &to,
		ReplyToken: nil,
		Expiration: uint64(time.Now().Add(expiration).Unix()),
	})
	if !nc.handleReply(fromID, MessageType_MessagePing, req) {
		//fmt.Printf("handlePing bond\n")

		// Note: we're ignoring the provided IP address right now
		go nc.kad.bond(true, fromID, &from, int(req.From.Port))
	}
	return nil
}

func (nc *NetCore) handlePong(req *Pong, fromID NodeID) error {
	if expired(req.Expiration) {
		return errExpired
	}
	if !nc.handleReply(fromID, MessageType_MessagePong, req) {
		return errUnsolicitedReply
	}
	return nil
}

func (nc *NetCore) handleFindNode(req *FindNode, fromID NodeID) error {

	if expired(req.Expiration) {
		return errExpired
	}
	// if !nc.db.hasBond(fromID) {
	// 	// No bond exists, we don't process the packet. This prevents
	// 	// an attack vector where the discovery protocol could be used
	// 	// to amplify traffic in a DDOS attack. A malicious actor
	// 	// would send a findnode request with the IP address and UDP
	// 	// port of the target as the source address. The recipient of
	// 	// the findnode packet would then send a neighbors packet
	// 	// (which is a much bigger packet than findnode) to the victim.
	// 	return errUnknownNode
	// }
	target := req.Target
	nc.kad.mutex.Lock()
	closest := nc.kad.closest(target, bucketSize).entries
	nc.kad.mutex.Unlock()

	p := Neighbors{Expiration: uint64(time.Now().Add(expiration).Unix())}
	var sent bool
	for _, n := range closest {
		//if netutil.CheckRelayIP(from.IP, n.IP) == nil {
		node := nodeToRPC(n)
		if len(node.IP) >0 && node.Port >0 {
			p.Nodes = append(p.Nodes, &node)
		}
		//fmt.Printf("handleFindNode id:%v      ip:%v   port:%v\n ", node.ID, node.IP, node.Port)

		//}
		if len(p.Nodes) == maxNeighbors {
			nc.Send(fromID, nil, MessageType_MessageNeighbors, &p)
			p.Nodes = p.Nodes[:0]
			sent = true
		}
	}
	if len(p.Nodes) > 0 || !sent {
		nc.Send(fromID, nil, MessageType_MessageNeighbors, &p)
	}
	//fmt.Printf("handleFindNode size:%v \n ", len(p.Nodes))

	return nil
}

func (nc *NetCore) handleNeighbors(req *Neighbors, fromID NodeID) error {
	if expired(req.Expiration) {
		return errExpired
	}
	if !nc.handleReply(fromID, MessageType_MessageNeighbors, req) {
		return errUnsolicitedReply
	}
	return nil
}

func (nc *NetCore) handleData(data []byte, fromID NodeID) error {
	id := fromID.GetHexString()
	//fmt.Printf("from:%v  len:%v \n", id, len(data))
	Server.handleMessage(data,id,time.Now())
	return nil
}

func expired(ts uint64) bool {
	return time.Unix(int64(ts), 0).Before(time.Now())
}
