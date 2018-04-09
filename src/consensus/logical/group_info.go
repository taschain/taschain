package logical

import (
	"common"
	"consensus/groupsig"
	"fmt"
	"math/big"
	"net"
)

type PubKeyInfo struct {
	id groupsig.ID //成员全局唯一ID
	//成员全局唯一公钥（哪怕不加入组，该公钥也存在，用于接收交易）/ 组成员公钥
	//看具体的实施场景
	pk groupsig.Pubkey
}

func (p *PubKeyInfo) IsValid() bool {
	return p.id.IsValid() && p.pk.IsValid()
}

//秘密分享片段，用于生成组成员私钥（进行组签名片段）
type SecKeyInfo struct {
	id groupsig.ID     //成员全局唯一ID
	sk groupsig.Seckey //秘密片段
}

func (s *SecKeyInfo) IsValid() bool {
	return s.id.IsValid() && s.sk.IsValid()
}

//组初始化结构
type InitGroupInfo struct {
	shares []SecKeyInfo    //第一步：聚合组成员（签名）私钥
	pubs   []PubKeyInfo    //第二步：聚合组公钥
	ssk    groupsig.Seckey //第一步的输出：组成员签名私钥
	gpk    groupsig.Pubkey //第二步的输出：组公钥
}

//取得待聚合的片段数量
//share=true，秘密聚合；share=false，公钥聚合
func (igi *InitGroupInfo) GetValidCount(share bool) int {
	var count int = 0
	if share {
		for _, share := range igi.shares {
			if share.IsValid() {
				count++
			}
		}
	} else {
		for _, pub := range igi.pubs {
			if pub.IsValid() {
				count++
			}
		}
	}
	return count
}

//收到share的处理，用于聚合组签名私钥
//返回：-1异常，0正常接收，1已经接收到所有的share，可以启动聚合。
func (igi *InitGroupInfo) UpdateShare(id groupsig.ID, s groupsig.Seckey) int {
	if !id.IsValid() || !s.IsValid() {
		return -1
	}
	for i := 0; i < len(igi.shares); i++ {
		if igi.shares[i].id == id {
			if !igi.shares[i].sk.IsValid() {
				igi.shares[i].sk = s
			} else {
				if igi.shares[i].sk != s {
					panic("UpdateShare failed, sec key diff.")
					return -1
				}
			}
		}
	}
	if igi.GetValidCount(true) == GROUP_MAX_MEMBERS {
		return 1 //已收到所有用户的秘密分享
	} else {
		return 0
	}
}

//生成组签名聚合私钥
func (igi *InitGroupInfo) AggrSignSecKey() bool {
	if igi.ssk.IsValid() { //已经生成
		return true
	}
	if len(igi.pubs) == GROUP_MAX_MEMBERS {
		secs := make([]groupsig.Seckey, GROUP_MAX_MEMBERS)
		for i := 0; i < len(igi.shares); i++ {
			if igi.shares[i].IsValid() {
				secs = append(secs, igi.shares[i].sk)
			} else {
				return false
			}
		}
		igi.ssk = *groupsig.AggregateSeckeys(secs) //聚合组成员签名私钥
		if !igi.ssk.IsValid() {
			fmt.Printf("InitGroupInfo::GenSignSecKey failed, AggregateSeckeys return false.\n")
		}
		return igi.ssk.IsValid()
	}
	return false
}

//生成组公钥片段。
//前提：组成员签名私钥已经聚合完成。
func (igi *InitGroupInfo) GenGroupPubKeyPiece() *groupsig.Pubkey {
	if !igi.ssk.IsValid() { //组成员签名私钥尚未生成
		return nil
	}
	return groupsig.NewPubkeyFromSeckey(igi.ssk)
}

//收集所有组成员公钥
//返回=-1异常，=0正常接收，=1已收到所有成员的公钥片段，可以启动组公钥聚合
func (igi *InitGroupInfo) UpdateMemberPubKey(id groupsig.ID, pk groupsig.Pubkey) int {
	if !id.IsValid() || !pk.IsValid() {
		return -1
	}
	found := false
	for i := 0; i < len(igi.pubs); i++ {
		if igi.pubs[i].id == id {
			if !igi.pubs[i].pk.IsValid() {
				igi.pubs[i].pk = pk
			} else {
				if igi.pubs[i].pk != pk {
					panic("UpdateMemberPubKey failed, pub key diff.")
				}
			}
			found = true
		}
	}
	if !found {
		return -1
	} else {
		if igi.GetValidCount(false) == GROUP_MAX_MEMBERS {
			return 1 //收到所有成员的公钥片段，可以启动聚合
		} else {
			return 0
		}
	}
}

