package p2p

import (
	inet "github.com/libp2p/go-libp2p-net"

	"utility"
	"github.com/libp2p/go-libp2p-host"
	"context"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/golang/protobuf/proto"

	"strings"
	"taslog"
	"github.com/libp2p/go-libp2p-protocol"
	"time"
	"io/ioutil"
	"common"
	"middleware/pb"
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

	REQ_BLOCK_INFO uint32 = 0x0e

	BLOCK_INFO uint32 = 0x0f

	//-----------组同步---------------------------------
	REQ_GROUP_CHAIN_HEIGHT_MSG uint32 = 0x10

	GROUP_CHAIN_HEIGHT_MSG uint32 = 0x11

	REQ_GROUP_MSG uint32 = 0x12

	GROUP_MSG uint32 = 0x13
	//-----------块链调整---------------------------------
	BLOCK_HASHES_REQ uint32 = 0x14

	BLOCK_HASHES uint32 = 0x15

	//广播自身上链过的BLOCK
	//ON_CHAIN_BLOCK_MSG uint32 = 0X16
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
	logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))
	host.SetStreamHandler(ProtocolTAS, swarmStreamHandler)
	Server = server{Host: host, Dht: dht, SelfNetInfo: node}
}

func (s *server) SendMessage(m Message, id string) {
	go func() {
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

		//beginTime := time.Now()
		s.send(b, id)
		//if (m.Code == CAST_VERIFY_MSG || m.Code == VARIFIED_CAST_MSG || m.Code == NEW_BLOCK_MSG) {
		//	logger.Debugf("[p2p] Send message to:%s,code:%d,message body hash is:%x,body length:%d,body length byte:%v,cost time:%v", id, m.Code, common.Sha256(m.Body), len(b), b2, time.Since(beginTime).String())
		//}
	}()

}

func (s *server) send(b []byte, id string) {
	if id == s.SelfNetInfo.Id {
		s.sendSelf(b, id)
		return
	}
	ctx := context.Background()
	context.WithTimeout(ctx, ContextTimeOut)
	//peerInfo, error := s.Dht.FindPeer(ctx, ConvertToPeerID(id))
	//if error != nil || string(peerInfo.ID) == "" {
	//	logger.Errorf("dht find peer error:%s,peer id:%s", error.Error(), id)
	//} else {
	//	s.Host.Network().Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, pstore.PermanentAddrTTL)
	//}

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
	r, err := stream.Write(b)
	if err != nil {
		logger.Errorf("Write stream for %s error:%s", id, err.Error())
		return
	}

	if r != l {
		logger.Errorf("Stream  should write %d byte ,bu write %d bytes", l, r)
		return
	}
}

func (s *server) sendSelf(b []byte, id string) {
	pkgBodyBytes := b[7:]
	s.handleMessage(pkgBodyBytes, id, b[3:7])
}

//TODO 考虑读写超时
func swarmStreamHandler(stream inet.Stream) {
	handleStream(stream)
}
func handleStream(stream inet.Stream) {

	beginTime := time.Now()
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
	b, err1 := ioutil.ReadAll(stream)
	if err1 != nil {
		logger.Errorf("Stream  read error:%s", err1.Error())
		return
	}
	if len(b) != pkgLength {
		logger.Errorf("Stream  should read %d byte,but received %d bytes,cost time:%v", pkgLength, len(b), time.Since(beginTime).String())
		return
	}

	Server.handleMessage(b, ConvertToID(stream.Conn().RemotePeer()), pkgLengthBytes)
}

func (s *server) handleMessage(b []byte, from string, lengthByte []byte) {
	message := new(tas_middleware_pb.Message)
	error := proto.Unmarshal(b, message)
	if error != nil {
		logger.Errorf("[Network]Proto unmarshal error:%s", error.Error())
	}
	//if (*message.Code == CAST_VERIFY_MSG || *message.Code == VARIFIED_CAST_MSG || *message.Code == NEW_BLOCK_MSG) {
	//	logger.Debugf("[p2p] Receive message from:%s,message body hash is:%x,body length is:%v", from, common.Sha256(message.Body), lengthByte)
	//}
	code := message.Code
	switch *code {
	case GROUP_MEMBER_MSG, GROUP_INIT_MSG, KEY_PIECE_MSG, SIGN_PUBKEY_MSG, GROUP_INIT_DONE_MSG, CURRENT_GROUP_CAST_MSG, CAST_VERIFY_MSG,
		VARIFIED_CAST_MSG:
		consensusHandler.HandlerMessage(*code, message.Body, from)
	case REQ_TRANSACTION_MSG, REQ_BLOCK_CHAIN_TOTAL_QN_MSG, BLOCK_CHAIN_TOTAL_QN_MSG, REQ_BLOCK_INFO, BLOCK_INFO,
		REQ_GROUP_CHAIN_HEIGHT_MSG, GROUP_CHAIN_HEIGHT_MSG, REQ_GROUP_MSG, GROUP_MSG, BLOCK_HASHES_REQ, BLOCK_HASHES:
		chainHandler.HandlerMessage(*code, message.Body, from)
	case NEW_BLOCK_MSG:
		consensusHandler.HandlerMessage(*code, message.Body, from)
	case TRANSACTION_MSG,TRANSACTION_GOT_MSG:
		_, e := chainHandler.HandlerMessage(*code, message.Body, from)
		if e != nil {
			return
		}
		consensusHandler.HandlerMessage(*code, message.Body, from)
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
