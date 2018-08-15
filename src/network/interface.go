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

	TransactionMsg uint32 = 0x0a

	NewBlockMsg uint32 = 0x0b

	NewBlockHeaderMsg uint32 = 0x0c

	BlockBodyReqMsg uint32 = 0x0d

	BlockBodyMsg uint32 = 0x0e

	//-----------块同步---------------------------------
	ReqBlockChainTotalQnMsg uint32 = 0x0f

	BlockChainTotalQnMsg uint32 = 0x10

	ReqBlockInfo uint32 = 0x11

	BlockInfo uint32 = 0x12

	//-----------组同步---------------------------------
	ReqGroupChainHeightMsg uint32 = 0x13

	GroupChainHeightMsg uint32 = 0x14

	ReqGroupMsg uint32 = 0x15

	GroupMsg uint32 = 0x16
	//-----------块链调整---------------------------------
	BlockHashesReq uint32 = 0x17

	BlockHashes uint32 = 0x18
	//---------------------组创建确认-----------------------
	CreateGroupaRaw uint32 = 0x19

	CreateGroupSign uint32 = 0x1a

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
	SendWithGroupRely(id string, groupId string,msg Message)error

	//Broadcast the message among the group which self belongs to
	Multicast(groupId string, msg Message) error

	//Broadcast the message to the group which self do not belong to
	SpreadOverGroup(groupId string, groupMembers []string,msg Message,digest MsgDigest) error

	//Send message to neighbor nodes
	TransmitToNeighbor(msg Message) error

	//Broadcast the message to all nodes and they will also broadcast the message to their neighbor util relayCount
	//if relayCount = -1,every node will relay this message to all connected node once
	Broadcast(msg Message,relayCount int32) error

	//Return all connections self has
	ConnInfo() []Conn

	BuildGroupNet(groupId string, members []string)

	DissolveGroupNet(groupId string)

}
