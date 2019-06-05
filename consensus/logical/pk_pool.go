//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package logical

import (
	lru "github.com/hashicorp/golang-lru"
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
