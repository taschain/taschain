package logical

import (
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/base"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/middleware/time"
	"github.com/taschain/taschain/middleware/types"
)

/*
**  Creator: pxf
**  Date: 2018/6/8 上午9:52
**  Description:
 */

func getCastExpireTime(base time.TimeStamp, deltaHeight uint64, castHeight uint64) time.TimeStamp {
	t := uint64(0)

	// When the cast height is 1, the expiration time is 5 times. In case the
	// node starts to be out of sync, the first proposed block expires prematurely,
	// causing the same node to propose the height 1 multiple times.
	if castHeight == 1 {
		t = 2
	}
	return base.Add(int64(t+deltaHeight) * int64(model.Param.MaxGroupCastTime))
}

func convertStaticGroup2CoreGroup(sgi *StaticGroupInfo) *types.Group {
	members := make([][]byte, sgi.GetMemberCount())
	for idx, miner := range sgi.GInfo.Mems {
		members[idx] = miner.Serialize()
	}
	return &types.Group{
		Header:    sgi.getGroupHeader(),
		ID:        sgi.GroupID.Serialize(),
		PubKey:    sgi.GroupPK.Serialize(),
		Signature: sgi.GInfo.GI.Signature.Serialize(),
		Members:   members,
	}
}

func deltaHeightByTime(bh *types.BlockHeader, preBH *types.BlockHeader) uint64 {
	var (
		deltaHeightByTime uint64
	)
	if bh.Height == 1 {
		d := time.TSInstance.Since(preBH.CurTime)
		deltaHeightByTime = uint64(d)/uint64(model.Param.MaxGroupCastTime) + 1
	} else {
		deltaHeightByTime = bh.Height - preBH.Height
	}
	return deltaHeightByTime
}

func expireTime(bh *types.BlockHeader, preBH *types.BlockHeader) time.TimeStamp {
	return getCastExpireTime(preBH.CurTime, deltaHeightByTime(bh, preBH), bh.Height)
}

func calcRandomHash(preBH *types.BlockHeader, height uint64) common.Hash {
	data := preBH.Random
	var hash common.Hash

	deltaHeight := height - preBH.Height
	for ; deltaHeight > 0; deltaHeight-- {
		hash = base.Data2CommonHash(data)
		data = hash.Bytes()
	}
	return hash
}

func isGroupDissmisedAt(gh *types.GroupHeader, h uint64) bool {
	return gh.DismissHeight <= h
}

// IsGroupWorkQualifiedAt check if the specified group is work qualified at the given height
func IsGroupWorkQualifiedAt(gh *types.GroupHeader, h uint64) bool {
	return !isGroupDissmisedAt(gh, h) && gh.WorkHeight <= h
}
