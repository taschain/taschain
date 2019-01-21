package logical

import (

	"time"
	"middleware/types"
	"consensus/model"
)

/*
**  Creator: pxf
**  Date: 2018/6/8 上午9:52
**  Description: 
*/

func GetCastExpireTime(base time.Time, deltaHeight uint64, castHeight uint64) time.Time {
	t := uint64(0)
	if castHeight == 1 {//铸高度1的时候，过期时间为5倍，以防节点启动不同步时，先提案的块过早过期导致同一节点对高度1提案多次
		t = 2
	}
	return base.Add(time.Second * time.Duration((t + deltaHeight) * uint64(model.Param.MaxGroupCastTime)))
}

func ConvertStaticGroup2CoreGroup(sgi *StaticGroupInfo) *types.Group {
	members := make([][]byte, sgi.GetMemberCount())
	for idx, miner := range sgi.GInfo.Mems {
		members[idx] = miner.Serialize()
	}
	return &types.Group{
		Header: sgi.getGroupHeader(),
		Id: 	sgi.GroupID.Serialize(),
		PubKey: sgi.GroupPK.Serialize(),
		Signature: sgi.GInfo.GI.Signature.Serialize(),
		Members: members,
	}
}

func DeltaHeightByTime(bh *types.BlockHeader, preBH *types.BlockHeader) uint64 {
	var (
		deltaHeightByTime uint64
	)
	if bh.Height == 1 {
		d := time.Since(preBH.CurTime)
		deltaHeightByTime = uint64(d.Seconds())/uint64(model.Param.MaxGroupCastTime) + 1
	} else {
		deltaHeightByTime = bh.Height - preBH.Height
	}
	return deltaHeightByTime
}

func VerifyBHExpire(bh *types.BlockHeader, preBH *types.BlockHeader) (time.Time, bool) {
	expire := GetCastExpireTime(preBH.CurTime, DeltaHeightByTime(bh, preBH), bh.Height)
	return expire, time.Now().After(expire)
}