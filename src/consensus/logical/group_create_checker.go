package logical

import (
	"consensus/groupsig"
	"consensus/model"
	"consensus/base"
	"math/big"
	"middleware/types"
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
	mem := len(group.Members)
	if biHash.BitLen() > 0 {
		index = int32(biHash.Mod(biHash, big.NewInt(int64(mem))).Int64())
	}
	newBizLog("selectKing").log("king index=%v, id=%v", index, group.GetMemberID(int(index)).ShortS())
	if index < 0 {
		return groupsig.ID{}
	}
	return group.GetMemberID(int(index))
}

//检查当前用户是否是属于建组的组, 返回组id
func (gchecker *GroupCreateChecker) checkCreateGroup(topHeight uint64) (create bool, sgi *StaticGroupInfo, castor groupsig.ID, theBH *types.BlockHeader) {
	blog := newBizLog("checkCreateGroup")
	defer func() {
		blog.log("topHeight=%v, create %v", topHeight, create)
	}()
	if topHeight <= model.Param.CreateGroupInterval {
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
	if !gchecker.processor.IsMinerGroup(castGID) {
		return
	}
	sgi = gchecker.processor.GetGroup(castGID)
	if !sgi.CastQualified(topHeight) {
		return
	}
	castor = gchecker.selectKing(theBH, sgi)
	blog.log("topHeight=%v, king=%v", topHeight, castor.ShortS())
	create = true
	return
}


func (gchecker *GroupCreateChecker) selectCandidates(theBH *types.BlockHeader, height uint64) (enough bool, cands []model.PubKeyInfo) {
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

	result := make([]model.PubKeyInfo, len(seqs))
	for i, seq := range seqs {
		result[i] = model.NewPubKeyInfo(candidates[seq].ID, candidates[seq].PK)
	}

	str := ""
	for _, id := range result {
		str += id.ID.ShortS() + ","
	}
	blog.log("=============selectCandidates %v", str)
	return true, result
}


func (gchecker *GroupCreateChecker) CheckGIS(gis *model.ConsensusGroupInitSummary, isGroupMember bool) bool {
	//topGroup := gchecker.groupChain.LastGroup()
	//topBH := gchecker.mainChain.QueryTopBlock()
	//
	//blog := newBizLog("CheckGIS")
	//deltaH := topBH.Height - gis.TopHeight
	//if deltaH < 0 || deltaH >= model.Param.CreateGroupInterval {
	//	blog.log("topHeight error. topHeight=%v, gis topHeight=%v",  topBH.Height, gis.TopHeight)
	//	return false
	//}
	//
	//create, group, bh := gchecker.checkCreateGroup(gis.TopHeight)
	//if group == nil {
	//	log.Printf("CheckGIS")
	//	return false
	//}

	return true
}