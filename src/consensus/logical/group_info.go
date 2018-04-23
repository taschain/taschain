package logical

import (
	"common"
	"consensus/groupsig"
	"consensus/rand"
	"fmt"
	"math/big"
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
	members  []PubKeyInfo              //组内成员的静态信息(严格按照链上次序，全网一致，不然影响组铸块)。to do : 组成员的公钥是否有必要保存在这里？
	mapCache map[string]int            //用ID查找成员信息(成员ID->members中的索引)
	gis      ConsensusGroupInitSummary //组的初始化凭证
}

//取得某个矿工在组内的排位
func (sgi StaticGroupInfo) GetMinerPos(id groupsig.ID) int {
	pos := -1
	if v, ok := sgi.mapCache[id.GetHexString()]; ok {
		pos = v
		//双重验证
		found := false
		for k, item := range sgi.members {
			if item.GetID().GetHexString() == id.GetHexString() {
				found = true
				if pos != k {
					panic("double check failed 1.\n")
				}
				break
			}
		}
		if !found {
			panic("double check failed 2.\n")
		}
	}
	return pos
}

func (sgi StaticGroupInfo) GenHash() common.Hash {
	str := sgi.GroupID.GetHexString()
	str += sgi.GroupPK.GetHexString()
	for _, v := range sgi.members {
		str += v.ID.GetHexString()
		str += v.PK.GetHexString()
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

//由父亲组的初始化消息生成SGI结构（组内和组外的节点都需要这个函数）
func NewSGIFromRawMessage(grm ConsensusGroupRawMessage) StaticGroupInfo {
	var sgi StaticGroupInfo
	sgi.gis = grm.GI
	sgi.members = make([]PubKeyInfo, GROUP_MAX_MEMBERS)
	sgi.mapCache = make(map[string]int, GROUP_MAX_MEMBERS)
	for _, v := range grm.MEMS {
		sgi.members = append(sgi.members, v)
		sgi.mapCache[v.GetID().GetHexString()] = len(sgi.members) - 1
	}
	return sgi
}

//创建一个未经过组初始化共识的静态组结构（尚未共识生成组私钥、组公钥和组ID）
//输入：组成员ID列表，该ID为组成员的私有ID（由该成员的交易私钥->公开地址处理而来），和组共识无关
func CreateWithRawMembers(mems []PubKeyInfo) StaticGroupInfo {
	var sgi StaticGroupInfo
	sgi.members = make([]PubKeyInfo, GROUP_MAX_MEMBERS)
	sgi.mapCache = make(map[string]int, GROUP_MAX_MEMBERS)
	for i := 0; i < len(mems); i++ {
		sgi.members = append(sgi.members, mems[i])
		sgi.mapCache[mems[i].GetID().GetHexString()] = len(sgi.members) - 1
	}
	return sgi
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
		ids = append(ids, sgi.members[i].GetID())
	}
	return ids
}

func (sgi *StaticGroupInfo) Addmember(m PubKeyInfo) {
	if m.GetID().IsValid() {
		_, ok := sgi.mapCache[m.GetID().GetHexString()]
		if !ok {
			sgi.members = append(sgi.members, m)
			sgi.mapCache[m.GetID().GetHexString()] = len(sgi.members) - 1
		}
	}
}

func (sgi *StaticGroupInfo) CanGroupSign() bool {
	return sgi.GroupPK.IsValid()
}

func (sgi StaticGroupInfo) MemExist(uid groupsig.ID) bool {
	_, ok := sgi.mapCache[uid.GetHexString()]
	return ok
}

func (sgi StaticGroupInfo) GetMember(uid groupsig.ID) (m PubKeyInfo, ok bool) {
	var i int
	i, ok = sgi.mapCache[uid.GetHexString()]
	fmt.Printf("data size=%v, cache size=%v.\n", len(sgi.members), len(sgi.mapCache))
	fmt.Printf("find node(%v) = %v, local all mems=%v, gpk=%v.\n", uid.GetHexString(), ok, len(sgi.members), sgi.GroupPK.GetHexString())
	if ok {
		m = sgi.members[i]
	} else {
		i := 0
		for k, _ := range sgi.mapCache {
			fmt.Printf("---mem(%v)=%v.\n", i, k)
			i++
		}
	}
	return
}

//取得某个成员在组内的排位
func (sgi StaticGroupInfo) GetPosition(uid groupsig.ID) int32 {
	i, ok := sgi.mapCache[uid.GetHexString()]
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
		m = sgi.members[i].GetID()
	}
	return m
}

