package logical

import (
	"consensus/groupsig"
	"github.com/hashicorp/golang-lru"
)

/*
**  Creator: pxf
**  Date: 2018/11/21 上午10:06
**  Description: 
*/

type pubkeyPool struct {
	pkCache *lru.Cache
	//vrfPKCache *lru.Cache		//idHex -> vrfPK
	minerAccess *MinerPoolReader
}

var pkPool pubkeyPool

func init() {
	pkc, _ := lru.New(100)
	//vrfc, _ := lru.New(100)
	pkPool = pubkeyPool{
		pkCache: pkc,
		//vrfPKCache: vrfc,
	}
}

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

	if v, ok := pkPool.pkCache.Get(id.GetHexString()); ok {
		return v.(*groupsig.Pubkey)
	}
	miner := pkPool.minerAccess.getLightMiner(id)
	if miner == nil {
		miner = pkPool.minerAccess.getProposeMiner(id)
	}
	if miner != nil {
		pkPool.pkCache.Add(id.GetHexString(), &miner.PK)
		return &miner.PK
	}
	return nil
}

