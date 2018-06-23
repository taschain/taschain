package logical

import (
	"common"
	"consensus/groupsig"
	"consensus/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
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
	MapCache map[string]int            //用ID查找成员信息(成员ID->members中的索引)
	GIS      ConsensusGroupInitSummary //组的初始化凭证
	BeginHeight uint64             	   //组开始参与铸块的高度
}

//取得某个矿工在组内的排位
func (sgi StaticGroupInfo) GetMinerPos(id groupsig.ID) int {
	pos := -1
	if v, ok := sgi.MapCache[id.GetHexString()]; ok {
		pos = v
		//双重验证
		found := false
		for k, item := range sgi.Members {
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
	for _, v := range sgi.Members {
		str += v.ID.GetHexString()
		str += v.PK.GetHexString()
	}
	//mapCache不进哈希
	gis_hash := sgi.GIS.GenHash()
	str += gis_hash.Str()
	str += strconv.FormatUint(sgi.BeginHeight,16)
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
		if sgi.GIS.IsExpired() {
			s = SGS_INIT_TIMEOUT
		} else {
			s = SGS_INITING
		}
	}
	return s
}

//由父亲组的初始化消息生成SGI结构（组内和组外的节点都需要这个函数）
func NewSGIFromRawMessage(grm *ConsensusGroupRawMessage) *StaticGroupInfo {
	sgi := &StaticGroupInfo{
		GIS: grm.GI,
		Members: make([]PubKeyInfo, 0),
		MapCache: make(map[string]int),
	}
	for _, v := range grm.MEMS {
		sgi.Members = append(sgi.Members, v)
		sgi.MapCache[v.GetID().GetHexString()] = len(sgi.Members) - 1
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
		ids = append(ids, sgi.Members[i].GetID())
	}
	return ids
}

func (sgi *StaticGroupInfo) addMember(m *PubKeyInfo) {
	if m.GetID().IsValid() {
		_, ok := sgi.MapCache[m.GetID().GetHexString()]
		if !ok {
			sgi.Members = append(sgi.Members, *m)
			sgi.MapCache[m.GetID().GetHexString()] = len(sgi.Members) - 1
		}
	}
}

func (sgi *StaticGroupInfo) CanGroupSign() bool {
	return sgi.GroupPK.IsValid()
}

func (sgi StaticGroupInfo) MemExist(uid groupsig.ID) bool {
	_, ok := sgi.MapCache[uid.GetHexString()]
	return ok
}

//ok:是否组内成员
//m:组内成员矿工公钥
func (sgi StaticGroupInfo) GetMember(uid groupsig.ID) (m PubKeyInfo, ok bool) {
	var i int
	i, ok = sgi.MapCache[uid.GetHexString()]
	fmt.Printf("data size=%v, cache size=%v.\n", len(sgi.Members), len(sgi.MapCache))
	fmt.Printf("find node(%v) = %v, local all mems=%v, gpk=%v.\n", uid.GetHexString(), ok, len(sgi.Members), sgi.GroupPK.GetHexString())
	if ok {
		m = sgi.Members[i]
	} else {
		i := 0
		for k, _ := range sgi.MapCache {
			fmt.Printf("---mem(%v)=%v.\n", i, k)
			i++
		}
	}
	return
}

//取得某个成员在组内的排位
func (sgi StaticGroupInfo) GetPosition(uid groupsig.ID) int32 {
	i, ok := sgi.MapCache[uid.GetHexString()]
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
		m = sgi.Members[i].GetID()
	}
	return m
}

func (sgi *StaticGroupInfo) CastQualified(height uint64) bool {
	return sgi.BeginHeight <= height
}
///////////////////////////////////////////////////////////////////////////////
//当前节点参与的铸块组（已初始化完成）
type JoinedGroup struct {
	GroupID groupsig.ID          //组ID
	SeedKey groupsig.Seckey      //（组相关性的）私密私钥
	SignKey groupsig.Seckey      //矿工签名私钥
	GroupPK groupsig.Pubkey      //组公钥（backup,可以从全局组上拿取）
	Members groupsig.PubkeyMapID //组成员签名公钥
	GroupSec GroupSecret
}

