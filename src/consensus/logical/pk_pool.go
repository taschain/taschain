package logical

import (
	"sync"
	"consensus/groupsig"
	"consensus/base"
)

/*
**  Creator: pxf
**  Date: 2018/11/21 上午10:06
**  Description: 
*/

type pubkeyPool struct {
	pkMap sync.Map			//idHex -> pubKey
	vrfPKMap sync.Map		//idHex -> vrfPK
	minerAccess *MinerPoolReader
}

var pkPool = pubkeyPool{}

func pkPoolInit(access *MinerPoolReader) {
	pkPool.minerAccess = access
}

func ready() bool {
	return pkPool.minerAccess != nil
}

func GetMinerPK(id groupsig.ID) *groupsig.Pubkey {
	if !ready() {
		return nil
	}

	if v, ok := pkPool.pkMap.Load(id.GetHexString()); ok {
		return v.(*groupsig.Pubkey)
	}
	miner := pkPool.minerAccess.getLightMiner(id)
	if miner == nil {
		miner = pkPool.minerAccess.getProposeMiner(id)
	}
	if miner != nil {
		pkPool.pkMap.Store(id.GetHexString(), &miner.PK)
		return &miner.PK
	}
	return nil
}
func GetMinerVrfPK(id groupsig.ID) *base.VRFPublicKey {
	if !ready() {
		return nil
	}

	if v, ok := pkPool.vrfPKMap.Load(id.GetHexString()); ok {
		return v.(*base.VRFPublicKey)
	}
	miner := pkPool.minerAccess.getProposeMiner(id)
	if miner != nil {
		pkPool.vrfPKMap.Store(id.GetHexString(), &miner.VrfPK)
		return &miner.VrfPK
	}
	return nil
}
