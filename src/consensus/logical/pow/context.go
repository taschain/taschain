package pow

import (
		"math/big"
	"sync"
	"consensus/groupsig"
	"common"
	"strconv"
	"consensus/base"
	"sort"
	"sync/atomic"
	"unsafe"
)

/*
**  Creator: pxf
**  Date: 2018/8/10 下午2:19
**  Description: 
*/

type powResult struct {
	minerID		groupsig.ID
	nonce      uint64
	value      *big.Int
	level      int32
}

func (pr *powResult) marshal() []byte {
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

func (prs powResults) totalLevel() int32 {
	t := int32(0)
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
	resultHash common.Hash
	results powResults
	totalLevel int32
	totalValue *big.Int
	signs 	map[string]groupsig.Signature
	gSign 	groupsig.Signature
}

func (pc *powConfirm) genHash() {
    bs := make([]byte, 0)
	for _, r := range pc.results {
		bs = append(bs, r.marshal()...)
	}
	bs = strconv.AppendInt(bs, int64(pc.totalLevel), 10)
	bs = append(bs, pc.totalValue.Bytes()...)
	hash := base.Data2CommonHash(bs)
	pc.resultHash = hash
}

type powContext struct {
	results   map[string]*powResult
	threshold int
	confirmed *powConfirm

	confirms  sync.Map	//hash -> *powConfirm
	lock sync.RWMutex
}

func newPowContext(threshold int) *powContext {
	return &powContext{
		results: make(map[string]*powResult),
		threshold: threshold,
	}
}

func (pc *powContext) add(r *powResult) bool {
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

func (pc *powContext) addConfirm(confirm *powConfirm) bool {
    _, ok := pc.confirms.LoadOrStore(confirm.resultHash, confirm)
    return !ok
}

func (pc *powContext) confirm()  {
	if !pc.threasholdReached() {
		return
	}
    results := make(powResults, len(pc.results))
	i := 0
	for _, r := range pc.results {
		results[i] = r
		i++
	}
	sort.Sort(results)

	results = results[:pc.threshold]

	confirm := &powConfirm{
		results: results,
		totalLevel: results.totalLevel(),
		totalValue: results.totalValue(),
		signs: map[string]groupsig.Signature{},
	}
	confirm.genHash()
	if pc.addConfirm(confirm) {
		p := unsafe.Pointer(pc.confirmed)
		atomic.StorePointer(&p, unsafe.Pointer(confirm))
	}
}

func (pc *powContext) hasConfirmed() bool {
	p := unsafe.Pointer(pc.confirmed)
    return atomic.LoadPointer(&p) != nil
}

func (pc *powContext) threasholdReached() bool {
	pc.lock.RLock()
	defer pc.lock.RUnlock()
    return len(pc.results) >= pc.threshold
}
