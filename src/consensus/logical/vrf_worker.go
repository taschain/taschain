package logical

import (
	"consensus/groupsig"
	"middleware/types"
	"time"
	"sync/atomic"
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

type vrfWorker struct {
	//read only
	baseBH *types.BlockHeader
	castHeight uint64
	expire time.Time

	//writable
	status int32
}

func newVRFWorker(bh *types.BlockHeader, castHeight uint64, expire time.Time) *vrfWorker {
    return &vrfWorker{
    	baseBH: bh,
    	castHeight: castHeight,
    	expire: expire,
    	status: prove,
	}
}

func (vrf *vrfWorker) prove() (bool, uint64) {
    return true, 0
}

func (vrf *vrfWorker) verify(pk groupsig.Pubkey, msg []byte) bool {
    return true
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
	return bh.Hash == vrf.baseBH.Hash && castHeight == vrf.castHeight
}