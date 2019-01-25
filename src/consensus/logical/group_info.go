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
	GroupID       groupsig.ID                     //组ID(可以由组公钥生成)
	GroupPK       groupsig.Pubkey                 //组公钥
	MemIndex      map[string]int                  //用ID查找成员信息(成员ID->members中的索引)
	//GIS           model.ConsensusGroupInitSummary //组的初始化凭证
	GInfo 			*model.ConsensusGroupInitInfo
	//WorkHeight    uint64                          //组开始参与铸块的高度
	//DismissHeight uint64                          //组解散的高度
	ParentId      groupsig.ID
	PrevGroupID   groupsig.ID				//前一块组id
	//Signature     groupsig.Signature
	//Authority     uint64      //权限相关数据（父亲组赋予）
	//Name          string    //父亲组取的名字
	//Extends       string      //带外数据
}


func NewSGIFromStaticGroupSummary(gid groupsig.ID, gpk groupsig.Pubkey, group *InitedGroup) *StaticGroupInfo {
	gInfo := group.gInfo
	sgi := &StaticGroupInfo{
		GroupID:       gid,
		GroupPK:       gpk,
		GInfo:     		gInfo,
		ParentId:      gInfo.GI.ParentID(),
		PrevGroupID:   gInfo.GI.PreGroupID(),
	}
	sgi.buildMemberIndex()
	return sgi
}

func NewSGIFromCoreGroup(coreGroup *types.Group) *StaticGroupInfo {
	gh := coreGroup.Header
	gis := model.ConsensusGroupInitSummary{
		Signature: *groupsig.DeserializeSign(coreGroup.Signature),
		GHeader: gh,
	}
	mems := make([]groupsig.ID, len(coreGroup.Members))
	for i, mem := range coreGroup.Members {
		mems[i] = groupsig.DeserializeId(mem)
	}
	gInfo := &model.ConsensusGroupInitInfo{
		GI: gis,
		Mems: mems,
	}
	sgi := &StaticGroupInfo{
		GroupID:       groupsig.DeserializeId(coreGroup.Id),
		GroupPK:       groupsig.DeserializePubkeyBytes(coreGroup.PubKey),
		ParentId:      groupsig.DeserializeId(gh.Parent),
		PrevGroupID:   groupsig.DeserializeId(gh.PreGroup),
		GInfo:			gInfo,
	}

	sgi.buildMemberIndex()
	return sgi
}

func (sgi *StaticGroupInfo) buildMemberIndex()  {
	if sgi.MemIndex == nil {
		sgi.MemIndex = make(map[string]int)
	}
	for index, mem := range sgi.GInfo.Mems {
		sgi.MemIndex[mem.GetHexString()] = index
	}
}

func (sgi *StaticGroupInfo) GetMembers() []groupsig.ID {
    return sgi.GInfo.Mems
}
//取得某个矿工在组内的排位
func (sgi StaticGroupInfo) GetMinerPos(id groupsig.ID) int {
	pos := -1
	if v, ok := sgi.MemIndex[id.GetHexString()]; ok {
		pos = v
		//双重验证
		if !sgi.GInfo.Mems[pos].IsEqual(id) {
			panic("double check fail!id=" + id.GetHexString())
		}
	}
	return pos
}


func (sgi StaticGroupInfo) GetPubKey() groupsig.Pubkey {
	return sgi.GroupPK
}


