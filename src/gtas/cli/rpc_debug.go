package cli

import (
	"consensus/mediator"
	"core"
	"consensus/groupsig"
	"fmt"
	"middleware/types"
)

/*
**  Creator: pxf
**  Date: 2019/1/11 下午1:39
**  Description: 
*/

type SysWorkSummary struct {
	BeginHeight uint64 `json:"begin_height"`
	ToHeight uint64 `json:"to_height"`
	GroupSummary map[string]*GroupVerifySummary `json:"group_summary"`
	AverCastTime float64 `json:"aver_cast_time"`
	MaxCastTime float64 `json:"max_cast_time"`
	HeightOfMaxCastTime uint64 `json:"height_of_max_cast_time"`
}

func (s *SysWorkSummary) getGroupSummary(gid groupsig.ID, top uint64, nextSelected bool) *GroupVerifySummary {
	gidStr := gid.GetHexString()
	if v, ok := s.GroupSummary[gidStr]; ok {
		return v
	}
	gvs := &GroupVerifySummary{
		LastJumpHeight: make([]uint64, 0),
	}
	g := core.GroupChainImpl.GetGroupById(gid.Serialize())
	gvs.fillGroupInfo(g, top)
	gvs.NextSelected = nextSelected
	s.GroupSummary[gidStr] = gvs


	return gvs
}


type GroupVerifySummary struct {
	DissmissHeight uint64 `json:"dissmiss_height"`
	Dissmissed bool `json:"dissmissed"`
	NumVerify int `json:"num_verify"`
	NumJump 	int `json:"num_jump"`
	LastJumpHeight []uint64 `json:"last_jump_height"`
	NextSelected bool `json:"next_selected"`
}

func (s *GroupVerifySummary) addJumpHeight(h uint64)  {
	if len(s.LastJumpHeight) < 50 {
		s.LastJumpHeight = append(s.LastJumpHeight, h)
	} else {
		for i := 1; i < len(s.LastJumpHeight); i++ {
			if s.LastJumpHeight[i-1] > s.LastJumpHeight[i] {
				s.LastJumpHeight[i] = h
			}
		}
	}
	s.NumJump+=1
}

func (s *GroupVerifySummary) fillGroupInfo(g *types.Group, top uint64)  {
    s.DissmissHeight = g.Header.DismissHeight
    s.Dissmissed = s.DissmissHeight <= top
}

func (api *GtasAPI) DebugContextSummary() (*Result, error) {
	s := mediator.Proc.BlockContextSummary()
	return successResult(s)
}


func (api *GtasAPI) DebugVerifySummary(from, to uint64) (*Result, error) {
	if from == 0 {
		from = 1
	}
	chain := core.BlockChainImpl
	top := chain.QueryTopBlock()
	topHeight := top.Height
	if to > topHeight {
		to = topHeight
	}

	summary := &SysWorkSummary{
		BeginHeight: from,
		ToHeight: to,
		GroupSummary: make(map[string]*GroupVerifySummary, 0),
	}
	nextGroupId := *mediator.Proc.CalcVerifyGroup(top, topHeight+1)
	preBH := chain.QueryBlockByHeight(from-1)

	t := float64(0)
	b := 0
	max := float64(0)
	maxHeight := uint64(0)
	for h := uint64(from); h <= to; h++ {
		bh := chain.QueryBlockByHeight(h)
		if bh == nil {
			nextGid := mediator.Proc.CalcVerifyGroup(preBH, h)

			gvs := summary.getGroupSummary(*nextGid, topHeight, nextGid.IsEqual(nextGroupId))
			gvs.addJumpHeight(h)
		} else {
			if bh.PreHash != preBH.Hash {
				e := fmt.Sprintf("not chain! pre %+v, curr %+v\n", preBH, bh)
				fmt.Printf(e)
				return failResult(e)
			}
			if h != 1 {
				b++
				cost := bh.CurTime.Sub(preBH.CurTime).Seconds()
				t += cost
				if cost > max {
					max = cost
					maxHeight = bh.Height
				}
			}
			preBH = bh
			gid := groupsig.DeserializeId(bh.GroupId)
			gvs := summary.getGroupSummary(gid, topHeight, gid.IsEqual(nextGroupId))
			gvs.NumVerify += 1
		}

	}
	summary.AverCastTime = t / float64(b)
	summary.MaxCastTime = max
	summary.HeightOfMaxCastTime = maxHeight
	return successResult(summary)
}
