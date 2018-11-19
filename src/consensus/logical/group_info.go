//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package logical

import (
	"common"
	"consensus/groupsig"
	"fmt"
	"math/big"
	"sync"
	"core"
	"middleware/types"
	"log"
	"consensus/model"
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
	Members       []model.PubKeyInfo              //组内成员的静态信息(严格按照链上次序，全网一致，不然影响组铸块)。to do : 组成员的公钥是否有必要保存在这里？
	MemIndex      map[string]int            //用ID查找成员信息(成员ID->members中的索引)
	GIS           model.ConsensusGroupInitSummary //组的初始化凭证
	BeginHeight   uint64                    //组开始参与铸块的高度
	DismissHeight uint64                    //组解散的高度
	ParentId      groupsig.ID
	PrevGroupID		groupsig.ID				//前一块组id
	Signature     groupsig.Signature
	Authority     uint64      //权限相关数据（父亲组赋予）
	Name          string    //父亲组取的名字
	Extends       string      //带外数据
}

func NewDummySGIFromGroupRawMessage(grm *model.ConsensusGroupRawMessage) *StaticGroupInfo {
	sgi := &StaticGroupInfo{
		GIS:           grm.GI,
		ParentId: 		grm.GI.ParentID,
		Members:       grm.MEMS,
		MemIndex:      make(map[string]int),
	}
	for index, mem := range sgi.Members {
		sgi.MemIndex[mem.ID.GetHexString()] = index
	}
	return sgi
}

