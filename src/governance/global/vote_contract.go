package global

import (
	"sync"
	"common"
	"governance/contract"
)

/*
**  Creator: pxf
**  Date: 2018/3/27 下午6:12
**  Description: 
*/

type VoteContract struct {
	Template *contract.TemplateCode
	Addr     common.Address
	Config   *VoteConfig
}

type VotePool struct {
	lock  sync.RWMutex
	votes []*VoteContract
}

type filter func(vc *VoteContract) bool

func NewVotePool() *VotePool {
	return &VotePool{
		votes: make([]*VoteContract, 0),
	}
}
func (v *VotePool) AddVote(vote *VoteContract) bool {
	v.lock.Lock()
	defer v.lock.Unlock()

    v.votes = append(v.votes, vote)
    return true
}

func (v *VotePool) findVote(ref common.Address) int {
	for idx, v := range v.votes {
		if v.Addr == ref {
			return idx
		}
	}
	return -1
}

func (v *VotePool) RemoveVote(ref common.Address) bool {
    v.lock.Lock()
    defer v.lock.Unlock()

	idx := v.findVote(ref)
	if idx < 0 {
		return false
	}

	if idx == len(v.votes)-1 {
		v.votes = v.votes[0:idx]
	} else {
		pre := v.votes[0:idx]
		v.votes = append(pre, v.votes[idx+1:]...)
	}
    return true
}

func (v *VotePool) subset(f filter) []*VoteContract {
	var result = make([]*VoteContract, 0)

	for _, v := range v.votes {
		if f(v) {
			result = append(result, v)
		}
	}

	return result
}

func (v *VotePool) GetByStatBlock(b uint64) []*VoteContract {
    return v.subset(func(vc *VoteContract) bool {
		return vc.Config.StatBlock == b
	})
}

func (v *VotePool) GetByEffectBlock(b uint64) []*VoteContract {
	return v.subset(func(vc *VoteContract) bool {
		return vc.Config.EffectBlock == b
	})
}