//生成组公钥
func (igi *InitGroupInfo) AggrGroupPubKey() bool {
	if igi.gpk.IsValid() {
		return true //已生成
	}
	if len(igi.pubs) == GROUP_MAX_MEMBERS {
		pieces := make([]groupsig.Pubkey, GROUP_MAX_MEMBERS)
		for i := 0; i < len(igi.pubs); i++ {
			if igi.pubs[i].IsValid() {
				pieces = append(pieces, igi.pubs[i].pk)
			} else {
				return false
			}
		}
		igi.gpk = *groupsig.AggregatePubkeys(pieces)
		if !igi.gpk.IsValid() {
			panic("InitGroupInfo::GenGroupPubKey failed, AggregatePubkeys error.")
		}
		return igi.gpk.IsValid()
	}
	return false
}

///////////////////////////////////////////////////////////////////////////////

type STATIC_GROUP_STATUS int

const (
	SGS_UNKNOWN      STATIC_GROUP_STATUS = iota //组状态未知
	SGS_INITING                                 //组已创建，在初始化中
	SGS_INIT_TIMEOUT                            //组初始化失败
	SGS_CASTOR                                  //合法的矿工组
	SGS_DISUSED                                 //组已废弃
)

//静态组结构（组创建即确定）
type StaticGroupInfo struct {
	GroupID  groupsig.ID               //组ID(可以由组公钥生成)
	GroupPK  groupsig.Pubkey           //组公钥
	members  []PubKeyInfo              //组内成员的静态信息(严格按照链上次序，全网一致，不然影响组铸块)
	mapCache map[groupsig.ID]uint32    //用ID查找成员信息(成员ID->members中的索引)
	gis      ConsensusGroupInitSummary //组的初始化凭证
}

func (sgi *StaticGroupInfo) GetGroupStatus() STATIC_GROUP_STATUS {
	s := SGS_UNKNOWN
	if len(sgi.members) > 0 && sgi.GroupPK.IsValid() {
		s = SGS_CASTOR
	} else {
		if sgi.gis.IsExpired() {
			s = SGS_INIT_TIMEOUT
		} else {
			s = SGS_INITING
		}
	}
	return s
}

//创建一个未经过组初始化共识的静态组结构（尚未共识生成组私钥、组公钥和组ID）
//输入：组成员ID列表，该ID为组成员的私有ID（由该成员的交易私钥->公开地址处理而来），和组共识无关
func CreateWithRawMembers(mems []PubKeyInfo) StaticGroupInfo {
	sgi := new(StaticGroupInfo)
	sgi.members = make([]PubKeyInfo, GROUP_MAX_MEMBERS)
	sgi.mapCache = make(map[groupsig.ID]uint32, GROUP_MAX_MEMBERS)
	for i := 0; i < len(mems); i++ {
		sgi.members = append(sgi.members, mems[i])
		sgi.mapCache[mems[i].id] = uint32(len(sgi.members)) - 1
	}
	return *sgi
}

func (sgi *StaticGroupInfo) GetLen() int {
	return len(sgi.members)
}

//组完成初始化，必须在一个组尚未初始化的时候调用有效。
//pk：初始化后的组公钥，id：初始化后生成的组ID
func (sgi *StaticGroupInfo) GroupConsensusInited(pk groupsig.Pubkey, id groupsig.ID) bool {
	if sgi.GroupID.IsValid() || sgi.GroupPK.IsValid() {
		return false
	}
	if !pk.IsValid() || !id.IsValid() {
		return false
	}
	sgi.GroupID = id
	sgi.GroupPK = pk
	return true
}

//按组内排位取得成员ID列表
func (sgi *StaticGroupInfo) GetIDSByOrder() []groupsig.ID {
	ids := make([]groupsig.ID, GROUP_MAX_MEMBERS)
	for i := 0; i < len(sgi.members); i++ {
		ids = append(ids, sgi.members[i].id)
	}
	return ids
}

func (sgi *StaticGroupInfo) Addmember(m PubKeyInfo) {
	if m.id.IsValid() {
		_, ok := sgi.mapCache[m.id]
		if !ok {
			sgi.members = append(sgi.members, m)
			sgi.mapCache[m.id] = uint32(len(sgi.members)) - 1
		}
	}
}

func (sgi *StaticGroupInfo) CanGroupSign() bool {
	return sgi.GroupPK.IsValid()
}

func (sgi StaticGroupInfo) MemExist(uid groupsig.ID) bool {
	_, ok := sgi.mapCache[uid]
	return ok
}

func (sgi StaticGroupInfo) GetMember(uid groupsig.ID) (m PubKeyInfo, result bool) {
	var i uint32
	i, result = sgi.mapCache[uid]
	if result {
		m = sgi.members[i]
	}
	return
}

