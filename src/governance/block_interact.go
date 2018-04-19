package governance

import (
	"core"
	"governance/contract"
	"governance/param"
	"vm/core/vm"
	"common"
	"governance/global"
)

/*
**  Creator: pxf
**  Date: 2018/3/28 下午2:50
**  Description: 提供与区块链交互的方法
*/

/** 
* @Description: 根据目标名称以及输入获取真实部署的code 
* @Param:  
* @return:  
*/ 
func GetRealCode(b *core.Block, db vm.StateDB, name string, input []byte) ([]byte, error) {
	gov := global.Gov()
	
	callctx := contract.NewCallContext(b, gov.BlockChain, db)
	tc := gov.NewTemplateCodeInst(callctx)
	tp, err := tc.Template(common.StringToAddress(name))
	if err != nil {
		return nil, err
	}
	
	return append(tp.Code, input...), nil
	
}

func updateTransCredit(callctx *contract.CallContext, b *core.Block)  {
	tmp := make(map[*common.Address]uint32)

	//更新交易相关的信用信息
	credit := global.Gov().NewTasCreditInst(callctx)
	for _, tx := range b.Transactions {
		if cnt, ok := tmp[tx.Source]; ok {
			tmp[tx.Source] = cnt + 1
		} else {
			tmp[tx.Source] = 1
		}
	}

	for addr, cnt := range tmp {
		credit.AddTransCnt(*addr, cnt)
		credit.SetLatestTransBlock(*addr, b.Header.Height)
	}
}

//上链前触发
func OnInsertChain(b *core.Block, stateDB vm.StateDB) {
	ctx := global.Gov()

	callctx := contract.NewCallContext(b, ctx.BlockChain, stateDB)

	updateTransCredit(callctx, b)

	//先处理到达唱票区块的投票
	statVotes := ctx.VotePool.GetByStatBlock(b.Header.Height)
	for _, vc := range statVotes {
		vote := ctx.NewVoteInst(callctx, vc.Ref.Address())
		//调用合约函数检查投票是否通过
		pass, err := vote.CheckResult()
		if err != nil {
			//TODO: log
			continue
		}
		if pass {
			if vc.Config.Custom {
				//TODO: 自定义投票, 暂不实现处理
			} else {
				p := ctx.ParamManager.GetParamByIndex(vc.Config.PIndex)
				if p != nil {
					meta := param.NewMeta(vc.Config.PValue)
					meta.VoteAddr = vc.Ref.Address()
					meta.Block = b.Header.Height
					meta.ValidBlock = vc.Config.EffectBlock
					p.AddFuture(meta)
				}
			}
		}
		//更新信用信息
		vote.UpdateCredit(pass)

	}

	//处理生效的区块
	effectVotes := ctx.VotePool.GetByEffectBlock(b.Header.Height)
	for _, vc := range effectVotes {
		vote := ctx.NewVoteInst(callctx, vc.Ref.Address())

		//处理保证金
		err := vote.HandleDeposit()
		if err != nil {
			//TODO: log
			continue
		}
		//使参数值生效
		if vc.Config.Custom {
			//TODO: 自定义投票, 暂不实现处理
		} else {
			ctx.ParamManager.GetParamByIndex(vc.Config.PIndex).CurrentValue(ctx.ParamManager.CurrentBlockHeight())
		}
	}
}
