package param

import "time"

/*
**  Creator: pxf
**  Date: 2018/3/26 下午5:17
**  Description: 
*/



type ParamManager struct {
	defs *ParamDefs
	db *ParamDB
}


func NewParamManager() *ParamManager {
	pm := &ParamManager{
		db: &ParamDB{},
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

func (pm *ParamManager) getParamByIndex(idx int) *ParamDef {
	return pm.defs.GetParamByIndex(idx)
}

func (pm *ParamManager) getUint64ByIndex(idx int) uint64 {
	return pm.getParamByIndex(idx).CurrentValue().(uint64)
}

func (pm *ParamManager) GetGasPriceMin() uint64 {
	return pm.getUint64ByIndex(IDX_GASPRICE_MIN)
}

func (pm *ParamManager) GetFixBlockAward() uint64 {
	return pm.getUint64ByIndex(IDX_BLOCK_FIX_AWARD)
}

func (pm *ParamManager) GetVoterCntMin() uint64 {
	return pm.getUint64ByIndex(IDX_VOTER_CNT_MIN)
}

func (pm *ParamManager) GetVoterDepositMin() uint64 {
	return pm.getUint64ByIndex(IDX_VOTER_DEPOSIT_MIN)
}

func (pm *ParamManager) GetVoterTotalDepositMin() uint64 {
	return pm.getUint64ByIndex(IDX_VOTER_TOTAL_DEPOSIT_MIN)
}

