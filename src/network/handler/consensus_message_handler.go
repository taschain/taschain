package handler

import (
	"core"
	"consensus/logical"
	"consensus/groupsig"
	)

//groupsig.ID  全网唯一 可作为节点在网络内的标识

type ConsensusMessageHandler struct{

}

//----------------------------------------------------组初始化-----------------------------------------------------------
//广播 组初始化消息  组内广播
// param： id slice
//         signData
func (h *ConsensusMessageHandler)sendGroupInitMessage(is []groupsig.ID,sd logical.SignData){

}

//接收组初始化消息 生成针对其他成员的密钥片段
//param: signData
func (h *ConsensusMessageHandler)onMessageGroupInit(sd logical.SignData)  {
	
}


//组内广播密钥   for each定向发送 组内广播
//param：密钥片段map
//       signData
func (h *ConsensusMessageHandler)sendKeyPiece(km map[groupsig.ID]groupsig.Pubkey,sd logical.SignData){

}

//组内节点接收密钥片段 保存，收到所有密钥片段后 生成用来签名的  组用户私钥
//param:keyPiece
//      signData
func (h *ConsensusMessageHandler)onMessageKeyPiece(kp groupsig.Pubkey,sd logical.SignData){

}


//广播组用户公钥  组内广播
//param:组用户公钥 memberPubkey
// signData
func (h *ConsensusMessageHandler)sendMemberPubkey(mk groupsig.Pubkey,sd logical.SignData){

}

//接收组用户公钥 单位时间内超过k个 生成组公钥
//参数:组用户公钥
//     signData
func (h *ConsensusMessageHandler)onMessageMemberPubKey(mk groupsig.Pubkey,sd logical.SignData){

}

//组初始化完成 广播组信息 全网广播
//参数: groupInfo senderId
func (h *ConsensusMessageHandler)broadcastGroupInfo(gi logical.StaticGroupInfo, sd logical.SignData){
 //上链 本地写数据库
}

//接收组  不验证了 进行上链 广播
//参数: groupInfo senderId
// todo  存在单点问题
func (h *ConsensusMessageHandler)onMessageGroupInfo(gi logical.StaticGroupInfo, sd logical.SignData){

}

//-----------------------------------------------------------------组铸币----------------------------------------------

//组内成员发现自己所在组成为铸币组 发消息通知全组 组内广播
//param: 组信息
//      SignData
func (h *ConsensusMessageHandler)sendCurrentGroupCast(gi logical.StaticGroupInfo, sd logical.SignData){

}

//接收开始铸币的消息，判定自己是否为铸币节点，如果是  启动铸币流程  如果不是 保持静默
//param:SignData
func (h *ConsensusMessageHandler)onMessageCurrentGroupCast(sd logical.SignData){

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


//铸币节点完成铸币，将blockheader  签名后发送至组内其他节点进行验证。组内广播
//param BlockHeader 组信息 signData
func (h *ConsensusMessageHandler)sendCastVerify(bh core.BlockHeader,gi logical.StaticGroupInfo,sd logical.SignData){}

//内节点接收 待验证的blockheader 进行验证
//param :BlockHeader
//        senderId
func (h *ConsensusMessageHandler)onMessageCastVerify(bh core.BlockHeader,sd logical.SignData){}




//组内节点  验证通过后 自身签名 广播验证块 组内广播  验证不通过 保持静默
//param :BlockHeader
//       member signature
//       signData
func (h *ConsensusMessageHandler)sendVerifiedCast(bh core.BlockHeader,gi logical.StaticGroupInfo,sd logical.SignData){}

//接收组内广播的验证块，签名数量累加，某一事件窗口内 接收超过k个签名 进行组签名
//param :BlockHeader
//       member signature
//       signData
func (h *ConsensusMessageHandler)onMessageVerifiedCast(bh core.BlockHeader,sd logical.SignData){}