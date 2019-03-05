//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package network

import (
	"github.com/golang/protobuf/proto"

	"middleware/pb"

	"common"
	"golang.org/x/crypto/sha3"
	"middleware/notify"
	"middleware/statistics"
	"strconv"
	"time"
	mrand "math/rand"
)

type server struct {
	Self *Node

	netCore *NetCore

	consensusHandler MsgHandler

	chainHandler MsgHandler

}

func (n *server) Send(id string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}
	if id == n.Self.Id.GetHexString() {
		n.sendSelf(bytes)
		return nil
	}
	go n.netCore.Send(NewNodeID(id), nil, bytes, msg.Code)
	//Logger.Debugf("[Sender]Send to id:%s,code:%d,msg size:%d", id, msg.Code, len(msg.Body)+4)
	return nil
}

func (n *server) SendWithGroupRelay(id string, groupId string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}

	n.netCore.SendGroupMember(groupId, bytes, msg.Code, NewNodeID(id))
	//Logger.Debugf("[Sender]SendWithGroupRely to id:%s,code:%d,msg size:%d", id, msg.Code, len(msg.Body)+4)
	return nil
}

func (n *server) RandomSpreadInGroup(groupId string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}

	n.netCore.SendGroup(groupId, bytes, msg.Code, true, 1)
	//Logger.Debugf("Multicast to group:%s,code:%d,msg size:%d", groupId, msg.Code, len(msg.Body)+4)
	return nil
}

func (n *server) SpreadAmongGroup(groupId string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}

	n.netCore.SendGroup(groupId, bytes, msg.Code, true, -1)
	//Logger.Debugf("Multicast to group:%s,code:%d,msg size:%d", groupId, msg.Code, len(msg.Body)+4)
	return nil
}

func (n *server) SpreadToRandomGroupMember(groupId string, groupMembers []string, msg Message) error {
	if Logger == nil {
		Logger.Errorf("Logger is nil!")
		return nil
	}
	if groupMembers == nil || len(groupMembers) == 0 {
		Logger.Errorf("group members is empty!")
		return errGroupEmpty
	}

	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}

	rand := mrand.New(mrand.NewSource(time.Now().Unix()))
	entranceIndex := rand.Intn(len(groupMembers))
	entranceNodes := groupMembers[entranceIndex:]
	Logger.Debugf("SpreadToRandomGroupMember group:%s,groupMembers:%d,index:%d", groupId, len(groupMembers),entranceIndex)

	n.netCore.GroupBroadcastWithMembers(groupId, bytes, msg.Code, nil, entranceNodes, 1)
	return nil
}

func (n *server) SpreadToGroup(groupId string, groupMembers []string, msg Message, digest MsgDigest) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}

	Logger.Debugf("SpreadToGroup :%s,code:%d,msg size:%d", groupId, msg.Code, len(msg.Body)+4)
	n.netCore.GroupBroadcastWithMembers(groupId, bytes, msg.Code, digest, groupMembers, -1)

	return nil
}

func (n *server) TransmitToNeighbor(msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}

	n.netCore.SendAll(bytes, msg.Code, false, nil, -1)

	//Logger.Debugf("[Sender]TransmitToNeighbor,code:%d,msg size:%d", msg.Code, len(msg.Body)+4)
	return nil
}

func (n *server) Relay(msg Message, relayCount int32) error {

	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}
	//n.netCore.SendAll(bytes, true,nil,-1)
	n.netCore.BroadcastRandom(bytes, msg.Code, relayCount)
	//Logger.Debugf("[Sender]Relay,code:%d,msg size:%d", msg.Code, len(msg.Body)+4)
	return nil
}

func (n *server) Broadcast(msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}
	n.netCore.SendAll(bytes, msg.Code, true, nil, -1)
	//Logger.Debugf("[Sender]Broadcast,code:%d,msg size:%d", msg.Code, len(msg.Body)+4)
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
		nodes = append(nodes, NewNodeID(id))
	}
	n.netCore.groupManager.buildGroup(groupId, nodes)
}

func (n *server) DissolveGroupNet(groupId string) {
	n.netCore.groupManager.removeGroup(groupId)
}

func (n *server) AddGroup(groupId string, members []string) *Group {
	nodes := make([]NodeID, 0)
	for _, id := range members {
		nodes = append(nodes, NewNodeID(id))
	}
	return n.netCore.groupManager.buildGroup(groupId, nodes)
}

//RemoveGroup 移除组
func (n *server) RemoveGroup(ID string) {
	n.netCore.groupManager.removeGroup(ID)
}

func (n *server) sendSelf(b []byte) {
	n.handleMessage(b, n.Self.Id.GetHexString())
}

