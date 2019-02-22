package logical

import (
	"consensus/groupsig"
	"consensus/model"
	"consensus/base"
	"middleware/types"
	"sync"
	"math"
	"bytes"
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
func checkCreate(h uint64) bool {
    return h > 0 && h % model.Param.CreateGroupInterval == 0
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

//只要选择一半人就行了。每个人的权重按顺序递减
func (gchecker *GroupCreateChecker) selectKing(theBH *types.BlockHeader, group *StaticGroupInfo) (kings []groupsig.ID, isKing bool) {
	num := int(math.Ceil(float64(group.GetMemberCount()/2)))
	if num < 1 {
		num = 1
	}

	rand := base.RandFromBytes(theBH.Random)

	isKing = false

	selectIndexs := rand.RandomPerm(group.GetMemberCount(), num)
	kings = make([]groupsig.ID, len(selectIndexs))
	for i, idx := range selectIndexs {
		kings[i] = group.GetMemberID(idx)
		if gchecker.processor.GetMinerID().IsEqual(kings[i]) {
			isKing = true
		}
	}

	newBizLog("selectKing").log("king index=%v, ids=%v, isKing %v", selectIndexs, kings, isKing)
	return
}

func (gchecker *GroupCreateChecker) availableGroupsAt(h uint64) []*types.Group {
    iter := gchecker.processor.GroupChain.NewIterator()
    gs := make([]*types.Group, 0)
    for g := iter.Current(); g != nil; g = iter.MovePre() {
		if g.Header.DismissHeight > h {
			gs = append(gs, g)
		} else {
			genesis := gchecker.processor.GroupChain.GetGroupByHeight(0)
			gs = append(gs, genesis)
			break
		}
	}
	return gs
}


func (gchecker *GroupCreateChecker) selectCandidates(theBH *types.BlockHeader) (enough bool, cands []groupsig.ID) {
	min := model.Param.CreateGroupMinCandidates()
	blog := newBizLog("selectCandidates")
	height := theBH.Height
	allCandidates := gchecker.access.getCanJoinGroupMinersAt(height)

	ids := make([]string, len(allCandidates))
	for idx, can := range allCandidates {
		ids[idx] = can.ID.ShortS()
	}
	blog.log("=======allCandidates height %v, %v size %v", height, ids, len(allCandidates))
	if len(allCandidates) < min {
		return
	}
	groups := gchecker.availableGroupsAt(theBH.Height)

	blog.log("available groupsize %v", len(groups))

	candidates := make([]model.MinerDO, 0)
	for _, cand := range allCandidates {
		joinedNum := 0
		for _, g := range groups {
			for _, mem := range g.Members {
				if bytes.Equal(mem, cand.ID.Serialize()) {
					joinedNum++
					break
				}
			}
		}
		if joinedNum < model.Param.MinerMaxJoinGroup {
			candidates = append(candidates, cand)
		}
	}
	num := len(candidates)

	selectNum := model.Param.CreateGroupMemberCount(num)
	if selectNum <= 0 {
		blog.log("not enough candidates, got %v", len(candidates))
		return
	}

	rand := base.RandFromBytes(theBH.Random)
	seqs := rand.RandomPerm(num, selectNum)

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
