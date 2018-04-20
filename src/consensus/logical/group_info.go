package logical

import (
	"common"
	"consensus/groupsig"
	"consensus/rand"
	"fmt"
	"math/big"
	"net"
)

type STATIC_GROUP_STATUS int

const (
	SGS_UNKNOWN      STATIC_GROUP_STATUS = iota //组状态未知
	SGS_INITING                                 //组已创建，在初始化中
	SGS_INIT_TIMEOUT                            //组初始化失败
	SGS_CASTOR                                  //合法的矿工组
	SGS_DISUSED                                 //组已废弃
)

//静态组结构（组创建成功后加入到GlobalGroups）
type StaticGroupInfo struct {
	GroupID  groupsig.ID               //组ID(可以由组公钥生成)
	GroupPK  groupsig.Pubkey           //组公钥
	Members  []PubKeyInfo              //组内成员的静态信息(严格按照链上次序，全网一致，不然影响组铸块)。to do : 组成员的公钥是否有必要保存在这里？
	mapCache map[groupsig.ID]uint32    //用ID查找成员信息(成员ID->members中的索引)
	gis      ConsensusGroupInitSummary //组的初始化凭证
}

func (sgi StaticGroupInfo) GenHash() common.Hash {
	str := sgi.GroupID.GetHexString()
	str += sgi.GroupPK.GetHexString()
	for _, v := range sgi.Members {
		str += v.Id.GetHexString()
		str += v.pk.GetHexString()
	}
	//mapCache不进哈希
	gis_hash := sgi.gis.GenHash()
	str += gis_hash.Str()
	all_hash := rand.Data2CommonHash([]byte(str))
	return all_hash
}

func (sgi StaticGroupInfo) GetPubKey() groupsig.Pubkey {
	return sgi.GroupPK
}

func (sgi *StaticGroupInfo) GetGroupStatus() STATIC_GROUP_STATUS {
	s := SGS_UNKNOWN
	if len(sgi.Members) > 0 && sgi.GroupPK.IsValid() {
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

//由父亲组的初始化消息生成SGI结构（组内和组外的节点都需要这个函数）
func NewSGIFromRawMessage(grm ConsensusGroupRawMessage) StaticGroupInfo {
	var sgi StaticGroupInfo
	sgi.gis = grm.gi
	sgi.Members = make([]PubKeyInfo, GROUP_MAX_MEMBERS)
	sgi.mapCache = make(map[groupsig.ID]uint32, GROUP_MAX_MEMBERS)
	var pki PubKeyInfo
	for _, v := range grm.Ids {
		pki.Id = v
		//pki.pk =
		sgi.Members = append(sgi.Members, pki)
		sgi.mapCache[pki.Id] = uint32(len(sgi.Members)) - 1
	}
	return sgi
}

//创建一个未经过组初始化共识的静态组结构（尚未共识生成组私钥、组公钥和组ID）
//输入：组成员ID列表，该ID为组成员的私有ID（由该成员的交易私钥->公开地址处理而来），和组共识无关
func CreateWithRawMembers(mems []PubKeyInfo) StaticGroupInfo {
	var sgi StaticGroupInfo
	sgi.Members = make([]PubKeyInfo, GROUP_MAX_MEMBERS)
	sgi.mapCache = make(map[groupsig.ID]uint32, GROUP_MAX_MEMBERS)
	for i := 0; i < len(mems); i++ {
		sgi.Members = append(sgi.Members, mems[i])
		sgi.mapCache[mems[i].Id] = uint32(len(sgi.Members)) - 1
	}
	return sgi
}

func (sgi *StaticGroupInfo) GetLen() int {
	return len(sgi.Members)
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
	for i := 0; i < len(sgi.Members); i++ {
		ids = append(ids, sgi.Members[i].Id)
	}
	return ids
}

func (sgi *StaticGroupInfo) Addmember(m PubKeyInfo) {
	if m.Id.IsValid() {
		_, ok := sgi.mapCache[m.Id]
		if !ok {
			sgi.Members = append(sgi.Members, m)
			sgi.mapCache[m.Id] = uint32(len(sgi.Members)) - 1
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
		m = sgi.Members[i]
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
	if i >= 0 && i < len(sgi.Members) {
		m = sgi.Members[i].Id
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
	//全网组的静态信息列表，用slice而不是map是为了求模定位(to do:组之间不需要求模，可以直接使用map)
	sgi      []StaticGroupInfo
	mapCache map[groupsig.ID]uint32 //用ID查找组信息
	ngg      NewGroupGenerator      //新组处理器(组外处理器)
}

func (gg *GlobalGroups) GroupInitedMessage(id GroupMinerID, ngmd NewGroupMemberData) int {
	result := gg.ngg.ReceiveData(id, ngmd)
	switch result {
	case 1: //收到组内相同消息>阈值，可上链
		//to do : 上链已初始化的组
		//to do ：从待初始化组中删除
		//to do : 是否全网广播该组的生成？广播的意义？
	case -1: //该组初始化异常，且无法恢复
		//to do : 从待初始化组中删除
	case 0:
		//继续等待下一包数据
	}
	return result
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
