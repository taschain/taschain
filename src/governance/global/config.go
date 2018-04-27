package global

/*
**  Creator: pxf
**  Date: 2018/3/27 下午5:47
**  Description: 
*/


type VoteConfig struct {
	TemplateName        string  //合约模板名称
	PIndex              uint32         //投票参数索引
	PValue              string		 //投票值
	Custom              bool        //'是否自定义投票合约', true时, pIndex pValue无效
	Desc                string      //描述
	DepositMin          uint64      //每个投票人最低缴纳保证金
	TotalDepositMin     uint64      //最低总保证金
	VoterCntMin         uint64      //最低参与投票人
	ApprovalDepositMin  uint64      //通过的最低保证金
	ApprovalVoterCntMin uint64      //通过的最低投票人
	DeadlineBlock       uint64      //投票截止的最高区块高度
	StatBlock           uint64     //唱票区块高度
	EffectBlock         uint64      //生效高度
	DepositGap          uint64         //缴纳保证金后到可以投票的间隔, 用区块高度差表示
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

//对输入的config进行abi编码, 此编码借助vote_config_abi_helper.sol合约的构造函数进行编码
func (cfg *VoteConfig) AbiEncode() ([]byte, error) {
	return abiEncode(cfg)
}

func AbiDecodeConfig(data []byte) (*VoteConfig, error) {
	return abiDecode(data)
}

func (cfg *VoteConfig) convert() ([]byte, error) {
	_abi := gov.VoteContract.GetAbi()
	return _abi.Pack(
		"",
		gov.CreditContract.GetAddress(),
		cfg.DepositMin,
		cfg.TotalDepositMin,
		max(cfg.VoterCntMin, gov.VoterCntMin),
		cfg.ApprovalDepositMin,
		max(cfg.ApprovalVoterCntMin, gov.ApprovalVoterMin),
		cfg.DeadlineBlock,
		cfg.StatBlock,
		cfg.EffectBlock,
		cfg.DepositGap,
		gov.VoteScoreMin,
		gov.LaunchVoteScoreMin,
	)
}
