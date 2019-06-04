package logical

import (
	"bytes"
	"github.com/hashicorp/golang-lru"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/base"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/core"
)

type proveChecker struct {
	proposalVrfHashs *lru.Cache // Recently proposed vrf prve hash
	proveRootCaches  *lru.Cache // Full account verification cache
	chain            core.BlockChain
}

type prootCheckResult struct {
	ok  bool
	err error
}

func newProveChecker(chain core.BlockChain) *proveChecker {
	return &proveChecker{
		proveRootCaches:  common.MustNewLRUCache(50),
		proposalVrfHashs: common.MustNewLRUCache(50),
		chain:            chain,
	}
}

func (p *proveChecker) proveExists(pi base.VRFProve) bool {
	hash := common.BytesToHash(base.VRFProof2hash(pi))
	_, ok := p.proposalVrfHashs.Get(hash)
	return ok
}

func (p *proveChecker) addProve(pi base.VRFProve) {
	hash := common.BytesToHash(base.VRFProof2hash(pi))
	p.proposalVrfHashs.Add(hash, 1)
}

func (p *proveChecker) genVerifyHash(b []byte, id groupsig.ID) common.Hash {
	buf := bytes.NewBuffer([]byte{})
	if b != nil {
		buf.Write(b)
	}
	buf.Write(id.Serialize())

	h := base.Data2CommonHash(buf.Bytes())
	return h
}

// sampleBlockHeight performs block sampling on the id
func (p *proveChecker) sampleBlockHeight(heightLimit uint64, rand []byte, id groupsig.ID) uint64 {
	// Randomly extract the blocks before 10 blocks to ensure that
	// the blocks on the forks are not extracted.
	if heightLimit > 2*model.Param.Epoch {
		heightLimit -= 2 * model.Param.Epoch
	}
	return base.RandFromBytes(rand).DerivedRand(id.Serialize()).ModuloUint64(heightLimit)
}

func (p *proveChecker) genProveHash(heightLimit uint64, rand []byte, id groupsig.ID) common.Hash {
	h := p.sampleBlockHeight(heightLimit, rand, id)
	bs := p.chain.QueryBlockBytesFloor(h)
	hash := p.genVerifyHash(bs, id)

	return hash
}

func (p *proveChecker) genProveHashs(heightLimit uint64, rand []byte, ids []groupsig.ID) (proves []common.Hash) {
	hashs := make([]common.Hash, len(ids))

	for idx, id := range ids {
		hashs[idx] = p.genProveHash(heightLimit, rand, id)
	}
	proves = hashs

	return
}

func (p *proveChecker) addPRootResult(hash common.Hash, ok bool, err error) {
	p.proveRootCaches.Add(hash, &prootCheckResult{ok: ok, err: err})
}

func (p *proveChecker) getPRootResult(hash common.Hash) (exist bool, result bool, err error) {
	v, ok := p.proveRootCaches.Get(hash)
	if ok {
		r := v.(*prootCheckResult)
		return true, r.ok, r.err
	}
	return false, false, nil
}
