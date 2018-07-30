package model

import (
	"common"
	"consensus/groupsig"
	"consensus/base"
	"strconv"
	"time"
	"middleware/types"
)

type ISignedMessage interface {
	GenSign(ski SecKeyInfo, hasher Hasher) bool
	VerifySign(pk groupsig.Pubkey) bool
}

type Hasher interface {
	GenHash() common.Hash
}

type BaseSignedMessage struct {
	SI SignData
}


func (sign *BaseSignedMessage) GenSign(ski SecKeyInfo, hasher Hasher) bool {
	if !ski.IsValid() {
		return false
	}
	sign.SI = GenSignData(hasher.GenHash(), ski.GetID(), ski.SK)
	return true
}

func (sign *BaseSignedMessage) VerifySign(pk groupsig.Pubkey) bool {
	if !sign.SI.GetID().IsValid() {
		return false
	}
	return sign.SI.VerifySign(pk)
}


//收到父亲组的启动组初始化消息
//to do : 组成员ID列表在哪里提供
type ConsensusGroupRawMessage struct {
	GI   ConsensusGroupInitSummary //组初始化共识
	MEMS []PubKeyInfo              //组成员列表，该次序不可变更，影响组内铸块排位。
	//SI   SignData                  //矿工（父亲组成员）个人签名
	BaseSignedMessage
}

func (msg *ConsensusGroupRawMessage) GenHash() common.Hash {
	return msg.GI.GenHash()
}

func (msg *ConsensusGroupRawMessage) MemberExist(id groupsig.ID) bool {
	for _, mem := range msg.MEMS {
		if mem.ID.IsEqual(id) {
			return true
		}
	}
	return false
}

//向所有组内成员发送秘密片段消息（不同成员不同）
type ConsensusSharePieceMessage struct {
	GISHash common.Hash //组初始化共识（ConsensusGroupInitSummary）的哈希
	DummyID groupsig.ID //父亲组指定的新组（哑元）ID，即ConsensusGroupInitSummary的DummyID
	Dest    groupsig.ID //接收者（矿工）的ID
	Share   SharePiece  //消息明文（由传输层用接收者公钥对消息进行加密和解密）
	//SI      SignData    //矿工个人签名
	BaseSignedMessage
}

func (msg *ConsensusSharePieceMessage) GenHash() common.Hash {
	buf := msg.GISHash.Bytes()
	buf = append(buf, msg.DummyID.Serialize()...)
	buf = append(buf, msg.Dest.Serialize()...)
	buf = append(buf, msg.Share.Pub.Serialize()...)
	buf = append(buf, msg.Share.Share.Serialize()...)
	return base.Data2CommonHash(buf)
}
//向组内成员发送签名公钥消息（所有成员相同）
type ConsensusSignPubKeyMessage struct {
	GISHash common.Hash //组初始化共识的哈希
	DummyID groupsig.ID
	SignPK  groupsig.Pubkey    //组成员签名公钥
	GISSign groupsig.Signature //用组成员签名私钥对GIS进行的签名（用于验证组成员签名公钥的正确性）
	//SI      SignData           //矿工个人签名
	BaseSignedMessage
}

func (msg *ConsensusSignPubKeyMessage) GenGISSign(sk groupsig.Seckey) {
	msg.GISSign = groupsig.Sign(sk, msg.GISHash.Bytes())
}


func (msg *ConsensusSignPubKeyMessage) VerifyGISSign(pk groupsig.Pubkey) bool {
	return groupsig.VerifySig(pk, msg.GISHash.Bytes(), msg.GISSign)
}

func (msg *ConsensusSignPubKeyMessage) GenHash() common.Hash {
	buf := msg.GISHash.Bytes()
	buf = append(buf, msg.DummyID.Serialize()...)
	buf = append(buf, msg.SignPK.Serialize()...)
	return base.Data2CommonHash(buf)
}


//向组外广播该组已经初始化完成(组外节点要收到门限个消息相同，才进行上链)
type ConsensusGroupInitedMessage struct {
	GI StaticGroupSummary //组初始化完成后的上链组信息（辅助map不用传输和上链）
	//SI SignData        //用户个人签名
	BaseSignedMessage
}

func (msg *ConsensusGroupInitedMessage) GenHash() common.Hash {
	return msg.GI.GenHash()
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
	BaseSignedMessage
}

func (msg *ConsensusCurrentMessage) GenHash() common.Hash {
	buf := msg.PreHash.Str()
	buf += string(msg.GroupID[:])
	buf += msg.PreTime.String()
	buf += strconv.FormatUint(msg.BlockHeight, 10)
	return base.Data2CommonHash([]byte(buf))
}

type ConsensusBlockMessageBase struct {
	BH types.BlockHeader
	//GroupID groupsig.ID
	BaseSignedMessage
}

func (msg *ConsensusBlockMessageBase) GenHash() common.Hash {
	return msg.BH.GenHash()
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
	Block   types.Block
	GroupID groupsig.ID
	BaseSignedMessage
}

func (msg *ConsensusBlockMessage) GenHash() common.Hash {
	buf := msg.Block.Header.GenHash().Bytes()
	buf = append(buf, msg.GroupID.Serialize()...)
	return base.Data2CommonHash(buf)
}


//====================================父组建组共识消息================================

type ConsensusCreateGroupRawMessage struct {
	GI   ConsensusGroupInitSummary //组初始化共识
	IDs []groupsig.ID              //组成员列表，该次序不可变更，影响组内铸块排位。
	BaseSignedMessage
}

func (msg *ConsensusCreateGroupRawMessage) GenHash() common.Hash {
    return msg.GI.GenHash()
}


type ConsensusCreateGroupSignMessage struct {
	GI 	ConsensusGroupInitSummary
	BaseSignedMessage
	Launcher groupsig.ID
}

func (msg *ConsensusCreateGroupSignMessage) GenHash() common.Hash {
	return msg.GI.GenHash()
}
