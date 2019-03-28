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
	"common"
	"consensus/base"
	"consensus/groupsig"
	"consensus/logical"
	"consensus/model"
	"fmt"
	"math/big"
	"middleware/types"
	"math"
)

///////////////////////////////////////////////////////////////////////////////
/*
//主链提供给共识模块的接口
//根据哈希取得某个交易
//h:交易的哈希; forced:如本地不存在是否要发送异步网络请求
//int=0，返回合法的交易；=-1，交易异常；=1，本地不存在，已发送网络请求；=2，本地不存在
type GetTransactionByHash func(h common.Hash, forced bool) (int, core.Transaction)

//构建一个铸块（组内当前铸块人同步操作）
//to do : 细化入参(铸块人，QN，盐)
type CastingBlock func() (core.Block, error)

//验证一个铸块（如本地缺少交易，则异步网络请求该交易）
//返回:=0, 验证通过；=-1，验证失败；=1，缺少交易，已发送网络请求
type VerifyCastingBlock func(bh core.BlockHeader) int

//铸块成功，上链
//返回:=0,上链成功；=-1，验证失败；=1,上链成功，上链过程中发现分叉并进行了权重链调整
type AddBlockOnChain func(b core.Block) int

//查询最高块
type QueryTopBlock func() core.BlockHeader

//根据指定哈希查询块，不存在则返回nil。
type QueryBlockHeaderByHash func() *core.BlockHeader

//根据指定高度查询块，不存在则返回nil。
type QueryBlockByHeight func() *core.BlockHeader

//加载组信息
//to do :
*/
///////////////////////////////////////////////////////////////////////////////
//共识模块提供给外部的数据

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
	return base.VRF_proof2hash(base.VRFProve(prove)).Big()
}

func (helper *ConsensusHelperImpl) CalculateQN(bh *types.BlockHeader) uint64 {
	return Proc.CalcBlockHeaderQN(bh)
}

func (helper *ConsensusHelperImpl) CheckProveRoot(bh *types.BlockHeader) (bool, error) {
	return Proc.CheckProveRoot(bh)
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
	sign_bytes := tx.Sign
	if len(sign_bytes) < common.SignLength {
		return false, fmt.Errorf("not enough bytes for bonus signature, sign =%v", sign_bytes)
	}
	groupID, _, _, _ := Proc.MainChain.GetBonusManager().ParseBonusTransaction(tx)
	group :=Proc.GroupChain.GetGroupById(groupID)
	if group == nil {
		return false, fmt.Errorf("VerifyBonusTransaction fail, Can't get groupinfo(gid=%x)", groupID)
	}
	gpk := groupsig.DeserializePubkeyBytes(group.PubKey)
	//AcceptRewardPiece Function store groupsig in common sign buff, here will recover the groupsig
	gsign := groupsig.DeserializeSign(sign_bytes[0:33]) //size of groupsig == 33
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
    castTime := bh.CurTime.Sub(bh.PreTime).Seconds()
    return height - uint64(math.Ceil(castTime/float64(model.Param.MaxGroupCastTime)))
}