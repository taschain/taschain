package cli

import (
	"consensus/mediator"
	"core"
	"consensus/groupsig"
	"fmt"
	"middleware/types"
	"sort"
	"consensus/base"
	"common"
	"math/big"
	"consensus/logical"
)

/*
**  Creator: pxf
**  Date: 2019/1/11 下午1:39
**  Description: 
*/

type SysWorkSummary struct {
	BeginHeight         uint64                `json:"begin_height"`
	ToHeight            uint64                `json:"to_height"`
	GroupSummary        []*GroupVerifySummary `json:"group_summary"`
	AverCastTime        float64               `json:"aver_cast_time"`
	MaxCastTime         float64               `json:"max_cast_time"`
	HeightOfMaxCastTime uint64                `json:"height_of_max_cast_time"`
	JumpRate            float64               `json:"jump_rate"`
	summaryMap          map[string]*GroupVerifySummary
	allGroup            map[string]*types.Group
}

func (s *SysWorkSummary) Len() int {
	return len(s.GroupSummary)
}

func (s *SysWorkSummary) Less(i, j int) bool {
	return s.GroupSummary[i].DissmissHeight < s.GroupSummary[j].DissmissHeight
}

func (s *SysWorkSummary) Swap(i, j int) {
	s.GroupSummary[i], s.GroupSummary[j] = s.GroupSummary[j], s.GroupSummary[i]
}

func (s *SysWorkSummary) sort() {
	if len(s.summaryMap) == 0 {
		return
	}
	tmp := make([]*GroupVerifySummary, 0)
	for _, s := range s.summaryMap {
		tmp = append(tmp, s)
	}
	s.GroupSummary = tmp
	sort.Sort(s)
}

type groupArray []*types.Group

func (g groupArray) Len() int {
	return len(g)
}

func (g groupArray) Less(i, j int) bool {
	return g[i].Header.WorkHeight < g[j].Header.WorkHeight
}

func (g groupArray) Swap(i, j int) {
	g[i], g[j] = g[j], g[i]
}

func (s *SysWorkSummary) getGroupSummary(gid groupsig.ID, top uint64, nextSelected bool) *GroupVerifySummary {
	gidStr := gid.GetHexString()
	if v, ok := s.summaryMap[gidStr]; ok {
		return v
	}
	gvs := &GroupVerifySummary{
		Gid:            gidStr,
		LastJumpHeight: make([]uint64, 0),
	}
	g := s.allGroup[gidStr]
	gvs.fillGroupInfo(g, top)
	gvs.NextSelected = nextSelected
	s.summaryMap[gidStr] = gvs

	return gvs
}

type GroupVerifySummary struct {
	Gid            string   `json:"gid"`
	DissmissHeight uint64   `json:"dissmiss_height"`
	Dissmissed     bool     `json:"dissmissed"`
	NumVerify      int      `json:"num_verify"`
	NumJump        int      `json:"num_jump"`
	LastJumpHeight []uint64 `json:"last_jump_height"`
	NextSelected   bool     `json:"next_selected"`
	JumpRate       float64  `json:"jump_rate"`
}

func (s *GroupVerifySummary) addJumpHeight(h uint64) {
	if len(s.LastJumpHeight) < 50 {
		s.LastJumpHeight = append(s.LastJumpHeight, h)
	} else {
		find := false
		for i := 1; i < len(s.LastJumpHeight); i++ {
			if s.LastJumpHeight[i-1] > s.LastJumpHeight[i] {
				s.LastJumpHeight[i] = h
				find = true
				break
			}
		}
		if !find {
			s.LastJumpHeight[0] = h
		}
	}
	s.NumJump += 1
}

func (s *GroupVerifySummary) calJumpRate() {
	s.JumpRate = float64(s.NumJump) / float64(s.NumVerify+s.NumJump)
}

func (s *GroupVerifySummary) fillGroupInfo(g *types.Group, top uint64) {
	if g == nil {
		return
	}
	s.DissmissHeight = g.Header.DismissHeight
	s.Dissmissed = s.DissmissHeight <= top
}

func (api *GtasAPI) DebugContextSummary() (*Result, error) {
	s := mediator.Proc.BlockContextSummary()
	return successResult(s)
}

func getAllGroup() map[string]*types.Group {
	iterator := mediator.Proc.GroupChain.NewIterator()
	gs := make(map[string]*types.Group)
	for coreGroup := iterator.Current(); coreGroup != nil; coreGroup = iterator.MovePre() {
		id := groupsig.DeserializeId(coreGroup.Id)
		gs[id.GetHexString()] = coreGroup
	}

	return gs
}

