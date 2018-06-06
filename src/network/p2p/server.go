package p2p

import (
	inet "github.com/libp2p/go-libp2p-net"

	"utility"
	"github.com/libp2p/go-libp2p-host"
	"context"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/golang/protobuf/proto"
	"pb"
	"strings"
	"taslog"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p-protocol"
	"time"
	"common"
)

const (
	PACKAGE_MAX_SIZE = 1024 * 1024

	PACKAGE_LENGTH_SIZE = 4

	CODE_SIZE = 4

	//-----------组初始化---------------------------------
	GROUP_MEMBER_MSG uint32 = 0x00

	GROUP_INIT_MSG uint32 = 0x01

	KEY_PIECE_MSG uint32 = 0x02

	SIGN_PUBKEY_MSG uint32 = 0x03

	GROUP_INIT_DONE_MSG uint32 = 0x04

	//-----------组铸币---------------------------------
	CURRENT_GROUP_CAST_MSG uint32 = 0x05

	CAST_VERIFY_MSG uint32 = 0x06

	VARIFIED_CAST_MSG uint32 = 0x07

	REQ_TRANSACTION_MSG uint32 = 0x08

	TRANSACTION_GOT_MSG uint32 = 0x09

	TRANSACTION_MSG uint32 = 0x0a

	NEW_BLOCK_MSG uint32 = 0x0b

	//-----------块同步---------------------------------
	REQ_BLOCK_CHAIN_TOTAL_QN_MSG uint32 = 0x0c

	BLOCK_CHAIN_TOTAL_QN_MSG uint32 = 0x0d

	REQ_BLOCK_MSG uint32 = 0x0e

	BLOCK_MSG uint32 = 0x0f

	//-----------组同步---------------------------------
	REQ_GROUP_CHAIN_HEIGHT_MSG uint32 = 0x10

	GROUP_CHAIN_HEIGHT_MSG uint32 = 0x11

	REQ_GROUP_MSG uint32 = 0x12

	GROUP_MSG uint32 = 0x13
	//-----------块链调整---------------------------------
	BLOCK_CHAIN_HASHES_REQ uint32 = 0x14

	BLOCK_CHAIN_HASHES uint32 = 0x15
)

var ProtocolTAS protocol.ID = "/tas/1.0.0"

var ContextTimeOut = time.Minute * 5

var logger taslog.Logger

var Server server

type server struct {
	SelfNetInfo *Node

	Host host.Host

	Dht *dht.IpfsDHT
}

func InitServer(host host.Host, dht *dht.IpfsDHT, node *Node) {
	host.SetStreamHandler(ProtocolTAS, swarmStreamHandler)

	Server = server{Host: host, Dht: dht, SelfNetInfo: node}
}

