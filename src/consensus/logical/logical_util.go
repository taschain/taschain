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

func GetCastExpireTime(base time.Time, deltaHeight uint64) time.Time {
	return base.Add(time.Second * time.Duration(deltaHeight * uint64(model.Param.MaxGroupCastTime)))
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