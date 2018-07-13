package p2p

import (
	//"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/golang/protobuf/proto"

	"strings"
	"taslog"
	"github.com/libp2p/go-libp2p-protocol"
	//"common"
	"middleware/pb"

	"strconv"
	"errors"
	"net"
	"time"
	"common"
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
}

func InitServer(seeds []*Node, self *Node) {
	//logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))
	Server = server{SelfNetInfo: self}

	netConfig := Config{PrivateKey: &self.PrivateKey, ID: self.ID, ListenAddr: &net.UDPAddr{IP: self.IP, Port: self.Port}, Bootnodes: seeds, Unhandled: make(chan<- ReadPacket)}

	GetNetCore().Init(netConfig)

}

func (s *server) SendMessage(m Message, id string) {
	go func() {
		bytes, e := MarshalMessage(m)
		if e != nil {
			logger.Errorf("[Network]Marshal message error:%s", e.Error())
			return
		}
		s.send(bytes, id)
	}()

}

func (s *server) SendMessageToAll(m Message) {
	//go func() {
	bytes, e := MarshalMessage(m)
	if e != nil {
		logger.Errorf("[Network]Marshal message error:%s", e.Error())
		return
	}

	//s.send(b, id)

	GetNetCore().SendDataToAll(bytes)
	//s.sendSelf(bytes, s.SelfNetInfo.ID.B58String() )
	//}()

}

//AddGroup 添加组
func (s *server) AddGroup(groupID string, members []string) *Group {
	nodes := []NodeID{}
	for _,id:= range (members) {
		nodes = append(nodes,MustB58ID(id))
	}

	return GetNetCore().GM.AddGroup(groupID,nodes)
}

//RemoveGroup 移除组
func (s *server) RemoveGroup(ID string) {
	GetNetCore().GM.RemoveGroup(ID)
}

func (s *server) send(b []byte, id string) {

	if id == s.SelfNetInfo.ID.Str() {
		s.sendSelf(b, id)
		return
	}
	GetNetCore().SendData(common.StringToAddress(id), nil, b)
}

func (s *server) sendSelf(b []byte, id string) {

	s.handleMessage(b, id, time.Now())
}

func (s *server) handleMessage(b []byte, from string, beginTime time.Time) {
	message := new(tas_middleware_pb.Message)
	error := proto.Unmarshal(b, message)
	if error != nil {
		logger.Errorf("[Network]Proto unmarshal error:%s", error.Error())
		return
	}

	//fmt.Printf("message.Code:%v body:%v from:%v \n ", *message.Code,message.Body,from)

	if *message.Code == CAST_VERIFY_MSG {
		logger.Debugf("receive CAST_VERIFY_MSG from %s ,byte:%d,read message cost time %v", from, len(b), time.Since(beginTime))
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
	result := []ConnInfo{}
	peers := GetNetCore().PM.peers;
	for _, p := range peers {
		if p.seesionID > 0 && p.IP != nil && p.Port > 0{
			c := ConnInfo{Id: p.ID.B58String(), Ip: p.IP.String(), TcpPort: strconv.Itoa(p.Port)}

			//fmt.Printf("id:%v ip：%v port:%v \n ", c.Id,c.Ip,c.TcpPort)
			result = append(result, c)
		}
	}


	return result
}

func GetIPPortFromAddr(addr string) (string, int, error) {
	//addr /ip4/127.0.0.1/udp/1234"
	split := strings.Split(addr, "/")
	if len(split) != 5 {
		return "", 0, errors.New("wrong addr")
	}
	ip := split[2]
	port, _ := strconv.Atoi(split[4])
	return ip, port, nil

}
