package network

import (
	"github.com/golang/protobuf/proto"

	"middleware/pb"

	"strconv"
	"common"
	"time"
	"golang.org/x/crypto/sha3"
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

type server struct {
	Self *Node

	netCore *NetCore

	consensusHandler MsgHandler

	chainHandler MsgHandler
}

func (n *server) Send(targetId string, msg Message) error {
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

func (n *server) SendWithGroupRely(id string, groupId string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("[Network]Marshal message error:%s", err.Error())
		return err
	}

	n.netCore.SendGroupMember(groupId, bytes, common.HexStringToAddress(id))
	return nil
}

func (n *server) Multicast(groupId string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("[Network]Marshal message error:%s", err.Error())
		return err
	}

	n.netCore.SendGroup(groupId, bytes, true)
	return nil
}

func (n *server) TransmitToNeighbor(msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("[Network]Marshal message error:%s", err.Error())
		return err
	}
	n.netCore.SendAll(bytes, false)
	return nil
}

func (n *server) Broadcast(msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("[Network]Marshal message error:%s", err.Error())
		return err
	}
	n.netCore.SendAll(bytes, true)
	return nil
}

func (n *server) ConnInfo() []Conn {
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

func (n *server) BuildGroupNet(groupId string, members []string) {
	nodes := make([]NodeID, 0)
	for _, id := range members {
		nodes = append(nodes, common.HexStringToAddress(id))
	}
	n.netCore.groupManager.addGroup(groupId, nodes)
}

func (n *server) DissolveGroupNet(groupId string) {
	n.netCore.groupManager.removeGroup(groupId)
}

func (n *server) AddGroup(groupId string, members []string) *Group {
	nodes := make([]NodeID, 0)
	for _, id := range members {
		nodes = append(nodes, common.HexStringToAddress(id))
	}
	return n.netCore.groupManager.addGroup(groupId, nodes)
}

//RemoveGroup 移除组
func (n *server) RemoveGroup(ID string) {
	n.netCore.groupManager.removeGroup(ID)
}

func (n *server) sendSelf(b []byte) {
	n.handleMessage(b, n.Self.Id.GetHexString())
}

func (n *server) handleMessage(b []byte, from string) {
	begin := time.Now()
	message, error := unMarshalMessage(b)
	if error != nil {
		Logger.Errorf("[Network]Proto unmarshal error:%s", error.Error())
		return
	}

	code := message.Code
	if code == KEY_PIECE_MSG {
		Logger.Debugf("Receive KEY_PIECE_MSG from %s,hash:%s", from, message.Hash())
	}

	if code == SIGN_PUBKEY_MSG {
		Logger.Debugf("Receive SIGN_PUBKEY_MSG from %s,hash:%s", from, message.Hash())
	}

	if code == CAST_VERIFY_MSG {
		Logger.Debugf("Receive CAST_VERIFY_MSG from%s,hash:%s", from, message.Hash())
	}

	if code == GROUP_INIT_DONE_MSG {
		Logger.Debugf("Receive GROUP_INIT_DONE_MSG from %s,hash:%s", from, message.Hash())
	}

	if code == NEW_BLOCK_MSG {
		Logger.Debugf("Receive NEW_BLOCK_MSG from %s,hash:%s", from, message.Hash())
	}

	if code == CAST_VERIFY_MSG {
		Logger.Debugf("Receive GROUP_INIT_MSG from %s,hash:%s", from, message.Hash())
	}

	if code == GROUP_INIT_DONE_MSG {
		Logger.Debugf("Receive GROUP_INIT_MSG from %s,hash:%s", from, message.Hash())
	}

	defer Logger.Debugf("handle message cost time:%v,hash:%s", time.Since(begin), message.Hash())
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
	bytes, err := marshalMessage(m)
	if err != nil {
		return ""
	}

	var h common.Hash
	sha3Hash := sha3.Sum256(bytes)
	if len(sha3Hash) == common.HashLength {
		copy(h[:], sha3Hash[:])
	} else {
		panic("Data2Hash failed, size error.")
	}
	return h.String()
}
