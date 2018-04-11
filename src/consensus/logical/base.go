package logical

import (
	"common"
	"core"
	"time"
	"hash"
	"strconv"
	"consensus/groupsig"
	"consensus/rand"
)

const NORMAL_FAILED int = -1
const NORMAL_SUCCESS int = 0

//矿工ID信息
type MinerID struct {
	gid groupsig.ID //组ID
	uid groupsig.ID //成员ID
}

func (id MinerID) IsValid() bool {
	return id.gid.IsValid() && id.uid.IsValid()
}

//数据签名结构
type SignData struct {
	DataHash   common.Hash        //哈希值
	DataSign   groupsig.Signature //签名
	SignMember groupsig.ID        //用户ID或组ID，看消息类型
}

func (sd SignData) GetID() groupsig.ID {
	return sd.SignMember
}

//用pk验证签名，验证通过返回true，否则false。
func (sd SignData) VerifySign(pk groupsig.Pubkey) bool {
	b := pk.IsValid()
	if b {
		b = groupsig.VerifySig(pk, sd.DataHash.Bytes(), sd.DataSign)
	}
	return b
}

//是否已有签名数据
func (sd SignData) HasSign() bool {
	return sd.DataSign.IsValid() && sd.SignMember.IsValid()
}

//id->公钥对
type PubKeyInfo struct {
	id groupsig.ID //矿工ID
	pk groupsig.Pubkey	
}

func (p PubKeyInfo) IsValid() bool {
	return p.id.IsValid() && p.pk.IsValid()
}

//id->私钥对
type SecKeyInfo struct {
	id groupsig.ID     //矿工ID
	sk groupsig.Seckey
}

func (s SecKeyInfo) IsValid() bool {
	return s.id.IsValid() && s.sk.IsValid()
}

//铸块共识摘要
type ConsensusBlockSummary struct {
	Castor      groupsig.ID        //铸块人
	DataHash    common.Hash        //待共识的区块头哈希
	QueueNumber int64              //出块序号
	CastTime    time.Time          //铸块时间（铸块人的出块时间）
}

func GenConsensusSummary(bh core.BlockHeader) ConsensusBlockSummary {
	var cs ConsensusBlockSummary
	if cs.Castor.Deserialize(bh.Castor) != nil {
		panic("ID Deserialize failed.")
	}
	cs.DataHash = bh.GenHash()
	cs.QueueNumber = int64(bh.QueueNumber)
	cs.CastTime = bh.CurTime
	return cs
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

//区块摘要
type BlockSummary struct {
	HeaderHash hash.Hash
	BlockHeight uint
	CastTime	time.Time
}