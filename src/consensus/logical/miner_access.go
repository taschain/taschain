package logical

import (
	"consensus/model"
	"consensus/groupsig"
	"core"
	"middleware/types"
	"consensus/vrf_ed25519"
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
		PK: groupsig.DeserializePubkeyBytes(miner.PublicKey),
		VrfPK: vrf_ed25519.PublicKey(miner.VrfPublicKey),
		Stake: miner.Stake,
		NType: miner.Type,
		ApplyHeight: miner.ApplyHeight,
		AbortHeight: miner.AbortHeight,
	}
	md.ID = *groupsig.NewIDFromPubkey(md.PK)
	return md
}

func (access *MinerPoolReader) getProposeMiner(id groupsig.ID) *model.MinerDO {
	miner, err := access.minerPool.GetMinerById(id.Serialize(), types.MinerTypeHeavy)
	if err != nil {
		access.blog.log("getMinerById error %v, id %v", err, GetIDPrefix(id))
		return nil
	}
	return convert2MinerDO(miner)
}

func (access *MinerPoolReader) getAllMinerDOByType(ntype byte) []model.MinerDO {
	iter := access.minerPool.MinerIterator(ntype)
	mds := make([]model.MinerDO, 0)
	for iter.Next() {
		if curr, err := iter.Current(); err != nil {
			access.blog.log("minerManager iterator error %v", err)
		} else {
			md := convert2MinerDO(curr)
			mds = append(mds, *md)
		}
	}
    return mds
}

func (access *MinerPoolReader) getCanJoinGroupMinersAt(h uint64) []model.MinerDO {
    miners := access.getAllMinerDOByType(types.MinerTypeLight)
    rets := make([]model.MinerDO, 0)

	for _, md := range miners {
		if md.CanJoinGroupAt(h) {
			rets = append(rets, md)
		}
	}
	return rets
}

func (access *MinerPoolReader) getTotalStake(h uint64) uint64 {
	return 1
}