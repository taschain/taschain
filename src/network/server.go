package network

import (
	"github.com/golang/protobuf/proto"

	"middleware/pb"

	"strconv"
	"common"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

const (
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
	//---------------------组创建确认-----------------------
	CREATE_GROUP_RAW uint32 = 0x16

	CREATE_GROUP_SIGN uint32 = 0x17
)

type network struct {
	Self *Node

	netCore *NetCore

	consensusHandler MsgHandler

	chainHandler MsgHandler
}

func (n *network) Send(targetId string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("[Network]Marshal message error:%s", err.Error())
		return err
	}
	if targetId == n.Self.Id.GetHexString() {
		go n.sendSelf(bytes)
		return nil
	}
	go n.netCore.Send(common.HexStringToAddress(targetId), nil, bytes)
	return nil
}

func (n *network) Multicast(groupId string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("[Network]Marshal message error:%s", err.Error())
		return err
	}

	n.netCore.SendGroup(groupId,bytes)
	return nil
}

func (n *network) Broadcast(msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("[Network]Marshal message error:%s", err.Error())
		return err
	}
	n.netCore.SendAll(bytes)
	return nil
}

func (n *network) ConnInfo() []Conn {
	result := make([]Conn, 0)
	peers := n.netCore.peerManager.peers
	for _, p := range peers {
		if p.seesionId > 0 && p.Ip != nil && p.Port > 0 {
			c := Conn{Id: p.Id.GetHexString(), Ip: p.Ip.String(), Port: strconv.Itoa(p.Port)}
			result = append(result, c)
		}
	}
	return result
}

func (n *network)BuildGroupNet(groupId string, members []string){
	nodes := make([]NodeID, 0)
	for _, id := range members {
		nodes = append(nodes, common.HexStringToAddress(id))
	}
	n.netCore.groupManager.AddGroup(groupId, nodes)
}


func (n *network)DissolveGroupNet(groupId string){
	n.netCore.groupManager.RemoveGroup(groupId)
}


func (n *network) AddGroup(groupId string, members []string) *Group {
	nodes := make([]NodeID, 0)
	for _, id := range members {
		nodes = append(nodes, common.HexStringToAddress(id))
	}
	return n.netCore.groupManager.AddGroup(groupId, nodes)
}

//RemoveGroup 移除组
func (n *network) RemoveGroup(ID string) {
	n.netCore.groupManager.RemoveGroup(ID)
}

func (n *network) sendSelf(b []byte) {
	n.handleMessage(b, n.Self.Id.GetHexString())
}

func (n *network) handleMessage(b []byte, from string) {
	begin:= time.Now()
	message, error := unMarshalMessage(b)
	if error != nil {
		Logger.Errorf("[Network]Proto unmarshal error:%s", error.Error())
		return
	}

	code := message.Code
	defer Logger.Debugf("code:%d,cost time:%v",code,time.Since(begin))
	switch code {
	case GROUP_MEMBER_MSG, GROUP_INIT_MSG, KEY_PIECE_MSG, SIGN_PUBKEY_MSG, GROUP_INIT_DONE_MSG, CURRENT_GROUP_CAST_MSG, CAST_VERIFY_MSG,
		VARIFIED_CAST_MSG, CREATE_GROUP_RAW, CREATE_GROUP_SIGN:
		n.consensusHandler.Handle(from, *message)
	case REQ_TRANSACTION_MSG, REQ_BLOCK_CHAIN_TOTAL_QN_MSG, BLOCK_CHAIN_TOTAL_QN_MSG, REQ_BLOCK_INFO, BLOCK_INFO,
		REQ_GROUP_CHAIN_HEIGHT_MSG, GROUP_CHAIN_HEIGHT_MSG, REQ_GROUP_MSG, GROUP_MSG, BLOCK_HASHES_REQ, BLOCK_HASHES:
		n.chainHandler.Handle(from, *message)
	case NEW_BLOCK_MSG:
		n.consensusHandler.Handle(from, *message)
	case TRANSACTION_MSG, TRANSACTION_GOT_MSG:
		error := n.chainHandler.Handle(from, *message)
		if error != nil {
			return
		}
		n.consensusHandler.Handle(from, *message)
	}
}

func marshalMessage(m Message) ([]byte, error) {
	message := tas_middleware_pb.Message{Code: &m.Code, Body: m.Body}
	return proto.Marshal(&message)
}

func unMarshalMessage(b []byte) (*Message, error) {
	message := new(tas_middleware_pb.Message)
	e := proto.Unmarshal(b, message)
	if e != nil {
		return nil, e
	}
	m := Message{Code: *message.Code, Body: message.Body}
	return &m, nil
}

func (m Message) Hash() string {
	hash := sha256.New()
	bytes, err := marshalMessage(m)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(hash.Sum(bytes))
}
