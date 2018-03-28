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
	//TODO: 继承普通合约
	//Contract
	Config VoteConfig
}

type VotePool struct {
	mu	sync.Mutex
	votes	[]*VoteContract
}

func (v *VotePool) AddVoteContract(vote *VoteContract) {
	v.mu.Lock()
	defer v.mu.Unlock()

    v.votes = append(v.votes, vote)
}

func (v *VotePool) StoreVoteContract(vote *VoteContract)  {
    v.AddVoteContract(vote)
    //TODO: 持久化
}

func (v *VotePool) GetVoteContract(ref ContractRef) *VoteContract  {
	//TODO: 根据地址查找合约
    return nil
}