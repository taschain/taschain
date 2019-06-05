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

package mediator

import (
	"fmt"
	"math"
	"math/big"

	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/base"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/consensus/logical"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/middleware/types"
)

// ConsensusHelperImpl is consensus module provides external data
type ConsensusHelperImpl struct {
	ID groupsig.ID
}

func NewConsensusHelper(id groupsig.ID) types.ConsensusHelper {
	return &ConsensusHelperImpl{ID: id}
}

func (helper *ConsensusHelperImpl) ProposalBonus() *big.Int {
	return new(big.Int).SetUint64(model.Param.ProposalBonus)
}

func (helper *ConsensusHelperImpl) PackBonus() *big.Int {
	return new(big.Int).SetUint64(model.Param.PackBonus)
}

func (helper *ConsensusHelperImpl) GenerateGenesisInfo() *types.GenesisInfo {
	return logical.GenerateGenesis()
}

func (helper *ConsensusHelperImpl) VRFProve2Value(prove []byte) *big.Int {
	if len(prove) == 0 {
		return big.NewInt(0)
	}
	return base.VRFProof2hash(base.VRFProve(prove)).Big()
}

func (helper *ConsensusHelperImpl) CalculateQN(bh *types.BlockHeader) uint64 {
	return Proc.CalcBlockHeaderQN(bh)
}

func (helper *ConsensusHelperImpl) CheckProveRoot(bh *types.BlockHeader) (bool, error) {
	// No longer check when going up, only check at consensus
	return true, nil
}

func (helper *ConsensusHelperImpl) VerifyNewBlock(bh *types.BlockHeader, preBH *types.BlockHeader) (bool, error) {
	return Proc.VerifyBlock(bh, preBH)
}

func (helper *ConsensusHelperImpl) VerifyBlockHeader(bh *types.BlockHeader) (bool, error) {
	return Proc.VerifyBlockHeader(bh)
}

func (helper *ConsensusHelperImpl) CheckGroup(g *types.Group) (ok bool, err error) {
	return Proc.VerifyGroup(g)
}

func (helper *ConsensusHelperImpl) VerifyBonusTransaction(tx *types.Transaction) (ok bool, err error) {
	signBytes := tx.Sign
	if len(signBytes) < common.SignLength {
		return false, fmt.Errorf("not enough bytes for bonus signature, sign =%v", signBytes)
	}
	groupID, _, _, _ := Proc.MainChain.GetBonusManager().ParseBonusTransaction(tx)
	group := Proc.GroupChain.GetGroupByID(groupID)
	if group == nil {
		return false, common.ErrGroupNil
	}
	gpk := groupsig.DeserializePubkeyBytes(group.PubKey)
	gsign := groupsig.DeserializeSign(signBytes[0:33]) //size of groupsig == 33
	if !groupsig.VerifySig(gpk, tx.Hash.Bytes(), *gsign) {
		return false, fmt.Errorf("verify bonus sign fail, gsign=%v", gsign.GetHexString())
	}
	return true, nil
}

func (helper *ConsensusHelperImpl) EstimatePreHeight(bh *types.BlockHeader) uint64 {
	height := bh.Height
	if height == 1 {
		return 0
	}
	return height - uint64(math.Ceil(float64(bh.Elapsed)/float64(model.Param.MaxGroupCastTime)))
}
