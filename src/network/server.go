package network

import (
	"github.com/golang/protobuf/proto"

	"middleware/pb"

	"strconv"
	"common"
	"golang.org/x/crypto/sha3"
	"middleware/statistics"
	"middleware/notify"
)

type server struct {
	Self *Node

	netCore *NetCore

	consensusHandler MsgHandler

	chainHandler MsgHandler

	//workinglist *concurrent.Queue
}

type workingdata struct {
	message *Message
	from    string
}

func (n *server) Send(id string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("[Network]Marshal message error:%s", err.Error())
		return err
	}
	if id == n.Self.Id.GetHexString() {
		n.sendSelf(bytes)
		return nil
	}
	go n.netCore.Send(common.HexStringToAddress(id), nil, bytes)
	//Logger.Debugf("[Sender]Send to id:%s,code:%d,msg size:%d", id, msg.Code, len(msg.Body)+4)
	return nil
}

func (n *server) SendWithGroupRelay(id string, groupId string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("[Network]Marshal message error:%s", err.Error())
		return err
	}

	n.netCore.SendGroupMember(groupId, bytes, common.HexStringToAddress(id))
	//Logger.Debugf("[Sender]SendWithGroupRely to id:%s,code:%d,msg size:%d", id, msg.Code, len(msg.Body)+4)
	return nil
}

func (n *server) Multicast(groupId string, msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("[Network]Marshal message error:%s", err.Error())
		return err
	}

	n.netCore.SendGroup(groupId, bytes, true)
	//Logger.Debugf("[Sender]Multicast to group:%s,code:%d,msg size:%d", groupId, msg.Code, len(msg.Body)+4)
	return nil
}

func (n *server) SpreadOverGroup(groupId string, groupMembers []string, msg Message, digest MsgDigest) error {

	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("[Network]Marshal message error:%s", err.Error())
		return err
	}

	n.netCore.GroupBroadcastWithMembers(groupId, bytes, digest, groupMembers)
	//Logger.Debugf("[Sender]SpreadOverGroup to group:%s,code:%d,msg size:%d", groupId, msg.Code, len(msg.Body)+4)

	return nil
}

func (n *server) TransmitToNeighbor(msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("[Network]Marshal message error:%s", err.Error())
		return err
	}

	n.netCore.SendAll(bytes, false, nil, -1)

	//Logger.Debugf("[Sender]TransmitToNeighbor,code:%d,msg size:%d", msg.Code, len(msg.Body)+4)
	return nil
}

func (n *server) Relay(msg Message, relayCount int32) error {

	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("[Network]Marshal message error:%s", err.Error())
		return err
	}
	//n.netCore.SendAll(bytes, true,nil,-1)
	n.netCore.BroadcastRandom(bytes, relayCount)
	//Logger.Debugf("[Sender]Relay,code:%d,msg size:%d", msg.Code, len(msg.Body)+4)
	return nil
}

func (n *server) Broadcast(msg Message) error {
	bytes, err := marshalMessage(msg)
	if err != nil {
		Logger.Errorf("[Network]Marshal message error:%s", err.Error())
		return err
	}
	n.netCore.SendAll(bytes, true, nil, -1)
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
	message, error := unMarshalMessage(b)
	if error != nil {
		Logger.Errorf("[Network]Proto unmarshal error:%s", error.Error())
		return
	}
	Logger.Debugf("Receive message from %s,code:%d,msg size:%d,hash:%s", from, message.Code, len(b), message.Hash())
	statistics.AddCount("server.handleMessage", message.Code, uint64(len(b)))

	// 快速释放b
	go n.handleMessageInner(message, from)
}

func (n *server) handleMessageInner(message *Message, from string) {
	//begin := time.Now()
	code := message.Code

	//defer Logger.Debugf("handle message cost time:%v,hash:%s", time.Since(begin), message.Hash())
	switch code {
	case GroupInitMsg, KeyPieceMsg, SignPubkeyMsg, GroupInitDoneMsg, CurrentGroupCastMsg, CastVerifyMsg,
		VerifiedCastMsg, CreateGroupaRaw, CreateGroupSign:
		n.consensusHandler.Handle(from, *message)
	case ReqTransactionMsg, ReqBlockChainTotalQnMsg, BlockChainTotalQnMsg, ReqBlockInfo, BlockInfo,
		ReqGroupChainCountMsg, GroupChainCountMsg, ReqGroupMsg, GroupMsg, BlockHashesReq, BlockHashes:
		n.chainHandler.Handle(from, *message)
	case NewBlockMsg:
		n.consensusHandler.Handle(from, *message)
	case TransactionMsg, TransactionGotMsg:
		error := n.chainHandler.Handle(from, *message)
		if error != nil {
			return
		}
		n.consensusHandler.Handle(from, *message)
	case NewBlockHeaderMsg:
		//Logger.Debugf("Receive NewBlockHeaderMsg from %s", from)
		msg := notify.BlockHeaderNotifyMessage{HeaderByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.NewBlockHeader, &msg)
	case BlockBodyReqMsg:
		//Logger.Debugf("Receive BlockBodyReqMsg from %s", from)
		msg := notify.BlockBodyReqMessage{BlockHashByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.BlockBodyReq, &msg)
	case BlockBodyMsg:
		//Logger.Debugf("Receive BlockBodyMsg from %s", from)
		msg := notify.BlockBodyNotifyMessage{BodyByte: message.Body, Peer: from}
		notify.BUS.Publish(notify.BlockBody, &msg)
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
