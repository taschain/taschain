package logical

import (
	"consensus/base"
	"consensus/model"
	"consensus/groupsig"
	"middleware/types"
	"core"
)

/*
**  Creator: pxf
**  Date: 2018/9/11 下午3:24
**  Description: 
*/

type MinerPoolReader struct {
	minerPool *core.MinerManager
	blog  *bizLog
}

func newMinerPoolReader(mp *core.MinerManager) *MinerPoolReader {
    return &MinerPoolReader{
    	minerPool: mp,
    	blog: newBizLog("MinerPoolReader"),
	}
}

func convert2MinerDO(miner *types.Miner) *model.MinerDO {
	if miner == nil {
		return nil
	}
	md := &model.MinerDO{
		ID: groupsig.DeserializeId(miner.Id),
		PK: groupsig.DeserializePubkeyBytes(miner.PublicKey),
		VrfPK: base.VRFPublicKey(miner.VrfPublicKey),
		Stake: miner.Stake,
		NType: miner.Type,
		ApplyHeight: miner.ApplyHeight,
		AbortHeight: miner.AbortHeight,
	}
	return md
}

func (access *MinerPoolReader) getProposeMiner(id groupsig.ID) *model.MinerDO {
	miner := access.minerPool.GetMinerById(id.Serialize(), types.MinerTypeHeavy)
	if miner == nil {
		//access.blog.log("getMinerById error id %v", id.ShortS())
		return nil
	}
	return convert2MinerDO(miner)
}

func (access *MinerPoolReader) getAllMinerDOByType(ntype byte) []*model.MinerDO {
	iter := access.minerPool.MinerIterator(ntype,nil)
	mds := make([]*model.MinerDO, 0)
	for iter.Next() {
		if curr, err := iter.Current(); err != nil {
			continue
			access.blog.log("minerManager iterator error %v", err)
		} else {
			md := convert2MinerDO(curr)
			mds = append(mds, md)
		}
	}
    return mds
}

func (access *MinerPoolReader) getCanJoinGroupMinersAt(h uint64) []model.MinerDO {
    miners := access.getAllMinerDOByType(types.MinerTypeLight)
    rets := make([]model.MinerDO, 0)
	access.blog.log("all light nodes size %v", len(miners))
	for _, md := range miners {
		access.blog.log("%v %v %v %v", md.ID.ShortS(), md.ApplyHeight, md.NType, md.CanJoinGroupAt(h))
		if md.CanJoinGroupAt(h) {
			rets = append(rets, *md)
		}
	}
	return rets
}

func (access *MinerPoolReader) getTotalStake(h uint64) uint64 {
	return access.minerPool.GetTotalStakeByHeight(h)
	//return 30
}

//func (access *MinerPoolReader) genesisMiner(miners []*types.Miner)  {
//    access.minerPool.AddGenesesMiner(miners)
//}