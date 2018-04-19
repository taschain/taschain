package p2p

import (
	inet "github.com/libp2p/go-libp2p-net"

	"utility"
	"github.com/libp2p/go-libp2p-host"
	"context"
	gpeer "github.com/libp2p/go-libp2p-peer"
	"network/biz"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/golang/protobuf/proto"
	"pb"
	"taslog"
	pstore "github.com/libp2p/go-libp2p-peerstore"

	"strings"
	"consensus/groupsig"
)

const (
	PACKAGE_MAX_SIZE = 1024 * 1024

	PACKAGE_LENGTH_SIZE = 4

	CODE_SIZE = 4

	//-----------组初始化---------------------------------
	GROUP_INIT_MSG uint32 = 0x00

	KEY_PIECE_MSG uint32 = 0x01

	GROUP_INIT_DONE_MSG uint32 = 0x02

	//-----------组铸币---------------------------------
	CURRENT_GROUP_CAST_MSG uint32 = 0x03

	CAST_VERIFY_MSG uint32 = 0x04

	VARIFIED_CAST_MSG uint32 = 0x05

	REQ_TRANSACTION_MSG uint32 = 0x06

	TRANSACTION_MSG uint32 = 0x07

	NEW_BLOCK_MSG uint32 = 0x08

	//-----------块同步---------------------------------
	REQ_BLOCK_CHAIN_HEIGHT_MSG uint32 = 0x09

	BLOCK_CHAIN_HEIGHT_MSG uint32 = 0x0a

	REQ_BLOCK_MSG uint32 = 0x0b

	BLOCK_MSG uint32 = 0x0c

	//-----------组同步---------------------------------
	REQ_GROUP_CHAIN_HEIGHT_MSG uint32 = 0x0d

	GROUP_CHAIN_HEIGHT_MSG uint32 = 0x0e

	REQ_GROUP_MSG uint32 = 0x0f

	GROUP_MSG uint32 = 0x10
)

var Server server

type server struct {
	host host.Host

	dht *dht.IpfsDHT

	bHandler biz.BlockChainMessageHandler

	cHandler biz.ConsensusMessageHandler
}

func InitServer(host host.Host, dht *dht.IpfsDHT,	bHandler biz.BlockChainMessageHandler,	cHandler biz.ConsensusMessageHandler) {

	host.Network().SetStreamHandler(swarmStreamHandler)

	Server = server{host: host, dht: dht, bHandler: bHandler, cHandler: cHandler}
}

func (s *server) SendMessage(m Message, id string) {
	bytes, e := MarshalMessage(m)
	if e != nil {
		taslog.P2pLogger.Errorf("Marshal message error:%s\n", e.Error())
		return
	}

	length := len(bytes)
	b2 := utility.UInt32ToByte(uint32(length))

	b := make([]byte, len(bytes)+len(b2))
	copy(b[:4], b2)
	copy(b[4:], bytes)

	s.send(b, id)
}

func (s *server) send(b []byte, id string) {
	peerInfo, error := s.dht.FindPeer(context.Background(), gpeer.ID(id))
	if error != nil || peerInfo.ID.String() == "" {
		taslog.P2pLogger.Errorf("dht find peer error:%s,peer id:%s\n", error.Error(), id)
		panic("DHT find peer error!")
	}
	s.host.Network().Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, pstore.PermanentAddrTTL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, e := s.host.Network().NewStream(ctx, gpeer.ID(id))
	defer stream.Close()
	if e != nil {
		taslog.P2pLogger.Errorf("New stream for %s error:%s\n", id, error.Error())
		panic("New stream error!")
	}
	l := len(b)
	if l < PACKAGE_MAX_SIZE {
		r, err := stream.Write(b)
		if r != l || err != nil {
			taslog.P2pLogger.Errorf("Write stream for %s error:%s\n", id, error.Error())
			panic("Writew stream error!")
		}
	} else {
		n := l / PACKAGE_MAX_SIZE
		left, right := 0, PACKAGE_MAX_SIZE
		for i := 0; i <= n; i++ {
			a := make([]byte, PACKAGE_MAX_SIZE)
			copy(a, b[left:right])
			r, err := stream.Write(a)
			if r != PACKAGE_MAX_SIZE || err != nil {

			}
			left += PACKAGE_MAX_SIZE
			right += PACKAGE_MAX_SIZE
			if right > l {
				right = l
			}
		}
	}
}

