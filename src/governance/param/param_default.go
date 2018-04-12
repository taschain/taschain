package param

import (
	"math"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2018/4/11 上午10:16
**  Description: 
*/

//参数校验函数
type ValidateFunc func(input interface{}) error

type DefaultValueDef struct {
	name string
	init interface{}
	min  interface{}
	max  interface{}
}

const PARAM_CNT = 5

const (
	DEFAULT_GASPRICE_MIN            = 10000 //gasprice min
	DEFAULT_BLOCK_FIX_AWARD         = 5     //区块固定奖励
	DEFAULT_VOTER_CNT_MIN           = 100   //最小参与投票人
	DEFAULT_VOTER_DEPOSIT_MIN       = 1     //每个投票人最低缴纳的保证金
	DEFAULT_VOTER_TOTAL_DEPOSIT_MIN = 100   //所有投票人最低保证金总和
)

const (
	IDX_GASPRICE_MIN	= iota
	IDX_BLOCK_FIX_AWARD
	IDX_VOTER_CNT_MIN
	IDX_VOTER_DEPOSIT_MIN
	IDX_VOTER_TOTAL_DEPOSIT_MIN
)

var DEFAULT_DEFS  [PARAM_CNT]DefaultValueDef

func init() {
	DEFAULT_DEFS[IDX_GASPRICE_MIN] = DefaultValueDef{
		name: "gas_price_min",
		init: uint64(DEFAULT_GASPRICE_MIN),
		min: uint64(1),
		max: uint64(math.MaxUint64)}

	DEFAULT_DEFS[IDX_BLOCK_FIX_AWARD] = DefaultValueDef{
		name: "block_fix_award_min",
		init: uint64(DEFAULT_BLOCK_FIX_AWARD),
		min: uint64(1),
		max: uint64(math.MaxInt8)}

	DEFAULT_DEFS[IDX_VOTER_CNT_MIN] = DefaultValueDef{
		name: "voter_cnt_min",
		init: uint64(DEFAULT_VOTER_CNT_MIN),
		min: uint64(1),
		max: uint64(math.MaxInt32)}

	DEFAULT_DEFS[IDX_VOTER_DEPOSIT_MIN] = DefaultValueDef{
		name: "voter_deposit_min",
		init: uint64(DEFAULT_VOTER_DEPOSIT_MIN),
		min: uint64(0),
		max: uint64(math.MaxUint64)}

	DEFAULT_DEFS[IDX_VOTER_TOTAL_DEPOSIT_MIN] = DefaultValueDef{
		name: "voter_total_deposit_min",
		init: uint64(DEFAULT_VOTER_TOTAL_DEPOSIT_MIN),
		min: uint64(0),
		max: uint64(math.MaxUint64)}

}

func intRangeCheck(input uint64, min uint64, max uint64) bool {
	return input >= min && input < max
}

func getDefaultValueDefs(idx int) *DefaultValueDef {
	if !intRangeCheck(uint64(idx), 0, uint64(len(DEFAULT_DEFS))) {
		panic("idx out of bound!")
	}
	return &DEFAULT_DEFS[idx]
}


func getValidateFunc(def *DefaultValueDef) ValidateFunc {
	return func(input interface{}) error {
		v := input.(uint64)
		ret := intRangeCheck(v, def.min.(uint64), def.max.(uint64))
		if !ret {
			return fmt.Errorf("%v must be between %v and %v", def.name, def.min, def.max)
		}
		return nil
	}
}