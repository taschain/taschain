package model

import (
	"bytes"
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/base"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/middleware/types"
	"github.com/taschain/taschain/taslog"
	"strconv"
	"time"
)

var SlowLog taslog.Logger

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

func (sign *BaseSignedMessage) VerifySign(pk groupsig.Pubkey) (ok bool) {
	if !sign.SI.GetID().IsValid() {
		return false
	}
	ok = sign.SI.VerifySign(pk)
	if !ok {
		fmt.Printf("verifySign fail, pk=%v, id=%v, sign=%v, data=%v\n", pk.GetHexString(), sign.SI.SignMember.GetHexString(), sign.SI.DataSign.GetHexString(), sign.SI.DataHash.Hex())
	}
	return
}

//收到父亲组的启动组初始化消息
//to do : 组成员ID列表在哪里提供
type ConsensusGroupRawMessage struct {
	GInfo ConsensusGroupInitInfo //组初始化共识
	BaseSignedMessage
}

func (msg *ConsensusGroupRawMessage) GenHash() common.Hash {
	return msg.GInfo.GI.GetHash()
}

func (msg *ConsensusGroupRawMessage) MemberExist(id groupsig.ID) bool {
	return msg.GInfo.MemberExists(id)
}

//向所有组内成员发送秘密片段消息（不同成员不同）
type ConsensusSharePieceMessage struct {
	GHash common.Hash //组初始化共识（ConsensusGroupInitSummary）的哈希
	//GHash   common.Hash //父亲组指定的新组hash，GroupHeader的hash
	Dest  groupsig.ID //接收者（矿工）的ID
	Share SharePiece  //消息明文（由传输层用接收者公钥对消息进行加密和解密）
	//SI      SignData    //矿工个人签名
	MemCnt int32
	BaseSignedMessage
}

func (msg *ConsensusSharePieceMessage) GenHash() common.Hash {
	buf := msg.GHash.Bytes()
	//buf = append(buf, msg.GHash.Bytes()...)
	buf = append(buf, msg.Dest.Serialize()...)
	buf = append(buf, msg.Share.Pub.Serialize()...)
	buf = append(buf, msg.Share.Share.Serialize()...)
	return base.Data2CommonHash(buf)
}

//向组内成员发送签名公钥消息（所有成员相同）
type ConsensusSignPubKeyMessage struct {
	GHash   common.Hash
	GroupID groupsig.ID     //组id
	SignPK  groupsig.Pubkey //组成员签名公钥
	//GSign  groupsig.Signature //用组成员签名私钥对GIS进行的签名（用于验证组成员签名公钥的正确性）
	//SI      SignData           //矿工个人签名
	MemCnt int32
	BaseSignedMessage
}

//func (msg *ConsensusSignPubKeyMessage) GenGSign(sk groupsig.Seckey) {
//	msg.GSign = groupsig.Sign(sk, msg.GHash.Bytes())
//}
//
//
//func (msg *ConsensusSignPubKeyMessage) VerifyGSign(pk groupsig.Pubkey) bool {
//	return groupsig.VerifySig(pk, msg.GHash.Bytes(), msg.GSign)
//}

func (msg *ConsensusSignPubKeyMessage) GenHash() common.Hash {
	buf := msg.GHash.Bytes()
	buf = append(buf, msg.GroupID.Serialize()...)
	buf = append(buf, msg.SignPK.Serialize()...)
	return base.Data2CommonHash(buf)
}

//请求签名公钥
type ConsensusSignPubkeyReqMessage struct {
	BaseSignedMessage
	GroupID groupsig.ID
}

func (m *ConsensusSignPubkeyReqMessage) GenHash() common.Hash {
	return base.Data2CommonHash(m.GroupID.Serialize())
}

