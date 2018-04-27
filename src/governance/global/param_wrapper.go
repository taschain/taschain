package global

import (
	"governance/contract"
	"strconv"
)

/*
**  Creator: pxf
**  Date: 2018/4/26 下午4:41
**  Description: 
*/

const (
	GASPRICE_MIN = iota
	BLOCK_FIX_AWARD
	VOTER_COUNT_MIN
)

type ParamWrapper struct {
	params []*contract.ParamMeta
}

func NewParamWrapper() *ParamWrapper {
	return &ParamWrapper{
		params: make([]*contract.ParamMeta, 3),
	}
}

func (pc *ParamWrapper) load(ps *contract.ParamStore) {
	pc.refresh(GASPRICE_MIN, ps)
	pc.refresh(BLOCK_FIX_AWARD, ps)
	pc.refresh(VOTER_COUNT_MIN, ps)
}

func (pc ParamWrapper) refresh(pindex uint32, ps *contract.ParamStore)  {
	meta, err := ps.GetCurrentMeta(pindex)
	if err != nil {
		gov.Logger.Error("load param error", pindex, err)
	}
	pc.params[pindex] = meta
}

func (pc *ParamWrapper) addFuture(pindex uint32, ps *contract.ParamStore, meta *contract.ParamMeta) bool {
    err := ps.AddFuture(pindex, meta)
	if err != nil {
		gov.Logger.Error("add future err", pindex, err)
		return false
	}
	pc.refresh(pindex, ps)
    return true
}

func (pc *ParamWrapper) getUint64Value(pindex uint32, ps *contract.ParamStore) uint64 {
    meta := pc.params[pindex]
	if meta == nil {
		pc.refresh(pindex, ps)
		meta = pc.params[pindex]
	}
	if meta == nil {
		return 0
	}

	ret, err := strconv.ParseUint(meta.Value, 10, 64)
	if err != nil {
		gov.Logger.Error("parse meta value err", meta.Value, err)
		return 0
	}
	return ret
}


func (pc *ParamWrapper) GetGasPriceMin(ps *contract.ParamStore) uint64 {
	return pc.getUint64Value(GASPRICE_MIN, ps)
}

func (pc *ParamWrapper) GetBlockFixAward(ps *contract.ParamStore) uint64 {
	return pc.getUint64Value(BLOCK_FIX_AWARD, ps)
}

func (pc *ParamWrapper) GetVoterCountMin(ps *contract.ParamStore) uint64 {
	return pc.getUint64Value(VOTER_COUNT_MIN, ps)
}
