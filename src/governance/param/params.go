package param

/*
**  Creator: pxf
**  Date: 2018/3/26 下午5:17
**  Description: 
*/

type ParamDefs []*ParamDef

var Params ParamDefs

func init() {
	Params = ParamDefs{}

	gasPriceMin := newMeta(DEFAULT_GASPRICE_MIN)
	blockFixAward := newMeta(DEFAULT_BLOCK_FIX_AWARD)
	voteCntMin := newMeta(DEFAULT_VOTER_CNT_MIN)
	voteDepositMin := newMeta(DEFAULT_VOTER_DEPOSIT_MIN)
	voteTotalDepositMin := newMeta(DEFAULT_VOTER_TOTAL_DEPOSIT_MIN)

	Params = append(Params, newParamDef(gasPriceMin))
	Params = append(Params, newParamDef(blockFixAward))
	Params = append(Params, newParamDef(voteCntMin))
	Params = append(Params, newParamDef(voteDepositMin))
	Params = append(Params, newParamDef(voteTotalDepositMin))
}

func (p *ParamDefs) GetParamByIndex(index int) *ParamDef {
	return Params[index]
}

func (p *ParamDefs) UpdateParam(index int, def *ParamDef) {
	if len(Params) <= index {
		return
	}
	Params[index] = def
}
