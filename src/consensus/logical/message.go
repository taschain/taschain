package logical

import (
	"common"
	"consensus/groupsig"
	"core"
	"time"
)

//铸块消息族
//成为当前处理组消息 - 由第一个发现当前组成为铸块组的成员发出
type ConsensusCurrentMessage struct {
	PreHash       common.Hash    //上一块哈希
	PreTime       time.Time      //上一块完成时间
	//ConsensusType CONSENSUS_TYPE //共识类型
	BlockHeight   uint64         //铸块高度
	Instigator    groupsig.ID    //发起者（发现者）
	si            SignData
}

type ConsensusBlockMessageBase struct {
	bh core.BlockHeader
	si SignData
}

//铸块消息 - 由成为KING的组成员发出
type ConsensusCastMessage ConsensusBlockMessageBase

//验证消息 - 由组内的验证人发出（对KING的铸块进行验证）
type ConsensusVerifyMessage ConsensusBlockMessageBase

//出块消息 - 该组成功完成了一个出块，由组内任意一个收集到k个签名的成员发出
type ConsensusBlockMessage ConsensusBlockMessageBase

//组初始化消息族
//收到父亲组的启动组初始化消息
type ConsensusGroupRawMessage struct {
	gi ConsensusGroupInitSummary //组初始化共识
	si SignData                  //用户个人签名
}

//向所有组内成员发送秘密片段消息（不同成员不同）
type ConsensusSharePieceMessage struct {
	cd []byte   //用接收者私人公钥加密的数据（只有接收者可以解开）。解密后的结构为SecKeyInfo
	si SignData //用户个人签名
}

//向所有组内成员发送自己的（片段）签名公钥消息（所有成员相同）
type ConsensusPubKeyPieceMessage struct {
	pk groupsig.Pubkey //组公钥片段
	si SignData        //用户个人签名（发送者ID）
}

//向组外广播该组已经初始化完成(组外节点要收到门限个消息相同，才进行上链)
type ConsensusGroupInitedMessage struct {
	gi StaticGroupInfo //组初始化完成后的上链组信息（辅助map不用传输和上链）
	si SignData        //用户个人签名
}
