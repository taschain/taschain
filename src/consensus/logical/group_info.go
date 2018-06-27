package logical

import (
	"common"
	"consensus/groupsig"
	"fmt"
	"math/big"
	"sync"
	"sort"
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
	GroupID       groupsig.ID               //组ID(可以由组公钥生成)
	GroupPK       groupsig.Pubkey           //组公钥
	Members       []PubKeyInfo              //组内成员的静态信息(严格按照链上次序，全网一致，不然影响组铸块)。to do : 组成员的公钥是否有必要保存在这里？
	MemIndex      map[string]int            //用ID查找成员信息(成员ID->members中的索引)
	GIS           ConsensusGroupInitSummary //组的初始化凭证
	BeginHeight   uint64                    //组开始参与铸块的高度
	DismissHeight uint64                    //组解散的高度
	ParentId      groupsig.ID
	Signature     groupsig.Signature
	Authority     uint64      //权限相关数据（父亲组赋予）
	Name          string    //父亲组取的名字
	Extends       string      //带外数据
}

func NewSGIFromStaticGroupSummary(summary *StaticGroupSummary, group *InitingGroup) *StaticGroupInfo {
	sgi := &StaticGroupInfo{
		GroupID:       summary.GroupID,
		GroupPK:       summary.GroupPK,
		GIS:           summary.GIS,
		Members:       group.mems,
		MemIndex:      make(map[string]int),
		BeginHeight:   summary.GIS.BeginCastHeight,
		DismissHeight: summary.GIS.DismissHeight,
		ParentId:      summary.GIS.ParentID,
		Signature:     summary.GIS.Signature,
		Authority:     summary.GIS.Authority,
		Name:          string(summary.GIS.Name[:]),
		Extends:       summary.GIS.Extends,
	}
	for index, mem := range group.mems {
		sgi.MemIndex[mem.ID.GetHexString()] = index
	}
	return sgi
}

