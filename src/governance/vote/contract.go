package vote

import (
	"sync"
	"common"
)

/*
**  Creator: pxf
**  Date: 2018/3/27 下午6:12
**  Description: 
*/

type ContractRef common.Address

type VoteContract struct {
	template *VoteTemplate
	addr common.Address
	config *VoteConfig
}

type VotePool struct {
	mu	sync.Mutex
	votes map[common.Address]*VoteContract
}

func (v *VotePool) AddVoteContract(vote *VoteContract) {
	v.mu.Lock()
	defer v.mu.Unlock()

    v.votes[vote.addr] = vote
}

