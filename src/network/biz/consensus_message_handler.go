package biz

import (
	"consensus/logical"
)

//groupsig.ID  全网唯一 可作为节点在网络内的标识

//-----------------------------------------------------回调函数定义------------------------------------------------------

//----------------------------------------------------组初始化-----------------------------------------------------------
//接收组初始化消息 生成针对其他成员的密钥片段
//processer.go OnMessageGroupInit
type onMessageGroupInitFn func(grm logical.ConsensusGroupRawMessage)

//组内节点接收密钥片段 保存，收到所有密钥片段后 生成组公钥
// processer.go OnMessageSharePiece
type onMessageSharePieceFn func(spm logical.ConsensusSharePieceMessage)

//接收组  不验证了 进行上链 广播
// todo  存在单点问题
// processer.go OnMessageGroupInited
type onMessageGroupInitedFn func(gim logical.ConsensusGroupInitedMessage)

//-----------------------------------------------------------------组铸币----------------------------------------------
//接收开始铸币的消息，判定自己是否为铸币节点，如果是  启动铸币流程  如果不是 保持静默
//processer.go OnMessageCurrent
type onMessageCurrentGroupCastFn func(ccm logical.ConsensusCurrentMessage)

//组内节点验证接收到的刚铸出来的币
//processer.go OnMessageCast
type onMessageCastFn func(ccm logical.ConsensusCastMessage)

//组内验证过的 块消息监听到达
//processer.go OnMessageVerify
type onMessageVerifiedCastFn func(cvm logical.ConsensusVerifyMessage)



//---------------------------------------------------------------------------------------------------------------------
type ConsensusMessageHandler struct {
	OnMessageGroupInitFn   onMessageGroupInitFn
	OnMessageSharePieceFn  onMessageSharePieceFn
	OnMessageGroupInitedFn onMessageGroupInitedFn

	OnMessageCurrentGroupCastFn onMessageCurrentGroupCastFn
	OnMessageCastFn             onMessageCastFn
	OnMessageVerifiedCastFn     onMessageVerifiedCastFn
}

func NewConsensusMessageHandler(onMessageGroupInitFn onMessageGroupInitFn, onMessageSharePieceFn onMessageSharePieceFn,
	onMessageGroupInitedFn onMessageGroupInitedFn, onMessageCurrentGroupCastFn onMessageCurrentGroupCastFn, onMessageCastFn onMessageCastFn, onMessageVerifiedCastFn onMessageVerifiedCastFn) ConsensusMessageHandler {

	return ConsensusMessageHandler{
		OnMessageGroupInitFn:        onMessageGroupInitFn,
		OnMessageSharePieceFn:       onMessageSharePieceFn,
		OnMessageGroupInitedFn:      onMessageGroupInitedFn,
		OnMessageCurrentGroupCastFn: onMessageCurrentGroupCastFn,
		OnMessageCastFn:             onMessageCastFn,
		OnMessageVerifiedCastFn:     onMessageVerifiedCastFn,
	}
}

/**
共识铸块过程中请求父节点哈希不需要走网络请求，默认本地都有，由日常块的同步机制保证实现
////如果本地没有父区块哈希值，请求最新区块哈希值 全网广播
//func requestPreHash(){}
//
////接收父区块哈希值请求 如果本地有 则返回
//func onPreHashRequest()common.Hash{
//	var a common.Hash
//	return a
//}
//
////接收父区块哈希值
//func onMessagePreHash(h common.Hash){}
 */
