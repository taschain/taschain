package p2p

import (
	inet "github.com/libp2p/go-libp2p-net"

	"utility"
	"github.com/libp2p/go-libp2p-host"
	"context"
	"github.com/libp2p/go-libp2p-peer"
	"network/biz"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/golang/protobuf/proto"
	"pb"
	"taslog"
	pstore "github.com/libp2p/go-libp2p-peerstore"

	"strings"
)

const (
	PACKAGE_MAX_SIZE = 1024 * 1024

	PACKAGE_LENGTH_SIZE = 4

	CODE_SIZE = 4

	//-----------组初始化---------------------------------
	GROUP_INIT_MSG uint32 = 0x00

	KEY_PIECE_MSG uint32 = 0x01

	MEMBER_PUBKEY_MSG uint32 = 0x02

	GROUP_INIT_DONE_MSG uint32 = 0x03

	//-----------组铸币---------------------------------
	CURRENT_GROUP_CAST_MSG uint32 = 0x04

	CAST_VERIFY_MSG uint32 = 0x05

	VARIFIED_CAST_MSG uint32 = 0x06

	REQ_TRANSACTION_MSG uint32 = 0x07

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

var Server server

type server struct {
	host host.Host

	dht *dht.IpfsDHT

	bHandler biz.BlockChainMessageHandler

	cHandler biz.ConsensusMessageHandler
}

func InitServer(host host.Host, dht *dht.IpfsDHT) {



	bHandler := biz.NewBlockChainMessageHandler(nil, nil, nil, nil,
		nil, nil, )

	cHandler := biz.NewConsensusMessageHandler(nil, nil, nil,nil,nil,nil)

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
	peerInfo, error := s.dht.FindPeer(context.Background(), peer.ID(id))
	if error != nil || peerInfo.ID.String() == "" {
		taslog.P2pLogger.Errorf("dht find peer error:%s,peer id:%s\n", error.Error(), id)
		panic("DHT find peer error!")
	}
	s.host.Network().Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, pstore.PermanentAddrTTL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, e := s.host.Network().NewStream(ctx, peer.ID(id))
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
	pkgLengthBytes := make([]byte, PACKAGE_LENGTH_SIZE)
	n, err := stream.Read(pkgLengthBytes)
	if n != 4 || err != nil {
		taslog.P2pLogger.Errorf("Stream  read %d byte error:%s,received %d bytes\n", 4, err.Error(), n)
		return
	}
	pkgLength := utility.ByteToInt(pkgLengthBytes)
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
	handleMessage(pkgBodyBytes, (string)(stream.Conn().RemotePeer()))
}

func handleMessage(pkgBodyBytes []byte, from string) {
	if len(pkgBodyBytes) < 4 {
		taslog.P2pLogger.Errorf("Message  format error!\n")
		return
	}
	message := new(tas_pb.Message)
	error := proto.Unmarshal(pkgBodyBytes, message)
	if error != nil {
		taslog.P2pLogger.Errorf("Proto unmarshal error:%s\n", error.Error())
	}

	code := message.Code
	switch *code {
	case GROUP_INIT_MSG:
		//todo
	default:
		taslog.P2pLogger.Errorf("Message not support! Code:%d\n", code)
	}
}

type ConnInfo struct {
	Id      string
	Ip      string
	TcpPort string
}

func (s *server)GetConnInfo() []ConnInfo {
	conns := s.host.Network().Conns()
	result := []ConnInfo{}
	for _, conn := range conns {
		id := conn.RemotePeer().Pretty()
		if id ==""{
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
