package cli

func voteConfigHelp() string {
	return `format like key=value, keys include: 
							TemplateId          string      //合约模板id
							PIndex              int         //投票参数索引
							PValue              interface{} //投票值
							Custom              bool        //'是否自定义投票合约', true时, pIndex pValue无效
							Desc                string      //描述
							DepositMin          uint64      //每个投票人最低缴纳保证金
							TotalDepositMin     uint64      //最低总保证金
							VoterCntMin         uint64      //最低参与投票人
							ApprovalDepositMin  uint64      //通过的最低保证金
							ApprovalVoterCntMin uint64      //通过的最低投票人
							DeadlineBlock       uint64      //投票截止的最高区块高度
							StatBlock           uint64      //唱票区块高度
							EffectBlock         uint64      //生效高度
							DepositGap          int         //缴纳保证金后到可以投票的间隔, 用区块高度差表示`
}
