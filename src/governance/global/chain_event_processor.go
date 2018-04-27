package global

import (
	"core"
	"vm/core/vm"
	"vm/core/types"
	"governance/contract"
	"common"
	"governance/util"
)

/*
**  Creator: pxf
**  Date: 2018/4/24 下午5:05
**  Description: 
*/

type ChainEventProcessorImpl struct {
	
}

func NewChainEventProcessor() *ChainEventProcessorImpl {
	return &ChainEventProcessorImpl{}
}

func (p *ChainEventProcessorImpl) AfterAllTransactionExecuted(b *core.Block, stateDB vm.StateDB, receipts types.Receipts) error {
	onInsertChain(b, stateDB, &receipts)
	return nil
}

func (p *ChainEventProcessorImpl) BeforeExecuteTransaction(b *core.Block, db vm.StateDB, tx *core.Transaction) ([]byte, error) {
	if !isVote(tx) {
		return tx.Data, nil
	}
	return getRealCode(b, db, tx.Data)
}




/** 
* @Description: 根据目标名称以及输入获取真实部署的code 
* @Param:  
* @return:  
*/
func getRealCode(b *core.Block, db vm.StateDB, input []byte) ([]byte, error) {
	gov := GetGOV()

	callctx := contract.NewCallContext(b.Header, gov.BlockChain, db)
	tc := gov.NewTemplateCodeInst(callctx)

	cfg, err := AbiDecodeConfig(input)
	if err != nil {
		return nil, nil
	}

	tp, err := tc.Template(cfg.TemplateName)
	if err != nil {
		return nil, err
	}

	var convert []byte
	convert, err = cfg.convert()
	if err != nil {
		return nil, err
	}

	return append(tp.Code, convert...), nil

}

//更新本区块里每个账号的交易相关的信用信息
func updateTransCredit(callctx *contract.CallContext, b *core.Block)  {
	tmp := make(map[*common.Address]uint32)

	//更新交易相关的信用信息
	credit := GetGOV().NewTasCreditInst(callctx)
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

func getContractAddressByTxHash(hash common.Hash) common.Address {
	if receipt := GetGOV().BlockChain.GetTransactionPool().GetExecuted(hash); receipt != nil {
		return util.ToTASAddress(receipt.Receipt.ContractAddress)
	}
	return common.Address{}
}

//检查到达唱票区块的投票
func checkStatVotes(callctx *contract.CallContext, b *core.Block) {
	ctx := GetGOV()

	vap := gov.NewVoteAddrPoolInst(callctx)
	hs, err := vap.GetCurrentStatVoteHashes()
	if err != nil {
		gov.Logger.Error("get current stat vote hash err", b.Header.Height, err)
	}

	paramStore := gov.NewParamStoreInst(callctx)

	for _, h := range hs {
		tx, err := ctx.BlockChain.GetTransactionByHash(h)
		if err != nil || tx == nil {
			gov.Logger.Error("get transaction fail", h, err)
			continue
		}

		cfg, err := AbiDecodeConfig(tx.Data)
		if err != nil {
			gov.Logger.Error("decode tx data fail", err)
			continue
		}

		addr := getContractAddressByTxHash(h)

		vote := ctx.NewVoteInst(callctx, addr)
		//调用合约函数检查投票是否通过
		pass, err := vote.CheckResult()
		if err != nil {
			ctx.Logger.Warnf("check vote result fail, voteAddr %v, err %v", addr, err.Error())
			continue
		}
		if pass {
			if cfg.Custom {
				//TODO: 自定义投票, 暂不实现处理
			} else {
				meta := &contract.ParamMeta{
					Value: cfg.PValue,
					TxHash: h,
					EffectBlock: cfg.EffectBlock,
				}
				ctx.ParamWrapper.addFuture(cfg.PIndex, paramStore, meta)
			}
		}
		//更新信用信息
		err = vote.UpdateCredit()
		if err != nil {
			gov.Logger.Warn("update credit fail", err)
		}

	}
}

//处理那些已经到达生效区块高度的投票的保证金, 以及参数正式生效
func checkEffectVotes(callctx *contract.CallContext, b *core.Block) {
	gov := GetGOV()

	vap := gov.NewVoteAddrPoolInst(callctx)
	hs, err := vap.GetCurrentEffectVoteHashes()
	if err != nil {
		gov.Logger.Error("get current effect vote hash err", b.Header.Height, err)
	}

	paramStore := gov.NewParamStoreInst(callctx)

	for _, h := range hs {
		tx, err := gov.BlockChain.GetTransactionByHash(h)
		if err != nil || tx == nil {
			gov.Logger.Error("get transaction fail", h, err)
			continue
		}

		cfg, err := AbiDecodeConfig(tx.Data)
		if err != nil {
			gov.Logger.Error("decode tx data fail", err)
			continue
		}

		addr := getContractAddressByTxHash(h)

		vote := gov.NewVoteInst(callctx, addr)

		//处理保证金
		err = vote.HandleDeposit()
		if err != nil {
			gov.Logger.Warnf("handle vote deposit fail, voteAddr %v, err %v", addr, err.Error())
			continue
		}
		//使参数值生效
		if cfg.Custom {
			//TODO: 自定义投票, 暂不实现处理
		} else {
			gov.ParamWrapper.refresh(cfg.PIndex, paramStore)
		}

		//移除投票信息
		b, err := vap.RemoveVote(addr)
		if err != nil || !b {
			gov.Logger.Error("remove vote fail", b, err, addr)
		}
	}
}

func isVote(tx *core.Transaction) bool {
	return tx.ExtraDataType == 1
}

func voteAddr(tx *core.Transaction, receipts *types.Receipts) common.Address {
	for _, r := range *receipts {
		if r.TxHash == util.ToETHHash(tx.Hash) {
			return util.ToTASAddress(r.ContractAddress)
		}
	}

	return common.Address{}
}

func handleVoteCreate(callctx *contract.CallContext, b *core.Block, receipts *types.Receipts) {
	gov := GetGOV()

	for _, tx := range b.Transactions {
		if isVote(tx) {
			cfg, err := AbiDecodeConfig(tx.Data)
			if err != nil {
				gov.Logger.Warnf("abi decode config fail, txHash %v, err %v", tx.Hash, err.Error())
				continue
			}

			vap := gov.NewVoteAddrPoolInst(callctx)

			v := &contract.VoteAddr{
				Addr: voteAddr(tx, receipts),
				StatBlock: cfg.StatBlock,
				EffectBlock: cfg.EffectBlock,
				TxHash: tx.Hash,
			}
			if ok, err := vap.AddVote(v); err != nil || !ok {
				gov.Logger.Warn("add vote fail", ok, err)
			}
		}
	}
}

//上链前触发
func onInsertChain(b *core.Block, stateDB vm.StateDB, recepts *types.Receipts) {
	ctx := GetGOV()

	//生成智能合约调用的上下文
	callctx := contract.NewCallContext(b.Header, ctx.BlockChain, stateDB)

	//处理投票部署
	handleVoteCreate(callctx, b, recepts)

	//更新交易相关的信用信息
	updateTransCredit(callctx, b)

	//唱票
	checkStatVotes(callctx, b)

	//生效
	checkEffectVotes(callctx, b)
}