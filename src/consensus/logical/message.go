package logical

import (
	"common"
	"consensus/groupsig"
	"time"
)

type CONSENSUS_TYPE uint8

const (
	CAST_BLOCK   CONSENSUS_TYPE = iota //铸块
	CREATE_GROUP                       //建组
)


//共识摘要
type ConsensusSummary struct {
	Castor      groupsig.ID        //铸块人
	DataHash    common.Hash        //待共识的哈希
	QueueNumber int64              //出块序号
	CastTime    time.Time          //铸块时间（铸块人的出块时间）
	Witness     groupsig.ID        //见证人
	Sign        groupsig.Signature //见证人签名
}

func (cs ConsensusSummary) IsValid() bool {
	return cs.Castor.IsValid() && cs.DataHash.IsValid() && cs.QueueNumber >= 0 && cs.Witness.IsValid() && cs.Sign.IsValid() && !cs.CastTime.IsZero()
}

func (cs ConsensusSummary) IsKing() bool {
	return cs.Castor == cs.Witness
}

//成为当前处理组消息 - 由第一个发现当前组成为铸块组的成员发出
type ConsensusCurrentMessage struct {
	PreHash       common.Hash    //上一块哈希
	PreTime       time.Time      //上一块完成时间
	ConsensusType CONSENSUS_TYPE //共识类型
	BlockHeight   uint64         //铸块高度
	Instigator    groupsig.ID    //发起者（发现者）
	si            SignData
}

type ConsensusBlockMessageBase struct {
	bh BlockHeader
	si SignData
}

//铸块消息 - 由成为KING的组成员发出
type ConsensusCastMessage ConsensusBlockMessageBase

//验证消息 - 由组内的验证人发出（对KING的铸块进行验证）
type ConsensusVerifyMessage ConsensusBlockMessageBase

//出块消息 - 该组成功完成了一个出块，由组内任意一个收集到k个签名的成员发出
type ConsensusBlockMessage ConsensusBlockMessageBase

func GenConsensusSummary(bh BlockHeader, si SignData) ConsensusSummary {
	var cs ConsensusSummary
	cs.Castor = bh.Castor
	cs.DataHash = si.DataHash
	cs.QueueNumber = int64(bh.QueueNumber)
	cs.Witness = si.SignMember
	cs.Sign = si.DataSign
	cs.CastTime = bh.CurTime
	return cs
}