func (jg *JoinedGroup) Init() {
	jg.Members = make(groupsig.PubkeyMapID, 0)
}

//取得组内某个成员的签名公钥
func (jg JoinedGroup) GetMemSignPK(mid groupsig.ID) groupsig.Pubkey {
	return jg.Members[mid.GetHexString()]
}

func (jg *JoinedGroup) setGroupSecretHeight(height uint64)  {
    jg.GroupSec.effectHeight = height
}

///////////////////////////////////////////////////////////////////////////////
//父亲组节点已经向外界宣布，但未完成初始化的组也保存在这个结构内。
//未完成初始化的组用独立的数据存放，不混入groups。因groups的排位影响下一个铸块组的选择。
type GlobalGroups struct {
	groups      []StaticGroupInfo
	mapCache    map[string]int             //string(ID)->索引
	ngg         NewGroupGenerator          //新组处理器(组外处理器)
	//dummyGroups map[string]StaticGroupInfo //未完成初始化的组(str(DUMMY ID)->组信息)
}

func (gg *GlobalGroups) Load() bool {
	fmt.Printf("begin GlobalGroups::Load, gc=%v, mcc=%v, dgc=%v...\n", len(gg.groups), len(gg.mapCache), 1)
	cc := common.GlobalConf.GetSectionManager("consensus")
	str := cc.GetString("GLOBAL_GROUPS", "")
	if len(str) == 0 {
		return false
	}
	fmt.Printf("gg groups unmarshal str=%v.\n", str)
	gg.groups = make([]StaticGroupInfo, 0)
	var buf = []byte(str)
	err := json.Unmarshal(buf, &gg.groups)
	if err != nil {
		fmt.Println("error:", err)
		panic("GlobalGroups::Load Unmarshal failed.")
	}
	fmt.Printf("after Ummarshal, group_count=%v.\n", len(gg.groups))
	gg.mapCache = make(map[string]int, 0)
	for k, v := range gg.groups {
		fmt.Printf("---static group: gid=%v, gpk=%v, mems=%v.\n", GetIDPrefix(v.GroupID), GetPubKeyPrefix(v.GroupPK), len(v.Members))
		gg.mapCache[v.GroupID.GetHexString()] = k
	}

	fmt.Printf("end GlobalGroups::Load, gc=%v, mcc=%v, dgc=%v.\n", len(gg.groups), len(gg.mapCache), 0)
	return true
}

func (gg GlobalGroups) Save() {
	fmt.Printf("begin GlobalGroups::Save, group_count=%v...\n", len(gg.groups))
	for _, v := range gg.groups {
		fmt.Printf("---static group: gid=%v, gpk=%v, mems=%v.\n", GetIDPrefix(v.GroupID), GetPubKeyPrefix(v.GroupPK), len(v.Members))
	}

	b, err := json.Marshal(gg.groups)
	if err != nil {
		fmt.Println("error:", err)
		panic("GlobalGroups::Save Marshal failed.")
	}

	str := string(b[:])
	fmt.Printf("gg groups marshal str=%v.\n", str)
	cc := common.GlobalConf.GetSectionManager("consensus")
	cc.SetString("GLOBAL_GROUPS", str)
	fmt.Printf("end GlobalGroups::Save.\n")
	return
}

func (gg *GlobalGroups) Init() {
	gg.groups = make([]StaticGroupInfo, 0)
	gg.mapCache = make(map[string]int, 0)
	//gg.dummyGroups = make(map[string]StaticGroupInfo, 0)
	gg.ngg.Init(gg)
}

func (gg GlobalGroups) GetGroupSize() int {
	return len(gg.groups)
}

