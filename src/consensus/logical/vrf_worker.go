package logical

import (
	"middleware/types"
	"time"
	"sync/atomic"
	"consensus/vrf_ed25519"
	"consensus/model"
	"math/big"
	"errors"
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

var max256 *big.Int

func init() {
	max256 = new(big.Int)
	max256.SetString("ffffffffffffffffffffffffffffffff", 16)
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

func (vrf *vrfWorker) prove(totalStake uint64) (vrf_ed25519.VRFProve, error) {
	pi, err := vrf_ed25519.ECVRF_prove(vrf.miner.VrfPK, vrf.miner.VrfSK, vrf.baseBH.Random)
	if err != nil {
		return nil, err
	}
	if vrfSatisfy(pi, vrf.miner.Stake, totalStake) {
		return pi, nil
	}
    return nil, errors.New("proof fail")
}

func vrfSatisfy(pi vrf_ed25519.VRFProve, stake uint64, totalStake uint64) bool {
	value := vrf_ed25519.ECVRF_proof2hash(pi)
	bi := new(big.Int).SetBytes(value)
	v := bi.Div(bi, max256)

	biTStake := new(big.Int).SetUint64(totalStake)
	vs := new(big.Int).Div(new(big.Int).SetUint64(stake), biTStake)

	return v.Cmp(vs) < 0
}

func vrfVerifyBlock(bh *types.BlockHeader, preBH *types.BlockHeader, miner *model.MinerDO, totalStake uint64) (bool, error) {
	pi := vrf_ed25519.VRFProve(bh.ProveValue.Bytes())
	ok, err := vrf_ed25519.ECVRF_verify(miner.VrfPK, pi, preBH.Random)
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