func NewSGIFromStaticGroupSummary(summary *model.StaticGroupSummary, group *InitingGroup) *StaticGroupInfo {
	sgi := &StaticGroupInfo{
		GroupID:       summary.GroupID,
		GroupPK:       summary.GroupPK,
		GIS:           summary.GIS,
		Members:       group.mems,
		MemIndex:      make(map[string]int),
		BeginHeight:   summary.GIS.BeginCastHeight,
		DismissHeight: summary.GIS.DismissHeight,
		ParentId:      summary.GIS.ParentID,
		PrevGroupID:   summary.GIS.PrevGroupID,
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

func NewSGIFromCoreGroup(coreGroup *types.Group) *StaticGroupInfo {
	sgi := &StaticGroupInfo{
		GroupID:     groupsig.DeserializeId(coreGroup.Id),
		GroupPK:     groupsig.DeserializePubkeyBytes(coreGroup.PubKey),
		BeginHeight: coreGroup.BeginHeight,
		Members:     make([]model.PubKeyInfo, 0),
		MemIndex:    make(map[string]int),
		DismissHeight: coreGroup.DismissHeight,
		ParentId:      groupsig.DeserializeId(coreGroup.Parent),
		PrevGroupID:   groupsig.DeserializeId(coreGroup.PreGroup),
		Signature:     *groupsig.DeserializeSign(coreGroup.Signature),
		Authority:     coreGroup.Authority,
		Name:          coreGroup.Name,
		Extends:       coreGroup.Extends,
	}

	for _, cMem := range coreGroup.Members {
		id := groupsig.DeserializeId(cMem.Id)
		pk := groupsig.DeserializePubkeyBytes(cMem.PubKey)
		pkInfo := model.NewPubKeyInfo(id, pk)
		sgi.addMember(&pkInfo)
	}
	return sgi
}

//取得某个矿工在组内的排位
func (sgi StaticGroupInfo) GetMinerPos(id groupsig.ID) int {
	pos := -1
	if v, ok := sgi.MemIndex[id.GetHexString()]; ok {
		pos = v
		//双重验证
		if !sgi.Members[pos].ID.IsEqual(id) {
			panic("double check fail!id=" + id.GetHexString())
		}
	}
	return pos
}


func (sgi StaticGroupInfo) GetPubKey() groupsig.Pubkey {
	return sgi.GroupPK
}


func (sgi *StaticGroupInfo) GetMemberCount() int {
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

func (sgi *StaticGroupInfo) addMember(m *model.PubKeyInfo) {
	if m.GetID().IsValid() {
		_, ok := sgi.MemIndex[m.GetID().GetHexString()]
		if !ok {
			sgi.Members = append(sgi.Members, *m)
			sgi.MemIndex[m.GetID().GetHexString()] = len(sgi.Members) - 1
		}
	}
}

func (sgi StaticGroupInfo) MemExist(uid groupsig.ID) bool {
	_, ok := sgi.MemIndex[uid.GetHexString()]
	return ok
}

//ok:是否组内成员
//m:组内成员矿工公钥
//func (sgi StaticGroupInfo) GetMember(uid groupsig.ID) (m model.PubKeyInfo, ok bool) {
//	var i int
//	i, ok = sgi.MemIndex[uid.GetHexString()]
//	fmt.Printf("data size=%v, cache size=%v.\n", len(sgi.Members), len(sgi.MemIndex))
//	fmt.Printf("find node(%v) = %v, local all mems=%v, gpk=%v.\n", uid.GetHexString(), ok, len(sgi.Members), sgi.GroupPK.GetHexString())
//	if ok {
//		m = sgi.Members[i]
//	} else {
//		i := 0
//		for k, _ := range sgi.MemIndex {
//			fmt.Printf("---mem(%v)=%v.\n", i, k)
//			i++
//		}
//	}
//	return
//}


//取得指定位置的铸块人
func (sgi *StaticGroupInfo) GetMemberID(i int) groupsig.ID {
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
	chain 	  	*core.GroupChain
	groups    []*StaticGroupInfo
	gIndex    map[string]int     //string(ID)->索引
	generator *NewGroupGenerator //新组处理器(组外处理器)
	lock      sync.RWMutex
}

func NewGlobalGroups(chain *core.GroupChain) *GlobalGroups {
	return &GlobalGroups{
		groups:    make([]*StaticGroupInfo, 1),
		gIndex:    make(map[string]int),
		generator: CreateNewGroupGenerator(),
		chain: 		chain,
	}
}

func (gg *GlobalGroups) GetGroupSize() int {
	gg.lock.RLock()
	defer gg.lock.RUnlock()
	return len(gg.groups)
}

func (gg *GlobalGroups) AddInitingGroup(g *InitingGroup) bool {
    return gg.generator.addInitingGroup(g)
}

func (gg *GlobalGroups) removeInitingGroup(dummyId groupsig.ID) {
	gg.generator.removeInitingGroup(dummyId)
}

func (gg *GlobalGroups) lastGroup() *StaticGroupInfo {
	return gg.groups[len(gg.groups)-1]
}

func (gg *GlobalGroups) findPos(g *StaticGroupInfo) (idx int, right bool){
	cnt := len(gg.groups)
	if cnt == 1 {
		return 1, true
	}
	last := gg.lastGroup()
	if g.PrevGroupID.IsEqual(last.GroupID) {	//刚好能与最后一个连接上，大部分时候是这个情况
		return cnt, true
	}
	if g.BeginHeight > last.BeginHeight {	//属于更后面的组， 先append到最后
		return cnt, false
	}
	for i := 1; i < cnt; i++	{
		if gg.groups[i].BeginHeight > g.BeginHeight {
			return i, g.GroupID.IsEqual(gg.groups[i].PrevGroupID) && (i ==1 || g.PrevGroupID.IsEqual(gg.groups[i-1].GroupID))
		}
	}
	return -1, false
}


func (gg *GlobalGroups) append(g *StaticGroupInfo) bool {
	gg.groups = append(gg.groups, g)
	gg.gIndex[g.GroupID.GetHexString()] = len(gg.groups) - 1
	return true
}

//增加一个合法铸块组
//在组同步的时候，该方法有可能并发调用，导致组的顺序也是乱序的
func (gg *GlobalGroups) AddStaticGroup(g *StaticGroupInfo) bool {
	gg.lock.Lock()
	defer gg.lock.Unlock()

	result := ""
	blog := newBizLog("AddStaticGroup")
	defer func() {
		blog.log("id=%v, beginHeight=%v, result=%v\n", g.GroupID.ShortS(), g.BeginHeight, result)
	}()

	if _, ok := gg.gIndex[g.GroupID.GetHexString()]; !ok {
		if g.BeginHeight == 0 {	//创世组
			gg.groups[0] = g
			gg.gIndex[g.GroupID.GetHexString()] = 0
			result = "success"
			return true
		}
		if idx, right := gg.findPos(g); idx >= 0 {
			cnt := len(gg.groups)
			if idx == cnt {
				gg.append(g)
				result = "append"
			} else {
				gg.groups = append(gg.groups, g)
				for i:= cnt; i > idx; i-- {
					gg.groups[i] = gg.groups[i-1]
					gg.gIndex[gg.groups[i].GroupID.GetHexString()] = i
				}
				gg.groups[idx] = g
				gg.gIndex[g.GroupID.GetHexString()] = idx
				result = "insert"
			}
			if right {
				result += "and linked"
			} else {
				result += "but not linked"
			}
			return true
		} else {
			result = "can't find insert pos"
		}
	} else {
		result = "already exist this group, ignored"
	}
	return false
}

func (gg *GlobalGroups) GroupInitedMessage(sgs *model.StaticGroupSummary, sender groupsig.ID, height uint64) int32 {
	return gg.generator.ReceiveData(sgs, sender, height)
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

func (gg *GlobalGroups) getGroupFromCache(id groupsig.ID) (g *StaticGroupInfo, err error) {
	gg.lock.RLock()
	defer gg.lock.RUnlock()

	index, ok := gg.gIndex[id.GetHexString()]
	if ok {
		g, err = gg.getGroupByIndex(index)
		if !g.GroupID.IsEqual(id) {
			panic("ggIndex error")
		}
	}
	return
}

func (gg *GlobalGroups) GetGroupByID(id groupsig.ID) (g *StaticGroupInfo, err error) {
	if g, err = gg.getGroupFromCache(id); err != nil {
		return
	} else {
		if g == nil {
			chainGroup := gg.chain.GetGroupById(id.Serialize())
			if chainGroup != nil {
				g = NewSGIFromCoreGroup(chainGroup)
			}
		}
	}
	if g == nil {
		log.Printf("^^^^^^^^^^^^^^^^^^GetGroupByID nil, gid=%v\n", id.ShortS())
		for _, g := range gg.groups {
			log.Printf("^^^^^^^^^^^^^^^^^^GetGroupByID cached groupid %v\n", g.GroupID.ShortS())
		}
		g = &StaticGroupInfo{}
	}
	return
}

//根据上一块哈希值，确定下一块由哪个组铸块
func (gg *GlobalGroups) SelectNextGroup(h common.Hash, height uint64) (groupsig.ID, error) {
	qualifiedGS := gg.GetCastQualifiedGroups(height)

	var ga groupsig.ID
	value := h.Big()

	gids := make([]string, 0)
	for _, g := range qualifiedGS {
		gids = append(gids, g.GroupID.ShortS())
	}

	if value.BitLen() > 0 && len(qualifiedGS) > 0 {
		index := value.Mod(value, big.NewInt(int64(len(qualifiedGS))))
		ga = qualifiedGS[index.Int64()].GroupID
		log.Printf("height %v SelectNextGroup qualified groups %v, index %v\n", height, gids, index)
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
		if g == nil {
			continue
		}
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
		if g == nil {
			continue
		}
		if !g.Dismissed(height) {
			gs = append(gs, g)
		}
	}
	return gs
}

func (gg *GlobalGroups) GetInitingGroup(dummyId groupsig.ID) *InitingGroup {
    return gg.generator.getInitingGroup(dummyId)
}

func (gg *GlobalGroups) DismissGroups(height uint64) []groupsig.ID {
    gg.lock.RLock()
    defer gg.lock.RUnlock()

    ids := make([]groupsig.ID, 0)
	for _, g := range gg.groups {
		if g == nil {
			continue
		}
		if g.Dismissed(height) {
			ids = append(ids, g.GroupID)
		} else {
			break
		}
	}
	return ids
}

func (gg *GlobalGroups) RemoveGroups(gids []groupsig.ID) {
	if len(gids) == 0 {
		return
	}
	removeIdMap := make(map[string]bool)
	for _, gid := range gids {
		removeIdMap[gid.GetHexString()] = true
	}
	newGS := make([]*StaticGroupInfo, 0)
	for _, g := range gg.groups {
		if g == nil {
			continue
		}
		if _, ok := removeIdMap[g.GroupID.GetHexString()]; !ok {
			newGS = append(newGS, g)
		}
	}
	indexMap := make(map[string]int)
	for idx, g := range newGS {
		indexMap[g.GroupID.GetHexString()] = idx
	}

	gg.lock.Lock()
	defer gg.lock.Unlock()

	gg.groups = newGS
	gg.gIndex = indexMap
}