//func (gg *GlobalGroups) AddDummyGroup(g *StaticGroupInfo) bool {
//	var add bool
//	fmt.Printf("gg AddDummyGroup, dummy_id=%v, exist_dummy_groups=%v.\n", GetIDPrefix(g.GIS.DummyID), len(gg.dummyGroups))
//	if _, ok := gg.dummyGroups[g.GIS.DummyID.GetHexString()]; !ok {
//		gg.dummyGroups[g.GIS.DummyID.GetHexString()] = *g
//		add = true
//	} else {
//		fmt.Printf("already exist this dummy group in gg.\n")
//	}
//	return add
//}

//增加一个合法铸块组
func (gg *GlobalGroups) AddGroup(g StaticGroupInfo) bool {
	if len(g.Members) != len(g.MapCache) {
		//重构cache
		g.MapCache = make(map[string]int, len(g.Members))
		for i, v := range g.Members {
			if v.PK.GetHexString() == "0x0" {
				panic("GlobalGroups::AddGroup failed, group member has no pub key.")
			}
			g.MapCache[v.GetID().GetHexString()] = i
		}
	}
	fmt.Printf("begin GlobalGroups::AddGroup, id=%v, mems 1=%v, mems 2=%v...\n", GetIDPrefix(g.GroupID), len(g.Members), len(g.MapCache))
	if idx, ok := gg.mapCache[g.GroupID.GetHexString()]; !ok {
		gg.groups = append(gg.groups, g)
		gg.mapCache[g.GroupID.GetHexString()] = len(gg.groups) - 1
		fmt.Printf("*****Group(%v) BeginHeight(%v)*****\n", GetIDPrefix(g.GroupID),g.BeginHeight)
		return true
	} else {
		if gg.groups[idx].BeginHeight < g.BeginHeight {
			gg.groups[idx].BeginHeight = g.BeginHeight
			fmt.Printf("Group(%v) BeginHeight change from (%v) to (%v)\n", GetIDPrefix(g.GroupID),gg.groups[idx].BeginHeight,g.BeginHeight)
		} else {
			fmt.Printf("already exist this group, ignored.\n")
		}

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

//func (gg GlobalGroups) GetGroupByDummyID(id groupsig.ID) (g StaticGroupInfo, err error) {
//	fmt.Printf("gg GetGroupByDummyID, dummy_id=%v, exist_dummy_groups=%v.\n", GetIDPrefix(id), len(gg.dummyGroups))
//	if v, ok := gg.dummyGroups[id.GetHexString()]; ok {
//		g = v
//	} else {
//		err = fmt.Errorf("out of range")
//	}
//	return
//}

//根据上一块哈希值，确定下一块由哪个组铸块
func (gg GlobalGroups) SelectNextGroup(h common.Hash, height uint64) (groupsig.ID, error) {
	var ga groupsig.ID
	value := h.Big()
	var vgroups = make([]int,0)
	for i := 0; i<gg.GetGroupSize(); i++ {
		if gg.groups[i].CastQualified(height) {
			vgroups = append(vgroups, i)
		}
	}
	if value.BitLen() > 0 && len(vgroups) > 0 {
		index := value.Mod(value, big.NewInt(int64(len(vgroups))))
		ga = gg.groups[vgroups[index.Uint64()]].GroupID
		return ga, nil
	} else {
		return ga, fmt.Errorf("selectNextGroup failed, arg error")
	}
}

//取得当前铸块组信息
//pre_hash : 上一个铸块哈希
func (gg GlobalGroups) GetCastGroup(preHS common.Hash,height uint64) (g StaticGroupInfo) {
	gid, e := gg.SelectNextGroup(preHS, height)
	if e == nil {
		g, e = gg.GetGroupByID(gid)
	}
	return
}

//判断pub_key是否为合法铸块组的公钥
//h：上一个铸块的哈希
func (gg GlobalGroups) IsCastGroup(pre_h common.Hash, pub_key groupsig.Pubkey, height uint64) (result bool) {
	g := gg.GetCastGroup(pre_h, height)
	result = g.GroupPK == pub_key
	return
}
