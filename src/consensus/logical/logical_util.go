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

func ConvertStaticGroup2CoreGroup(sgi *StaticGroupInfo, isDummy bool) *types.Group {
	members := make([]types.Member, len(sgi.Members))
	for idx, miner := range sgi.Members {
		member := types.Member{Id: miner.ID.Serialize(), PubKey: miner.PK.Serialize()}
		members[idx] = member
	}

	if isDummy {
		return &types.Group{
			Dummy: sgi.GIS.DummyID.Serialize(),
			Members: members,
			Signature: sgi.Signature.Serialize(),
			Parent: sgi.ParentId.Serialize(),
			PreGroup: sgi.PrevGroupID.Serialize(),
			BeginHeight: sgi.BeginHeight,
			DismissHeight: sgi.DismissHeight,
			Authority: sgi.Authority,
			Name: sgi.Name,
			Extends: sgi.Extends,
		}
	} else {
		return &types.Group{
			Id: sgi.GroupID.Serialize(),
			Members: members,
			PubKey: sgi.GroupPK.Serialize(),
			Signature: sgi.Signature.Serialize(),
			Parent: sgi.ParentId.Serialize(),
			PreGroup: sgi.PrevGroupID.Serialize(),
			BeginHeight: sgi.BeginHeight,
			DismissHeight: sgi.DismissHeight,
			Authority: sgi.Authority,
			Name: sgi.Name,
			Extends: sgi.Extends,
		}
	}
}