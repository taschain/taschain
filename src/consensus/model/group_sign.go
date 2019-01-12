package model

import (
	"consensus/groupsig"
	"sync"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2018/8/17 下午2:04
**  Description:
*/

type GroupSignGenerator struct {
	witnesses map[string]groupsig.Signature //见证人列表
	threshold int                           //阈值
	gSign     groupsig.Signature            //输出的组签名
	lock      sync.RWMutex
}

func NewGroupSignGenerator(threshold int) *GroupSignGenerator {
	return &GroupSignGenerator{
		witnesses: make(map[string]groupsig.Signature, 0),
		threshold: threshold,
	}
}

func (gs *GroupSignGenerator) Threshold() int {
    return gs.threshold
}

func (gs *GroupSignGenerator) GetWitness(id groupsig.ID) (groupsig.Signature, bool) {
	gs.lock.RLock()
	defer gs.lock.RUnlock()
	if s, ok := gs.witnesses[id.GetHexString()]; ok {
		return s, true
	}
	return groupsig.Signature{}, false
}

func (gs *GroupSignGenerator) AddWitness(id groupsig.ID, signature groupsig.Signature) (add bool, generated bool) {
	if gs.Recovered() {
		return false, true
	}

	gs.lock.Lock()
	defer gs.lock.Unlock()

	key := id.GetHexString()
	if _, ok := gs.witnesses[key]; ok {
		return false, false
	}
	gs.witnesses[key] = signature

	if len(gs.witnesses) >= gs.threshold {
		return true, gs.generate()
	}
	return true, false
}

func (gs *GroupSignGenerator) generate() bool {
	if gs.gSign.IsValid() {
		return true
	}

	sig := groupsig.RecoverSignatureByMapI(gs.witnesses, gs.threshold)
	if sig == nil {
		return false
	}
	gs.gSign = *sig
	if len(gs.gSign.Serialize()) == 0 {
		//stdL("!!!!!!!!!!!!!!!!!!!!!!!!!!!1sign is empty!")
	}
	return true
}

func (gs *GroupSignGenerator) GetGroupSign() groupsig.Signature {
	gs.lock.RLock()
	defer gs.lock.RUnlock()

	return gs.gSign
}

func (gs *GroupSignGenerator) VerifyGroupSign(gpk groupsig.Pubkey, data []byte) bool {
	return groupsig.VerifySig(gpk, data, gs.GetGroupSign())
}

func (gs *GroupSignGenerator) WitnessSize() int {
	gs.lock.RLock()
	defer gs.lock.RUnlock()
	return len(gs.witnesses)
}

func (gs *GroupSignGenerator) ThresholdReached() bool {
	return gs.WitnessSize() >= gs.threshold
}

func (gs *GroupSignGenerator) Recovered() bool {
    gs.lock.RLock()
    defer gs.lock.RUnlock()
    return gs.gSign.IsValid()
}

func (gs *GroupSignGenerator) ForEachWitness(f func(id string, sig groupsig.Signature) bool) {
    gs.lock.RLock()
    defer gs.lock.RUnlock()

	for ids, sig := range gs.witnesses {
		if !f(ids, sig) {
			break
		}
	}
}

func (gs *GroupSignGenerator) Brief() string {
	return fmt.Sprintf("当前分片数%v，需分片数%v", gs.WitnessSize(), gs.threshold)
}