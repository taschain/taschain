package logical

import (
	"github.com/taschain/taschain/consensus/base"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/core"
	"github.com/taschain/taschain/middleware/types"
)

/*
**  Creator: pxf
**  Date: 2018/9/11 下午3:24
**  Description:
 */

type MinerPoolReader struct {
	minerPool       *core.MinerManager
	blog            *bizLog
	totalStakeCache uint64
}

func newMinerPoolReader(mp *core.MinerManager) *MinerPoolReader {
	return &MinerPoolReader{
		minerPool: mp,
		blog:      newBizLog("MinerPoolReader"),
	}
}

func convert2MinerDO(miner *types.Miner) *model.MinerDO {
	if miner == nil {
		return nil
	}
	md := &model.MinerDO{
		ID:          groupsig.DeserializeId(miner.ID),
		PK:          groupsig.DeserializePubkeyBytes(miner.PublicKey),
		VrfPK:       base.VRFPublicKey(miner.VrfPublicKey),
		Stake:       miner.Stake,
		NType:       miner.Type,
		ApplyHeight: miner.ApplyHeight,
		AbortHeight: miner.AbortHeight,
	}
	if !md.ID.IsValid() {
		stdLogger.Debugf("invalid id %v, %v", miner.ID, md.ID.GetHexString())
		panic("id not valid")
	}
	return md
}

func (access *MinerPoolReader) getLightMiner(id groupsig.ID) *model.MinerDO {
	miner := access.minerPool.GetMinerByID(id.Serialize(), types.MinerTypeLight, nil)
	if miner == nil {
		//access.blog.log("getMinerById error id %v", id.ShortS())
		return nil
	}
	return convert2MinerDO(miner)
}

func (access *MinerPoolReader) getProposeMiner(id groupsig.ID) *model.MinerDO {
	miner := access.minerPool.GetMinerByID(id.Serialize(), types.MinerTypeHeavy, nil)
	if miner == nil {
		//access.blog.log("getMinerById error id %v", id.ShortS())
		return nil
	}
	return convert2MinerDO(miner)
}

func (access *MinerPoolReader) getAllMinerDOByType(ntype byte, h uint64) []*model.MinerDO {
	iter := access.minerPool.MinerIterator(ntype, h)
	mds := make([]*model.MinerDO, 0)
	for iter.Next() {
		if curr, err := iter.Current(); err != nil {
			continue
		} else {
			md := convert2MinerDO(curr)
			mds = append(mds, md)
		}
	}
	return mds
}

func (access *MinerPoolReader) getCanJoinGroupMinersAt(h uint64) []model.MinerDO {
	miners := access.getAllMinerDOByType(types.MinerTypeLight, h)
	rets := make([]model.MinerDO, 0)
	access.blog.log("all light nodes size %v", len(miners))
	for _, md := range miners {
		//access.blog.log("%v %v %v %v", md.ID.ShortS(), md.ApplyHeight, md.NType, md.CanJoinGroupAt(h))
		if md.CanJoinGroupAt(h) {
			rets = append(rets, *md)
		}
	}
	return rets
}

func (access *MinerPoolReader) getTotalStake(h uint64, cache bool) uint64 {
	if cache && access.totalStakeCache > 0 {
		return access.totalStakeCache
	}
	st := access.minerPool.GetTotalStake(h)
	access.totalStakeCache = st
	return st
	//return 30
}

//func (access *MinerPoolReader) genesisMiner(miners []*types.Miner)  {
//    access.minerPool.AddGenesesMiner(miners)
//}