func (sgi *StaticGroupInfo) GetMemberCount() int {
	return sgi.GInfo.MemberSize()
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

func (sgi *StaticGroupInfo) getGroupHeader() *types.GroupHeader {
    return sgi.GInfo.GI.GHeader
}

func (sgi StaticGroupInfo) MemExist(uid groupsig.ID) bool {
	_, ok := sgi.MemIndex[uid.GetHexString()]
	return ok
}

//取得指定位置的铸块人
func (sgi *StaticGroupInfo) GetMemberID(i int) groupsig.ID {
	var m groupsig.ID
	if i >= 0 && i < len(sgi.MemIndex) {
		m = sgi.GInfo.Mems[i]
	}
	return m
}

func (sgi *StaticGroupInfo) CastQualified(height uint64) bool {
	gh := sgi.getGroupHeader()
	return gh.WorkHeight <= height && height < gh.DismissHeight
}

//是否已解散
func (sgi *StaticGroupInfo) Dismissed(height uint64) bool {
	return height >= sgi.getGroupHeader().DismissHeight
}

func (sgi *StaticGroupInfo) GetReadyTimeout(height uint64) bool {
    return sgi.getGroupHeader().ReadyHeight <= height
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

//func (gg *GlobalGroups) AddInitingGroup(g *InitingGroup) bool {
//    return gg.generator.ad(g)
//}

func (gg *GlobalGroups) removeInitedGroup(gHash common.Hash) {
	gg.generator.removeInitedGroup(gHash)
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
	if g.getGroupHeader().WorkHeight > last.getGroupHeader().WorkHeight { //属于更后面的组， 先append到最后
		return cnt, false
	}
	for i := 1; i < cnt; i++	{
		if gg.groups[i].getGroupHeader().WorkHeight > g.getGroupHeader().WorkHeight {
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
		blog.log("id=%v, hash=%v, beginHeight=%v, result=%v\n", g.GroupID.ShortS(), g.getGroupHeader().Hash.ShortS(), g.getGroupHeader().WorkHeight, result)
	}()

	if _, ok := gg.gIndex[g.GroupID.GetHexString()]; !ok {
		if g.getGroupHeader().WorkHeight == 0 { //创世组
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

//func (gg *GlobalGroups) GroupInitedMessage(msg *model.ConsensusGroupInitedMessage, threshold int) int32 {
//	return gg.generator.ReceiveData(msg, threshold)
//}

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
		stdLogger.Debugf("^^^^^^^^^^^^^^^^^^GetGroupByID nil, gid=%v\n", id.ShortS())
		for _, g := range gg.groups {
			stdLogger.Debugf("^^^^^^^^^^^^^^^^^^GetGroupByID cached groupid %v\n", g.GroupID.ShortS())
		}
		g = &StaticGroupInfo{}
	}
	return
}

func (gg *GlobalGroups) selectIndex(num int, hash common.Hash) int64 {
	value := hash.Big()
	index := value.Mod(value, big.NewInt(int64(num)))
	return index.Int64()
}

//根据上一块哈希值，确定下一块由哪个组铸块
func (gg *GlobalGroups) SelectNextGroupFromCache(h common.Hash, height uint64) (groupsig.ID, error) {
	qualifiedGS := gg.GetCastQualifiedGroups(height)

	var ga groupsig.ID

	gids := make([]string, 0)
	for _, g := range qualifiedGS {
		gids = append(gids, g.GroupID.ShortS())
	}

	if h.Big().BitLen() > 0 && len(qualifiedGS) > 0 {
		index := gg.selectIndex(len(qualifiedGS), h)
		ga = qualifiedGS[index].GroupID
		stdLogger.Debugf("height %v SelectNextGroupFromCache qualified groups %v, index %v\n", height, gids, index)
		return ga, nil
	} else {
		return ga, fmt.Errorf("selectNextGroupFromCache failed, hash %v, qualified group %v", h.ShortS(), gids)
	}
}

func (gg *GlobalGroups) getCastQualifiedGroupFromChains(height uint64) []*types.Group {
	iter := gg.chain.NewIterator()
	groups := make([]*types.Group, 0)
	for g := iter.Current(); g != nil; g = iter.MovePre() {
		if g.Header.WorkHeight <= height && g.Header.DismissHeight > height {
			groups = append(groups, g)
		}
		if g.Header.DismissHeight <= height {
			g = gg.chain.GetGroupByHeight(0)
			groups = append(groups, g)
			break
		}
	}
	n := len(groups)
	reverseGroups := make([]*types.Group, n)
	for i := 0; i < n; i++ {
		reverseGroups[n-i-1] = groups[i]
	}
	return groups
}

func (gg *GlobalGroups) SelectNextGroupFromChain(h common.Hash, height uint64) (groupsig.ID, error) {
	quaulifiedGS := gg.getCastQualifiedGroupFromChains(height)
	idshort := make([]string, len(quaulifiedGS))
	for idx, g := range quaulifiedGS {
		idshort[idx] = groupsig.DeserializeId(g.Id).ShortS()
	}

	var ga groupsig.ID
	if h.Big().BitLen() > 0 && len(quaulifiedGS) > 0 {
		index := gg.selectIndex(len(quaulifiedGS), h)
		ga = groupsig.DeserializeId(quaulifiedGS[index].Id)
		stdLogger.Debugf("height %v SelectNextGroupFromChain qualified groups %v, index %v\n", height, idshort, index)
		return ga, nil
	} else {
		return ga, fmt.Errorf("SelectNextGroupFromChain failed, arg error")
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

func (gg *GlobalGroups) GetInitedGroup(gHash common.Hash) *InitedGroup {
    return gg.generator.getInitedGroup(gHash)
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
