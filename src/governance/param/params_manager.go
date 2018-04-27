package param

import (
	"time"
	"core"
)

/*
**  Creator: pxf
**  Date: 2018/3/26 下午5:17
**  Description: 
*/



type ParamManager struct {
	defs *ParamDefs
	db *ParamDB
	bc *core.BlockChain
}


func NewParamManager(bc *core.BlockChain) *ParamManager {
	pm := &ParamManager{
		db: &ParamDB{},
		bc: bc,
	}
	
	pm.defs = pm.db.LoadParams()
	if pm.defs == nil {
		pm.defs = initParamDefs()
		pm.db.StoreAll(pm.defs)
	}

	go pm.updateLoop()
	return pm
}

func (pm *ParamManager) updateLoop()  {
	for {
		for _, def := range pm.defs.defs {
			select {
			case <-def.update :
				pm.db.StoreParam(def)
			default:
				continue
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func (pm *ParamManager) GetParamByIndex(idx int) *ParamDef {
	return pm.defs.GetParamByIndex(idx)
}

func (pm *ParamManager) getUint64ByIndex(height uint64, idx int) uint64 {
	return pm.GetParamByIndex(idx).CurrentValue(height).(uint64)
}

func (pm *ParamManager) GetGasPriceMin(height uint64, ) uint64 {
	return pm.getUint64ByIndex(height, IDX_GASPRICE_MIN)
}

func (pm *ParamManager) GetFixBlockAward(height uint64, ) uint64 {
	return pm.getUint64ByIndex(height, IDX_BLOCK_FIX_AWARD)
}

func (pm *ParamManager) GetVoterCntMin(height uint64, ) uint64 {
	return pm.getUint64ByIndex(height, IDX_VOTER_CNT_MIN)
}

func (pm *ParamManager) GetVoterDepositMin(height uint64, ) uint64 {
	return pm.getUint64ByIndex(height, IDX_VOTER_DEPOSIT_MIN)
}

func (pm *ParamManager) GetVoterTotalDepositMin(height uint64, ) uint64 {
	return pm.getUint64ByIndex(height, IDX_VOTER_TOTAL_DEPOSIT_MIN)
}

