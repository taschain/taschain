package governance

import (
	"core"
	"governance/contract"
	"governance/param"
	"vm/core/vm"
	"common"
	"governance/global"
)

/** 
* @Description: 根据目标名称以及输入获取真实部署的code 
* @Param:  
* @return:  
*/ 
func GetRealCode(b *core.Block, db vm.StateDB, name string, input []byte) ([]byte, error) {
	gov := global.GetGOV()
	
	callctx := contract.NewCallContext(b, gov.BlockChain, db)
	tc := gov.NewTemplateCodeInst(callctx)
	tp, err := tc.Template(name)
	if err != nil {
		return nil, err
	}

	var convert []byte
	convert, err = global.ConvertToVoteAbi(input)
	if err != nil {
		return nil, err
	}
	
	return append(tp.Code, convert...), nil
	
}

//更新本区块里每个账号的交易相关的信用信息
func updateTransCredit(callctx *contract.CallContext, b *core.Block)  {
	tmp := make(map[*common.Address]uint32)

	//更新交易相关的信用信息
	credit := global.GetGOV().NewTasCreditInst(callctx)
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

//检查到达唱票区块的投票
func checkStatVotes(callctx *contract.CallContext, b *core.Block) {
	ctx := global.GetGOV()

	statVotes := ctx.VotePool.GetByStatBlock(b.Header.Height)
	for _, vc := range statVotes {
		vote := ctx.NewVoteInst(callctx, vc.Addr)
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
				p := ctx.ParamManager.GetParamByIndex(int(vc.Config.PIndex))
				if p != nil {
					meta := param.NewMeta(p.Convertor(vc.Config.PValue))
					meta.VoteAddr = vc.Addr
					meta.Block = b.Header.Height
					meta.ValidBlock = vc.Config.EffectBlock
					p.AddFuture(meta)
				}
			}
		}
		//更新信用信息
		vote.UpdateCredit(pass)

	}
}

//处理那些已经到达生效区块高度的投票的保证金, 以及参数正式生效
func checkEffectVotes(callctx *contract.CallContext, b *core.Block) {
	gov := global.GetGOV()

	effectVotes := gov.VotePool.GetByEffectBlock(b.Header.Height)
	for _, vc := range effectVotes {
		vote := gov.NewVoteInst(callctx, vc.Addr)

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
			gov.ParamManager.GetParamByIndex(int(vc.Config.PIndex)).CurrentValue(gov.ParamManager.CurrentBlockHeight())
		}

		//移除投票信息
		gov.VotePool.RemoveVote(vc.Addr)
	}
}

func isVote(tx *core.Transaction) bool {
	return tx.ExtraData != nil
}

func voteAddr(tx *core.Transaction) common.Address {
	return common.BytesToAddress(tx.ExtraData)
}

func handleVoteCreate(b *core.Block) {
	gov := global.GetGOV()

	for _, tx := range b.Transactions {
		if isVote(tx) {
			cfg, err := global.AbiDecodeConfig(tx.Data)
			if err != nil {
				//TODO: log
				continue
			}
			v := &global.VoteContract{
				Addr: voteAddr(tx),
				Config: cfg,
			}
			gov.VotePool.AddVote(v)
		}
	}
}

//上链前触发
func OnInsertChain(b *core.Block, stateDB vm.StateDB) {
	ctx := global.GetGOV()

	//处理投票部署
	handleVoteCreate(b)

	//生成智能合约调用的上下文
	callctx := contract.NewCallContext(b, ctx.BlockChain, stateDB)

	//更新交易相关的信用信息
	updateTransCredit(callctx, b)

	//唱票
	checkStatVotes(callctx, b)

	//生效
	checkEffectVotes(callctx, b)
}