///////////////////////////////////////////////////////////////////////////////
//父亲组节点已经向外界宣布，但未完成初始化的组也保存在这个结构内。
//未完成初始化的组用独立的数据存放，不混入groups。因groups的排位影响下一个铸块组的选择。
type GlobalGroups struct {
	groups       []StaticGroupInfo
	mapCache     map[string]int             //string(ID)->索引
	ngg          NewGroupGenerator          //新组处理器(组外处理器)
	dummy_groups map[string]StaticGroupInfo //未完成初始化的组(str(DUMMY ID)->组信息)
}

func (gg *GlobalGroups) Init() {
	gg.groups = make([]StaticGroupInfo, 0)
	gg.mapCache = make(map[string]int, 0)
	gg.dummy_groups = make(map[string]StaticGroupInfo, 0)
	gg.ngg.Init(gg)
}

func (gg GlobalGroups) GetGroupSize() int {
	return len(gg.groups)
}

//增加一个合法铸块组
func (gg *GlobalGroups) AddGroup(g StaticGroupInfo) bool {
	if len(g.members) != len(g.mapCache) {
		//重构cache
		g.mapCache = make(map[string]int, len(g.members))
		for i, v := range g.members {
			if v.PK.GetHexString() == "0x0" {
				panic("GlobalGroups::AddGroup failed, group member has no pub key.")
			}
			g.mapCache[v.GetID().GetHexString()] = i
		}
	}
	fmt.Printf("begin GlobalGroups::AddGroup, id=%v, mems 1=%v, mems 2=%v...\n", g.GroupID.GetHexString(), len(g.members), len(g.mapCache))
	if _, ok := gg.mapCache[g.GroupID.GetHexString()]; !ok {
		gg.groups = append(gg.groups, g)
		gg.mapCache[g.GroupID.GetHexString()] = len(gg.groups) - 1
		return true
	} else {
		fmt.Printf("already exist this group, ignored.\n")
	}
	return false
}

func (gg *GlobalGroups) GroupInitedMessage(id GroupMinerID, ngmd NewGroupMemberData) int {
	result := gg.ngg.ReceiveData(id, ngmd)
	switch result {
	case 1: //收到组内相同消息>阈值，可上链
		//to do : 上链已初始化的组
		//to do ：从待初始化组中删除
		//to do : 是否全网广播该组的生成？广播的意义？
		//b := gg.AddGroup(ngmd.)
	case -1: //该组初始化异常，且无法恢复
		//to do : 从待初始化组中删除
	case 0:
		//继续等待下一包数据
	}
	return result
}

//取得矿工的公钥
func (gg *GlobalGroups) GetMinerPubKey(gid groupsig.ID, uid groupsig.ID) *groupsig.Pubkey {
	if index, ok := gg.mapCache[gid.GetHexString()]; ok {
		g := gg.groups[index]
		m, b := g.GetMember(uid)
		if b {
			return &m.PK
		}
	}
	return nil
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
	if i >= 0 && i < len(gg.groups) {
		g = gg.groups[i]
	} else {
		err = fmt.Errorf("out of range")
	}
	return
}

func (gg GlobalGroups) GetGroupByID(id groupsig.ID) (g StaticGroupInfo, err error) {
	index, ok := gg.mapCache[id.GetHexString()]
	if ok {
		g, err = gg.GetGroupByIndex(index)
	}
	return
}

func (gg GlobalGroups) GetGroupByDummyID(id groupsig.ID) (g StaticGroupInfo, err error) {
	if v, ok := gg.dummy_groups[id.GetHexString()]; ok {
		g = v
	} else {
		err = fmt.Errorf("out of range")
	}
	return
}

//根据上一块哈希值，确定下一块由哪个组铸块
func (gg GlobalGroups) SelectNextGroup(h common.Hash) (groupsig.ID, error) {
	var ga groupsig.ID
	value := h.Big()
	if value.BitLen() > 0 && len(gg.groups) > 0 {
		index := value.Mod(value, big.NewInt(int64(len(gg.groups))))
		ga = gg.groups[index.Uint64()].GroupID
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
