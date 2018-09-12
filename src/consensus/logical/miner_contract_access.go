package logical

import (
	"consensus/model"
)

/*
**  Creator: pxf
**  Date: 2018/9/11 下午3:24
**  Description: 
*/

type MinerContractAccess struct {

}

func NewMinerContractAccess() *MinerContractAccess {
    return &MinerContractAccess{}
}

func (access *MinerContractAccess) getProposeMiner(idbytes []byte) *model.MinerDO {
    return &model.MinerDO{}
}

func (access *MinerContractAccess) getAllMinerDOByType(ntype model.NodeType) []model.MinerDO {
    return []model.MinerDO{}
}

func (access *MinerContractAccess) getCanJoinGroupMinersAt(h uint64) []model.MinerDO {
    miners := access.getAllMinerDOByType(model.LightNode)
    rets := make([]model.MinerDO, 0)

	for _, md := range miners {
		if md.CanJoinGroupAt(h) {
			rets = append(rets, md)
		}
	}
	return rets
}