func selectNextVerifyGroup(gs map[string]*types.Group, preBH *types.BlockHeader, deltaHeight uint64) (groupsig.ID, []*types.Group) {
	qualifiedGs := make(groupArray, 0)
	h := preBH.Height + deltaHeight
	for _, g := range gs {
		if logical.IsGroupWorkQualifiedAt(g.Header, h) {
			qualifiedGs = append(qualifiedGs, g)
		}
	}
	sort.Sort(qualifiedGs)

	var hash common.Hash
	data := preBH.Random
	for ; deltaHeight > 0; deltaHeight-- {
		hash = base.Data2CommonHash(data)
		data = hash.Bytes()
	}
	value := hash.Big()
	index := value.Mod(value, big.NewInt(int64(len(qualifiedGs))))
	gid := qualifiedGs[index.Int64()].Id
	return groupsig.DeserializeId(gid), qualifiedGs
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

	allGroup := getAllGroup()

	summary := &SysWorkSummary{
		BeginHeight: from,
		ToHeight:    to,
		summaryMap:  make(map[string]*GroupVerifySummary, 0),
		allGroup:    allGroup,
	}
	nextGroupId, _ := selectNextVerifyGroup(allGroup, top, 1)
	preBH := chain.QueryBlockByHeight(from - 1)

	t := float64(0)
	b := 0
	max := float64(0)
	maxHeight := uint64(0)
	jump := 0
	for h := uint64(from); h <= to; h++ {
		bh := chain.QueryBlockByHeight(h)
		if bh == nil {
			expectGid, _ := selectNextVerifyGroup(allGroup, preBH, h-preBH.Height)
			gvs := summary.getGroupSummary(expectGid, topHeight, expectGid.IsEqual(nextGroupId))
			gvs.addJumpHeight(h)
			jump++
		} else {
			if preBH == nil {
				preBH = chain.QueryBlockHeaderByHash(bh.PreHash)
			}
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
			//expectGid, gs := selectNextVerifyGroup(allGroup, preBH, h-preBH.Height)
			gid := groupsig.DeserializeId(bh.GroupId)
			//if !expectGid.IsEqual(gid) {
			//	fmt.Printf("bh %+v\n", bh)
			//	fmt.Printf("pre %+v\n", preBH)
			//	for _, g := range gs {
			//		fmt.Printf("g workheight=%v, id=%v, pre=%v\n", g.Header.WorkHeight, groupsig.DeserializeId(g.Id).ShortS(), groupsig.DeserializeId(g.Header.PreGroup))
			//	}
			//	return failResult(fmt.Sprintf("expect gid not equal, height=%v, expect %v, real %v", bh.Height, expectGid.GetHexString(), gid.GetHexString()))
			//}
			preBH = bh
			gvs := summary.getGroupSummary(gid, topHeight, gid.IsEqual(nextGroupId))
			gvs.NumVerify += 1
		}

	}
	summary.AverCastTime = t / float64(b)
	summary.MaxCastTime = max
	summary.HeightOfMaxCastTime = maxHeight
	summary.sort()
	summary.JumpRate = float64(jump) / float64(to-from+1)
	for _, v := range summary.GroupSummary {
		v.calJumpRate()
	}
	return successResult(summary)
}

func (api *GtasAPI) DebugJoinGroupInfo(gid string) (*Result, error) {
	jg := mediator.Proc.GetJoinGroupInfo(gid)
	return successResult(jg)
}

func (api *GtasAPI) DebugRemoveBlock(h uint64) (*Result, error) {
	bh := core.BlockChainImpl.QueryBlockByHeight(h)
	if bh != nil {
		b := core.BlockChainImpl.QueryBlockByHash(bh.Hash)
		if b != nil {
			ret := mediator.Proc.MainChain.Remove(b)
			return successResult(ret)
		}
	}
	return successResult("not exist")
}

func (api *GtasAPI) DebugGetTxs(limit int) (*Result, error) {
    txs := core.BlockChainImpl.GetTransactionPool().GetReceived()

    hashs := make([]string, 0)
	for _, tx := range txs {
		hashs = append(hashs, tx.Hash.String())
		if len(hashs) >= limit {
			break
		}
	}
	return successResult(hashs)
}

func (api *GtasAPI) DebugGetBonusTxs(limit int) (*Result, error) {
	txs := core.BlockChainImpl.GetTransactionPool().GetBonusTxs()

	type bonusTxHash struct {
		TxHash, BlockHash common.Hash
	}

	hashs := make([]*bonusTxHash, 0)
	for _, tx := range txs {
		btx := &bonusTxHash{
			TxHash: tx.Hash,
			BlockHash: common.BytesToHash(tx.Data),
		}
		hashs = append(hashs, btx)
		if len(hashs) >= limit {
			break
		}
	}
	return successResult(hashs)
}