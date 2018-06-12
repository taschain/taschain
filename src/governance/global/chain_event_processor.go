package global

import (
	"vm/core/vm"
	vtypes "vm/core/types"
	"governance/contract"
	"common"
	"governance/util"
	"middleware/types"
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

func (p *ChainEventProcessorImpl) AfterAllTransactionExecuted(b *types.Block, stateDB vm.StateDB, receipts vtypes.Receipts) error {
	return afterExecute(b, stateDB, &receipts)
}

func (p *ChainEventProcessorImpl) BeforeExecuteTransaction(b *types.Block, db vm.StateDB, tx *types.Transaction) ([]byte, error) {
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
func getRealCode(b *types.Block, db vm.StateDB, input []byte) ([]byte, error) {
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
func updateTransCredit(callctx *contract.CallContext, b *types.Block) error {
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
		if err := credit.AddTransCnt(*addr, cnt); err != nil {
			return gov.Logger.Errorf("add trans cnt fail, addr %v, err %v", addr, err)
		}
		if err := credit.SetLatestTransBlock(*addr, b.Header.Height); err != nil {
			return gov.Logger.Errorf("set latestransblock fail, addr %v, err %v", addr, err)
		}
	}
	return nil
}

func getContractAddressByTxHash(hash common.Hash) common.Address {
	if receipt := GetGOV().BlockChain.GetTransactionPool().GetExecuted(hash); receipt != nil {
		return util.ToTASAddress(receipt.Receipt.ContractAddress)
	}
	return common.Address{}
}

//检查到达唱票区块的投票
func checkStatVotes(callctx *contract.CallContext, b *types.Block) error {
	ctx := GetGOV()

	vap := gov.NewVoteAddrPoolInst(callctx)
	hs, err := vap.GetCurrentStatVoteHashes()
	if err != nil {
		return gov.Logger.Error("get current stat vote hash err", b.Header.Height, err)
	}

	paramStore := gov.NewParamStoreInst(callctx)

	for _, h := range hs {
		tx, err := ctx.BlockChain.GetTransactionByHash(h)
		if err != nil || tx == nil {
			return gov.Logger.Error("get transaction fail", h, err)
		}

		cfg, err := AbiDecodeConfig(tx.Data)
		if err != nil {
			return gov.Logger.Error("decode tx data fail", err)
		}

		addr := getContractAddressByTxHash(h)

		vote := ctx.NewVoteInst(callctx, addr)
		//调用合约函数检查投票是否通过
		pass, err := vote.CheckResult()
		if err != nil {
			return ctx.Logger.Warnf("check vote result fail, voteAddr %v, err %v", addr, err)
		}
		ctx.Logger.Infof("check voteAddr %v, result %v", common.Bytes2Hex(addr.Bytes()), pass)
		if pass {
			if cfg.Custom {
				//TODO: 自定义投票, 暂不实现处理
			} else {
				meta := &contract.ParamMeta{
					Value:       cfg.PValue,
					TxHash:      h,
					EffectBlock: cfg.EffectBlock,
				}
				ctx.ParamWrapper.addFuture(cfg.PIndex, paramStore, meta)
			}
		}
		//更新信用信息
		err = vote.UpdateCredit()
		if err != nil {
			return gov.Logger.Warn("update credit fail", err)
		}

	}
	return nil
}

//处理那些已经到达生效区块高度的投票的保证金, 以及参数正式生效
func checkEffectVotes(callctx *contract.CallContext, b *types.Block) error {
	gov := GetGOV()

	vap := gov.NewVoteAddrPoolInst(callctx)
	hs, err := vap.GetCurrentEffectVoteHashes()
	if err != nil {
		return gov.Logger.Error("get current effect vote hash err", b.Header.Height, err)
	}

	paramStore := gov.NewParamStoreInst(callctx)

	for _, h := range hs {
		tx, err := gov.BlockChain.GetTransactionByHash(h)
		if err != nil || tx == nil {
			return gov.Logger.Error("get transaction fail", h, err)
		}

		cfg, err := AbiDecodeConfig(tx.Data)
		if err != nil {
			return gov.Logger.Error("decode tx data fail", err)
		}

		addr := getContractAddressByTxHash(h)

		vote := gov.NewVoteInst(callctx, addr)

		//处理保证金
		err = vote.HandleDeposit()
		if err != nil {
			return gov.Logger.Warnf("handle vote deposit fail, voteAddr %v, err %v", addr, err)
		}

		oldValue := gov.ParamWrapper.getUint64Value(cfg.PIndex, paramStore)

		//使参数值生效
		if cfg.Custom {
			//TODO: 自定义投票, 暂不实现处理
		} else {
			gov.ParamWrapper.refresh(cfg.PIndex, paramStore)
		}
		newValue := gov.ParamWrapper.getUint64Value(cfg.PIndex, paramStore)
		gov.Logger.Infof("vote effect param, from %v to %v ", oldValue, newValue)

		//移除投票信息
		b, err := vap.RemoveVote(addr)
		if err != nil || !b {
			return gov.Logger.Error("remove vote fail", b, err, addr)
		}
	}
	return nil
}

func isVote(tx *types.Transaction) bool {
	return tx.ExtraDataType == 1
}

func voteAddr(tx *types.Transaction, receipts *vtypes.Receipts) common.Address {
	for _, r := range *receipts {
		if r.TxHash == util.ToETHHash(tx.Hash) {
			return util.ToTASAddress(r.ContractAddress)
		}
	}

	return common.Address{}
}

func handleVoteCreate(callctx *contract.CallContext, b *types.Block, receipts *vtypes.Receipts) error {
	gov := GetGOV()

	for _, tx := range b.Transactions {
		if isVote(tx) {
			cfg, err := AbiDecodeConfig(tx.Data)
			if err != nil {
				return gov.Logger.Warnf("abi decode config fail, txHash %v, err %v", tx.Hash, err)
			}

			vap := gov.NewVoteAddrPoolInst(callctx)

			v := &contract.VoteAddr{
				Addr:        voteAddr(tx, receipts),
				StatBlock:   cfg.StatBlock,
				EffectBlock: cfg.EffectBlock,
				TxHash:      tx.Hash,
			}
			if ok, err := vap.AddVote(v); err != nil || !ok {
				return gov.Logger.Warn("add vote fail", ok, err)
			}

			gov.Logger.Infof("vote contract found!, address %v", common.Bytes2Hex(v.Addr.Bytes()))
		}
	}
	return nil
}

//执行后触发
func afterExecute(b *types.Block, stateDB vm.StateDB, recepts *vtypes.Receipts) error {
	ctx := GetGOV()

	//生成智能合约调用的上下文
	callctx := contract.NewCallContext(b.Header, ctx.BlockChain, stateDB)

	//处理投票部署
	if err := handleVoteCreate(callctx, b, recepts); err != nil {
		return err
	}

	//更新交易相关的信用信息
	if err := updateTransCredit(callctx, b); err != nil {
		return err
	}

	//唱票
	if err := checkStatVotes(callctx, b); err != nil {
		return err
	}

	//生效
	if err := checkEffectVotes(callctx, b); err != nil {
		return err
	}
	return nil
}
