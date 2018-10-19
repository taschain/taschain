package logical

import (
	"middleware/types"
	"time"
	"sync/atomic"
	"consensus/model"
	"math/big"
	"errors"
	"common"
	"math"
	"log"
)

/*
**  Creator: pxf
**  Date: 2018/9/12 上午11:46
**  Description: 
*/

const (
	prove int32 = 0
	proposed  = 1
	success  = 2
)

var max256 *big.Rat
var rat1 	*big.Rat

func init() {
	t := new(big.Int)
	t.SetString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)
	max256 = new(big.Rat).SetInt(t)
	rat1 = new(big.Rat).SetInt64(1)
}

type vrfWorker struct {
	//read only
	miner  *model.SelfMinerDO
	baseBH *types.BlockHeader
	castHeight uint64
	expire time.Time
	//writable
	status int32
}

func newVRFWorker(miner *model.SelfMinerDO, bh *types.BlockHeader, castHeight uint64, expire time.Time) *vrfWorker {
    return &vrfWorker{
    	miner: miner,
    	baseBH: bh,
    	castHeight: castHeight,
    	expire: expire,
    	status: prove,
	}
}

func (vrf *vrfWorker) prove(totalStake uint64) (common.VRFProve, uint64, error) {
	pi, err := common.VRF_prove(vrf.miner.VrfPK, vrf.miner.VrfSK, vrf.baseBH.Random)
	if err != nil {
		return nil, 0, err
	}
	if ok, pr, sr := vrfSatisfy(pi, vrf.miner.Stake, totalStake); ok {
		//计算qn
		if sr.Cmp(rat1) > 0 {
			sr.Set(rat1)
		}
		s1, _ := pr.Float64()
		s2, _ := sr.Float64()

		step := sr.Quo(sr, new(big.Rat).SetInt64(int64(model.Param.MaxQN)))

		st, _ := step.Float64()

		r, _ := pr.Quo(pr, step).Float64()
		qn := uint64(math.Floor(r) + 1)

		log.Printf("vrf prove pr %v, sr %v, step %v, qn %v", s1, s2, st, r, qn)
		return pi, qn, nil
	}
    return nil, 0, errors.New("proof fail")
}

func vrfSatisfy(pi common.VRFProve, stake uint64, totalStake uint64) (ok bool, piRat *big.Rat, sRat *big.Rat) {
	value := common.VRF_proof2hash(pi)

	br := new(big.Rat).SetInt(new(big.Int).SetBytes(value))
	v := br.Quo(br, max256)

	brTStake := new(big.Rat).SetInt64(int64(totalStake))
	vs := new(big.Rat).Quo(new(big.Rat).SetInt64(int64(stake*uint64(model.Param.PotentialProposal))), brTStake)

	s1, _ := v.Float64()
	s2, _ := vs.Float64()
	//blog := newBizLog("vrfSatisfy")
	//blog.log("value %v stake %v", s1, s2)
	log.Printf("varffffffff 1 pr %v, sr %v", s1, s2)

	return v.Cmp(vs) < 0, v, vs
	//return true
}

func vrfVerifyBlock(bh *types.BlockHeader, preBH *types.BlockHeader, miner *model.MinerDO, totalStake uint64) (bool, error) {
	pi := common.VRFProve(bh.ProveValue.Bytes())
	ok, err := common.VRF_verify(miner.VrfPK, pi, preBH.Random)
	//blog := newBizLog("vrfVerifyBlock")
	//blog.log("pi %v", pi.ShortS())
	if !ok {
		return ok ,err
	}
	if ok, _, _ := vrfSatisfy(pi, miner.Stake, totalStake); ok {
		return true, nil
	}
	return false, errors.New("proof not satisfy")
}

func (vrf *vrfWorker) markProposed()  {
	atomic.CompareAndSwapInt32(&vrf.status, prove, proposed)
}

func (vrf *vrfWorker) markSuccess()  {
	atomic.CompareAndSwapInt32(&vrf.status, proposed, success)
}

func (vrf *vrfWorker) getBaseBH() *types.BlockHeader {
	return vrf.baseBH
}

func (vrf *vrfWorker) isSuccess() bool {
	return vrf.getStatus() == success
}

func (vrf *vrfWorker) isProposed() bool {
	return vrf.getStatus() == proposed
}

func (vrf *vrfWorker) getStatus() int32 {
    return atomic.LoadInt32(&vrf.status)
}

func (vrf *vrfWorker) workingOn(bh *types.BlockHeader, castHeight uint64) bool {
	return bh.Hash == vrf.baseBH.Hash && castHeight == vrf.castHeight && !time.Now().After(vrf.expire)
}

func (vrf *vrfWorker) timeout() bool {
    return time.Now().After(vrf.expire)
}