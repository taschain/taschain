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

var Server server

type server struct {
	host host.Host

	dht *dht.IpfsDHT

	bHandler biz.BlockChainMessageHandler

	cHandler biz.ConsensusMessageHandler
}

func InitServer(host host.Host, dht *dht.IpfsDHT, bHandler biz.BlockChainMessageHandler, cHandler biz.ConsensusMessageHandler) {

	host.Network().SetStreamHandler(swarmStreamHandler)

	Server = server{host: host, dht: dht, bHandler: bHandler, cHandler: cHandler}
}

func (s *server) SendMessage(m Message, id string) {
	bytes, e := MarshalMessage(m)
	if e != nil {
		logger.Errorf("Marshal message error:%s\n", e.Error())
		return
	}

	length := len(bytes)
	b2 := utility.UInt32ToByte(uint32(length))

	b := make([]byte,len(bytes)+len(b2))
	copy(b[:4], b2)
	copy(b[4:], bytes)

	s.send(b, id)
}

func (s *server) send(b []byte, id string) {
	peerInfo, error := s.dht.FindPeer(context.Background(), gpeer.ID(id))
	if error != nil || peerInfo.ID.String() == "" {
		logger.Errorf("dht find peer error:%s,peer id:%s\n", error.Error(), id)
		panic("DHT find peer error!")
	}
	s.host.Network().Peerstore().AddAddrs(peerInfo.ID, peerInfo.Addrs, pstore.PermanentAddrTTL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, e := s.host.Network().NewStream(ctx, gpeer.ID(id))
	defer stream.Close()
	if e != nil {
		logger.Errorf("New stream for %s error:%s\n", id, error.Error())
		panic("New stream error!")
	}
	l := len(b)
	if l < PACKAGE_MAX_SIZE {
		r, err := stream.Write(b)
		if r != l || err != nil {
			logger.Errorf("Write stream for %s error:%s\n", id, error.Error())
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
		logger.Errorf("Stream  read %d byte error:%s,received %d bytes\n", 4, err.Error(), n)
		return
	}
	pkgLength := int(utility.ByteToUInt32(pkgLengthBytes))
	pkgBodyBytes := make([]byte, pkgLength)
	if pkgLength < PACKAGE_MAX_SIZE {
		n1, err1 := stream.Read(pkgBodyBytes)
		if n1 != pkgLength || err1 != nil {
			logger.Errorf("Stream  read %d byte error:%s,received %d bytes\n", pkgLength, err.Error(), n)
			return
		}
	} else {
		c := pkgLength / PACKAGE_MAX_SIZE
		left, right := 0, PACKAGE_MAX_SIZE
		for i := 0; i <= c; i++ {
			a := make([]byte, PACKAGE_MAX_SIZE)
			n1, err1 := stream.Read(a)
			if n1 != PACKAGE_MAX_SIZE || err1 != nil {
				logger.Errorf("Stream  read %d byte error:%s,received %d bytes\n", PACKAGE_MAX_SIZE, err.Error(), n1)
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
		logger.Errorf("Message  format error!\n")
		return
	}
	message := new(tas_pb.Message)
	error := proto.Unmarshal(b, message)
	if error != nil {
		logger.Errorf("Proto unmarshal error:%s\n", error.Error())
	}

	code := message.Code
	switch *code {
	case GROUP_INIT_MSG:
		m, e := UnMarshalConsensusGroupRawMessage(message.Body)
		if e != nil {
			logger.Error("Discard ConsensusGroupRawMessage because of unmarshal error!\n")
			return
		}
		s.cHandler.OnMessageGroupInitFn(*m)
	case KEY_PIECE_MSG:
		m, e := UnMarshalConsensusSharePieceMessage(message.Body)
		if e != nil {
			logger.Error("Discard ConsensusSharePieceMessage because of unmarshal error!\n")
			return
		}
		s.cHandler.OnMessageSharePieceFn(*m)
	case GROUP_INIT_DONE_MSG:
		m, e := UnMarshalConsensusGroupInitedMessage(message.Body)
		if e != nil {
			logger.Error("Discard ConsensusGroupInitedMessage because of unmarshal error!\n")
			return
		}
		s.cHandler.OnMessageGroupInitedFn(*m)

	case CURRENT_GROUP_CAST_MSG:
		m, e := UnMarshalConsensusCurrentMessage(message.Body)
		if e != nil {
			logger.Error("Discard ConsensusCurrentMessage because of unmarshal error!\n")
			return
		}
		s.cHandler.OnMessageCurrentGroupCastFn(*m)
	case CAST_VERIFY_MSG:
		m, e := UnMarshalConsensusCastMessage(message.Body)
		if e != nil {
			logger.Error("Discard ConsensusCastMessage because of unmarshal error!\n")
			return
		}
		s.cHandler.OnMessageCastFn(*m)
	case VARIFIED_CAST_MSG:
		m, e := UnMarshalConsensusVerifyMessage(message.Body)
		if e != nil {
			logger.Error("Discard ConsensusVerifyMessage because of unmarshal error!\n")
			return
		}
		s.cHandler.OnMessageVerifiedCastFn(*m)

	case REQ_TRANSACTION_MSG:
		m, e := UnMarshalTransactionRequestMessage(message.Body)
		if e != nil {
			logger.Error("Discard TransactionRequestMessage because of unmarshal error!\n")
			return
		}
		s.bHandler.OnTransactionRequest(m)
	case TRANSACTION_GOT_MSG:
		m, e := UnMarshalTransactions(message.Body)
		if e != nil {
			logger.Error("Discard TRANSACTION_MSG because of unmarshal error!\n")
			return
		}
		s.bHandler.OnMessageTransaction(m)

	case TRANSACTION_MSG:
		m, e := UnMarshalTransactions(message.Body)
		if e != nil {
			logger.Error("Discard TRANSACTION_MSG because of unmarshal error!\n")
			return
		}
		s.bHandler.OnNewTransaction(m)

	case REQ_BLOCK_CHAIN_HEIGHT_MSG:
		BlockSyncer.HeightRequestCh <- from

	case BLOCK_CHAIN_HEIGHT_MSG:
		height := utility.ByteToUInt64(message.Body)
		s := blockHeight{height: height, sourceId: from}
		BlockSyncer.HeightCh <- s
	case REQ_BLOCK_MSG:
		m, e := UnMarshalBlockOrGroupRequestEntity(message.Body)
		if e != nil {
			logger.Error("Discard REQ_BLOCK_MSG because of unmarshal error!\n")
			return
		}

		enetity := BlockOrGroupRequestEntity{SourceHeight: m.SourceHeight, SourceCurrentHash: m.SourceCurrentHash}
		s := blockRequest{bre: enetity, sourceId: from}
		BlockSyncer.BlockRequestCh <- s
	case BLOCK_MSG:
		m, e := UnMarshalBlockEntity(message.Body)
		if e != nil {
			logger.Error("Discard BLOCK_MSG because of unmarshal error!\n")
			return
		}
		s := blockArrived{blockEntity: *m, sourceId: from}
		BlockSyncer.BlockArrivedCh <- s

	default:
		logger.Errorf("Message not support! Code:%d\n", code)
	}
}

type ConnInfo struct {
	Id      string `json:"id"`
	Ip      string `json:"ip"`
	TcpPort string `json:"tcp_port"`
}

func (s *server) GetConnInfo() []ConnInfo {
	conns := s.host.Network().Conns()
	result := []ConnInfo{}
	for _, conn := range conns {
		id := string(conn.RemotePeer())
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
