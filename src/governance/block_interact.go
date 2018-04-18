package governance

import (
	"core"
	"governance/contract"
	"governance/param"
)

/*
**  Creator: pxf
**  Date: 2018/3/28 下午2:50
**  Description: 提供与区块链交互的方法
*/

/**
* @Description: 返回当前区块高度
* @Param:
* @return:  uint64
*/
func CurrentBlock() uint64 {
	return 2
}

func OnVoteDeployed() {

}

//上链时调用
func OnInsertChain(ctx *GovContext, b *core.Block) {
	//先处理到达唱票区块的投票
	statVotes := ctx.VotePool.GetByStatBlock(b.Header.Height)
	for _, vc := range statVotes {
		vote, err := contract.NewVote(vc.Ref.Address())
		if err != nil {
			//TODO: log
			continue
		}
		var pass bool
		//调用合约函数检查投票是否通过
		pass, err = vote.CheckResult()
		if err != nil {
			//TODO: log
			continue
		}
		if pass {
			if vc.Config.Custom {
				//TODO: 自定义投票, 暂不实现处理
			} else {
				p := ctx.pm.GetParamByIndex(vc.Config.PIndex)
				if p != nil {
					meta := param.NewMeta(vc.Config.PValue)
					meta.VoteAddr = vc.Ref.Address()
					meta.Block = b.Header.Height
					meta.ValidBlock = vc.Config.EffectBlock
					p.AddFuture(meta)
				}
			}
		}
	}

	//处理生效的区块
	effectVotes := ctx.VotePool.GetByEffectBlock(b.Header.Height)
	for _, vc := range effectVotes {
		vote, err := contract.NewVote(vc.Ref.Address())
		if err != nil {
			//TODO: log
			continue
		}
		//处理保证金
		err = vote.HandleDeposit()
		if err != nil {
			//TODO: log
			continue
		}
		//使参数值生效
		if vc.Config.Custom {
			//TODO: 自定义投票, 暂不实现处理
		} else {
			ctx.pm.GetParamByIndex(vc.Config.PIndex).CurrentValue()
		}
	}
}
