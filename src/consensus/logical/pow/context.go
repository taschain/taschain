package pow

import (
	"common"
	"consensus/groupsig"
	"consensus/model"
	"math/big"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"unsafe"
)

/*
**  Creator: pxf
**  Date: 2018/8/10 下午2:19
**  Description:
 */

type powResult struct {
	minerID groupsig.ID
	nonce   uint64
	value   *big.Int
	level   uint32
}

func (pr *powResult) toBytes() []byte {
	bs := pr.minerID.Serialize()
	bs = strconv.AppendUint(bs, pr.nonce, 10)
	bs = append(bs, pr.value.Bytes()...)
	bs = strconv.AppendInt(bs, int64(pr.level), 10)
	return bs
}

type powResults []*powResult

func (prs powResults) Len() int {
	return len(prs)
}

func (prs powResults) Less(i, j int) bool {
	return prs[i].value.Cmp(prs[j].value) < 0
}

func (prs powResults) Swap(i, j int) {
	tmp := prs[i]
	prs[i] = prs[j]
	prs[j] = tmp
}

func (prs powResults) totalLevel() uint32 {
	t := uint32(0)
	for _, pr := range prs {
		t += pr.level
	}
	return t
}

func (prs powResults) totalValue() *big.Int {
	tv := new(big.Int).SetInt64(0)
	for _, pr := range prs {
		tv = tv.Add(tv, pr.value)
	}
	return tv
}

type powConfirm struct {
	resultHash     common.Hash
	results        powResults
	totalLevel     uint32
	totalValue     *big.Int
	gSignGenerator *model.GroupSignGenerator
	gSign          *groupsig.Signature
}

func (pc *powConfirm) genNonceSeq() []model.MinerNonce {
	mns := make([]model.MinerNonce, 0)
	for _, r := range pc.results {
		mns = append(mns, model.MinerNonce{MinerID: r.minerID, Nonce: r.nonce})
	}
	return mns
}

func (pc *powConfirm) genHash(blockHash common.Hash) {
	msgTmp := model.ConsensusPowConfirmMessage{
		BaseHash: blockHash,
		NonceSeq: pc.genNonceSeq(),
	}
	pc.resultHash = msgTmp.GenHash()
}

func (pc *powConfirm) addSign(uid groupsig.ID, signature groupsig.Signature) bool {
	add, gen := pc.gSignGenerator.AddWitness(uid, signature)
	if add && gen {
		sig := pc.gSignGenerator.GetGroupSign()
		pc.gSign = &sig
		return true
	}
	return false
}

type powContext struct {
	blockHash common.Hash
	results   map[string]*powResult
	resultK   int
	confirmed *powConfirm

	//confirms  sync.Map	//hash -> *powConfirm
	lock sync.RWMutex
}

func newPowContext(blockHash common.Hash, members int) *powContext {
	return &powContext{
		blockHash: blockHash,
		results:   make(map[string]*powResult),
		resultK:   model.Param.GetPowGroupK(members),
	}
}

func (pc *powContext) addResult(r *powResult) bool {
	pc.lock.Lock()
	defer pc.lock.Unlock()
	hex := r.minerID.GetHexString()
	if _, ok := pc.results[hex]; ok {
		return false
	} else {
		pc.results[hex] = r
		return true
	}
}

func (pc *powContext) thresholdReached() bool {
	pc.lock.RLock()
	defer pc.lock.RUnlock()
	return len(pc.results) >= pc.resultK
}

func (pc *powContext) getResult(mn *model.MinerNonce) *powResult {
	pc.lock.RLock()
	defer pc.lock.RUnlock()
	if r, ok := pc.results[mn.MinerID.GetHexString()]; ok && r.nonce == mn.Nonce {
		return r
	}
	return nil
}

func (pc *powContext) newPowConfirm(results powResults) *powConfirm {
	confirm := &powConfirm{
		results:        results,
		totalLevel:     results.totalLevel(),
		totalValue:     results.totalValue(),
		gSignGenerator: model.NewGroupSignGenerator(pc.resultK),
	}
	confirm.genHash(pc.blockHash)
	return confirm
}

func (pc *powContext) confirm() bool {
	if !pc.thresholdReached() {
		return false
	}
	if pc.hasConfirmed() {
		return false
	}
	results := make(powResults, len(pc.results))
	i := 0
	for _, r := range pc.results {
		results[i] = r
		i++
	}
	sort.Sort(results)

	results = results[:pc.resultK]

	confirm := pc.newPowConfirm(results)
	p := unsafe.Pointer(pc.confirmed)
	return atomic.CompareAndSwapPointer(&p, nil, unsafe.Pointer(confirm))
}

func (pc *powContext) setConfirmed(cf *powConfirm) {
	p := unsafe.Pointer(pc.confirmed)
	atomic.StorePointer(&p, unsafe.Pointer(cf))
}

func (pc *powContext) hasConfirmed() bool {
	p := unsafe.Pointer(pc.confirmed)
	return atomic.LoadPointer(&p) != nil
}

func (pc *powContext) getConfirmed() *powConfirm {
	p := unsafe.Pointer(pc.confirmed)
	if rp := atomic.LoadPointer(&p); rp == nil {
		return nil
	} else {
		return (*powConfirm)(rp)
	}
}



func (pc *powContext) addSign(uid groupsig.ID, signature groupsig.Signature) bool {
	return pc.getConfirmed().addSign(uid, signature)
}