//TODO 考虑读写超时
func swarmStreamHandler(stream inet.Stream) {
	defer stream.Close()
	pkgLengthBytes := make([]byte, PACKAGE_LENGTH_SIZE)
	n, err := stream.Read(pkgLengthBytes)
	if n != 4 || err != nil {
		taslog.P2pLogger.Errorf("Stream  read %d byte error:%s,received %d bytes\n", 4, err.Error(), n)
		return
	}
	pkgLength := int(utility.ByteToUInt32(pkgLengthBytes))
	pkgBodyBytes := make([]byte, pkgLength)
	if pkgLength < PACKAGE_MAX_SIZE {
		n1, err1 := stream.Read(pkgBodyBytes)
		if n1 != pkgLength || err1 != nil {
			taslog.P2pLogger.Errorf("Stream  read %d byte error:%s,received %d bytes\n", pkgLength, err.Error(), n)
			return
		}
	} else {
		c := pkgLength / PACKAGE_MAX_SIZE
		left, right := 0, PACKAGE_MAX_SIZE
		for i := 0; i <= c; i++ {
			a := make([]byte, PACKAGE_MAX_SIZE)
			n1, err1 := stream.Read(a)
			if n1 != PACKAGE_MAX_SIZE || err1 != nil {
				taslog.P2pLogger.Errorf("Stream  read %d byte error:%s,received %d bytes\n", PACKAGE_MAX_SIZE, err.Error(), n1)
				return
			}
			copy(pkgBodyBytes[left:right], a)
			left += PACKAGE_MAX_SIZE
			right += PACKAGE_MAX_SIZE
			if right > pkgLength {
				right = pkgLength
			}
		}
	}
	Server.handleMessage(pkgBodyBytes, (string)(stream.Conn().RemotePeer()))
}

func (s *server) handleMessage(b []byte, from string) {
	if len(b) < 4 {
		taslog.P2pLogger.Errorf("Message  format error!\n")
		return
	}
	message := new(tas_pb.Message)
	error := proto.Unmarshal(b, message)
	if error != nil {
		taslog.P2pLogger.Errorf("Proto unmarshal error:%s\n", error.Error())
	}

	code := message.Code
	switch *code {
	case GROUP_INIT_MSG:
		m, e := UnMarshalConsensusGroupRawMessage(b)
		if e != nil {
			taslog.P2pLogger.Error("Discard ConsensusGroupRawMessage because of unmarshal error!\n")
			return
		}
		Peer.KeyMap = make(map[groupsig.ID]string)
		for i := 0; i < len(m.Ids); i++ {
			userId := m.UserIds[i]
			if userId == "" {
				taslog.P2pLogger.Error("Bad ConsensusGroupRawMessage:uers is is null.Discard!\n")
				return
			}
			Peer.KeyMap[m.Ids[i]] = m.UserIds[i]
		}
		s.cHandler.OnMessageGroupInitFn(*m)
	case KEY_PIECE_MSG:
		m, e := UnMarshalConsensusSharePieceMessage(b)
		if e != nil {
			taslog.P2pLogger.Error("Discard ConsensusSharePieceMessage because of unmarshal error!\n")
			return
		}
		s.cHandler.OnMessageSharePieceFn(*m)
	case GROUP_INIT_DONE_MSG:
		m, e := UnMarshalConsensusGroupInitedMessage(b)
		if e != nil {
			taslog.P2pLogger.Error("Discard ConsensusGroupInitedMessage because of unmarshal error!\n")
			return
		}
		s.cHandler.OnMessageGroupInitedFn(*m)
	case CURRENT_GROUP_CAST_MSG:
		m,e := UnMarshalConsensusCurrentMessage(b)
		if e != nil {
			taslog.P2pLogger.Error("Discard ConsensusCurrentMessage because of unmarshal error!\n")
			return
		}
		s.cHandler.OnMessageCurrentGroupCastFn(*m)
	case CAST_VERIFY_MSG:
		m,e := UnMarshalConsensusCastMessage(b)
		if e != nil {
			taslog.P2pLogger.Error("Discard ConsensusCastMessage because of unmarshal error!\n")
			return
		}
		s.cHandler.OnMessageCastFn(*m)
	case VARIFIED_CAST_MSG:
		m,e := UnMarshalConsensusVerifyMessage(b)
		if e != nil {
			taslog.P2pLogger.Error("Discard ConsensusVerifyMessage because of unmarshal error!\n")
			return
		}
		s.cHandler.OnMessageVerifiedCastFn(*m)

	default:
		taslog.P2pLogger.Errorf("Message not support! Code:%d\n", code)
	}
}

type ConnInfo struct {
	Id      string
	Ip      string
	TcpPort string
}

func (s *server) GetConnInfo() []ConnInfo {
	conns := s.host.Network().Conns()
	result := []ConnInfo{}
	for _, conn := range conns {
		id := conn.RemotePeer().Pretty()
		if id == "" {
			continue
		}
		addr := conn.RemoteMultiaddr().String()
		//addr /ip4/127.0.0.1/udp/1234"
		split := strings.Split(addr, "/")
		if len(split) != 5 {
			continue
		}
		ip := split[2]
		port := split[4]
		c := ConnInfo{Id: id, Ip: ip, TcpPort: port}
		result = append(result, c)
	}
	return result
}