//向组外广播该组已经初始化完成(组外节点要收到门限个消息相同，才进行上链)
type ConsensusGroupInitedMessage struct {
	GHash        common.Hash
	GroupID      groupsig.ID     //组ID(可以由组公钥生成)
	GroupPK      groupsig.Pubkey //组公钥
	CreateHeight uint64          //组开始创建时的高度
	ParentSign   groupsig.Signature
	MemMask      []byte //组成员mask，值为1的位表名该candidate在组成员列表中,根据该mask表和candidate集合可恢复出组成员列表
	MemCnt       int32
	BaseSignedMessage
}

func (msg *ConsensusGroupInitedMessage) GenHash() common.Hash {
	buf := bytes.Buffer{}
	buf.Write(msg.GHash.Bytes())
	buf.Write(msg.GroupID.Serialize())
	buf.Write(msg.GroupPK.Serialize())
	buf.Write(common.Uint64ToByte(msg.CreateHeight))
	buf.Write(msg.ParentSign.Serialize())
	buf.Write(msg.MemMask)
	return base.Data2CommonHash(buf.Bytes())
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
	buf := msg.PreHash.String()
	buf += string(msg.GroupID[:])
	buf += msg.PreTime.String()
	buf += strconv.FormatUint(msg.BlockHeight, 10)
	return base.Data2CommonHash([]byte(buf))
}

type ConsensusCastMessage struct {
	BH types.BlockHeader
	//GroupID groupsig.ID
	ProveHash common.Hash
	BaseSignedMessage
}

func (msg *ConsensusCastMessage) GenHash() common.Hash {
	//buf := bytes.Buffer{}
	//buf.Write(msg.BH.GenHash().Bytes())
	//for _, h := range msg.ProveHash {
	//	buf.Write(h.Bytes())
	//}
	//return base.Data2CommonHash(buf.Bytes())
	return msg.BH.GenHash()
}

//func (msg *ConsensusCastMessage) GenRandomSign(skey groupsig.Seckey, preRandom []byte)  {
//	sig := groupsig.Sign(skey, preRandom)
//    msg.BH.Random = sig.Serialize()
//}

func (msg *ConsensusCastMessage) VerifyRandomSign(pkey groupsig.Pubkey, preRandom []byte) bool {
	sig := groupsig.DeserializeSign(msg.BH.Random)
	if sig == nil || sig.IsNil() {
		return false
	}
	return groupsig.VerifySig(pkey, preRandom, *sig)
}

//出块消息 - 由成为KING的组成员发出
//type ConsensusCastMessage struct {
//	ConsensusBlockMessageBase
//}

//验证消息 - 由组内的验证人发出（对KING的出块进行验证）
type ConsensusVerifyMessage struct {
	BlockHash  common.Hash
	RandomSign groupsig.Signature
	BaseSignedMessage
}

func (msg *ConsensusVerifyMessage) GenHash() common.Hash {
	//buf := bytes.Buffer{}
	//buf.Write(msg.BH.GenHash().Bytes())
	//for _, h := range msg.ProveHash {
	//	buf.Write(h.Bytes())
	//}
	//return base.Data2CommonHash(buf.Bytes())
	return msg.BlockHash
}

func (msg *ConsensusVerifyMessage) GenRandomSign(skey groupsig.Seckey, preRandom []byte) {
	sig := groupsig.Sign(skey, preRandom)
	msg.RandomSign = sig
}

//func (msg *ConsensusVerifyMessage) VerifyRandomSign(pkey groupsig.Pubkey, preRandom []byte) bool {
//	sig := msg.RandomSign
//	if sig.IsNil() {
//		return false
//	}
//	return groupsig.VerifySig(pkey, preRandom, sig)
//}

//铸块成功消息 - 该组成功完成了一个铸块，由组内任意一个收集到k个签名的成员发出
type ConsensusBlockMessage struct {
	Block types.Block
}

func (msg *ConsensusBlockMessage) GenHash() common.Hash {
	buf := msg.Block.Header.GenHash().Bytes()
	buf = append(buf, msg.Block.Header.GroupId...)
	return base.Data2CommonHash(buf)
}

