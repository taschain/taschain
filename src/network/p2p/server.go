package p2p

import (
	inet "github.com/libp2p/go-libp2p-net"

	"utility"
	"github.com/libp2p/go-libp2p-host"
	"context"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/golang/protobuf/proto"
	"pb"
	pstore "github.com/libp2p/go-libp2p-peerstore"

	"strings"
	"taslog"
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

	TRANSACTION_GOT_MSG uint32 = 0x07

	TRANSACTION_MSG uint32 = 0x08

	NEW_BLOCK_MSG uint32 = 0x09

	//-----------块同步---------------------------------
	REQ_BLOCK_CHAIN_HEIGHT_MSG uint32 = 0x0a

	BLOCK_CHAIN_HEIGHT_MSG uint32 = 0x0b

	REQ_BLOCK_MSG uint32 = 0x0c

	BLOCK_MSG uint32 = 0x0d

	//-----------组同步---------------------------------
	REQ_GROUP_CHAIN_HEIGHT_MSG uint32 = 0x0e

	GROUP_CHAIN_HEIGHT_MSG uint32 = 0x0f

	REQ_GROUP_MSG uint32 = 0x10

	GROUP_MSG uint32 = 0x11
)

var logger = taslog.GetLogger(taslog.P2PConfig)

var Server server

type server struct {
	SelfNetInfo *Node

	Host host.Host

	Dht *dht.IpfsDHT
}

func InitServer(host host.Host, dht *dht.IpfsDHT, node *Node) {

	host.Network().SetStreamHandler(swarmStreamHandler)

	Server = server{Host: host, Dht: dht, SelfNetInfo: node}
}

func (s *server) SendMessage(m Message, id string) {
	bytes, e := MarshalMessage(m)
	if e != nil {
		logger.Errorf("Marshal message error:%s\n", e.Error())
		return
	}

	length := len(bytes)
	b2 := utility.UInt32ToByte(uint32(length))

	//"TAS"的byte
	header := []byte{84, 65, 83}

	b := make([]byte, len(bytes)+len(b2)+3)
	copy(b[:3], header[:])
	copy(b[3:7], b2)
	copy(b[7:], bytes)


	s.send(b, id)
}

func (s *server) send(b []byte, id string) {
	peerInfo, error := s.Dht.FindPeer(context.Background(), ConvertToPeerID(id))
	if error != nil || string(peerInfo.ID) == "" {
		logger.Errorf("dht find peer error:%s,peer id:%s\n", error.Error(), id)
		panic("DHT find peer error!")
	}
	s.Host.Network().Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, pstore.PermanentAddrTTL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, e := s.Host.Network().NewStream(ctx, ConvertToPeerID(id))
	defer stream.Close()
	if e != nil {
		logger.Errorf("New stream for %s error:%s\n", id, error.Error())
		panic("New stream error!")
	}
	l := len(b)
	if l < PACKAGE_MAX_SIZE {
		r, err := stream.Write(b)
		if err != nil {
			logger.Errorf("Write stream for %s error:%s\n", id, error.Error())
			return
		}

		if r != l {
			logger.Errorf("Stream  should write %d byte ,bu write %d bytes\n", l, r)
			return
		}
	} else {
		n := l / PACKAGE_MAX_SIZE
		left, right := 0, PACKAGE_MAX_SIZE
		for i := 0; i <= n; i++ {
			a := make([]byte, PACKAGE_MAX_SIZE)
			copy(a, b[left:right])
			r, err := stream.Write(a)
			if err != nil {
				logger.Errorf("Write stream for %s error:%s\n", id, error.Error())
				return
			}
			if r != PACKAGE_MAX_SIZE {
				logger.Errorf("Stream  should write %d byte ,bu write %d bytes\n", PACKAGE_MAX_SIZE, r)
				return
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
	headerBytes := make([]byte, 3)
	h, e1 := stream.Read(headerBytes)
	if e1 != nil {
		logger.Errorf("Stream  read error:%s\n", e1.Error())
		return
	}
	if h != 3 {
		return
	}
	//校验 header
	if !(headerBytes[0] == byte(84) && headerBytes[1] == byte(65) && headerBytes[2] == byte(83)) {
		return
	}

	pkgLengthBytes := make([]byte, PACKAGE_LENGTH_SIZE)
	n, err := stream.Read(pkgLengthBytes)
	if err != nil {
		logger.Errorf("Stream  read error:%s\n", err.Error())
		return
	}
	if n != 4 {
		logger.Errorf("Stream  should read %d byte, but received %d bytes\n", 4, n)
		return
	}
	pkgLength := int(utility.ByteToUInt32(pkgLengthBytes))
	pkgBodyBytes := make([]byte, pkgLength)
	if pkgLength < PACKAGE_MAX_SIZE {
		n1, err1 := stream.Read(pkgBodyBytes)
		if err1 != nil {
			logger.Errorf("Stream  read error:%s\n", err.Error())
			return
		}
		if n1 != pkgLength {
			logger.Errorf("Stream  should read %d byte,but received %d bytes\n", pkgLength, n1)
			return
		}
	} else {
		c := pkgLength / PACKAGE_MAX_SIZE
		left, right := 0, PACKAGE_MAX_SIZE
		for i := 0; i <= c; i++ {
			a := make([]byte, PACKAGE_MAX_SIZE)
			n1, err1 := stream.Read(a)
			if err1 != nil {
				logger.Errorf("Stream  read error:%s\n", err.Error())
				return
			}

			if n1 != PACKAGE_MAX_SIZE {
				logger.Errorf("Stream should  read %d byte,but received %d bytes\n", PACKAGE_MAX_SIZE, n1)
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
	Server.handleMessage(pkgBodyBytes, ConvertToID(stream.Conn().RemotePeer()))
}

func (s *server) handleMessage(b []byte, from string) {
	message := new(tas_pb.Message)
	error := proto.Unmarshal(b, message)
	if error != nil {
		logger.Errorf("Proto unmarshal error:%s\n", error.Error())
	}

	code := message.Code
	switch *code {
	case GROUP_INIT_MSG, KEY_PIECE_MSG, GROUP_INIT_DONE_MSG, CURRENT_GROUP_CAST_MSG, CAST_VERIFY_MSG, VARIFIED_CAST_MSG:
		consensusHandler.HandlerMessage(*code, message.Body, from)
	case REQ_TRANSACTION_MSG, TRANSACTION_MSG, REQ_BLOCK_CHAIN_HEIGHT_MSG, BLOCK_CHAIN_HEIGHT_MSG, REQ_BLOCK_MSG, BLOCK_MSG,
		REQ_GROUP_CHAIN_HEIGHT_MSG, GROUP_CHAIN_HEIGHT_MSG, REQ_GROUP_MSG, GROUP_MSG:
		chainHandler.HandlerMessage(*code, message.Body, from)
	case NEW_BLOCK_MSG, TRANSACTION_GOT_MSG:
		e := chainHandler.HandlerMessage(*code, message.Body, from)
		if e != nil {
			consensusHandler.HandlerMessage(*code, message.Body, from)
		}
	}
}

type ConnInfo struct {
	Id      string `json:"id"`
	Ip      string `json:"ip"`
	TcpPort string `json:"tcp_port"`
}

func (s *server) GetConnInfo() []ConnInfo {
	conns := s.Host.Network().Conns()
	result := []ConnInfo{}
	for _, conn := range conns {
		id := ConvertToID(conn.RemotePeer())
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
