package logical

import (
	"common"
	"consensus/groupsig"
	"consensus/rand"
	"core"
	"hash"
	"strconv"
	"time"
)

//铸块共识摘要
type ConsensusBlockSummary struct {
	Castor      groupsig.ID        //铸块人
	DataHash    common.Hash        //待共识的区块头哈希
	QueueNumber int64              //出块序号
	CastTime    time.Time          //铸块时间（铸块人的出块时间）
	//si SignData							//验证签名
}

func (cs ConsensusBlockSummary) IsValid() bool {
	return cs.Castor.IsValid() && cs.DataHash.IsValid() && cs.QueueNumber >= 0 && !cs.CastTime.IsZero()
}

func (cs ConsensusBlockSummary) IsKing(uid groupsig.ID) bool {
	return uid == cs.Castor
}

func (cs ConsensusBlockSummary) GenHash() hash.Hash {
	buf := cs.Castor.GetHexString()
	buf += cs.DataHash.Hex()
	buf += strconv.FormatInt(cs.QueueNumber, 16)
	buf += cs.CastTime.Format(time.ANSIC)
	return rand.HashBytes([]byte(buf))
}

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

func GenConsensusSummary(bh core.BlockHeader/*, si SignData*/) ConsensusBlockSummary {
	var cs ConsensusBlockSummary
	cs.Castor = bh.Castor
	cs.DataHash = bh.GenHash()
	//cs.DataHash = si.DataHash
	cs.QueueNumber = int64(bh.QueueNumber)
	cs.CastTime = bh.CurTime
	//cs.si = si
	return cs
}

///////////////////////////////////////////////////////////////////////////////
//组初始化共识摘要
type ConsensusGroupInitSummary struct {
	ParentID  groupsig.ID //父亲组ID
	Authority uint64      //权限相关数据（父亲组赋予）
	Name      [64]byte    //父亲组取的名字
	DummyID   groupsig.ID //父亲组给的伪ID
	BeginTime time.Time   //初始化开始时间（必须在指定时间窗口内完成初始化）
}

//是否已超过允许的初始化共识时间窗口
func (gis *ConsensusGroupInitSummary) IsExpired() bool {
	if !gis.BeginTime.IsZero() && time.Since(gis.BeginTime).Seconds() <= float64(GROUP_INIT_MAX_SECONDS) {
		return false
	} else {
		return true
	}
}

//生成哈希
func (gis *ConsensusGroupInitSummary) GenHash() hash.Hash {
	buf := gis.ParentID.GetHexString()
	buf += strconv.FormatUint(gis.Authority, 16)
	buf += string(gis.Name[:])
	buf += gis.DummyID.GetHexString()
	buf += gis.BeginTime.Format(time.ANSIC)
	return rand.HashBytes([]byte(buf))
}

//组初始化消息族
type ConsensusGroupRawMessage struct {
	gi ConsensusGroupInitSummary //组初始化共识
	si SignData                  //用户个人签名
}

//向所有组内成员发送秘密片段消息（不同成员不同）
type ConsensusSharePieceMessage struct {
	cd []byte   //用接收者私人公钥加密的数据（只有接收者可以解开）。解密后的结构为group_info.go的SecKeyInfo
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
