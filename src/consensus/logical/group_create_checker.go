package logical

import (
	"consensus/groupsig"
	"consensus/model"
	"consensus/base"
	"middleware/types"
	"time"
	"fmt"
	"sync"
	"common"
)

/*
**  Creator: pxf
**  Date: 2018/9/11 上午11:19
**  Description: 
*/

type GroupCreateChecker struct {
	processor      *Processor
	access         *MinerPoolReader
	createdHeights [50]uint64 // 标识该建组高度是否已经创建过组了
	curr 			int
	lock           sync.RWMutex    // CreateHeightGroups的互斥锁，防止重复写入
}

func newGroupCreateChecker(proc *Processor) *GroupCreateChecker {
	return &GroupCreateChecker{
		processor:      proc,
		access:         proc.minerReader,
		curr: 0,
	}
}
func checkCreate(topBH *types.BlockHeader) bool {
    return topBH.Height % model.Param.CreateGroupInterval == 0
}

func (gchecker *GroupCreateChecker) heightCreated(h uint64) bool {
    gchecker.lock.RLock()
    defer gchecker.lock.RUnlock()
	for _, height := range gchecker.createdHeights {
		if h == height {
			return true
		}
	}
	return false
}

func (gchecker *GroupCreateChecker) addHeightCreated(h uint64)  {
	gchecker.lock.RLock()
	defer gchecker.lock.RUnlock()
	gchecker.createdHeights[gchecker.curr] = h
	gchecker.curr = (gchecker.curr+1)%len(gchecker.createdHeights)
}

func (gchecker *GroupCreateChecker) selectKing(theBH *types.BlockHeader, group *StaticGroupInfo) []groupsig.ID {
	memCnt := group.GetMemberCount()
	if memCnt <= 5 {
		return group.GetMembers()
	}

	rand := base.RandFromBytes(theBH.Random)
	num := 5

	selectIndexs := rand.RandomPerm(memCnt, num)
	ids := make([]groupsig.ID, len(selectIndexs))
	for i, idx := range selectIndexs {
		ids[i] = group.GetMemberID(idx)
	}

	newBizLog("selectKing").log("king index=%v, ids=%v", selectIndexs, ids)

	return ids
}


/**
* @Description:  检查是否开始建组
* @Param:
* @return:  create 当前是否应该建组, sgi 当前应该启动建组的组，即父亲组， castor 发起建组的组内成员， theBH 基于哪个块建组
*/
func (gchecker *GroupCreateChecker) checkCreateGroup(topHeight uint64) (create bool, sgi *StaticGroupInfo, castors []groupsig.ID, theBH *types.BlockHeader, err error) {
	blog := newBizLog("checkCreateGroup")
	blog.log("CreateHeightGroups = %v, topHeight = %v", gchecker.createdHeights, topHeight)
	defer func() {
		blog.log("topHeight=%v, create %v, err %v", topHeight, create, err)
	}()
	if topHeight <= model.Param.CreateGroupInterval {
		err = fmt.Errorf("topHeight samller than CreateGroupInterval")
		return
	}

	// 指定高度已经在组链上出现过
	if gchecker.heightCreated(topHeight) {
		err = fmt.Errorf("topHeight already created")
		return
	}

	h := topHeight-model.Param.CreateGroupInterval
	theBH = gchecker.processor.MainChain.QueryBlockByHeight(h)
	if theBH == nil {
		err = common.ErrCreateBlockNil
		return
	}
	if !checkCreate(theBH) {
		err = fmt.Errorf("cann't create group")
		return
	}

	castGID := groupsig.DeserializeId(theBH.GroupId)
	sgi = gchecker.processor.GetGroup(castGID)
	if !sgi.CastQualified(topHeight) {
		err = fmt.Errorf("group dont qualified creating group gid=%v", sgi.GroupID.ShortS())
		return
	}
	castors = gchecker.selectKing(theBH, sgi)
	blog.log("topHeight=%v, group=%v, king=%v", topHeight, sgi.GroupID.ShortS(), castors)
	create = true
	return
}


func (gchecker *GroupCreateChecker) selectCandidates(theBH *types.BlockHeader, height uint64) (enough bool, cands []groupsig.ID) {
	min := model.Param.CreateGroupMinCandidates()
	blog := newBizLog("selectCandidates")
	allCandidates := gchecker.access.getCanJoinGroupMinersAt(height)

	ids := make([]string, len(allCandidates))
	for idx, can := range allCandidates {
		ids[idx] = can.ID.ShortS()
	}
	blog.log("=======allCandidates height %v, %v size %v", height, ids, len(allCandidates))
	if len(allCandidates) < min {
		return
	}
	groups := gchecker.processor.GetAvailableGroupsAt(height)

	blog.log("available groupsize %v", len(groups))

	candidates := make([]model.MinerDO, 0)
	for _, cand := range allCandidates {
		joinedNum := 0
		for _, g := range groups {
			if g.MemExist(cand.ID) {
				joinedNum++
			}
		}
		if joinedNum < model.Param.MinerMaxJoinGroup {
			candidates = append(candidates, cand)
		}
	}
	num := len(candidates)
	if len(candidates) < min {
		blog.log("not enough candidates, expect %v, got %v", min, num)
		return
	}

	rand := base.RandFromBytes(theBH.Random)
	seqs := rand.RandomPerm(num, model.Param.GetGroupMemberNum())

	result := make([]groupsig.ID, len(seqs))
	for i, seq := range seqs {
		result[i] = candidates[seq].ID
	}

	str := ""
	for _, id := range result {
		str += id.ShortS() + ","
	}
	blog.log("=============selectCandidates %v", str)
	return true, result
}

func (checker *GroupCreateChecker) generateGroupHeader(createHeight uint64, createTime time.Time, lastGroup *types.Group) (gh *types.GroupHeader, mems []groupsig.ID, threshold int, kings []groupsig.ID, err error) {
	create, group, castors, theBH, e := checker.checkCreateGroup(createHeight)
	//指定高度不能建组
	if !create {
		err = e
		return
	}
	if !group.GroupID.IsValid() {
		panic("create group init summary failed")
	}
	//是否有足够候选人
	enough, memIds := checker.selectCandidates(theBH, createHeight)
	if !enough {
		err = fmt.Errorf("not enough candidates")
		return
	}

	gn := fmt.Sprintf("%s-%v", group.GroupID.GetHexString(), theBH.Height)

	gh = &types.GroupHeader{
		Parent: group.GroupID.Serialize(),
		PreGroup: lastGroup.Id,
		Name: gn,
		Authority: 777,
		BeginTime: createTime,
		CreateHeight: createHeight,
		ReadyHeight: createHeight + model.Param.GroupGetReadyGap,
		MemberRoot: model.GenMemberRootByIds(memIds),
		Extends: "",
	}
	gh.WorkHeight = gh.ReadyHeight + model.Param.GroupCastQualifyGap
	gh.DismissHeight = gh.WorkHeight + model.Param.GroupCastDuration

	gh.Hash = gh.GenHash()
	return gh, memIds, model.Param.GetGroupK(group.GetMemberCount()), castors,nil
}