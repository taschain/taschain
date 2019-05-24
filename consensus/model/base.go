package model

import (
	"bytes"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/base"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/middleware/types"
	"time"
)

//矿工ID信息
type GroupMinerID struct {
	Gid groupsig.ID //组ID
	Uid groupsig.ID //成员ID
}

func NewGroupMinerID(gid groupsig.ID, uid groupsig.ID) *GroupMinerID {
	return &GroupMinerID{
		Gid: gid,
		Uid: uid,
	}
}

func (id GroupMinerID) IsValid() bool {
	return id.Gid.IsValid() && id.Uid.IsValid()
}

//数据签名结构
type SignData struct {
	Version    int32
	DataHash   common.Hash        //哈希值
	DataSign   groupsig.Signature //签名
	SignMember groupsig.ID        //用户ID或组ID，看消息类型
}

func (sd SignData) IsEqual(rhs SignData) bool {
	return sd.DataHash == rhs.DataHash && sd.SignMember.IsEqual(rhs.SignMember) && sd.DataSign.IsEqual(rhs.DataSign)
}

func GenSignData(h common.Hash, id groupsig.ID, sk groupsig.Seckey) SignData {
	return SignData{
		DataHash:   h,
		DataSign:   groupsig.Sign(sk, h.Bytes()),
		SignMember: id,
		Version:    common.ConsensusVersion,
	}
}

/*
func GenSignDataEx(msg []byte, id groupsig.ID, sk groupsig.Seckey) SignData {
	var sd SignData
	if len(msg) <= common.HashLength {
		copy(sd.DataHash[:], msg[:])
		sd.DataSign = groupsig.Sign(sk, msg)
		sd.SignMember = id
	}
	return sd
}
*/

func (sd SignData) GetID() groupsig.ID {
	return sd.SignMember
}

//用sk生成签名
func (sd *SignData) GenSign(sk groupsig.Seckey) bool {
	b := sk.IsValid()
	if b {
		sd.DataSign = groupsig.Sign(sk, sd.DataHash.Bytes())
	}
	return b
}

//用pk验证签名，验证通过返回true，否则false。
func (sd SignData) VerifySign(pk groupsig.Pubkey) bool {
	return groupsig.VerifySig(pk, sd.DataHash.Bytes(), sd.DataSign)
}

//是否已有签名数据
func (sd SignData) HasSign() bool {
	return sd.DataSign.IsValid() && sd.SignMember.IsValid()
}

//id->公钥对
type PubKeyInfo struct {
	ID groupsig.ID
	PK groupsig.Pubkey
}

func NewPubKeyInfo(id groupsig.ID, pk groupsig.Pubkey) PubKeyInfo {
	return PubKeyInfo{
		ID: id,
		PK: pk,
	}
}

func (p PubKeyInfo) IsValid() bool {
	return p.ID.IsValid() && p.PK.IsValid()
}

func (p PubKeyInfo) GetID() groupsig.ID {
	return p.ID
}

//id->私钥对
type SecKeyInfo struct {
	ID groupsig.ID
	SK groupsig.Seckey
}

func NewSecKeyInfo(id groupsig.ID, sk groupsig.Seckey) SecKeyInfo {
	return SecKeyInfo{
		ID: id,
		SK: sk,
	}
}

func (s SecKeyInfo) IsValid() bool {
	return s.ID.IsValid() && s.SK.IsValid()
}

func (s SecKeyInfo) GetID() groupsig.ID {
	return s.ID
}

//组内秘密分享消息结构
type SharePiece struct {
	Share groupsig.Seckey //秘密共享
	Pub   groupsig.Pubkey //矿工（组私密）公钥
}

func (piece SharePiece) IsValid() bool {
	return piece.Share.IsValid() && piece.Pub.IsValid()
}

func (piece SharePiece) IsEqual(rhs SharePiece) bool {
	return piece.Share.IsEqual(rhs.Share) && piece.Pub.IsEqual(rhs.Pub)
}

//map(id->秘密分享)
type SharePieceMap map[string]SharePiece

//成为当前铸块组共识摘要
type CastGroupSummary struct {
	PreHash     common.Hash //上一块哈希
	PreTime     time.Time   //上一块完成时间
	BlockHeight uint64      //当前铸块高度
	GroupID     groupsig.ID //当前组ID
	Castor      groupsig.ID
	CastorPos   int32
}

//组初始化共识摘要
type ConsensusGroupInitSummary struct {
	Signature groupsig.Signature //父亲组签名
	GHeader   *types.GroupHeader
}

func (gis *ConsensusGroupInitSummary) GetHash() common.Hash {
	return gis.GHeader.Hash
}

func (gis *ConsensusGroupInitSummary) ParentID() groupsig.ID {
	return groupsig.DeserializeId(gis.GHeader.Parent)
}

func (gis *ConsensusGroupInitSummary) PreGroupID() groupsig.ID {
	return groupsig.DeserializeId(gis.GHeader.PreGroup)
}

func (gis *ConsensusGroupInitSummary) CreateHeight() uint64 {
	return gis.GHeader.CreateHeight
}

func GenMemberRootByIds(ids []groupsig.ID) common.Hash {
	data := bytes.Buffer{}
	for _, m := range ids {
		data.Write(m.Serialize())
	}
	return base.Data2CommonHash(data.Bytes())
}

func (gis *ConsensusGroupInitSummary) CheckMemberHash(mems []groupsig.ID) bool {
	return gis.GHeader.MemberRoot == GenMemberRootByIds(mems)
}

func (gis *ConsensusGroupInitSummary) ReadyTimeout(height uint64) bool {
	return gis.GHeader.ReadyHeight <= height
}

type ConsensusGroupInitInfo struct {
	GI   ConsensusGroupInitSummary
	Mems []groupsig.ID
}

func (gi *ConsensusGroupInitInfo) MemberExists(id groupsig.ID) bool {
	for _, mem := range gi.Mems {
		if mem.IsEqual(id) {
			return true
		}
	}
	return false
}

func (gi *ConsensusGroupInitInfo) MemberSize() int {
	return len(gi.Mems)
}

func (gi *ConsensusGroupInitInfo) GroupHash() common.Hash {
	return gi.GI.GetHash()
}

//type StaticGroupSummary struct {
//	GroupID  groupsig.ID               //组ID(可以由组公钥生成)
//	GroupPK  groupsig.Pubkey           //组公钥
//	//Members  []PubKeyInfo              //组内成员的静态信息(严格按照链上次序，全网一致，不然影响组铸块)。to do : 组成员的公钥是否有必要保存在这里？
//	//GIS 	ConsensusGroupInitSummary
//	GHash 	common.Hash
//}
//
//func (sgs *StaticGroupSummary) GenHash() common.Hash {
//	buf := sgs.GroupID.Serialize()
//	buf = append(buf, sgs.GroupPK.Serialize()...)
//
//	buf = append(buf, sgs.GHash.Bytes()...)
//	hash := base.Data2CommonHash(buf)
//	return hash
//}

type BlockProposalDetail struct {
	BH     *types.BlockHeader
	Proves []common.Hash
}
