package global

import (
	"sync"
	"common"
	"governance/contract"
)

/*
**  Creator: pxf
**  Date: 2018/3/27 下午6:12
**  Description: deprecated! use vote_addr_pool.go!
*/

type VoteContract struct {
	Template *contract.TemplateCode
	Addr     common.Address
	Config   *VoteConfig
}

type VotePool struct {
	lock  sync.RWMutex
	votes map[common.Address]*VoteContract
}

type filter func(vc *VoteContract) bool

func NewVotePool() *VotePool {
	return &VotePool{
		votes: make(map[common.Address]*VoteContract, 0),
	}
}
func (vp *VotePool) AddVote(vote *VoteContract) bool {
	vp.lock.Lock()
	defer vp.lock.Unlock()

    vp.votes[vote.Addr] = vote
    return true
}

func (vp *VotePool) RemoveVote(ref common.Address) bool {
    vp.lock.Lock()
    defer vp.lock.Unlock()

    delete(vp.votes, ref)
    return true
}

func (vp *VotePool) subset(f filter) []*VoteContract {
	var result = make([]*VoteContract, 0)

	for _, v := range vp.votes {
		if f(v) {
			result = append(result, v)
		}
	}

	return result
}

func (vp *VotePool) GetByStatBlock(b uint64) []*VoteContract {
    return vp.subset(func(vc *VoteContract) bool {
		return vc.Config.StatBlock == b
	})
}

func (vp *VotePool) GetByEffectBlock(b uint64) []*VoteContract {
	return vp.subset(func(vc *VoteContract) bool {
		return vc.Config.EffectBlock == b
	})
}