func (s *server) SendMessage(m Message, id string) {
	go func() {
		beginTime := time.Now()
		bytes, e := MarshalMessage(m)
		if e != nil {
			logger.Errorf("[Network]Marshal message error:%s", e.Error())
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
		logger.Debugf("[p2p] Send message to:%s,message body hash is:%x,cost time:%v",id,common.BytesToHash(m.Body),time.Since(beginTime).String())
	}()

}

func (s *server) send(b []byte, id string) {
	if id == s.SelfNetInfo.Id {
		s.sendSelf(b, id)
		return
	}
	ctx := context.Background()
	context.WithTimeout(ctx, ContextTimeOut)
	peerInfo, error := s.Dht.FindPeer(ctx, ConvertToPeerID(id))
	if error != nil || string(peerInfo.ID) == "" {
		logger.Errorf("dht find peer error:%s,peer id:%s", error.Error(), id)
	} else {
		s.Host.Network().Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, pstore.PermanentAddrTTL)
	}

	c, cancel := context.WithCancel(context.Background())
	context.WithTimeout(c, ContextTimeOut)
	defer cancel()

	stream, e := s.Host.NewStream(c, ConvertToPeerID(id), ProtocolTAS)
	if e != nil {
		logger.Errorf("New stream for %s error:%s", id, e.Error())
		return
	}
	defer stream.Close()
	l := len(b)
	if l < PACKAGE_MAX_SIZE {
		r, err := stream.Write(b)
		if err != nil {
			logger.Errorf("Write stream for %s error:%s", id, err.Error())
			return
		}

		if r != l {
			logger.Errorf("Stream  should write %d byte ,bu write %d bytes", l, r)
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
				logger.Errorf("Write stream for %s error:%s", id, err.Error())
				return
			}
			if r != PACKAGE_MAX_SIZE {
				logger.Errorf("Stream  should write %d byte ,bu write %d bytes", PACKAGE_MAX_SIZE, r)
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

func (s *server) sendSelf(b []byte, id string) {
	pkgBodyBytes := b[7:]
	s.handleMessage(pkgBodyBytes, id)
}

//TODO 考虑读写超时
func swarmStreamHandler(stream inet.Stream) {
	go handleStream(stream)
}
func handleStream(stream inet.Stream) {

	defer stream.Close()
	headerBytes := make([]byte, 3)
	h, e1 := stream.Read(headerBytes)
	if e1 != nil {
		logger.Errorf("Stream  read error:%s", e1.Error())
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
		logger.Errorf("Stream  read error:%s", err.Error())
		return
	}
	if n != 4 {
		logger.Errorf("Stream  should read %d byte, but received %d bytes", 4, n)
		return
	}
	pkgLength := int(utility.ByteToUInt32(pkgLengthBytes))
	pkgBodyBytes := make([]byte, pkgLength)
	if pkgLength < PACKAGE_MAX_SIZE {
		n1, err1 := stream.Read(pkgBodyBytes)
		if err1 != nil {
			logger.Errorf("Stream  read error:%s", err1.Error())
			return
		}
		if n1 != pkgLength {
			logger.Errorf("Stream  should read %d byte,but received %d bytes", pkgLength, n1)
			return
		}
	} else {
		c := pkgLength / PACKAGE_MAX_SIZE
		left, right := 0, PACKAGE_MAX_SIZE
		for i := 0; i <= c; i++ {
			a := make([]byte, PACKAGE_MAX_SIZE)
			n1, err1 := stream.Read(a)
			if err1 != nil {
				logger.Errorf("Stream  read error:%s", err1.Error())
				return
			}

			if n1 != PACKAGE_MAX_SIZE {
				logger.Errorf("Stream should  read %d byte,but received %d bytes", PACKAGE_MAX_SIZE, n1)
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
		logger.Errorf("[Network]Proto unmarshal error:%s", error.Error())
	}
	logger.Debugf("[p2p] Receive message from:%s,message body hash is:%x,cost time:%v",from,common.BytesToHash(message.Body))

	code := message.Code
	switch *code {
	case GROUP_MEMBER_MSG, GROUP_INIT_MSG, KEY_PIECE_MSG, SIGN_PUBKEY_MSG, GROUP_INIT_DONE_MSG, CURRENT_GROUP_CAST_MSG, CAST_VERIFY_MSG,
		VARIFIED_CAST_MSG:
		consensusHandler.HandlerMessage(*code, message.Body, from)
	case REQ_TRANSACTION_MSG, TRANSACTION_MSG, REQ_BLOCK_CHAIN_TOTAL_QN_MSG, BLOCK_CHAIN_TOTAL_QN_MSG, REQ_BLOCK_MSG, BLOCK_MSG,
		REQ_GROUP_CHAIN_HEIGHT_MSG, GROUP_CHAIN_HEIGHT_MSG, REQ_GROUP_MSG, GROUP_MSG, BLOCK_CHAIN_HASHES_REQ, BLOCK_CHAIN_HASHES:
		chainHandler.HandlerMessage(*code, message.Body, from)
	case NEW_BLOCK_MSG:
		consensusHandler.HandlerMessage(*code, message.Body, from)
	case TRANSACTION_GOT_MSG:
		_, e := chainHandler.HandlerMessage(*code, message.Body, from)
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
