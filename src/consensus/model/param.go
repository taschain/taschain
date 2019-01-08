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
	MAX_WAIT_BLOCK_TIME    int = 1            //广播出块前等待最大时间=2s
	CONSENSUS_VERSION          = 1            //共识版本号
	MAX_UNKNOWN_BLOCKS         = 5            //内存保存最大不能上链的未来块（中间块没有收到）
	GROUP_INIT_MAX_SECONDS     = 60 * 60 * 24 //10分钟内完成初始化，否则该组失败。不再有初始化机会。(测试改成一天)

	SSSS_THRESHOLD int = 51                 //1-100
	GROUP_MAX_MEMBERS int = 100             //一个组最大的成员数量
	MINER_MAX_JOINED_GROUP = 5	//一个矿工最多加入的组数
	CANDIDATES_MIN_RATIO = 1	//最小的候选人相对于组成员数量的倍数

	EPOCH int = 4
	GROUP_GET_READY_GAP = EPOCH * 3	//组准备就绪(建成组)的间隔为1个epoch
	GROUP_CAST_QUALIFY_GAP = EPOCH * 5	//组准备就绪后, 等待可以铸块的间隔为4个epoch
	GROUP_CAST_DURATION = EPOCH * 100	//组铸块的周期为100个epoch

)

type ConsensusParam struct {
	GroupMember int
	MaxQN 	int
	SSSSThreshold int
	MaxGroupCastTime int
	MaxWaitBlockTime int
	MaxFutureBlock int
	GroupInitMaxSeconds int
	Epoch	uint64
	CreateGroupInterval uint64
	MinerMaxJoinGroup int
	CandidatesMinRatio int
	GroupGetReadyGap uint64
	GroupCastQualifyGap uint64
	GroupCastDuration	uint64
	//EffectGapAfterApply uint64	//矿工申请后，到生效的高度间隔
	PotentialProposal	int 	//潜在提案者

	ProposalBonus 		uint64	//提案奖励
	PackBonus 			uint64	//打包一个分红交易奖励
	VerifyBonus 		uint64	//验证者总奖励
}


var Param ConsensusParam

func InitParam(cc common.SectionConfManager) {
	Param = ConsensusParam{
		GroupMember: cc.GetInt("group_member", GROUP_MAX_MEMBERS),
		SSSSThreshold: SSSS_THRESHOLD,
		MaxWaitBlockTime: cc.GetInt("max_wait_block_time", MAX_WAIT_BLOCK_TIME),
		MaxGroupCastTime: cc.GetInt("max_group_cast_time", MAX_GROUP_BLOCK_TIME),
		MaxQN: 5,
		MaxFutureBlock: MAX_UNKNOWN_BLOCKS,
		GroupInitMaxSeconds: GROUP_INIT_MAX_SECONDS,
		Epoch: uint64(cc.GetInt("epoch", EPOCH)),
		MinerMaxJoinGroup: cc.GetInt("miner_max_join_group", MINER_MAX_JOINED_GROUP),
		CandidatesMinRatio: cc.GetInt("candidates_min_ratio", CANDIDATES_MIN_RATIO),
		GroupGetReadyGap: uint64(cc.GetInt("group_ready_gap", GROUP_GET_READY_GAP)),
		GroupCastQualifyGap: uint64(cc.GetInt("group_cast_qualify_gap", GROUP_CAST_QUALIFY_GAP)),
		GroupCastDuration: uint64(cc.GetInt("group_cast_duration", GROUP_CAST_DURATION)),
		//EffectGapAfterApply: EPOCH,
		PotentialProposal: 5,
		ProposalBonus: 	50,
		PackBonus: 10,
		VerifyBonus: 100,
	}
	Param.CreateGroupInterval = Param.GroupCastQualifyGap + Param.GroupGetReadyGap
}

//取得门限值
func  (p *ConsensusParam) GetThreshold() int {
	return int(math.Ceil(float64(p.GroupMember*p.SSSSThreshold) / 100))
}

func  (p *ConsensusParam) GetGroupK(max int) int {
	return int(math.Ceil(float64(max*p.SSSSThreshold) / 100))
}

//获取组成员个数
func (p *ConsensusParam) GetGroupMemberNum() int {
	return p.GroupMember
}

func (p *ConsensusParam) CreateGroupMinCandidates() int {
    return p.GroupMember * p.CandidatesMinRatio
}

