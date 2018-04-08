package logical

import (
	"hash"
	//"common"
	"consensus/groupsig"
	"consensus/rand"
)

//新组的上链处理（全网节点需要处理）
//组的索引ID为DUMMY ID。
//待共识的数据由链上获取(公信力)，不由消息获取。
//消息提供4样东西，成员ID，共识数据哈希，组公钥，组ID。
type NewGroupMemberData struct {
	h   hash.Hash
	gpk groupsig.Pubkey
	gid groupsig.ID
}

type NewGroupChained struct {
	sgi  StaticGroupInfo
	mems map[groupsig.ID]NewGroupMemberData
}

//新组生成器
type NewGroupGenerator struct {
	groups map[groupsig.ID]NewGroupChained
}

///////////////////////////////////////////////////////////////////////////////
type GROUP_INIT_STATUS int

const (
	GIS_RAW    GROUP_INIT_STATUS = iota //组处于原始状态（知道有哪些人是一组的，但是组公钥和ID尚未生成）
	GIS_SHARED                          //当前节点已经生成秘密分享片段
	GIS_INITED                          //组公钥和ID已生成，可以进行铸块
)

//组共识上下文
//判断一个消息是否合法，在外层验证
//判断一个消息是否来自组内，由GroupContext验证
type GroupContext struct {
	igi InitGroupInfo //里面的公钥是组签名聚合公钥
	is  GROUP_INIT_STATUS
	//to do : 后面三个字段合并为一个结构
	pid  groupsig.ID //父亲组ID
	auth uint64
	name string
	sgi  StaticGroupInfo //组信息(里面的ID和公钥是组成员个人ID和公钥！)
}

func (gc *GroupContext) MemExist(id groupsig.ID) bool {
	return gc.sgi.MemExist(id)
}

func CreateGCByMems(mems []MemberInfo) *GroupContext {
	gc := new(GroupContext)
	gc.sgi = CreateWithRawMembers(mems)
	gc.is = GIS_RAW
	return gc
}

//收到RAW消息
func (gc *GroupContext) RawMeesage(grm ConsensusGroupRawMessage) bool {
	if !gc.sgi.MemExist(grm.si.SignMember) { //发消息的非组内成员
		return false //忽略该消息
	}
	if gc.is == GIS_RAW {
		gc.pid = grm.gi.ParentID
		gc.auth = grm.gi.Authority
		gc.name = string(grm.gi.Name[:])
		return true
	} else {
		//已经处于SHARED态或INITED态
		return false
	}
	return false
}

//收到一片秘密分享消息
//返回-1为异常，返回0为正常接收，返回1为已聚合出组成员私钥（用于签名）
func (gc *GroupContext) PieceMessage(spm ConsensusSharePieceMessage) int {
	if !gc.sgi.MemExist(spm.si.SignMember) { //非组内成员
		return -1
	}
	var piece groupsig.Seckey
	//to do : 吕博数据解密
	result := gc.igi.UpdateShare(spm.si.SignMember, piece)
	if result == 1 { //聚合成功

	}
	return result
}

//从已聚合的组成员签名私钥萃取出对应的公钥
func (gc *GroupContext) GetPiecePubKey() groupsig.Pubkey {
	var pk groupsig.Pubkey
	if gc.igi.ssk.IsValid() { //已经聚合出了组签名私钥
		pk = *groupsig.NewPubkeyFromSeckey(gc.igi.ssk)
	}
	return pk
}

//生成某个成员针对所有组内成员的秘密分享（私钥形式）
func (gc *GroupContext) GenSharePieces() []SecKeyInfo {
	var shares []SecKeyInfo
	if gc.sgi.GetLen() > 0 && gc.is == GIS_RAW {
		shares = make([]SecKeyInfo, gc.sgi.GetLen())

		master_seckeys := make([]groupsig.Seckey, gc.sgi.GetLen())
		seed := rand.NewRand() //每个组成员自己生成的随机数

		for i := 0; i < gc.sgi.GetLen(); i++ {
			master_seckeys[i] = *groupsig.NewSeckeyFromRand(seed.Deri(i)) //生成master私钥数组（bls库函数）
		}

		for i := 0; i < gc.sgi.GetLen(); i++ {
			var piece SecKeyInfo
			piece.id = gc.sgi.GetCastor(i)
			piece.sk = *groupsig.ShareSeckey(master_seckeys, piece.id) //对每个组成员生成秘密分享
			shares[i] = piece
		}
		gc.is = GIS_SHARED
	}
	return shares
}

//取得组信息（ID和公钥）。必须在已经完成组公钥的聚合后有效。
func (gc *GroupContext) GetGroupInfo() (gid groupsig.ID, gpk groupsig.Pubkey) {
	if gc.igi.gpk.IsValid() { //已经聚合出组公钥
		gpk = gc.igi.gpk
		gid = *groupsig.NewIDFromPubkey(gpk)
	}
	return
}

//收到一片组公钥片段
//返回-1为异常，返回0为正常接收，返回1为已聚合出组公钥和组ID
func (gc *GroupContext) PiecePubKey(ppm ConsensusPubKeyPieceMessage) int {
	if gc.is == GIS_INITED { //已经初始化完成
		return 1
	}
	if gc.is != GIS_SHARED {
		panic("GroupContext::PiecePubKey failed, group status error.")
	}
	result := gc.igi.UpdateMemberPubKey(ppm.si.SignMember, ppm.pk)
	if result == 1 { //可以聚合组公钥
		b := gc.igi.AggrGroupPubKey()
		if b {
			gc.is = GIS_INITED //组初始化完成
		} else {
			panic("GroupContext::PiecePubKey failed, GenGroupPubKey error.")
		}
	}
	return result
}
