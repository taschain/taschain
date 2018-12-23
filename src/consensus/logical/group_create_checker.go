package logical

import (
	"consensus/groupsig"
	"consensus/model"
	"consensus/base"
	"math/big"
	"middleware/types"
	"time"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2018/9/11 上午11:19
**  Description: 
*/

type GroupCreateChecker struct {
	processor *Processor
	access 	 *MinerPoolReader
}

func newGroupCreateChecker(proc *Processor) *GroupCreateChecker {
	return &GroupCreateChecker{
		processor: proc,
		access: proc.minerReader,
	}
}
func checkCreate(topBH *types.BlockHeader) bool {
    return topBH.Height % model.Param.CreateGroupInterval == 0
}

func (gchecker *GroupCreateChecker) selectKing(theBH *types.BlockHeader, group *StaticGroupInfo) groupsig.ID {
	data := make([]byte, 0)
	data = append(data, theBH.Random...)
	hash := base.Data2CommonHash(data)
	biHash := hash.Big()

	var index int32 = -1
	mem := group.GetMemberCount()
	if biHash.BitLen() > 0 {
		index = int32(biHash.Mod(biHash, big.NewInt(int64(mem))).Int64())
	}
	newBizLog("selectKing").log("king index=%v, id=%v", index, group.GetMemberID(int(index)).ShortS())
	if index < 0 {
		return groupsig.ID{}
	}
	return group.GetMemberID(int(index))
}


/**
* @Description:  检查是否开始建组
* @Param:
* @return:  create 当前是否应该建组, sgi 当前应该启动建组的组，即父亲组， castor 发起建组的组内成员， theBH 基于哪个块建组
*/
func (gchecker *GroupCreateChecker) checkCreateGroup(topHeight uint64) (create bool, sgi *StaticGroupInfo, castor groupsig.ID, theBH *types.BlockHeader) {
	blog := newBizLog("checkCreateGroup")
	defer func() {
		blog.log("topHeight=%v, create %v", topHeight, create)
	}()
	if topHeight <= model.Param.CreateGroupInterval {
		return
	}

	// 指定高度已经在组链上出现过
	if _, ok := gchecker.processor.CreateHeightGroups[topHeight]; ok {
		return
	}

	h := topHeight-model.Param.CreateGroupInterval
	theBH = gchecker.processor.MainChain.QueryBlockByHeight(h)
	if theBH == nil {
		blog.log("theBH is nil, height=%v", h)
		return
	}
	if !checkCreate(theBH) {
		return
	}

	castGID := groupsig.DeserializeId(theBH.GroupId)
	sgi = gchecker.processor.GetGroup(castGID)
	if !sgi.CastQualified(topHeight) {
		return
	}
	castor = gchecker.selectKing(theBH, sgi)
	blog.log("topHeight=%v, group=%v, king=%v", topHeight, sgi.GroupID.ShortS(), castor.ShortS())
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

func (checker *GroupCreateChecker) generateGroupHeader(createHeight uint64, createTime time.Time, lastGroup *types.Group) (gh *types.GroupHeader, mems []groupsig.ID, threshold int) {
	create, group, _, theBH := checker.checkCreateGroup(createHeight)
	//指定高度不能建组
	if !create {
		return
	}
	if !group.GroupID.IsValid() {
		panic("create group init summary failed")
	}
	//是否有足够候选人
	enough, memIds := checker.selectCandidates(theBH, createHeight)
	if !enough {
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
	return gh, memIds, model.Param.GetGroupK(group.GetMemberCount())
}