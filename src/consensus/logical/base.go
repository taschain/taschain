package logical

import (
	"common"
	"consensus/groupsig"
	"consensus/rand"
	"core"
	"hash"
	"math"
	"strconv"
	"time"
)

const NORMAL_FAILED int = -1
const NORMAL_SUCCESS int = 0

//共识名词定义：
//铸块组：轮到铸当前高度块的组。
//铸块时间窗口：给该组完成当前高度铸块的时间。如无法完成，则当前高度为空块，由上个铸块的哈希的哈希决定铸下一个块的组。
//铸块槽：组内某个时间窗口的出块和验证。一个铸块槽如果能走到最后，则会成功产出该组的一个铸块。一个铸块组会有多个铸块槽。
//铸块槽权重：组内第一个铸块槽权重最高=1，第二个=2，第三个=4...。铸块槽权重<=0为无效。权重在>0的情况下，越小越好（有特权）。
//出块时间窗口：一个铸块槽的完成时间。如当前铸块槽在出块时间窗口内无法组内达成共识（上链，组外广播），则当前槽的King的组内排位后继成员成为下一个铸块槽的King，启动下一个铸块槽。
//出块人（King）：铸块槽内的出块节点，一个铸块槽只有一个出块人。
//见证人（Witness）：铸块槽内的验证节点，一个铸块槽有多个见证人。见证人包括组内除了出块人之外的所有成员。
//出块消息：在一个铸块槽内，出块人广播到组内的出块消息（类似比特币的铸块）。
//验块消息：在一个铸块槽内，见证人广播到组内的验证消息（对出块人出块消息的验证）。出块人的出块消息同时也是他的验块消息。
//合法铸块：就某个铸块槽组内达成一致，完成上链和组外广播。
//最终铸块：权重=1的铸块槽完成合法铸块；或当期高度铸块时间窗口内的组内权重最小的合法铸块。
//插槽替换规则（插槽可以容纳铸块槽，如插槽容量=5，则同时最多能容纳5个铸块槽）：
//1. 每过一个铸块槽时间窗口，如组内无法对上个铸块槽达成共识（完成组签名，向组外宣布铸块完成），则允许新启动一个铸块槽。新槽的King为上一个槽King的组内排位后继。
//2. 一个铸块高度在同一时间内可能会有多个铸块槽同时运行，如插槽未满，则所有满足规则1的出块消息或验块消息都允许新生成一个铸块槽。
//3. 如插槽已满，则时间窗口更早的铸块槽替换时间窗口较晚的铸块槽。
//4. 如某个铸块槽已经完成该高度的铸块（上链，组外广播），则只允许时间窗口更早的铸块槽更新该高度的铸块（上链，组外广播）。
//组内第一个KING的QN值=0。
/*
bls曲线使用情况：
使用：CurveFP382_1曲线，初始化参数枚举值=1.
ID长度（即地址）：48*8=384位。底层同私钥结构。
私钥长度：48*8=384位。底层结构Fr。
公钥长度：96*8=768位。底层结构G2。
签名长度：48*8=384位。底层结构G1。
*/
const CONSENSUS_VERSION = 1        //共识版本号
const SSSS_THRESHOLD = 51          //1-100
const GROUP_MAX_MEMBERS = 5        //一个组最大的成员数量
const GROUP_INIT_MAX_SECONDS = 600 //10分钟内完成初始化，否则该组失败。不再有初始化机会。
const MAX_UNKNOWN_BLOCKS = 5       //内存保存最大不能上链的未来块（中间块没有收到）
const MAX_SYNC_CASTORS = 3         //最多同时支持几个铸块验证
const INVALID_QN = -1              //无效的队列序号
//const GROUP_MIN_WITNESSES = GROUP_MAX_MEMBERS * SSSS_THRESHOLD / 100 //阈值绝对值
const TIMER_INTEVAL_SECONDS time.Duration = time.Second * 2          //定时器间隔
const MAX_GROUP_BLOCK_TIME int32 = 10                                //组铸块最大允许时间=10s
const MAX_USER_CAST_TIME int32 = 2                                   //个人出块最大允许时间=2s
const MAX_QN int32 = (MAX_GROUP_BLOCK_TIME - 1) / MAX_USER_CAST_TIME //组内能出的最大QN值

//取得门限值
func GetGroupK() int {
	return int(math.Ceil(float64(GROUP_MAX_MEMBERS*SSSS_THRESHOLD) / 100))
}

//矿工ID信息
type GroupMinerID struct {
	gid groupsig.ID //组ID
	uid groupsig.ID //成员ID
}

func (id GroupMinerID) IsValid() bool {
	return id.gid.IsValid() && id.uid.IsValid()
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
	var sd SignData
	sd.DataHash = h
	sd.DataSign = groupsig.Sign(sk, h.Bytes())
	sd.SignMember = id
	return sd
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
	ID groupsig.ID
	PK groupsig.Pubkey
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
	SecretSeed rand.Rand   //私密随机数
}

func NewMinerInfo(id string, secert string) MinerInfo {
	var mi MinerInfo
	mi.MinerID = *groupsig.NewIDFromString(id)
	mi.SecretSeed = rand.RandFromString(secert)
	return mi
}

func (mi *MinerInfo) Init(id groupsig.ID, secert rand.Rand) {
	mi.MinerID = id
	mi.SecretSeed = secert
	return
}

func (mi MinerInfo) GetMinerID() groupsig.ID {
	return mi.MinerID
}

func (mi MinerInfo) GetSecret() rand.Rand {
	return mi.SecretSeed
}

func (mi MinerInfo) GetDefaultSecKey() groupsig.Seckey {
	return *groupsig.NewSeckeyFromRand(mi.SecretSeed)
}

func (mi MinerInfo) GetDefaultPubKey() groupsig.Pubkey {
	return *groupsig.NewPubkeyFromSeckey(mi.GetDefaultSecKey())
}

func (mi MinerInfo) GenSecretForGroup(h common.Hash) rand.Rand {
	r := rand.RandFromBytes(h.Bytes())
	return mi.SecretSeed.DerivedRand(r[:])
}

//流化函数
func (mi MinerInfo) Serialize() []byte {
	buf := make([]byte, groupsig.IDLENGTH+rand.RandLength)
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
}

//铸块共识摘要
type ConsensusBlockSummary struct {
	Castor      groupsig.ID //铸块人
	DataHash    common.Hash //待共识的区块头哈希
	QueueNumber int64       //出块序号
	CastTime    time.Time   //铸块时间（铸块人的出块时间）
}

//根据区块头生成铸块共识摘要
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

//生成测试数据
func genDummyGIS(parent MinerInfo, group_name string) ConsensusGroupInitSummary {
	var gis ConsensusGroupInitSummary
	gis.ParentID = parent.GetMinerID()
	gis.DummyID = *groupsig.NewIDFromString(group_name)
	gis.Authority = 777
	copy(gis.Name[:], group_name[:])
	gis.BeginTime = time.Now()
	if !gis.ParentID.IsValid() || !gis.DummyID.IsValid() {
		panic("create group init summary failed")
	}
	return gis
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
func (gis *ConsensusGroupInitSummary) GenHash() common.Hash {
	buf := gis.ParentID.GetHexString()
	buf += strconv.FormatUint(gis.Authority, 16)
	buf += string(gis.Name[:])
	buf += gis.DummyID.GetHexString()
	buf += gis.BeginTime.Format(time.ANSIC)
	return rand.Data2CommonHash([]byte(buf))
}
