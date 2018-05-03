package logical

import (
	"common"
	"consensus/groupsig"
	"consensus/rand"
	"core"
	"strconv"
	"time"
)

//组初始化消息族
//组初始化消息族的SI用矿工公钥验签（和组无关）

//收到父亲组的启动组初始化消息
//to do : 组成员ID列表在哪里提供
type ConsensusGroupRawMessage struct {
	GI   ConsensusGroupInitSummary //组初始化共识
	MEMS []PubKeyInfo              //组成员列表，该次序不可变更，影响组内铸块排位。
	SI   SignData                  //矿工（父亲组成员）个人签名
}

func (msg *ConsensusGroupRawMessage) GenSign(ski SecKeyInfo) bool {
	if !ski.IsValid() {
		return false
	}
	msg.SI.SignMember = ski.GetID()
	msg.SI.DataHash = msg.GI.GenHash()
	return msg.SI.GenSign(ski.SK)
}

func (msg ConsensusGroupRawMessage) VerifySign(pk groupsig.Pubkey) bool {
	if !msg.SI.GetID().IsValid() {
		return false
	}
	return msg.SI.VerifySign(pk)
}

//向所有组内成员发送秘密片段消息（不同成员不同）
type ConsensusSharePieceMessage struct {
	GISHash common.Hash //组初始化共识（ConsensusGroupInitSummary）的哈希
	DummyID groupsig.ID //父亲组指定的新组（哑元）ID，即ConsensusGroupInitSummary的DummyID
	Dest    groupsig.ID //接收者（矿工）的ID
	Share   SharePiece  //消息明文（由传输层用接收者公钥对消息进行加密和解密）
	SI      SignData    //矿工个人签名
}

func (msg *ConsensusSharePieceMessage) GenSign(ski SecKeyInfo) bool {
	if !ski.IsValid() {
		return false
	}
	buf := msg.GISHash.Str()
	buf += msg.DummyID.GetHexString()
	buf += msg.Dest.GetHexString()
	buf += msg.Share.Pub.GetHexString()
	buf += msg.Share.Share.GetHexString()
	msg.SI.DataHash = rand.Data2CommonHash([]byte(buf))
	msg.SI.SignMember = ski.GetID()
	return msg.SI.GenSign(ski.SK)
}

func (msg ConsensusSharePieceMessage) VerifySign(pk groupsig.Pubkey) bool {
	if !msg.SI.GetID().IsValid() {
		return false
	}
	return msg.SI.VerifySign(pk)
}

//向组内成员发送签名公钥消息（所有成员相同）
type ConsensusSignPubKeyMessage struct {
	GISHash common.Hash //组初始化共识的哈希
	DummyID groupsig.ID
	SignPK  groupsig.Pubkey //组成员签名公钥
	SI      SignData        //矿工个人签名
}

func (msg *ConsensusSignPubKeyMessage) GenSign(ski SecKeyInfo) bool {
	if !ski.IsValid() {
		return false
	}
	buf := msg.GISHash.Str()
	buf += msg.DummyID.GetHexString()
	buf += msg.SignPK.GetHexString()
	msg.SI.DataHash = rand.Data2CommonHash([]byte(buf))
	msg.SI.SignMember = ski.GetID()
	return msg.SI.GenSign(ski.SK)
}

func (msg ConsensusSignPubKeyMessage) VerifySign(pk groupsig.Pubkey) bool {
	if !msg.SI.GetID().IsValid() {
		return false
	}
	return msg.SI.VerifySign(pk)
}

//向组外广播该组已经初始化完成(组外节点要收到门限个消息相同，才进行上链)
type ConsensusGroupInitedMessage struct {
	GI StaticGroupInfo //组初始化完成后的上链组信息（辅助map不用传输和上链）
	SI SignData        //用户个人签名
}

func (msg *ConsensusGroupInitedMessage) GenSign(ski SecKeyInfo) bool {
	if !ski.IsValid() {
		return false
	}
	msg.SI.SignMember = ski.GetID()
	msg.SI.DataHash = msg.GI.GenHash()
	return msg.SI.GenSign(ski.SK)
}

func (msg ConsensusGroupInitedMessage) VerifySign(pk groupsig.Pubkey) bool {
	if !msg.SI.GetID().IsValid() {
		return false
	}
	return msg.SI.VerifySign(pk)
}

///////////////////////////////////////////////////////////////////////////////
//铸块消息族
//铸块消息族的SI用组成员签名公钥验签

//成为当前处理组消息 - 由第一个发现当前组成为铸块组的成员发出
type ConsensusCurrentMessage struct {
	GroupID     []byte      //铸块组
	PreHash     common.Hash //上一块哈希
	PreTime     time.Time   //上一块完成时间
	BlockHeight uint64      //铸块高度
	SI          SignData    //签名者即为发现者
}

func (msg *ConsensusCurrentMessage) GenSign(ski SecKeyInfo) bool {
	if !ski.IsValid() {
		return false
	}
	buf := msg.PreHash.Str()
	buf += string(msg.GroupID[:])
	buf += msg.PreTime.String()
	buf += strconv.FormatUint(msg.BlockHeight, 10)
	msg.SI.DataHash = rand.Data2CommonHash([]byte(buf))
	msg.SI.SignMember = ski.ID
	return msg.SI.GenSign(ski.SK)
}

func (msg ConsensusCurrentMessage) VerifySign(pk groupsig.Pubkey) bool {
	if !msg.SI.GetID().IsValid() {
		return false
	}
	return msg.SI.VerifySign(pk)
}

type ConsensusBlockMessageBase struct {
	BH      core.BlockHeader
	GroupID groupsig.ID
	SI      SignData
}

func (msg *ConsensusBlockMessageBase) GenSign(ski SecKeyInfo) bool {
	if !ski.IsValid() {
		return false
	}
	buf := msg.BH.GenHash().Str()
	buf += msg.GroupID.GetHexString()
	msg.SI.DataHash = rand.Data2CommonHash([]byte(buf))
	msg.SI.SignMember = ski.ID
	return msg.SI.GenSign(ski.SK)
}

func (msg ConsensusBlockMessageBase) VerifySign(pk groupsig.Pubkey) bool {
	if !msg.SI.GetID().IsValid() {
		return false
	}
	return msg.SI.VerifySign(pk)
}

//出块消息 - 由成为KING的组成员发出
type ConsensusCastMessage struct {
	ConsensusBlockMessageBase
}

//验证消息 - 由组内的验证人发出（对KING的出块进行验证）
type ConsensusVerifyMessage struct {
	ConsensusBlockMessageBase
}

//铸块成功消息 - 该组成功完成了一个铸块，由组内任意一个收集到k个签名的成员发出
type ConsensusBlockMessage struct {
	Block   core.Block
	GroupID groupsig.ID
	SI      SignData
}

func (msg *ConsensusBlockMessage) GenSign(ski SecKeyInfo) bool {
	if !ski.IsValid() {
		return false
	}
	buf := msg.Block.Header.GenHash().Str()
	buf += msg.GroupID.GetHexString()
	msg.SI.DataHash = rand.Data2CommonHash([]byte(buf))
	msg.SI.SignMember = ski.ID
	return msg.SI.GenSign(ski.SK)
}

func (msg ConsensusBlockMessage) VerifySign(pk groupsig.Pubkey) bool {
	if !msg.SI.GetID().IsValid() {
		return false
	}
	return msg.SI.VerifySign(pk)
}
