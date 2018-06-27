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
	"common"
	"middleware/pb"
	"sync"
	"bufio"
	"fmt"
	"time"
	"network"
)

const (
	PACKAGE_MAX_SIZE = 4 * 1024

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

var ContextTimeOut = -1

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

		beginTime := time.Now()
		s.send(b, id)
		if m.Code == CAST_VERIFY_MSG {
			network.Logger.Debugf("send CAST_VERIFY_MSG to %s, byte:%d,send message cost time %v", id, len(b), time.Since(beginTime))
		}
	}()

}

func (s *server) send(b []byte, id string) {
	if id == s.SelfNetInfo.Id {
		s.sendSelf(b, id)
		return
	}
	c := context.Background()

	//peerInfo, e := s.Dht.FindPeer(c, ConvertToPeerID(id))
	//if e != nil || string(peerInfo.ID) == "" {
	//	logger.Errorf("dht find peer error:%s,peer id:%s", e.Error(), id)
	//} else {
	//	s.Host.Network().Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, pstore.PermanentAddrTTL)
	//}

	s.streamMapLock.Lock()
	stream := s.streams[id]
	var e1 error
	if stream == nil {
		stream, e1 = s.Host.NewStream(c, ConvertToPeerID(id), ProtocolTAS)
		if e1 != nil {
			logger.Errorf("New stream for %s error:%s", id, e1.Error())
			s.streamMapLock.Unlock()
			return
		}
		s.streams[id] = stream
	}
	e2 := s.writePackage(stream, b, id)

	if e2 != nil {
		stream.Close()
		stream, e1 = s.Host.NewStream(c, ConvertToPeerID(id), ProtocolTAS)
		if e1 != nil {
			logger.Errorf("New stream for %s error:%s", id, e1.Error())
			s.streamMapLock.Unlock()
			return
		}
		s.streams[id] = stream
		s.streamMapLock.Unlock()
		s.send(b, id)
		return
	}
	s.streamMapLock.Unlock()
	//fmt.Printf("send to %s, size:%d\n", id, len(b))
}

func (s *server) writePackage(stream inet.Stream, body []byte, id string) error {
	l := len(body)
	var r int
	var err error
	if l < PACKAGE_MAX_SIZE {
		r, err = stream.Write(body)
		if err != nil {
			logger.Errorf("Write stream for %s error:%s", id, err.Error())
			return err
		}

		if r != l {
			logger.Errorf("stream should write %d byte ,bu write %d bytes", l, r)
			return fmt.Errorf("stream write length error")
		}
	} else {
		n := l / PACKAGE_MAX_SIZE
		left, right := 0, PACKAGE_MAX_SIZE
		for i := 0; i <= n; i++ {
			a := make([]byte, right-left)
			copy(a, body[left:right])
			r, err = stream.Write(a)
			if err != nil {
				logger.Errorf("Write stream for %s error:%s", id, err.Error())
				return err
			}
			if r != len(a) {
				logger.Errorf("stream should write %d byte ,but write %d bytes", PACKAGE_MAX_SIZE, r)
				return fmt.Errorf("stream write length error")
			}
			left += PACKAGE_MAX_SIZE
			right += PACKAGE_MAX_SIZE
			if right > l {
				right = l
			}
		}
	}
	return nil
}

func (s *server) sendSelf(b []byte, id string) {
	pkgBodyBytes := b[7:]
	s.handleMessage(pkgBodyBytes, id, time.Now())
}

//TODO 考虑读写超时
func swarmStreamHandler(stream inet.Stream) {
	go func() {
		reader := bufio.NewReader(stream)
		id := ConvertToID(stream.Conn().RemotePeer())
		for {
			e := handleStream(reader, id)
			if e != nil {
				stream.Close()
				break
			}
		}
	}()
}
func handleStream(reader *bufio.Reader, id string) error {
	headerBytes := make([]byte, 3)
	e1 := readPackage(reader, headerBytes)
	if e1 != nil {
		logger.Errorf("stream read 3 from %s error:%s!", id, e1.Error())
		return e1
	}

	//校验 header
	if !(headerBytes[0] == byte(84) && headerBytes[1] == byte(65) && headerBytes[2] == byte(83)) {
		logger.Errorf("stream validate header error from %s! ", id)
		return fmt.Errorf("validate header error")
	}

	pkgLengthBytes := make([]byte, PACKAGE_LENGTH_SIZE)
	err := readPackage(reader, pkgLengthBytes)
	if err != nil {
		logger.Errorf("stream  read4 error:%s", err.Error())
		return err
	}

	pkgLength := int(utility.ByteToUInt32(pkgLengthBytes))
	b := make([]byte, pkgLength)

	e := readPackage(reader, b)
	if e != nil {
		logger.Errorf("stream  readPackage error:%s", e.Error())
		return e
	}

	//fmt.Printf("revceive from %s, byte len:%d\n", id, len(b))
	go Server.handleMessage(b, id, time.Now())
	return nil
}

func readPackage(reader *bufio.Reader, body []byte) error {
	l := len(body)
	if l < PACKAGE_MAX_SIZE {
		err1 := readAll(reader, body, 0)
		if err1 != nil {
			logger.Errorf("stream  read error:%s", err1.Error())
			return err1
		}
	} else {
		c := l / PACKAGE_MAX_SIZE
		left, right := 0, PACKAGE_MAX_SIZE
		for i := 0; i <= c; i++ {
			a := make([]byte, right-left)
			err1 := readAll(reader, a, 0)
			if err1 != nil {
				logger.Errorf("stream read error:%s", err1.Error())
				return err1
			}
			copy(body[left:right], a)
			left += PACKAGE_MAX_SIZE
			right += PACKAGE_MAX_SIZE
			if right > l {
				right = l
			}
		}
	}
	return nil
}

func readAll(reader *bufio.Reader, body []byte, index int) error {
	if index == 0 {
		n, err1 := reader.Read(body)
		if err1 != nil {
			return err1
		}
		if n != len(body) {
			return readAll(reader, body, n)
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
			return readAll(reader, body, index+n)
		}
		return nil
	}

}

func (s *server) handleMessage(b []byte, from string, beginTime time.Time) {
	message := new(tas_middleware_pb.Message)
	error := proto.Unmarshal(b, message)
	if error != nil {
		logger.Errorf("[Network]Proto unmarshal error:%s", error.Error())
		return
	}

	if *message.Code == CAST_VERIFY_MSG {
		network.Logger.Debugf("receive CAST_VERIFY_MSG from %s ,byte:%d,read message cost time %v", from, len(b), time.Since(beginTime))
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
