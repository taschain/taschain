package model

import (
	"math"
	"common"
)

/*
**  Creator: pxf
**  Date: 2018/7/27 下午7:22
**  Description: 
*/

const (
	MAX_GROUP_BLOCK_TIME   int = 10           //组铸块最大允许时间=10s
	MAX_WAIT_BLOCK_TIME    int = 2            //广播出块前等待最大时间=2s
	CONSENSUS_VERSION          = 1            //共识版本号
	MAX_UNKNOWN_BLOCKS         = 5            //内存保存最大不能上链的未来块（中间块没有收到）
	GROUP_INIT_MAX_SECONDS     = 60 * 60 * 24 //10分钟内完成初始化，否则该组失败。不再有初始化机会。(测试改成一天)
	MaxSlotSize 			= 3				//每一轮slot数

	SSSS_THRESHOLD int = 51                 //1-100
	GROUP_MAX_MEMBERS int = 100             //一个组最大的成员数量
	GROUP_MIN_MEMBERS int = 10             //一个组最大的成员数量
	MINER_MAX_JOINED_GROUP = 5	//一个矿工最多加入的组数
	CANDIDATES_MIN_RATIO = 1	//最小的候选人相对于组成员数量的倍数

	EPOCH                 int = 5
	Group_Create_Gap		= EPOCH * 2
	Group_Wait_Pong_Gap 	= Group_Create_Gap + EPOCH * 2
	GROUP_Ready_GAP           = Group_Create_Gap + EPOCH * 6   //组准备就绪(建成组)的间隔为1个epoch
	GROUP_Work_GAP            = Group_Create_Gap + EPOCH * 8   //组准备就绪后, 等待可以铸块的间隔为4个epoch
	GROUP_Work_DURATION       = EPOCH * 100 //组铸块的周期为100个epoch
	Group_Create_Interval     = EPOCH * 10

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
	GroupCreateGap		uint64
	GroupWaitPongGap 	uint64
	//EffectGapAfterApply uint64	//矿工申请后，到生效的高度间隔
	PotentialProposal	int 	//潜在提案者

	ProposalBonus 		uint64	//提案奖励
	PackBonus 			uint64	//打包一个分红交易奖励
	VerifyBonus 		uint64	//验证者总奖励

	VerifierStake		uint64 //

	MaxSlotSize			int
}


var Param ConsensusParam

func InitParam(cc common.SectionConfManager) {
	Param = ConsensusParam{
		GroupMemberMax:      cc.GetInt("group_member_max", GROUP_MAX_MEMBERS),
		GroupMemberMin:      cc.GetInt("group_member_min", GROUP_MIN_MEMBERS),
		SSSSThreshold:       SSSS_THRESHOLD,
		MaxWaitBlockTime:    cc.GetInt("max_wait_block_time", MAX_WAIT_BLOCK_TIME),
		MaxGroupCastTime:    cc.GetInt("max_group_cast_time", MAX_GROUP_BLOCK_TIME),
		MaxQN:               5,
		MaxFutureBlock:      MAX_UNKNOWN_BLOCKS,
		GroupInitMaxSeconds: GROUP_INIT_MAX_SECONDS,
		Epoch:               uint64(cc.GetInt("epoch", EPOCH)),
		MinerMaxJoinGroup:   cc.GetInt("miner_max_join_group", MINER_MAX_JOINED_GROUP),
		CandidatesMinRatio:  cc.GetInt("candidates_min_ratio", CANDIDATES_MIN_RATIO),
		GroupReadyGap:       uint64(cc.GetInt("group_ready_gap", GROUP_Ready_GAP)),
		GroupWorkGap:        uint64(cc.GetInt("group_cast_qualify_gap", GROUP_Work_GAP)),
		GroupworkDuration:   uint64(cc.GetInt("group_cast_duration", GROUP_Work_DURATION)),
		//EffectGapAfterApply: EPOCH,
		PotentialProposal: 10,
		ProposalBonus: 	common.TAS2RA(12),
		PackBonus: common.TAS2RA(3),
		VerifyBonus: common.TAS2RA(15),
		VerifierStake: common.VerifyStake,
		CreateGroupInterval: uint64(Group_Create_Interval),
		GroupCreateGap: uint64(Group_Create_Gap),
		GroupWaitPongGap: uint64(Group_Wait_Pong_Gap),
		MaxSlotSize: MaxSlotSize,
	}
}

//取得门限值
//func  (p *ConsensusParam) GetThreshold() int {
//	return p.GetGroupK(p.GroupMemberMax)
//}

func  (p *ConsensusParam) GetGroupK(max int) int {
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
    cnt := int(math.Ceil(float64(availCandidates /p.CandidatesMinRatio)))
	if cnt > p.GroupMemberMax {
		cnt = p.GroupMemberMax
	} else if cnt < p.GroupMemberMin {
		cnt = 0
	}
	return cnt
}

