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
	"github.com/taschain/taschain/common"
	"math"
)

/*
**  Creator: pxf
**  Date: 2018/7/27 下午7:22
**  Description:
 */

const (
	MaxGroupBlockTime   int = 10           //组铸块最大允许时间=10s
	MaxWaitBlockTime    int = 2            //广播出块前等待最大时间=2s
	ConsensusVersion        = 1            //共识版本号
	MaxUnknownBlocks        = 5            //内存保存最大不能上链的未来块（中间块没有收到）
	GroupInitMaxSeconds     = 60 * 60 * 24 //10分钟内完成初始化，否则该组失败。不再有初始化机会。(测试改成一天)
	MaxSlotSize             = 3            //每一轮slot数

	SSSSThreshold       int = 51  //1-100
	GroupMaxMembers     int = 100 //一个组最大的成员数量
	GroupMinMembers     int = 10  //一个组最大的成员数量
	MinerMaxJoinedGroup     = 5   //一个矿工最多加入的组数
	CandidatesMinRatio      = 1   //最小的候选人相对于组成员数量的倍数

	Epoch               int = 5
	GroupCreateGap          = Epoch * 2
	GroupWaitPongGap        = GroupCreateGap + Epoch*2
	GroupReadyGap           = GroupCreateGap + Epoch*6 //组准备就绪(建成组)的间隔为1个epoch
	GroupWorkGap            = GroupCreateGap + Epoch*8 //组准备就绪后, 等待可以铸块的间隔为4个epoch
	GroupWorkDuration       = Epoch * 100              //组铸块的周期为100个epoch
	GroupCreateInterval     = Epoch * 10
)

type ConsensusParam struct {
	GroupMemberMax      int
	GroupMemberMin      int
	MaxQN               int
	SSSSThreshold       int
	MaxGroupCastTime    int
	MaxWaitBlockTime    int
	MaxFutureBlock      int
	GroupInitMaxSeconds int
	Epoch               uint64
	CreateGroupInterval uint64
	MinerMaxJoinGroup   int
	CandidatesMinRatio  int
	GroupReadyGap       uint64
	GroupWorkGap        uint64
	GroupworkDuration   uint64
	GroupCreateGap      uint64
	GroupWaitPongGap    uint64
	//EffectGapAfterApply uint64	//矿工申请后，到生效的高度间隔
	PotentialProposal int //潜在提案者

	ProposalBonus uint64 //提案奖励
	PackBonus     uint64 //打包一个分红交易奖励
	VerifyBonus   uint64 //验证者总奖励

	VerifierStake uint64 //

	MaxSlotSize int
}

var Param ConsensusParam

func InitParam(cc common.SectionConfManager) {
	Param = ConsensusParam{
		GroupMemberMax:      cc.GetInt("group_member_max", GroupMaxMembers),
		GroupMemberMin:      cc.GetInt("group_member_min", GroupMinMembers),
		SSSSThreshold:       SSSSThreshold,
		MaxWaitBlockTime:    cc.GetInt("max_wait_block_time", MaxWaitBlockTime),
		MaxGroupCastTime:    cc.GetInt("max_group_cast_time", MaxGroupBlockTime),
		MaxQN:               5,
		MaxFutureBlock:      MaxUnknownBlocks,
		GroupInitMaxSeconds: GroupInitMaxSeconds,
		Epoch:               uint64(cc.GetInt("Epoch", Epoch)),
		MinerMaxJoinGroup:   cc.GetInt("miner_max_join_group", MinerMaxJoinedGroup),
		CandidatesMinRatio:  cc.GetInt("candidates_min_ratio", CandidatesMinRatio),
		GroupReadyGap:       uint64(cc.GetInt("group_ready_gap", GroupReadyGap)),
		GroupWorkGap:        uint64(cc.GetInt("group_cast_qualify_gap", GroupWorkGap)),
		GroupworkDuration:   uint64(cc.GetInt("group_cast_duration", GroupWorkDuration)),
		//EffectGapAfterApply: Epoch,
		PotentialProposal:   10,
		ProposalBonus:       common.TAS2RA(12),
		PackBonus:           common.TAS2RA(3),
		VerifyBonus:         common.TAS2RA(15),
		VerifierStake:       common.VerifyStake,
		CreateGroupInterval: uint64(GroupCreateInterval),
		GroupCreateGap:      uint64(GroupCreateGap),
		GroupWaitPongGap:    uint64(GroupWaitPongGap),
		MaxSlotSize:         MaxSlotSize,
	}
}

//取得门限值
//func  (p *ConsensusParam) GetThreshold() int {
//	return p.GetGroupK(p.GroupMemberMax)
//}

func (p *ConsensusParam) GetGroupK(max int) int {
	return int(math.Ceil(float64(max*p.SSSSThreshold) / 100))
}

//获取组成员个数
//func (p *ConsensusParam) GetGroupMemberNum() int {
//	return p.GroupMemberMax
//}
func (p *ConsensusParam) IsGroupMemberCountLegal(cnt int) bool {
	return p.GroupMemberMin <= cnt && cnt <= p.GroupMemberMax
}
func (p *ConsensusParam) CreateGroupMinCandidates() int {
	return p.GroupMemberMin * p.CandidatesMinRatio
}

func (p *ConsensusParam) CreateGroupMemberCount(availCandidates int) int {
	cnt := int(math.Ceil(float64(availCandidates / p.CandidatesMinRatio)))
	if cnt > p.GroupMemberMax {
		cnt = p.GroupMemberMax
	} else if cnt < p.GroupMemberMin {
		cnt = 0
	}
	return cnt
}
