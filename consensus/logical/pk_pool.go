package logical

import (
	"github.com/hashicorp/golang-lru"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/groupsig"
)

type pubkeyPool struct {
	pkCache     *lru.Cache
	minerAccess *MinerPoolReader
}

var pkPool pubkeyPool

func init() {
	pkPool = pubkeyPool{
		pkCache: common.MustNewLRUCache(100),
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