//取得某个成员在组内的排位
func (sgi StaticGroupInfo) GetPosition(uid groupsig.ID) int32 {
	i, ok := sgi.mapCache[uid]
	if ok {
		return int32(i)
	} else {
		return int32(-1)
	}
}

//取得指定位置的铸块人
func (sgi StaticGroupInfo) GetCastor(i int) groupsig.ID {
	var m groupsig.ID
	if i >= 0 && i < len(sgi.members) {
		m = sgi.members[i].id
	}
	return m
}

//动态组结构（运行时变化）
type DynamicGroupInfo struct {
	members map[string]net.TCPAddr //组内成员的网络地址
}

//取得组成员网络地址
func (dgi DynamicGroupInfo) GetNetIP(ma string) string {
	addr, ok := dgi.members[ma]
	if ok {
		return addr.IP.String()
	}
	return ""
}

func (dgi DynamicGroupInfo) GetNetPort(ma string) int32 {
	addr, ok := dgi.members[ma]
	if ok {
		return int32(addr.Port)
	}
	return 0
}

///////////////////////////////////////////////////////////////////////////////
//如一个组还在初始化中，则以父亲组指定的dummy ID作为临时性group ID.
type GlobalGroups struct {
	//全网组的静态信息列表，用slice而不是map是为了求模定位
	sgi      []StaticGroupInfo
	mapCache map[groupsig.ID]uint32 //用ID查找组信息
}

//取得矿工的公钥
func (gg *GlobalGroups) GetMinerPubKey(gid groupsig.ID, uid groupsig.ID) *groupsig.Pubkey {
	if index, ok := gg.mapCache[gid]; ok {
		g := gg.sgi[index]
		m, b := g.GetMember(uid)
		if b {
			return &m.pk
		}
	}
	return nil
}

//组初始化完成后更新静态信息
//上链不在这里完成，由外部完成
func (gg *GlobalGroups) GroupInited(dummyid groupsig.ID, gid groupsig.ID, gpk groupsig.Pubkey) bool {
	//to do :
	return false
}

//检查某个用户是否某个组成员
func (gg GlobalGroups) IsGroupMember(gid groupsig.ID, uid groupsig.ID) bool {
	g, err := gg.GetGroupByID(gid)
	if err == nil {
		return g.MemExist(uid)
	}
	return false
}

//取得一个组的状态
//=-1不存在，=0正常状态，=1初始化中，=2初始化超时，=3已废弃
func (gg GlobalGroups) GetGroupStatus(gid groupsig.ID) STATIC_GROUP_STATUS {
	g, err := gg.GetGroupByID(gid)
	if err == nil {
		return g.GetGroupStatus()
	}
	return SGS_UNKNOWN
}

func (gg GlobalGroups) GetPosition(gid groupsig.ID) int32 {
	i, ok := gg.mapCache[gid]
	if ok {
		return int32(i)
	} else {
		return int32(-1)
	}
}

//由index取得组信息
func (gg GlobalGroups) GetGroupByIndex(i int) (g StaticGroupInfo, err error) {
	if i < len(gg.sgi) {
		g = gg.sgi[i]
	} else {
		err = fmt.Errorf("out of range")
	}
	return
}

func (gg GlobalGroups) GetGroupByID(id groupsig.ID) (g StaticGroupInfo, err error) {
	index, ok := gg.mapCache[id]
	if ok {
		g, err = gg.GetGroupByIndex(int(index))
	}
	return
}

//根据上一块哈希值，确定下一块由哪个组铸块
func (gg GlobalGroups) SelectNextGroup(h common.Hash) (groupsig.ID, error) {
	var ga groupsig.ID
	value := h.Big()
	if value.BitLen() > 0 && len(gg.sgi) > 0 {
		index := value.Mod(value, big.NewInt(int64(len(gg.sgi))))
		ga = gg.sgi[index.Uint64()].GroupID
		return ga, nil
	} else {
		return ga, fmt.Errorf("SelectNextGroup failed, arg error.")
	}
}

//取得当前铸块组信息
//pre_hash : 上一个铸块哈希
func (gg GlobalGroups) GetCastGroup(pre_h common.Hash) (g StaticGroupInfo) {
	gid, e := gg.SelectNextGroup(pre_h)
	if e == nil {
		g, e = gg.GetGroupByID(gid)
	}
	return
}

//判断pub_key是否为合法铸块组的公钥
//h：上一个铸块的哈希
func (gg GlobalGroups) IsCastGroup(pre_h common.Hash, pub_key groupsig.Pubkey) (result bool) {
	g := gg.GetCastGroup(pre_h)
	result = g.GroupPK == pub_key
	return
}