//取得某个矿工在组内的排位
func (sgi StaticGroupInfo) GetMinerPos(id groupsig.ID) int {
	pos := -1
	if v, ok := sgi.MemIndex[id.GetHexString()]; ok {
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
		GIS:      grm.GI,
		Members:  make([]PubKeyInfo, 0),
		MemIndex: make(map[string]int),
	}
	for _, v := range grm.MEMS {
		sgi.Members = append(sgi.Members, v)
		sgi.MemIndex[v.GetID().GetHexString()] = len(sgi.Members) - 1
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

func (sgi *StaticGroupInfo) addMember(m *PubKeyInfo) {
	if m.GetID().IsValid() {
		_, ok := sgi.MemIndex[m.GetID().GetHexString()]
		if !ok {
			sgi.Members = append(sgi.Members, *m)
			sgi.MemIndex[m.GetID().GetHexString()] = len(sgi.Members) - 1
		}
	}
}

func (sgi *StaticGroupInfo) CanGroupSign() bool {
	return sgi.GroupPK.IsValid()
}

func (sgi StaticGroupInfo) MemExist(uid groupsig.ID) bool {
	_, ok := sgi.MemIndex[uid.GetHexString()]
	return ok
}

//ok:是否组内成员
//m:组内成员矿工公钥
func (sgi StaticGroupInfo) GetMember(uid groupsig.ID) (m PubKeyInfo, ok bool) {
	var i int
	i, ok = sgi.MemIndex[uid.GetHexString()]
	fmt.Printf("data size=%v, cache size=%v.\n", len(sgi.Members), len(sgi.MemIndex))
	fmt.Printf("find node(%v) = %v, local all mems=%v, gpk=%v.\n", uid.GetHexString(), ok, len(sgi.Members), sgi.GroupPK.GetHexString())
	if ok {
		m = sgi.Members[i]
	} else {
		i := 0
		for k, _ := range sgi.MemIndex {
			fmt.Printf("---mem(%v)=%v.\n", i, k)
			i++
		}
	}
	return
}

//取得某个成员在组内的排位
func (sgi StaticGroupInfo) GetPosition(uid groupsig.ID) int32 {
	i, ok := sgi.MemIndex[uid.GetHexString()]
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
	return sgi.BeginHeight <= height && height < sgi.DismissHeight
}

//是否已解散
func (sgi *StaticGroupInfo) Dismissed(height uint64) bool {
	return height >= sgi.DismissHeight
}

func (sgi *StaticGroupInfo) GetReadyTimeout(height uint64) bool {
    return sgi.GIS.ReadyTimeout(height)
}


///////////////////////////////////////////////////////////////////////////////
//父亲组节点已经向外界宣布，但未完成初始化的组也保存在这个结构内。
//未完成初始化的组用独立的数据存放，不混入groups。因groups的排位影响下一个铸块组的选择。
type GlobalGroups struct {
	groups []*StaticGroupInfo
	gIndex map[string]int     //string(ID)->索引
	ngg    *NewGroupGenerator //新组处理器(组外处理器)
	lock sync.RWMutex
}

func NewGlobalGroups() *GlobalGroups {
	return &GlobalGroups{
		groups: make([]*StaticGroupInfo, 0),
		gIndex: make(map[string]int),
		ngg: CreateNewGroupGenerator(),
	}
}

func (gg *GlobalGroups) GetGroupSize() int {
	gg.lock.RLock()
	defer gg.lock.RUnlock()
	return len(gg.groups)
}

func (gg *GlobalGroups) AddInitingGroup(g *InitingGroup) bool {
    return gg.ngg.addInitingGroup(g)
}

func (gg *GlobalGroups) removeInitingGroup(dummyId groupsig.ID) {
	gg.ngg.removeInitingGroup(dummyId)
}

//增加一个合法铸块组
func (gg *GlobalGroups) AddStaticGroup(g *StaticGroupInfo) bool {
	gg.lock.Lock()
	defer gg.lock.Unlock()

	if len(g.Members) != len(g.MemIndex) {
		//重构cache
		g.MemIndex = make(map[string]int, len(g.Members))
		for i, v := range g.Members {
			if !v.PK.IsValid() {
				panic("GlobalGroups::AddStaticGroup failed, group member has no pub key.")
			}
			g.MemIndex[v.GetID().GetHexString()] = i
		}
	}
	fmt.Printf("begin GlobalGroups::AddStaticGroup, id=%v, mems 1=%v, mems 2=%v...\n", GetIDPrefix(g.GroupID), len(g.Members), len(g.MemIndex))
	if idx, ok := gg.gIndex[g.GroupID.GetHexString()]; !ok {
		gg.groups = append(gg.groups, g)
		gg.gIndex[g.GroupID.GetHexString()] = len(gg.groups) - 1
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

func (gg *GlobalGroups) GroupInitedMessage(sgs *StaticGroupSummary, sender groupsig.ID, height uint64) int32 {
	return gg.ngg.ReceiveData(sgs, sender, height)
}

//检查某个用户是否某个组成员
func (gg *GlobalGroups) IsGroupMember(gid groupsig.ID, uid groupsig.ID) bool {
	g, err := gg.GetGroupByID(gid)
	if err == nil {
		return g.MemExist(uid)
	}
	return false
}

func (gg *GlobalGroups) getGroupByIndex(i int) (g *StaticGroupInfo, err error) {
	if i >= 0 && i < len(gg.groups) {
		g = gg.groups[i]
	} else {
		err = fmt.Errorf("out of range")
	}
	return
}

func (gg *GlobalGroups) GetGroupByID(id groupsig.ID) (g *StaticGroupInfo, err error) {
	gg.lock.RLock()
	defer gg.lock.RUnlock()

	index, ok := gg.gIndex[id.GetHexString()]
	if ok {
		g, err = gg.getGroupByIndex(index)
	}
	return
}

//根据上一块哈希值，确定下一块由哪个组铸块
func (gg *GlobalGroups) SelectNextGroup(h common.Hash, height uint64) (groupsig.ID, error) {
	qualifiedGS := gg.GetCastQualifiedGroups(height)

	var ga groupsig.ID
	value := h.Big()

	if value.BitLen() > 0 && len(qualifiedGS) > 0 {
		index := value.Mod(value, big.NewInt(int64(len(qualifiedGS))))
		ga = qualifiedGS[index.Int64()].GroupID
		return ga, nil
	} else {
		return ga, fmt.Errorf("selectNextGroup failed, arg error")
	}
}

func (gg *GlobalGroups) GetCastQualifiedGroups(height uint64) []*StaticGroupInfo {
	gg.lock.RLock()
	defer gg.lock.RUnlock()

	gs := make([]*StaticGroupInfo, 0)
	for _, g := range gg.groups {
		if g.CastQualified(height) {
			gs = append(gs, g)
		}
	}
	return gs
}

func (gg *GlobalGroups) GetAvailableGroups(height uint64) []*StaticGroupInfo {
	gg.lock.RLock()
	defer gg.lock.RUnlock()

	gs := make([]*StaticGroupInfo, 0)
	for _, g := range gg.groups {
		if !g.Dismissed(height) {
			gs = append(gs, g)
		}
	}
	return gs
}

func (gg *GlobalGroups) GetInitingGroup(dummyId groupsig.ID) *InitingGroup {
    return gg.ngg.getInitingGroup(dummyId)
}

func (gg *GlobalGroups) DismissGroups(height uint64) []groupsig.ID {
    gg.lock.RLock()
    defer gg.lock.RUnlock()

    ids := make([]groupsig.ID, 0)
	for _, g := range gg.groups {
		if g.Dismissed(height) {
			ids = append(ids, g.GroupID)
		}
	}
	return ids
}

func (gg *GlobalGroups) RemoveGroups(gids []groupsig.ID) {
	if len(gids) == 0 {
		return
	}
	gg.lock.Lock()
	defer gg.lock.Unlock()

	newGS := make([]*StaticGroupInfo, 0)
	for _, g := range gids {
		delete(gg.gIndex, g.GetHexString())
	}
	idxs := make([]int, 0)
	for _, idx := range gg.gIndex {
		idxs = append(idxs, idx)
	}
	sort.Ints(idxs)
	for _, idx := range idxs {
		newGS = append(newGS, gg.groups[idx])
	}
	gg.groups = newGS
}
