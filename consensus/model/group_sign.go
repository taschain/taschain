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

package model

import (
	"fmt"
	"github.com/taschain/taschain/consensus/groupsig"
	"sync"
)

type GroupSignGenerator struct {
	witnesses map[string]groupsig.Signature // Witness list
	threshold int                           // Threshold
	gSign     groupsig.Signature            // Output group signature
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

// AddWitnessForce do not check if it has been recovered, just add the shard
func (gs *GroupSignGenerator) AddWitnessForce(id groupsig.ID, signature groupsig.Signature) (add bool, generated bool) {
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

func (gs *GroupSignGenerator) AddWitness(id groupsig.ID, signature groupsig.Signature) (add bool, generated bool) {
	if gs.Recovered() {
		return false, true
	}

	return gs.AddWitnessForce(id, signature)
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
