package p2p

import (
	"consensus/logical"
	"common"
	"taslog"
	"consensus/groupsig"
	"core"
)

const (
	KNOWN_PEER_SIZE = 100

	KNOWN_TX_SIZE = 100

	KNOWN_BLOCK_SIZE = 100

	KNOWN_GROUP_SIZE = 100
)

var Peer peer

type peer struct {
	KeyMap map[groupsig.ID]string

	KnownTx map[groupsig.ID][]common.Hash

	KnownBlock map[groupsig.ID][]common.Hash

	KnownGroup map[groupsig.ID][]string

	SelfNetInfo *Node
}

func InitPeer(self *Node) {
	keyMap := map[groupsig.ID]string{}
	knownTx := make(map[groupsig.ID][]common.Hash)
	knownBlock := make(map[groupsig.ID][]common.Hash)
	knownGroup := make(map[groupsig.ID][]string)
	Peer = peer{KeyMap: keyMap, KnownTx: knownTx, KnownBlock: knownBlock, KnownGroup: knownGroup, SelfNetInfo: self}
}

//----------------------------------------------------组初始化-----------------------------------------------------------
//广播 组初始化消息  组内广播
// param： id slice
//         signData
func (p *peer) SendGroupInitMessage(grm logical.ConsensusGroupRawMessage) {
	body, e := MarshalConsensusGroupRawMessage(&grm)
	if e != nil {
		taslog.P2pLogger.Error("Discard ConsensusGroupRawMessage because of marshal error!\n")
		return
	}
	m := Message{Code: GROUP_INIT_MSG, Body: body}
	for _, userId := range grm.UserIds {
		Server.SendMessage(m, userId)
	}
}

//组内广播密钥   for each定向发送 组内广播
func (p *peer) SendKeySharePiece(spm logical.ConsensusSharePieceMessage) {
	body, e := MarshalConsensusSharePieceMessage(&spm)
	if e != nil {
		taslog.P2pLogger.Error("Discard ConsensusSharePieceMessage because of marshal error!\n")
		return
	}
	gId := spm.Dest
	userId := p.KeyMap[gId]
	if userId == "" {
		taslog.P2pLogger.Errorf("Can not get userId from miner id:%s.Discard!\n", gId)
		return
	}
	m := Message{Code: KEY_PIECE_MSG, Body: body}
	Server.SendMessage(m, userId)

}

//组初始化完成 广播组信息 全网广播 //todo  这里是否上链 还是班德去上链
func (p *peer) BroadcastGroupInfo(cgm logical.ConsensusGroupInitedMessage) {

	body, e := MarshalConsensusGroupInitedMessage(&cgm)
	if e != nil {
		taslog.P2pLogger.Error("Discard ConsensusGroupInitedMessage because of marshal error!\n")
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
	//todo 从鸠兹获得 可做缓存
	body, e := MarshalConsensusCurrentMessagee(ccm)
	if e != nil {
		taslog.P2pLogger.Error("Discard ConsensusCurrentMessage because of marshal error!\n")
		return
	}
	m := Message{Code: CURRENT_GROUP_CAST_MSG, Body: body}
	for _, memberId := range memberIds {
		userId := p.KeyMap[memberId]
		if userId == "" {
			taslog.P2pLogger.Errorf("Can not get userId from miner id:%s.Discard!\n", memberId)
			continue
		}
		Server.SendMessage(m, userId)
	}
}

//铸币节点完成铸币，将blockheader  签名后发送至组内其他节点进行验证。组内广播
func (p *peer) SendCastVerify(ccm *logical.ConsensusCastMessage) {
	//groupId := ccm.GroupID
	var memberIds []groupsig.ID
	//todo 从鸠兹获得 可做缓存

	body, e := MarshalConsensusCastMessage(ccm)
	if e != nil {
		taslog.P2pLogger.Error("Discard ConsensusCastMessage because of marshal error!\n")
		return
	}
	m := Message{Code: CAST_VERIFY_MSG, Body: body}
	for _, memberId := range memberIds {
		userId := p.KeyMap[memberId]
		if userId == "" {
			taslog.P2pLogger.Errorf("Can not get userId from miner id:%s.Discard!\n", memberId)
			continue
		}
		Server.SendMessage(m, userId)
	}
}

//组内节点  验证通过后 自身签名 广播验证块 组内广播  验证不通过 保持静默
func (p *peer) SendVerifiedCast(cvm *logical.ConsensusVerifyMessage) {
	//groupId := ccm.GroupID
	var memberIds []groupsig.ID
	//todo 从鸠兹获得 可做缓存

	body, e := MarshalConsensusVerifyMessage(cvm)
	if e != nil {
		taslog.P2pLogger.Error("Discard ConsensusVerifyMessage because of marshal error!\n")
		return
	}
	m := Message{Code: VARIFIED_CAST_MSG, Body: body}
	for _, memberId := range memberIds {
		userId := p.KeyMap[memberId]
		if userId == "" {
			taslog.P2pLogger.Errorf("Can not get userId from miner id:%s.Discard!\n", memberId)
			continue
		}
		Server.SendMessage(m, userId)
	}
}

//对外广播经过组签名的block 全网广播
//param: block
//       member signature
//       signData
func BroadcastNewBlock(b core.Block, sd logical.SignData) {}




//验证节点 交易集缺失，索要、特定交易 全网广播
//param:hash slice of transaction slice
//      signData
func (p *peer) RequestTransactionByHash(hs []common.Hash, sig []byte) {}

//收到交易 全网扩散
func (p *peer) BroadcastTransaction(hs []common.Hash, sig []byte) {}

//自己调
func (p *peer) BroadcastTransactionRequest(hs []common.Hash, sig []byte) {}
