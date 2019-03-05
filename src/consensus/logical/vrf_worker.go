package logical

import (
	"consensus/model"
	"errors"
	"math/big"
	"middleware/types"
	"sync/atomic"
	"time"
	"math"
	"fmt"
	"consensus/base"
)

/*
**  Creator: pxf
**  Date: 2018/9/12 上午11:46
**  Description:
 */

const (
	prove    int32 = 0
	proposed       = 1
	success        = 2
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
	miner      *model.SelfMinerDO
	baseBH     *types.BlockHeader
	castHeight uint64
	expire     time.Time
	//writable
	status int32
}

func NewVRFWorker(miner *model.SelfMinerDO, bh *types.BlockHeader, castHeight uint64, expire time.Time) *vrfWorker {
	return &vrfWorker{
		miner:      miner,
		baseBH:     bh,
		castHeight: castHeight,
		expire:     expire,
		status:     prove,
	}
}

func (vrf *vrfWorker) m() []byte {
	h := vrf.castHeight - vrf.baseBH.Height
	return vrfM(vrf.baseBH.Random, h)
}

func vrfM(random []byte, h uint64) []byte {
	if h <= 0 {
		panic(fmt.Sprintf("vrf height error! deltaHeight=%v", h))
	}
	data := random
	for h > 1 {
		h--
		hash := base.Data2CommonHash(data)
		data = hash.Bytes()
	}
	return data
}

func (vrf *vrfWorker) Prove(totalStake uint64) (base.VRFProve, uint64, error) {
	pi, err := base.VRF_prove(vrf.miner.VrfPK, vrf.miner.VrfSK, vrf.m())
	if err != nil {
		return nil, 0, err
	}
	if ok, qn := vrfSatisfy(pi, vrf.miner.Stake, totalStake); ok {

		return pi, qn, nil
	}
    return nil, 0, errors.New("proof fail")
}

func vrfThreshold(stake, totalStake uint64) *big.Rat {
	brTStake := new(big.Rat).SetFloat64(float64(totalStake))
	return new(big.Rat).Quo(new(big.Rat).SetInt64(int64(stake*uint64(model.Param.PotentialProposal))), brTStake)
}

func vrfSatisfy(pi base.VRFProve, stake uint64, totalStake uint64) (ok bool, qn uint64) {
	if totalStake == 0 {
		stdLogger.Errorf("total stake is 0!")
		return false, 0
	}
	value := base.VRF_proof2hash(pi)

	br := new(big.Rat).SetInt(new(big.Int).SetBytes(value))
	pr := br.Quo(br, max256)

	//brTStake := new(big.Rat).SetFloat64(float64(totalStake))
	vs := vrfThreshold(stake, totalStake)

	s1, _ := pr.Float64()
	s2, _ := vs.Float64()
	blog := newBizLog("vrfSatisfy")

	ok = pr.Cmp(vs) < 0
	//计算qn
	if vs.Cmp(rat1) > 0 {
		vs.Set(rat1)
	}

	step := vs.Quo(vs, new(big.Rat).SetInt64(int64(model.Param.MaxQN)))

	st, _ := step.Float64()

	r, _ := pr.Quo(pr, step).Float64()
	qn = uint64(math.Floor(r) + 1)

	blog.log("minerstake %v, totalstake %v, proveValue %v, stake %v, step %v, qn %v", stake, totalStake, s1, s2, st, qn)

	return
	//return true
}

func vrfVerifyBlock(bh *types.BlockHeader, preBH *types.BlockHeader, miner *model.MinerDO, totalStake uint64) (bool, error) {
	pi := base.VRFProve(bh.ProveValue.Bytes())
	ok, err := base.VRF_verify(miner.VrfPK, pi, vrfM(preBH.Random, bh.Height-preBH.Height))
	if !ok {
		return ok, err
	}
	if ok, qn := vrfSatisfy(pi, miner.Stake, totalStake); ok {
		if bh.TotalQN != qn + preBH.TotalQN {
			return false, errors.New(fmt.Sprintf("qn error.bh hash=%v, height=%v, qn=%v,totalQN=%v, preBH totalQN=%v", bh.Hash.ShortS(), bh.Height, qn, bh.TotalQN, preBH.TotalQN))
		}
		return true, nil
	}
	return false, errors.New("proof not satisfy")
}

func (vrf *vrfWorker) markProposed() {
	atomic.CompareAndSwapInt32(&vrf.status, prove, proposed)
}

func (vrf *vrfWorker) markSuccess() {
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
