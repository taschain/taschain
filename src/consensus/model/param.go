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
	MAX_GROUP_BLOCK_TIME int = 10                                //组铸块最大允许时间=10s
	MAX_USER_CAST_TIME int = 2                                   //个人出块最大允许时间=2s
	MAX_CAST_SLOT	= 3						//最大验证槽个数
	CONSENSUS_VERSION = 1                 //共识版本号
	MAX_UNKNOWN_BLOCKS = 5                //内存保存最大不能上链的未来块（中间块没有收到）
	INVALID_QN = -1                       //无效的队列序号
	GROUP_INIT_MAX_SECONDS = 60 * 60 * 24 //10分钟内完成初始化，否则该组失败。不再有初始化机会。(测试改成一天)

	SSSS_THRESHOLD int = 51                 //1-100
	GROUP_MAX_MEMBERS int = 20             //一个组最大的成员数量
	MINER_MAX_JOINED_GROUP = 5	//一个矿工最多加入的组数
	CANDIDATES_MIN_RATIO = 1	//最小的候选人相对于组成员数量的倍数

	EPOCH uint64 = 8
	GROUP_GET_READY_GAP = EPOCH * 3	//组准备就绪(建成组)的间隔为1个epoch
	GROUP_CAST_QUALIFY_GAP = EPOCH * 5	//组准备就绪后, 等待可以铸块的间隔为4个epoch
	GROUP_CAST_DURATION = EPOCH * 100	//组铸块的周期为100个epoch

)

type ConsensusParam struct {
	GroupMember int
	MaxQN 	int
	SSSSThreshold int
	MaxUserCastTime int
	MaxGroupCastTime int
	MaxFutureBlock int
	GroupInitMaxSeconds int
	Epoch	uint64
	CreateGroupInterval uint64
	MinerMaxJoinGroup int
	CandidatesMinRatio int
	GroupGetReadyGap uint64
	GroupCastQualifyGap uint64
	GroupCastDuration	uint64
}

var Param ConsensusParam

func InitParam() {
	cc := common.GlobalConf.GetSectionManager("consensus")
	Param = ConsensusParam{
		GroupMember: cc.GetInt("GROUP_MAX_MEMBERS", GROUP_MAX_MEMBERS),
		SSSSThreshold: cc.GetInt("SSSS_THRESHOLD", SSSS_THRESHOLD),
		MaxUserCastTime: cc.GetInt("MAX_USER_CAST_TIME", MAX_USER_CAST_TIME),
		MaxGroupCastTime: MAX_GROUP_BLOCK_TIME,
		MaxQN: (MAX_GROUP_BLOCK_TIME) / MAX_USER_CAST_TIME,
		MaxFutureBlock: MAX_UNKNOWN_BLOCKS,
		GroupInitMaxSeconds: GROUP_INIT_MAX_SECONDS,
		Epoch: EPOCH,
		MinerMaxJoinGroup: MINER_MAX_JOINED_GROUP,
		CandidatesMinRatio: CANDIDATES_MIN_RATIO,
		GroupGetReadyGap: GROUP_GET_READY_GAP,
		GroupCastQualifyGap: GROUP_CAST_QUALIFY_GAP,
		GroupCastDuration: GROUP_CAST_DURATION,
	}
	Param.MaxQN = Param.MaxGroupCastTime / Param.MaxUserCastTime
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
