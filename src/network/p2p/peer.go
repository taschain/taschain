package p2p

import (
	"core"
	"consensus/logical"
	"consensus/groupsig"
	"common"
)




//----------------------------------------------------组初始化-----------------------------------------------------------
//广播 组初始化消息  组内广播
// param： id slice
//         signData
func SendGroupInitMessage(is []groupsig.ID, sd logical.SignData) {

}

//组内广播密钥   for each定向发送 组内广播
//param：密钥片段map
//       signData
func SendKeyPiece(km map[groupsig.ID]groupsig.Pubkey, sd logical.SignData) {

}

//组初始化完成 广播组信息 全网广播
//参数: groupInfo senderId
func BroadcastGroupInfo(gi logical.StaticGroupInfo, sd logical.SignData) {
	//上链 本地写数据库
}

//-----------------------------------------------------------------组铸币----------------------------------------------
//组内成员发现自己所在组成为铸币组 发消息通知全组 组内广播
//param: 组信息
//      SignData
func SendCurrentGroupCast(gi logical.StaticGroupInfo, sd logical.SignData) {

}

//铸币节点完成铸币，将blockheader  签名后发送至组内其他节点进行验证。组内广播
//param BlockHeader 组信息 signData
func SendCastVerify(bh core.BlockHeader, gi logical.StaticGroupInfo, sd logical.SignData) {}

//组内节点  验证通过后 自身签名 广播验证块 组内广播  验证不通过 保持静默
//param :BlockHeader
//       member signature
//       signData
func SendVerifiedCast(bh core.BlockHeader, gi logical.StaticGroupInfo, sd logical.SignData) {}

//验证节点 交易集缺失，索要、特定交易 全网广播
//param:hash slice of transaction slice
//      signData
func RequestTransactionByHash(hs []common.Hash, sd logical.SignData) {}

//对外广播经过组签名的block 全网广播
//param: block
//       member signature
//       signData
func BroadcastNewBlock(b core.Block, sd logical.SignData) {}

/////////////////////////////////////////////////////链同步/////////////////////////////////////////////////////////////
//广播索要链高度
//param: signData
func RequestBlockChainHeight(sd logical.SignData) {}

//向某一节点请求Block
//param: target peer id
//       block height slice
//       sign data
func RequestBlockByHeight(id groupsig.ID, hs []int, sd logical.SignData) {}

////////////////////////////////////////////////////////组同步//////////////////////////////////////////////////////////
//广播索要组链高度
//param: signData
func RequestGroupChainHeight(sd logical.SignData) {}

//向某一节点请求GroupInfo
//param: target peer id
//       group height slice
//       sign data
func RequestGroupInfoByHeight(id groupsig.ID, gs []int, sd logical.SignData) {}
