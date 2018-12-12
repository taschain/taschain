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

const (
	//-----------组初始化---------------------------------

	GroupInitMsg uint32 = 0x01

	KeyPieceMsg uint32 = 0x02

	SignPubkeyMsg uint32 = 0x03

	GroupInitDoneMsg uint32 = 0x04

	//-----------组铸币---------------------------------
	CurrentGroupCastMsg uint32 = 0x05

	CastVerifyMsg uint32 = 0x06

	VerifiedCastMsg uint32 = 0x07

	ReqTransactionMsg uint32 = 0x08

	TransactionGotMsg uint32 = 0x09

	MinerTransactionMsg uint32 = 0x0a

	NewBlockMsg uint32 = 0x0b

	NewBlockHeaderMsg uint32 = 0x0c

	BlockBodyReqMsg uint32 = 0x0d

	BlockBodyMsg uint32 = 0x0e

	//-----------块同步---------------------------------
	BlockInfoNotifyMsg uint32 = 0x10

	ReqBlock uint32 = 0x11

	BlockMsg uint32 = 0x12

	//-----------组同步---------------------------------
	GroupChainCountMsg uint32 = 0x14

	ReqGroupMsg uint32 = 0x15

	GroupMsg uint32 = 0x16
	//-----------块链调整---------------------------------
	ChainPieceInfoReq uint32 = 0x17

	ChainPieceInfo uint32 = 0x18

	ReqChainPieceBlock uint32 = 0x22

	ChainPieceBlock uint32 = 0x23
	//---------------------组创建确认-----------------------
	CreateGroupaRaw uint32 = 0x19

	CreateGroupSign uint32 = 0x1a
	//---------------------轻节点状态同步-----------------------
	ReqStateInfoMsg uint32 = 0x1b

	StateInfoMsg uint32 = 0x1c

	//==================铸块分红=========
	CastRewardSignReq uint32 = 0x1d
	CastRewardSignGot uint32 = 0x1e

	//==================Trace=========
	RequestTraceMsg  uint32 = 0x20
	ResponseTraceMsg uint32 = 0x21

	FULL_NODE_VIRTUAL_GROUP_ID = "full_node_virtual_group_id"
)

type Message struct {
	Code uint32

	Body []byte
}

type Conn struct {
	Id   string
	Ip   string
	Port string
}

type MsgDigest []byte

type MsgHandler interface {
	Handle(sourceId string, msg Message) error
}

type Network interface {
	//Send message to the node which id represents.If self doesn't connect to the node,
	// resolve the kad net to find the node and then send the message
	Send(id string, msg Message) error

	//Send message to the node which id represents. If self doesn't connect to the node,
	// send message to the guys which belongs to the same group with the node and they will rely the message to the node
	SendWithGroupRelay(id string, groupId string, msg Message) error

	//Random broadcast the message to parts nodes in the group which self belongs to
	RandomSpreadInGroup(groupId string, msg Message) error

	//Broadcast the message among the group which self belongs to
	SpreadAmongGroup(groupId string, msg Message) error

	//send message to random memebers which in special group
	SpreadToRandomGroupMember(groupId string, groupMembers []string, msg Message) error

	//Broadcast the message to the group which self do not belong to
	SpreadToGroup(groupId string, groupMembers []string, msg Message, digest MsgDigest) error

	//Send message to neighbor nodes
	TransmitToNeighbor(msg Message) error

	//Send the message to part nodes it connects to and they will also broadcast the message to part of their neighbor util relayCount
	Relay(msg Message, relayCount int32) error

	//Send the message to all nodes it connects to and the node which receive the message also broadcast the message to their neighbor once
	Broadcast(msg Message) error

	//Return all connections self has
	ConnInfo() []Conn

	BuildGroupNet(groupId string, members []string)

	DissolveGroupNet(groupId string)
}
