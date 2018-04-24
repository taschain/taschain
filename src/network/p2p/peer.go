package p2p

import (
	"consensus/logical"
	"common"
	"taslog"
	"consensus/groupsig"
	"network/biz"
	"core"
)

const (
	KNOWN_PEER_SIZE = 100

	KNOWN_TX_SIZE = 100

	KNOWN_BLOCK_SIZE = 100

	KNOWN_GROUP_SIZE = 100
)

var Peer peer

var logger = taslog.GetLogger(taslog.P2PConfig)

type peer struct {
	KnownTx map[groupsig.ID][]common.Hash

	KnownBlock map[groupsig.ID][]common.Hash

	KnownGroup map[groupsig.ID][]string

	SelfNetInfo *Node
}

func InitPeer(self *Node) {
	knownTx := make(map[groupsig.ID][]common.Hash)
	knownBlock := make(map[groupsig.ID][]common.Hash)
	knownGroup := make(map[groupsig.ID][]string)
	Peer = peer{KnownTx: knownTx, KnownBlock: knownBlock, KnownGroup: knownGroup, SelfNetInfo: self}
}

//----------------------------------------------------组初始化-----------------------------------------------------------
//广播 组初始化消息  组内广播
// param： id slice
//         signData
func (p *peer) SendGroupInitMessage(grm logical.ConsensusGroupRawMessage) {
	body, e := MarshalConsensusGroupRawMessage(&grm)
	if e != nil {
		logger.Error("Discard ConsensusGroupRawMessage because of marshal error!\n")
		return
	}
	m := Message{Code: GROUP_INIT_MSG, Body: body}
	for _, member := range grm.MEMS {
		Server.SendMessage(m, member.ID.GetHexString())
	}
}

//组内广播密钥   for each定向发送 组内广播
func (p *peer) SendKeySharePiece(spm logical.ConsensusSharePieceMessage) {
	body, e := MarshalConsensusSharePieceMessage(&spm)
	if e != nil {
		logger.Error("Discard ConsensusSharePieceMessage because of marshal error!\n")
		return
	}
	id := spm.Dest.GetHexString()
	m := Message{Code: KEY_PIECE_MSG, Body: body}
	Server.SendMessage(m, id)

}

//组初始化完成 广播组信息 全网广播
func (p *peer) BroadcastGroupInfo(cgm logical.ConsensusGroupInitedMessage) {

	body, e := MarshalConsensusGroupInitedMessage(&cgm)
	if e != nil {
		logger.Error("Discard ConsensusGroupInitedMessage because of marshal error!\n")
		return
	}
	m := Message{Code: GROUP_INIT_DONE_MSG, Body: body}

	conns := Server.host.Network().Conns()
	for _, conn := range conns {
		id := conn.RemotePeer()
		if id != "" {
			Server.SendMessage(m, string(id))
		}
	}
}

//-----------------------------------------------------------------组铸币----------------------------------------------
//组内成员发现自己所在组成为铸币组 发消息通知全组 组内广播
//param: 组信息
//      SignData
func (p *peer) SendCurrentGroupCast(ccm *logical.ConsensusCurrentMessage) {
	//groupId := ccm.GroupID
	var memberIds []groupsig.ID
	//todo 从鸠兹获得
	body, e := MarshalConsensusCurrentMessagee(ccm)
	if e != nil {
		logger.Error("Discard ConsensusCurrentMessage because of marshal error!\n")
		return
	}
	m := Message{Code: CURRENT_GROUP_CAST_MSG, Body: body}
	for _, memberId := range memberIds {
		Server.SendMessage(m, memberId.GetHexString())
	}
}

//铸币节点完成铸币，将blockheader  签名后发送至组内其他节点进行验证。组内广播
func (p *peer) SendCastVerify(ccm *logical.ConsensusCastMessage) {
	//groupId := ccm.GroupID
	var memberIds []groupsig.ID
	//todo 从鸠兹获得 可做缓存

	body, e := MarshalConsensusCastMessage(ccm)
	if e != nil {
		logger.Error("Discard ConsensusCastMessage because of marshal error!\n")
		return
	}
	m := Message{Code: CAST_VERIFY_MSG, Body: body}
	for _, memberId := range memberIds {
		Server.SendMessage(m, memberId.GetHexString())
	}
}

//组内节点  验证通过后 自身签名 广播验证块 组内广播  验证不通过 保持静默
func (p *peer) SendVerifiedCast(cvm *logical.ConsensusVerifyMessage) {
	//groupId := ccm.GroupID
	var memberIds []groupsig.ID
	//todo 从鸠兹获得 可做缓存

	body, e := MarshalConsensusVerifyMessage(cvm)
	if e != nil {
		logger.Error("Discard ConsensusVerifyMessage because of marshal error!\n")
		return
	}
	m := Message{Code: VARIFIED_CAST_MSG, Body: body}
	for _, memberId := range memberIds {
		Server.SendMessage(m, memberId.GetHexString())
	}
}

//对外广播经过组签名的block 全网广播
//todo 此处参数留空 等班德构造
func BroadcastNewBlock() {}
//----------------------------------------------------------------------------------------------------------------------

//验证节点 交易集缺失，索要、特定交易 全网广播
func (p *peer) BroadcastTransactionRequest(m biz.TransactionRequestMessage) {
	if m.SourceId == "" {
		m.SourceId = p.SelfNetInfo.Id
	}

	body, e := MarshalTransactionRequestMessage(&m)
	if e != nil {
		logger.Error("Discard MarshalTransactionRequestMessage because of marshal error!\n")
		return
	}
	message := Message{Code: REQ_TRANSACTION_MSG, Body: body}

	conns := Server.host.Network().Conns()
	for _, conn := range conns {
		id := conn.RemotePeer()
		if id != "" {
			Server.SendMessage(message, string(id))
		}
	}
}

//本地查询到交易，返回请求方
func (p *peer) SendTransactions(txs []*core.Transaction, sourceId string) {
	body, e := MarshalTransactions(txs)
	if e != nil {
		logger.Error("Discard MarshalTransactions because of marshal error!\n")
		return
	}
	message := Message{Code: TRANSACTION_GOT_MSG, Body: body}
	Server.SendMessage(message, sourceId)
}

//收到交易 全网扩散
func (p *peer) BroadcastTransactions(txs []*core.Transaction) {

	body, e := MarshalTransactions(txs)
	if e != nil {
		logger.Error("Discard MarshalTransactions because of marshal error!\n")
		return
	}
	message := Message{Code: TRANSACTION_MSG, Body: body}

	conns := Server.host.Network().Conns()
	for _, conn := range conns {
		id := conn.RemotePeer()
		if id != "" {
			Server.SendMessage(message, string(id))
		}
	}
}

//-----------------------------------------------------------------组同步----------------------------------------------

//向某一节点请求Block
func (p *peer)RequestBlockByHeight(id string, localHeight uint64, currentHash common.Hash) {
	m := BlockOrGroupRequestEntity{SourceHeight: localHeight, SourceCurrentHash: currentHash}
	body, e := MarshalBlockOrGroupRequestEntity(&m)
	if e != nil {
		logger.Error("requestBlockByHeight marshal BlockOrGroupRequestEntity error:%s\n", e.Error())
		return
	}
	message := Message{Code: REQ_BLOCK_MSG, Body: body}
	Server.SendMessage(message, id)
}