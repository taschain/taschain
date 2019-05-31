//   Copyright (C) 2018 TASChain
//
//   This program is free software: you cas redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either versios 3 of the License, or
//   (at your option) any later versios.
//
//   This program is distributed is the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without eves the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package network

import (
	"github.com/golang/protobuf/proto"

	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/middleware/notify"
	"github.com/taschain/taschain/middleware/pb"
	"github.com/taschain/taschain/middleware/statistics"
	mrand "math/rand"
	"strconv"
	"time"

	"golang.org/x/crypto/sha3"
)

type Server struct {
	Self *Node

	netCore *NetCore

	consensusHandler MsgHandler
}

func (s *Server) Send(id string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		return err
	}
	if id == s.Self.Id.GetHexString() {
		s.sendSelf(bytes)
		return nil
	}
	go s.netCore.Send(NewNodeID(id), nil, bytes, msg.Code)

	return nil
}

func (s *Server) SendWithGroupRelay(id string, groupId string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}

	s.netCore.SendGroupMember(groupId, bytes, msg.Code, NewNodeID(id))
	return nil
}

func (s *Server) RandomSpreadInGroup(groupId string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}

	s.netCore.SendGroup(groupId, bytes, msg.Code, true, 1)

	return nil
}

func (s *Server) SpreadAmongGroup(groupId string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}

	s.netCore.SendGroup(groupId, bytes, msg.Code, true, -1)

	return nil
}

func (s *Server) SpreadToRandomGroupMember(groupId string, groupMembers []string, msg Message) error {
	if Logger == nil {
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
	Logger.Debugf("SpreadToRandomGroupMember group:%s,groupMembers:%d,index:%d", groupId, len(groupMembers), entranceIndex)

	s.netCore.GroupBroadcastWithMembers(groupId, bytes, msg.Code, nil, entranceNodes, 1)
	return nil
}

func (s *Server) SpreadToGroup(groupId string, groupMembers []string, msg Message, digest MsgDigest) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}

	Logger.Debugf("SpreadToGroup :%s,code:%d,msg size:%d", groupId, msg.Code, len(msg.Body)+4)
	s.netCore.GroupBroadcastWithMembers(groupId, bytes, msg.Code, digest, groupMembers, -1)

	return nil
}

func (s *Server) TransmitToNeighbor(msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}

	s.netCore.SendAll(bytes, msg.Code, false, nil, -1)

	return nil
}

func (s *Server) Relay(msg Message, relayCount int32) error {

	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}
	s.netCore.BroadcastRandom(bytes, msg.Code, relayCount)

	return nil
}

func (s *Server) Broadcast(msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}
	s.netCore.SendAll(bytes, msg.Code, true, nil, -1)

	return nil
}

func (s *Server) ConnInfo() []Conn {
	result := make([]Conn, 0)
	peers := s.netCore.peerManager.peers
	for _, p := range peers {
		if p.seesionId > 0 && p.Ip != nil && p.Port > 0 {
			c := Conn{Id: p.Id.GetHexString(), Ip: p.Ip.String(), Port: strconv.Itoa(p.Port)}
			result = append(result, c)
		}
	}
	return result
}

func (s *Server) BuildGroupNet(groupId string, members []string) {
	nodes := make([]NodeID, 0)
	for _, id := range members {
		nodes = append(nodes, NewNodeID(id))
	}
	s.netCore.groupManager.buildGroup(groupId, nodes)
}

func (s *Server) DissolveGroupNet(groupId string) {
	s.netCore.groupManager.removeGroup(groupId)
}

func (s *Server) AddGroup(groupId string, members []string) *Group {
	nodes := make([]NodeID, 0)
	for _, id := range members {
		nodes = append(nodes, NewNodeID(id))
	}
	return s.netCore.groupManager.buildGroup(groupId, nodes)
}

func (s *Server) RemoveGroup(ID string) {
	s.netCore.groupManager.removeGroup(ID)
}

func (s *Server) sendSelf(b []byte) {
	s.handleMessage(b, s.Self.Id.GetHexString(), s.netCore.chainId, s.netCore.protocolVersion)
}

func (s *Server) handleMessage(b []byte, from string, chaidId uint16, protocolVersion uint16) {

	message, error := unMarshalMessage(b)
	if error != nil {
		Logger.Errorf("Proto unmarshal error:%s", error.Error())
		return
	}
	message.ChainId = chaidId
	message.ProtocolVersion = protocolVersion
	Logger.Debugf("Receive message from %s,code:%d,msg size:%d,hash:%s, chainId:%v,protocolVersion:%v", from, message.Code, len(b), message.Hash(), chaidId, protocolVersion)
	statistics.AddCount("Server.handleMessage", message.Code, uint64(len(b)))
	s.netCore.flowMeter.recv(int64(message.Code), int64(len(b)))

	go s.handleMessageInner(message, from)
}

func newNotifyMessage(message *Message, from string) *notify.DefaultMessage {
	return notify.NewDefaultMessage(message.Body, from, message.ChainId, message.ProtocolVersion)
}

func (s *Server) handleMessageInner(message *Message, from string) {

	defer s.netCore.onHandleDataMessageDone(from)

	begin := time.Now()
	code := message.Code

	if code < 10000 {
		s.consensusHandler.Handle(from, *message)
	} else {
		topicId := ""
		switch code {
		case GroupChainCountMsg:
			topicId = notify.GroupHeight
		case ReqGroupMsg:
			topicId = notify.GroupReq
		case GroupMsg:
			topicId = notify.Group
		case TxSyncNotify:
			topicId = notify.TxSyncNotify
		case TxSyncReq:
			topicId = notify.TxSyncReq
		case TxSyncResponse:
			topicId = notify.TxSyncResponse
		case BlockInfoNotifyMsg:
			topicId = notify.BlockInfoNotify
		case ReqBlock:
			topicId = notify.BlockReq
		case BlockResponseMsg:
			topicId = notify.BlockResponse
		case NewBlockMsg:
			topicId = notify.NewBlock
		case ReqChainPieceBlock:
			topicId = notify.ChainPieceBlockReq
		case ChainPieceBlock:
			topicId = notify.ChainPieceBlock
		}
		if topicId != "" {
			msg := newNotifyMessage(message, from)
			notify.BUS.Publish(topicId, msg)
		}
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
	return h.Hex()
}