func (msg *ConsensusBlockMessage) VerifySig(gpk groupsig.Pubkey, preRandom []byte) bool {
	sig := groupsig.DeserializeSign(msg.Block.Header.Signature)
	if sig == nil {
		return false
	}
	b := groupsig.VerifySig(gpk, msg.Block.Header.Hash.Bytes(), *sig)
	if !b {
		return false
	}
	rsig := groupsig.DeserializeSign(msg.Block.Header.Random)
	if rsig == nil {
		return false
	}
	return groupsig.VerifySig(gpk, preRandom, *rsig)
}

//====================================父组建组共识消息================================

type ConsensusCreateGroupRawMessage struct {
	GInfo ConsensusGroupInitInfo //组初始化共识
	BaseSignedMessage
}

func (msg *ConsensusCreateGroupRawMessage) GenHash() common.Hash {
	return msg.GInfo.GI.GetHash()
}

type ConsensusCreateGroupSignMessage struct {
	GHash common.Hash
	BaseSignedMessage
	Launcher groupsig.ID
}

func (msg *ConsensusCreateGroupSignMessage) GenHash() common.Hash {
	return msg.GHash
}

//==============================奖励交易==============================
type CastRewardTransSignReqMessage struct {
	BaseSignedMessage
	Reward       types.Bonus
	SignedPieces []groupsig.Signature
	ReceiveTime  time.Time //收到请求信息的时间戳，加入future的msg，若对应的块长期未上链则需要根据此时间戳清除
}

func (msg *CastRewardTransSignReqMessage) GenHash() common.Hash {
	return msg.Reward.TxHash
}

type CastRewardTransSignMessage struct {
	BaseSignedMessage
	ReqHash   common.Hash
	BlockHash common.Hash

	//不序列化
	GroupID  groupsig.ID
	Launcher groupsig.ID
}

func (msg *CastRewardTransSignMessage) GenHash() common.Hash {
	return msg.ReqHash
}

type CreateGroupPingMessage struct {
	BaseSignedMessage
	FromGroupID groupsig.ID
	PingID      string
	BaseHeight  uint64
}

func (msg *CreateGroupPingMessage) GenHash() common.Hash {
	buf := msg.FromGroupID.Serialize()
	buf = append(buf, []byte(msg.PingID)...)
	buf = append(buf, common.Uint64ToByte(msg.BaseHeight)...)
	return base.Data2CommonHash(buf)
}

type CreateGroupPongMessage struct {
	BaseSignedMessage
	PingID string
	Ts     time.Time
}

func (msg *CreateGroupPongMessage) GenHash() common.Hash {
	buf := []byte(msg.PingID)
	tb, _ := msg.Ts.MarshalBinary()
	buf = append(buf, tb...)
	return base.Data2CommonHash(tb)
}

type ReqSharePieceMessage struct {
	BaseSignedMessage
	GHash common.Hash
}

func (msg *ReqSharePieceMessage) GenHash() common.Hash {
	return msg.GHash
}

//向所有组内成员发送秘密片段消息（不同成员不同）
type ResponseSharePieceMessage struct {
	GHash common.Hash //组初始化共识（ConsensusGroupInitSummary）的哈希
	Share SharePiece  //消息明文（由传输层用接收者公钥对消息进行加密和解密）
	BaseSignedMessage
}

func (msg *ResponseSharePieceMessage) GenHash() common.Hash {
	buf := msg.GHash.Bytes()
	//buf = append(buf, msg.GHash.Bytes()...)
	buf = append(buf, msg.Share.Pub.Serialize()...)
	buf = append(buf, msg.Share.Share.Serialize()...)
	return base.Data2CommonHash(buf)
}

type BlockSignAggrMessage struct {
	Hash   common.Hash
	Sign   groupsig.Signature
	Random groupsig.Signature
}

type ReqProposalBlock struct {
	Hash common.Hash
}

type ResponseProposalBlock struct {
	Hash         common.Hash
	Transactions []*types.Transaction
}
