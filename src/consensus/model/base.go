package model

import (
	"common"
	"strconv"
	"time"
	"consensus/base"
	"consensus/groupsig"
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
	DataHash   common.Hash        //哈希值
	DataSign   groupsig.Signature //签名
	SignMember groupsig.ID        //用户ID或组ID，看消息类型
}

func (sd SignData) IsEqual(rhs SignData) bool {
	return sd.DataHash.Str() == rhs.DataHash.Str() && sd.SignMember.IsEqual(rhs.SignMember) && sd.DataSign.IsEqual(rhs.DataSign)
}

func GenSignData(h common.Hash, id groupsig.ID, sk groupsig.Seckey) SignData {
	return SignData{
		DataHash: h,
		DataSign: groupsig.Sign(sk, h.Bytes()),
		SignMember: id,
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
	return pk.IsValid() && groupsig.VerifySig(pk, sd.DataHash.Bytes(), sd.DataSign)
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
		ID:id,
		PK:pk,
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
		ID:id,
		SK:sk,
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
type ShareMapID map[string]SharePiece

type MinerInfo struct {
	MinerID    groupsig.ID //矿工ID
	SecretSeed base.Rand   //私密随机数
}

func NewMinerInfo(id string, secert string) MinerInfo {
	var mi MinerInfo
	mi.MinerID = *groupsig.NewIDFromString(id)
	mi.SecretSeed = base.RandFromString(secert)
	return mi
}

func (mi *MinerInfo) Init(id groupsig.ID, secert base.Rand) {
	mi.MinerID = id
	mi.SecretSeed = secert
	return
}

func (mi MinerInfo) GetMinerID() groupsig.ID {
	return mi.MinerID
}

func (mi MinerInfo) GetSecret() base.Rand {
	return mi.SecretSeed
}

func (mi MinerInfo) GetDefaultSecKey() groupsig.Seckey {
	return *groupsig.NewSeckeyFromRand(mi.SecretSeed)
}

func (mi MinerInfo) GetDefaultPubKey() groupsig.Pubkey {
	return *groupsig.NewPubkeyFromSeckey(mi.GetDefaultSecKey())
}

func (mi MinerInfo) GenSecretForGroup(h common.Hash) base.Rand {
	r := base.RandFromBytes(h.Bytes())
	return mi.SecretSeed.DerivedRand(r[:])
}

//流化函数
func (mi MinerInfo) Serialize() []byte {
	buf := make([]byte, groupsig.IDLENGTH+base.RandLength)
	copy(buf[:groupsig.IDLENGTH], mi.MinerID.Serialize()[:])
	copy(buf[groupsig.IDLENGTH:], mi.SecretSeed[:])
	return buf
}

func (mi *MinerInfo) Deserialize(buf []byte) (err error) {
	id_buf := make([]byte, groupsig.IDLENGTH)
	copy(id_buf[:], buf[:groupsig.IDLENGTH])
	err = mi.MinerID.Deserialize(id_buf)
	if err != nil {
		return err
	}
	copy(mi.SecretSeed[:], buf[groupsig.IDLENGTH:])
	return
}

//成为当前铸块组共识摘要
type CastGroupSummary struct {
	PreHash     common.Hash //上一块哈希
	PreTime     time.Time   //上一块完成时间
	BlockHeight uint64      //当前铸块高度
	GroupID     groupsig.ID //当前组ID
	Castor 		groupsig.ID
	CastorPos	int32
}

//组初始化共识摘要
type ConsensusGroupInitSummary struct {
	ParentID  groupsig.ID //父亲组ID
	PrevGroupID	groupsig.ID
	Signature groupsig.Signature	//父亲组签名
	Authority uint64      //权限相关数据（父亲组赋予）
	Name      [64]byte    //父亲组取的名字
	DummyID   groupsig.ID //父亲组给的伪ID
	Members   uint64      //成员数量
	BeginTime time.Time   //初始化开始时间（必须在指定时间窗口内完成初始化）
	MemberHash	common.Hash	//成员数据哈希
	TopHeight		uint64	//当前块高
	GetReadyHeight	uint64	//准备就绪的高度,即组建成的最大高度, 超过该高度后, 组建成也无效
	BeginCastHeight	uint64	//可以开始铸块的高度
	DismissHeight	uint64	//解散的高度
	Extends   string      //带外数据
}

func GenMemberHash(mems []PubKeyInfo) common.Hash {
    ids := make([]groupsig.ID, 0)
	for _, m := range mems {
		ids = append(ids, m.ID)
	}
	return GenMemberHashByIds(ids)
}


func GenMemberHashByIds(ids []groupsig.ID) common.Hash {
	data := make([]byte, 0)
	for _, m := range ids {
		data = append(data, m.Serialize()...)
	}
	return base.Data2CommonHash(data)
}

func (gis *ConsensusGroupInitSummary) WithMemberPubs(mems []PubKeyInfo) {
    gis.Members = uint64(len(mems))
    gis.MemberHash = GenMemberHash(mems)
}

func (gis *ConsensusGroupInitSummary) WithMemberIds(mems []groupsig.ID) {
	gis.Members = uint64(len(mems))
	gis.MemberHash = GenMemberHashByIds(mems)
}

func (gis *ConsensusGroupInitSummary) CheckMemberHash(mems []PubKeyInfo) bool {
    return gis.MemberHash == GenMemberHash(mems)
}

//是否已超过允许的初始化共识时间窗口
func (gis *ConsensusGroupInitSummary) IsExpired() bool {
	if !gis.BeginTime.IsZero() && time.Since(gis.BeginTime).Seconds() <= float64(GROUP_INIT_MAX_SECONDS) {
		return false
	} else {
		return true
	}
}

func (gis *ConsensusGroupInitSummary) ReadyTimeout(height uint64) bool {
	return gis.IsExpired() && gis.GetReadyHeight <= height
}

//生成哈希
func (gis *ConsensusGroupInitSummary) GenHash() common.Hash {
	buf := gis.ParentID.Serialize()
	buf = append(buf, gis.PrevGroupID.Serialize()...)
	buf = strconv.AppendUint(buf, gis.Authority, 16)
	buf = append(buf, []byte(gis.Name[:])...)
	buf = append(buf, gis.DummyID.Serialize()...)
	buf = strconv.AppendUint(buf, gis.Members, 16)
	buf = gis.BeginTime.AppendFormat(buf, time.ANSIC)
	buf = append(buf, gis.MemberHash.Bytes()...)
	buf = strconv.AppendUint(buf, gis.GetReadyHeight, 16)
	buf = strconv.AppendUint(buf, gis.BeginCastHeight, 16)
	buf = strconv.AppendUint(buf, gis.DismissHeight, 16)
	buf = strconv.AppendUint(buf, gis.TopHeight, 16)
	if len(gis.Extends) <= 1024 {
		buf = append(buf, []byte(gis.Extends[:])...)
	} else {
		buf = append(buf, []byte(gis.Extends[:1024])...)
	}
	return base.Data2CommonHash([]byte(buf))
}

type StaticGroupSummary struct {
	GroupID  groupsig.ID               //组ID(可以由组公钥生成)
	GroupPK  groupsig.Pubkey           //组公钥
	//Members  []PubKeyInfo              //组内成员的静态信息(严格按照链上次序，全网一致，不然影响组铸块)。to do : 组成员的公钥是否有必要保存在这里？
	GIS 	ConsensusGroupInitSummary
}

func (sgs *StaticGroupSummary) GenHash() common.Hash {
	buf := sgs.GroupID.Serialize()
	buf = append(buf, sgs.GroupPK.Serialize()...)

	gisHash := sgs.GIS.GenHash()
	buf = append(buf, gisHash.Bytes()...)
	hash := base.Data2CommonHash(buf)
	return hash
}