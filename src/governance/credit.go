package governance

/*
**  Creator: pxf
**  Date: 2018/3/24 下午4:27
*/

import (
	"common"
	"time"
)

const ACCOUNT_VERSION = 1

const (
	MINER_WEIGHT 	= 10
	BALANCE_WEIGHT 	= 5
	TRANS_WEIGHT 	= 3
	VOTE_WEIGHT 	= 1
	VOTE_ACCEPT_WEIGHT 	= 3
)


type AccountCredit struct {
	Account        common.Address //账户地址
	Miner          bool           //是否矿工
	Balance        uint64       //账户余额, tas币表示
	TransCnt       uint32         //交易次数
	GmtLatestTrans time.Time      //最近交易时间戳
	VoteCnt        uint32         //投票次数
	VoteAcceptCnt  uint32         //被接受的投票次数
	BlockNumber    uint64     //当前信用统计时的区块高度
	Version        uint32         //版本号
}

func NewAccountCredit(ac common.Address) *AccountCredit {
	return &AccountCredit{
		Account:       ac,
		Miner:         false,
		Balance:       0,
		TransCnt:      0,
		VoteCnt:       0,
		VoteAcceptCnt: 0,
		Version:       ACCOUNT_VERSION,
	}
}


/** 
* @Description: 计算账户信用的最终得分 todo: 算法待定
* @Param:  
* @return:  
*/
func (ac *AccountCredit) CalculateScore() uint64 {
	score := uint64(0)
	if ac.Miner {
		score += MINER_WEIGHT
	}
	score += uint64(BALANCE_WEIGHT) * ac.Balance
	score += uint64(uint32(TRANS_WEIGHT) * ac.TransCnt)

	v := float64(ac.TransCnt >> 1) / 365
	score -= uint64(time.Now().Sub(ac.GmtLatestTrans).Hours() / 24 * v)

	score += uint64(uint32(VOTE_WEIGHT) * ac.VoteCnt)
	score += uint64(uint32(VOTE_ACCEPT_WEIGHT) * ac.VoteAcceptCnt)

	return score
}


