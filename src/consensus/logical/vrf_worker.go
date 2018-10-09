package logical

import (
	"middleware/types"
	"time"
	"sync/atomic"
	"consensus/model"
	"math/big"
	"errors"
	"consensus/base"
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

func init() {
	t := new(big.Int)
	t.SetString("ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)
	max256 = new(big.Rat).SetInt(t)
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

func (vrf *vrfWorker) prove(totalStake uint64) (base.VRFProve, error) {
	pi, err := base.VRF_prove(vrf.miner.VrfPK, vrf.miner.VrfSK, vrf.baseBH.Random)
	if err != nil {
		return nil, err
	}
	if vrfSatisfy(pi, vrf.miner.Stake, totalStake) {
		return pi, nil
	}
    return nil, errors.New("proof fail")
}

func vrfSatisfy(pi base.VRFProve, stake uint64, totalStake uint64) bool {
	value := base.VRF_proof2hash(pi)
	br := new(big.Rat).SetInt(new(big.Int).SetBytes(value))
	v := br.Quo(br, max256)

	brTStake := new(big.Rat).SetInt64(int64(totalStake))
	vs := new(big.Rat).Quo(new(big.Rat).SetInt64(int64(stake)), brTStake)

	s1, _ := v.Float64()
	s2, _ := vs.Float64()
	blog := newBizLog("vrfSatisfy")
	blog.log("value %v stake %v", s1, s2)

	//return v.Cmp(vs) < 0
	return true
}

func vrfVerifyBlock(bh *types.BlockHeader, preBH *types.BlockHeader, miner *model.MinerDO, totalStake uint64) (bool, error) {
	pi := base.VRFProve(bh.ProveValue.Bytes())
	ok, err := base.VRF_verify(miner.VrfPK, pi, preBH.Random)
	blog := newBizLog("vrfVerifyBlock")
	blog.log("pi %v", pi.ShortS())
	if !ok {
		return ok ,err
	}
	if vrfSatisfy(pi, miner.Stake, totalStake) {
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