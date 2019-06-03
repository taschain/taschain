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

func GetCastExpireTime(base time.TimeStamp, deltaHeight uint64, castHeight uint64) time.TimeStamp {
	t := uint64(0)
	if castHeight == 1 { //铸高度1的时候，过期时间为5倍，以防节点启动不同步时，先提案的块过早过期导致同一节点对高度1提案多次
		t = 2
	}
	return base.Add(int64(t+deltaHeight) * int64(model.Param.MaxGroupCastTime))
}

func ConvertStaticGroup2CoreGroup(sgi *StaticGroupInfo) *types.Group {
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

func DeltaHeightByTime(bh *types.BlockHeader, preBH *types.BlockHeader) uint64 {
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

func ExpireTime(bh *types.BlockHeader, preBH *types.BlockHeader) time.TimeStamp {
	return GetCastExpireTime(preBH.CurTime, DeltaHeightByTime(bh, preBH), bh.Height)
}

func CalcRandomHash(preBH *types.BlockHeader, height uint64) common.Hash {
	data := preBH.Random
	var hash common.Hash

	deltaHeight := height - preBH.Height
	for ; deltaHeight > 0; deltaHeight-- {
		hash = base.Data2CommonHash(data)
		data = hash.Bytes()
	}
	return hash
}

func IsGroupDissmisedAt(gh *types.GroupHeader, h uint64) bool {
	return gh.DismissHeight <= h
}
func IsGroupWorkQualifiedAt(gh *types.GroupHeader, h uint64) bool {
	return !IsGroupDissmisedAt(gh, h) && gh.WorkHeight <= h
}