package biz

import (
	"core"
	"consensus/logical"
	"consensus/groupsig"
)

//groupsig.ID  全网唯一 可作为节点在网络内的标识

//-----------------------------------------------------回调函数定义------------------------------------------------------

//对于每个组成员生成对应的PIECE
type genKeySharePieceFn func(sd logical.SignData)

//KEY PIECE 到达监听
type keyPieceArrivedNotifyFn func(kp groupsig.Pubkey, sd logical.SignData)

//组用户公钥到达监听
type memberPubkeyArrivedNotifyFn func(mk groupsig.Pubkey, sd logical.SignData)

//接收到新的组 本地上链
type addNewGroupToChainFn func(g logical.StaticGroupInfo, sd logical.SignData)

//接收组铸币请求 开始铸币判定和铸币流程
type beginCastFn func(sd logical.SignData)

//组内节点验证接收到的刚铸出来的币
type verifyCastFn func(bh core.BlockHeader, sd logical.SignData)

//组内验证过的 块消息监听到达
type verifiedCastMessageArrivedNotifyFn func(bh core.BlockHeader, sd logical.SignData)

//---------------------------------------------------------------------------------------------------------------------
type ConsensusMessageHandler struct {
	sv *signValidator

	genSHarePiece   genKeySharePieceFn
	keyPieceGot     keyPieceArrivedNotifyFn
	memberPubkeyGot memberPubkeyArrivedNotifyFn
	addNewGroup     addNewGroupToChainFn

	beginCast       beginCastFn
	verifyCast      verifyCastFn
	castVerifiedGot verifiedCastMessageArrivedNotifyFn
}

func NewConsensusMessageHandler(genSHarePiece genKeySharePieceFn, keyPieceGot keyPieceArrivedNotifyFn, memberPubkeyGot memberPubkeyArrivedNotifyFn,
	addNewGroup addNewGroupToChainFn, beginCast beginCastFn, verifyCast verifyCastFn, castVerifiedGot verifiedCastMessageArrivedNotifyFn) ConsensusMessageHandler {

	return ConsensusMessageHandler{
		sv:              GetSignValidatorInstance(),
		genSHarePiece:   genSHarePiece,
		keyPieceGot:     keyPieceGot,
		memberPubkeyGot: memberPubkeyGot,
		addNewGroup:     addNewGroup,
		beginCast:       beginCast,
		verifyCast:      verifyCast,
		castVerifiedGot: castVerifiedGot,
	}
}

//----------------------------------------------------组初始化-----------------------------------------------------------

//接收组初始化消息 生成针对其他成员的密钥片段
//param: signData
func (h ConsensusMessageHandler) onMessageGroupInit(sd logical.SignData) {
	h.genSHarePiece(sd)
}

//组内节点接收密钥片段 保存，收到所有密钥片段后 生成用来签名的  组用户私钥
//param:keyPiece
//      signData
func (h ConsensusMessageHandler) onMessageKeyPiece(kp groupsig.Pubkey, sd logical.SignData) {
	h.keyPieceGot(kp, sd)
}

//接收组用户公钥 单位时间内超过k个 生成组公钥
//参数:组用户公钥
//     signData
func (h ConsensusMessageHandler) onMessageMemberPubKey(mk groupsig.Pubkey, sd logical.SignData) {
	h.memberPubkeyGot(mk, sd)
}

//接收组  不验证了 进行上链 广播
//参数: groupInfo senderId
// todo  存在单点问题
func (h ConsensusMessageHandler) onMessageGroupInfo(gi logical.StaticGroupInfo, sd logical.SignData) {
	h.addNewGroup(gi, sd)
}

//-----------------------------------------------------------------组铸币----------------------------------------------

//接收开始铸币的消息，判定自己是否为铸币节点，如果是  启动铸币流程  如果不是 保持静默
//param:SignData
func (h ConsensusMessageHandler) onMessageCurrentGroupCast(sd logical.SignData) {
	h.beginCast(sd)
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

//内节点接收 待验证的blockheader 进行验证
//param :BlockHeader
//        senderId
func (h ConsensusMessageHandler) onMessageCastVerify(bh core.BlockHeader, sd logical.SignData) {
	h.verifyCast(bh, sd)
}

//接收组内广播的验证块，签名数量累加，某一事件窗口内 接收超过k个签名 进行组签名
//param :BlockHeader
//       member signature
//       signData
func (h ConsensusMessageHandler) onMessageVerifiedCast(bh core.BlockHeader, sd logical.SignData) {
	h.castVerifiedGot(bh, sd)
}
