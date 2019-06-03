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
	if id == s.Self.ID.GetHexString() {
		s.sendSelf(bytes)
		return nil
	}
	go s.netCore.Send(NewNodeID(id), nil, bytes, msg.Code)

	return nil
}

func (s *Server) SendWithGroupRelay(id string, groupID string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}

	s.netCore.SendGroupMember(groupID, bytes, msg.Code, NewNodeID(id))
	return nil
}

func (s *Server) RandomSpreadInGroup(groupID string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}

	s.netCore.SendGroup(groupID, bytes, msg.Code, true, 1)

	return nil
}

func (s *Server) SpreadAmongGroup(groupID string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}

	s.netCore.SendGroup(groupID, bytes, msg.Code, true, -1)

	return nil
}

func (s *Server) SpreadToRandomGroupMember(groupID string, groupMembers []string, msg Message) error {
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
	Logger.Debugf("SpreadToRandomGroupMember group:%s,groupMembers:%d,index:%d", groupID, len(groupMembers), entranceIndex)

	s.netCore.GroupBroadcastWithMembers(groupID, bytes, msg.Code, nil, entranceNodes, 1)
	return nil
}

func (s *Server) SpreadToGroup(groupID string, groupMembers []string, msg Message, digest MsgDigest) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("Marshal message error:%s", err.Error())
		return err
	}

	Logger.Debugf("SpreadToGroup :%s,code:%d,msg size:%d", groupID, msg.Code, len(msg.Body)+4)
	s.netCore.GroupBroadcastWithMembers(groupID, bytes, msg.Code, digest, groupMembers, -1)

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
		if p.sessionID > 0 && p.IP != nil && p.Port > 0 {
			c := Conn{ID: p.ID.GetHexString(), IP: p.IP.String(), Port: strconv.Itoa(p.Port)}
			result = append(result, c)
		}
	}
	return result
}

func (s *Server) BuildGroupNet(groupID string, members []string) {
	nodes := make([]NodeID, 0)
	for _, id := range members {
		nodes = append(nodes, NewNodeID(id))
	}
	s.netCore.groupManager.buildGroup(groupID, nodes)
}

func (s *Server) DissolveGroupNet(groupID string) {
	s.netCore.groupManager.removeGroup(groupID)
}

func (s *Server) AddGroup(groupID string, members []string) *Group {
	nodes := make([]NodeID, 0)
	for _, id := range members {
		nodes = append(nodes, NewNodeID(id))
	}
	return s.netCore.groupManager.buildGroup(groupID, nodes)
}

func (s *Server) RemoveGroup(ID string) {
	s.netCore.groupManager.removeGroup(ID)
}

func (s *Server) sendSelf(b []byte) {
	s.handleMessage(b, s.Self.ID.GetHexString(), s.netCore.chainID, s.netCore.protocolVersion)
}

func (s *Server) handleMessage(b []byte, from string, chainID uint16, protocolVersion uint16) {

	message, error := unMarshalMessage(b)
	if error != nil {
		Logger.Errorf("Proto unmarshal error:%s", error.Error())
		return
	}
	message.ChainID = chainID
	message.ProtocolVersion = protocolVersion
	Logger.Debugf("Receive message from %s,code:%d,msg size:%d,hash:%s, chainID:%v,protocolVersion:%v", from, message.Code, len(b), message.Hash(), chainID, protocolVersion)
	statistics.AddCount("Server.handleMessage", message.Code, uint64(len(b)))
	s.netCore.flowMeter.recv(int64(message.Code), int64(len(b)))

	go s.handleMessageInner(message, from)
}

func newNotifyMessage(message *Message, from string) *notify.DefaultMessage {
	return notify.NewDefaultMessage(message.Body, from, message.ChainID, message.ProtocolVersion)
}

func (s *Server) handleMessageInner(message *Message, from string) {

	defer s.netCore.onHandleDataMessageDone(from)

	begin := time.Now()
	code := message.Code

	if code < 10000 {
		s.consensusHandler.Handle(from, *message)
	} else {
		topicID := ""
		switch code {
		case GroupChainCountMsg:
			topicID = notify.GroupHeight
		case ReqGroupMsg:
			topicID = notify.GroupReq
		case GroupMsg:
			topicID = notify.Group
		case TxSyncNotify:
			topicID = notify.TxSyncNotify
		case TxSyncReq:
			topicID = notify.TxSyncReq
		case TxSyncResponse:
			topicID = notify.TxSyncResponse
		case BlockInfoNotifyMsg:
			topicID = notify.BlockInfoNotify
		case ReqBlock:
			topicID = notify.BlockReq
		case BlockResponseMsg:
			topicID = notify.BlockResponse
		case NewBlockMsg:
			topicID = notify.NewBlock
		case ReqChainPieceBlock:
			topicID = notify.ChainPieceBlockReq
		case ChainPieceBlock:
			topicID = notify.ChainPieceBlock
		}
		if topicID != "" {
			msg := newNotifyMessage(message, from)
			notify.BUS.Publish(topicID, msg)
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
	return h.String()
}