func (n *server) handleMessage(b []byte, from string) {

	message, error := unMarshalMessage(b)
	if error != nil {
		Logger.Errorf("Proto unmarshal error:%s", error.Error())
		return
	}
	Logger.Debugf("Receive message from %s,code:%d,msg size:%d,hash:%s", from, message.Code, len(b), message.Hash())
	statistics.AddCount("server.handleMessage", message.Code, uint64(len(b)))
	n.netCore.flowMeter.recv(int64(message.Code), int64(len(b)))
	// 快速释放b
	go n.handleMessageInner(message, from)
}

func (n *server) handleMessageInner(message *Message, from string) {

	defer n.netCore.onHandleDataMessageDone(from)

	begin := time.Now()
	code := message.Code
	switch code {
	case GroupInitMsg, KeyPieceMsg, SignPubkeyMsg, GroupInitDoneMsg, CurrentGroupCastMsg, CastVerifyMsg,
		VerifiedCastMsg2, CreateGroupaRaw, CreateGroupSign, CastRewardSignGot, CastRewardSignReq, AskSignPkMsg, AnswerSignPkMsg, GroupPing, GroupPong, ReqSharePiece, ResponseSharePiece:
		n.consensusHandler.Handle(from, *message)
	case ReqTransactionMsg:
		msg := notify.TransactionReqMessage{TransactionReqByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.TransactionReq, &msg)
	case GroupChainCountMsg:
		msg := notify.GroupHeightMessage{HeightByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.GroupHeight, &msg)
	case ReqGroupMsg:
		msg := notify.GroupReqMessage{GroupIdByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.GroupReq, &msg)
	case GroupMsg:
		Logger.Debugf("Rcv GroupMsg from %s", from)
		msg := notify.GroupInfoMessage{GroupInfoByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.Group, &msg)
	case TransactionGotMsg:
		msg := notify.TransactionGotMessage{TransactionGotByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.TransactionGot, &msg)
	case TransactionBroadcastMsg:
		msg := notify.TransactionBroadcastMessage{TransactionsByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.TransactionBroadcast, &msg)
	case BlockInfoNotifyMsg:
		msg := notify.BlockInfoNotifyMessage{BlockInfo: message.Body, Peer: from}
		notify.BUS.Publish(notify.BlockInfoNotify, &msg)
		//case NewBlockHeaderMsg:
		//	msg := notify.BlockHeaderNotifyMessage{HeaderByte: message.Body, Peer: from}
		//	notify.BUS.Publish(notify.NewBlockHeader, &msg)
		//case BlockBodyReqMsg:
		//	msg := notify.BlockBodyReqMessage{BlockHashByte: message.Body, Peer: from}
		//	notify.BUS.Publish(notify.BlockBodyReq, &msg)
		//case BlockBodyMsg:
		//	msg := notify.BlockBodyNotifyMessage{BodyByte: message.Body, Peer: from}
		//	notify.BUS.Publish(notify.BlockBody, &msg)
		//case ReqStateInfoMsg:
		//	msg := notify.StateInfoReqMessage{StateInfoReqByte: message.Body, Peer: from}
		//	notify.BUS.Publish(notify.StateInfoReq, &msg)
		//case StateInfoMsg:
		//	msg := notify.StateInfoMessage{StateInfoByte: message.Body, Peer: from}
		//	notify.BUS.Publish(notify.StateInfo, &msg)
	case ReqBlock:
		msg := notify.BlockReqMessage{HeightByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.BlockReq, &msg)
	case BlockResponseMsg:
		msg := notify.BlockResponseMessage{BlockResponseByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.BlockResponse, &msg)
	case NewBlockMsg:
		msg := notify.NewBlockMessage{BlockByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.NewBlock, &msg)
	case ChainPieceInfoReq:
		Logger.Debugf("Rcv ChainPieceInfoReq from %s", from)
		msg := notify.ChainPieceInfoReqMessage{HeightByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.ChainPieceInfoReq, &msg)
	case ChainPieceInfo:
		Logger.Debugf("Rcv ChainPieceInfo from %s", from)
		msg := notify.ChainPieceInfoMessage{ChainPieceInfoByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.ChainPieceInfo, &msg)
	case ReqChainPieceBlock:
		msg := notify.ChainPieceBlockReqMessage{ReqHeightByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.ChainPieceBlockReq, &msg)
	case ChainPieceBlock:
		msg := notify.ChainPieceBlockMessage{ChainPieceBlockMsgByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.ChainPieceBlock, &msg)
	}

	if time.Since(begin) > 100*time.Millisecond {
		Logger.Debugf("handle message cost time:%v,hash:%s,code:%d", time.Since(begin), message.Hash(), code)
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
