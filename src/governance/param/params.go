package param

import (
	"math"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2018/3/26 下午5:17
**  Description: 
*/

type ParamDefs []*ParamDef

var Params ParamDefs

const (
	DEFAULT_GASPRICE_MIN            = 10000 //gasprice min
	DEFAULT_BLOCK_FIX_AWARD         = 5     //区块固定奖励
	DEFAULT_VOTER_CNT_MIN           = 100   //最小参与投票人
	DEFAULT_VOTER_DEPOSIT_MIN       = 1     //每个投票人最低缴纳的保证金
	DEFAULT_VOTER_TOTAL_DEPOSIT_MIN = 100   //所有投票人最低保证金总和
)

type DefaultValueDef struct {
	name string
	init interface{}
	min  interface{}
	max  interface{}
}

func init() {
	Params = ParamDefs{}

	gasPriceMin := buildUint64ParamDef(DefaultValueDef{
		name: "gas_price_min",
		init: uint64(DEFAULT_GASPRICE_MIN),
		min: uint64(1),
		max: uint64(math.MaxUint64)},
		)
	blockFixAward := buildUint64ParamDef(DefaultValueDef{
		name: "block_fix_award_min",
		init: uint64(DEFAULT_BLOCK_FIX_AWARD),
		min: uint64(1),
		max: uint64(math.MaxInt8)},
		)
	voteCntMin := buildUint64ParamDef(DefaultValueDef{
		name: "voter_cnt_min",
		init: uint64(DEFAULT_VOTER_CNT_MIN),
		min: uint64(1),
		max: uint64(math.MaxInt32)},
		)
	voteDepositMin := buildUint64ParamDef(DefaultValueDef{
		name: "voter_deposit_min",
		init: uint64(DEFAULT_VOTER_DEPOSIT_MIN),
		min: uint64(0),
		max: uint64(math.MaxUint64)},
		)
	voteTotalDepositMin := buildUint64ParamDef(DefaultValueDef{
		name: "voter_total_deposit_min",
		init: uint64(DEFAULT_VOTER_TOTAL_DEPOSIT_MIN),
		min: uint64(0),
		max: uint64(math.MaxUint64)},
		)


	Params = append(Params, gasPriceMin)
	Params = append(Params, blockFixAward)
	Params = append(Params, voteCntMin)
	Params = append(Params, voteDepositMin)
	Params = append(Params, voteTotalDepositMin)
}

func buildUint64ParamDef(def DefaultValueDef) *ParamDef {
	return newParamDef(
		def.init,
		func(input interface{}) error {
			v := input.(uint64)
			ret := intRangeCheck(v, def.min.(uint64), def.max.(uint64))
			if !ret {
				return fmt.Errorf("%v must be between %v and %v", def.name, def.min, def.max)
			}
			return nil
		})
}

func intRangeCheck(input uint64, min uint64, max uint64) bool {
	return input >= min && input < max
}



func (p *ParamDefs) GetParamByIndex(index int) *ParamDef {
	return Params[index]
}

func (p *ParamDefs) UpdateParam(index int, def *ParamDef) error {
	if len(Params) <= index {
		return fmt.Errorf("error index %v", index)
	}
	if err := def.Validate(def.Current.Value); err != nil {
		return err
	}
	Params[index] = def
	return nil
}
