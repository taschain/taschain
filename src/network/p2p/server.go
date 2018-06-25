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
	"common"
	"middleware/pb"
	"sync"
	"bufio"
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

	streams map[string]inet.Stream

	streamMapLock sync.RWMutex
}

func InitServer(host host.Host, dht *dht.IpfsDHT, node *Node) {
	logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))
	host.SetStreamHandler(ProtocolTAS, swarmStreamHandler)
	Server = server{Host: host, Dht: dht, SelfNetInfo: node, streams: make(map[string]inet.Stream)}
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

		s.send(b, id)
	}()

}

func (s *server) send(b []byte, id string) {
	if id == s.SelfNetInfo.Id {
		s.sendSelf(b, id)
		return
	}
	ctx := context.Background()
	context.WithTimeout(ctx, ContextTimeOut)

	c, cancel := context.WithCancel(context.Background())
	context.WithTimeout(c, ContextTimeOut)
	defer cancel()

	s.streamMapLock.Lock()
	stream := s.streams[id]
	if stream == nil {
		var e error
		stream, e = s.Host.NewStream(c, ConvertToPeerID(id), ProtocolTAS)
		if e != nil {
			logger.Errorf("New stream for %s error:%s", id, e.Error())
			s.streamMapLock.Unlock()
			return
		}
		s.streams[id] = stream
	}

	//stream, e := s.Host.NewStream(c, ConvertToPeerID(id), ProtocolTAS)
	//if e != nil {
	//	logger.Errorf("New stream for %s error:%s", id, e.Error())
	//	return
	//}
	//defer stream.Close()
	l := len(b)
	r, err := stream.Write(b)
	if err != nil {
		logger.Errorf("Write stream for %s error:%s", id, err.Error())
		stream.Close()
		s.streams[id] = nil
		s.streamMapLock.Unlock()
		s.send(b,id)
		return
	}
	s.streamMapLock.Unlock()

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
	go func() {
		for {
			e := handleStream(stream)
			if e != nil {
				stream.Close()
				break
			}
		}
	}()
	//handleStream(stream)
}
func handleStream(stream inet.Stream) error {
	//defer stream.Close()
	reader := bufio.NewReader(stream)
	headerBytes := make([]byte, 3)
	h, e1 := reader.Read(headerBytes)
	if e1 != nil {
		logger.Errorf("steam read 3 from %d error:%d,! " ,ConvertToID(stream.Conn().RemotePeer()),e1.Error())
		return e1
	}
	if h != 3 {
		logger.Errorf("Stream  should read %d byte, but received %d bytes", 3, h)
		return nil
	}
	//校验 header
	if !(headerBytes[0] == byte(84) && headerBytes[1] == byte(65) && headerBytes[2] == byte(83)) {
		logger.Errorf("validate header error from %s! ",ConvertToID(stream.Conn().RemotePeer()))
		return nil
	}

	pkgLengthBytes := make([]byte, PACKAGE_LENGTH_SIZE)
	n, err := reader.Read(pkgLengthBytes)
	if err != nil {
		logger.Errorf("Stream  read4 error:%s", err.Error())
		return nil
	}
	if n != 4 {
		logger.Errorf("Stream  should read %d byte, but received %d bytes", 4, n)
		return nil
	}
	pkgLength := int(utility.ByteToUInt32(pkgLengthBytes))
	b := make([]byte, pkgLength)
	e := readMessageBody(reader, b, 0)
	if e != nil {
		logger.Errorf("Stream  readMessageBody error:%s", e.Error())
		return e
	}
	//b, err1 := ioutil.ReadAll(stream)
	//if err1 != nil {
	//	logger.Errorf("Stream  read error:%s", err1.Error())
	//	return
	//}
	//if len(b) != pkgLength {
	//	logger.Errorf("Stream  should read %d byte,but received %d bytes", pkgLength, len(b))
	//	return
	//}

	go Server.handleMessage(b, ConvertToID(stream.Conn().RemotePeer()), pkgLengthBytes)
	return nil
}

func readMessageBody(reader *bufio.Reader, body []byte, index int) error {
	if index == 0 {
		n, err1 := reader.Read(body)
		if err1 != nil {
			return err1
		}
		if n != len(body) {
			return readMessageBody(reader, body, n)
		}
		return nil
	} else {
		b := make([]byte, len(body)-index)
		n, err2 := reader.Read(b)
		if err2 != nil {
			return err2
		}
		copy(body[index:], b[:])
		if n != len(b) {
			return readMessageBody(reader, body, index+n)
		}
		return nil
	}

}
func (s *server) handleMessage(b []byte, from string, lengthByte []byte) {
	message := new(tas_middleware_pb.Message)
	error := proto.Unmarshal(b, message)
	if error != nil {
		logger.Errorf("[Network]Proto unmarshal error:%s", error.Error())
	}

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
	case TRANSACTION_MSG, TRANSACTION_GOT_MSG:
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
