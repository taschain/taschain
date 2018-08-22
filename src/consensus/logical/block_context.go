package logical

import (
	"common"
	"log"
	"time"
	"fmt"
	"sync"
	"middleware/types"
	"consensus/model"
	"consensus/logical/pow"
	"consensus/groupsig"
)

///////////////////////////////////////////////////////////////////////////////
//组铸块共识上下文结构
type BlockContext struct {
	Version         uint
	GroupMembers    int                        //组成员数量
	Proc    *Processor   //处理器
	MinerID model.GroupMinerID //矿工ID和所属组ID
	pos     int          //矿工在组内的排位
	groupInfo 		*StaticGroupInfo

	worker			*pow.PowWorker
	verifyContexts	[]*VerifyContext
	currentVerifyContext *VerifyContext //当前铸块的verifycontext
	ctrlCh			chan struct{}

	lock sync.RWMutex
}

func NewBlockContext(p *Processor, sgi *StaticGroupInfo) *BlockContext {
	bc := &BlockContext{
		Proc: p,
		MinerID: *model.NewGroupMinerID(sgi.GroupID, p.GetMinerID()),
		verifyContexts: make([]*VerifyContext, 0),
		GroupMembers: len(sgi.Members),
		Version: model.CONSENSUS_VERSION,
		worker: pow.NewPowWorker(p.storage),
		ctrlCh: make(chan struct{}),
		groupInfo: sgi,
	}

	go bc.powWorkerLoop()

	bc.reset()
	return bc
}


func (bc *BlockContext) powWorkerLoop()  {
	worker := bc.worker
	FOR:
	for {
		select {
		case cmd := <- worker.CmdCh:
			switch cmd {
			case pow.CMD_POW_RESULT:
				bc.Proc.onPowComputedDeadline(worker)
			case pow.CMD_POW_CONFIRM:
				bc.Proc.onPowConfirmDeadline(worker)
			}
		case <-bc.ctrlCh:
			break FOR
		}
	}
}

func (bc *BlockContext) threshold() int {
    return model.Param.GetGroupK(bc.GroupMembers)
}

func (bc *BlockContext) finalize()  {
    bc.ctrlCh <- struct{}{}
}

//func (bc *BlockContext) getKingCheckRoutineName() string {
//	return "king_check_routine_" + GetIDPrefix(bc.MinerID.Gid)
//}

func (bc *BlockContext) alreadyInCasting(height uint64, preHash common.Hash) bool {
	vctx := bc.GetCurrentVerifyContext()
	if vctx != nil {
		vctx.lock.Lock()
		defer vctx.lock.Unlock()
		return vctx.isCasting() && !vctx.castSuccess() && vctx.castHeight == height && vctx.prevHash == preHash && (!vctx.canProposal(bc.MinerID.Uid) || vctx.isProposal())
	} else {
		return false
	}
}

func (bc *BlockContext) getVerifyContext(height uint64, preHash common.Hash) (int32, *VerifyContext) {

	for idx, ctx := range bc.verifyContexts {
		if ctx.castHeight == height && ctx.prevHash == preHash {
			return int32(idx), ctx
		}
	}
	return -1, nil
}

func (bc *BlockContext) GetCurrentVerifyContext() *VerifyContext {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	return bc.currentVerifyContext
}

func (bc *BlockContext) GetOrNewVerifyContext(bh *types.BlockHeader, preBH *types.BlockHeader) (int32, *VerifyContext) {
	expireTime := GetCastExpireTime(bh.PreTime, bh.Height - preBH.Height)

	bc.lock.Lock()
	defer bc.lock.Unlock()

	if idx, vctx := bc.getVerifyContext(bh.Height, bh.PreHash); vctx == nil {
		vctx = newVerifyContext(bc, bh.Height, expireTime, preBH)
		bc.verifyContexts = append(bc.verifyContexts, vctx)
		return int32(len(bc.verifyContexts) - 1), vctx
	} else {
		return idx, vctx
	}
}

func (bc *BlockContext) CleanVerifyContext(height uint64)  {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	newCtxs := make([]*VerifyContext, 0)
	for _, ctx := range bc.verifyContexts {
		if !ctx.ShouldRemove(height) {
			newCtxs = append(newCtxs, ctx)
		} else {
			if bc.currentVerifyContext == ctx {
				bc.reset()
			}
			log.Printf("CleanVerifyContext: ctx.castHeight=%v, ctx.prevHash=%v\n", ctx.castHeight, GetHashPrefix(ctx.prevHash))
		}
	}
	bc.verifyContexts = newCtxs
}

func (bc *BlockContext) SafeGetVerifyContexts() []*VerifyContext {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	return bc.verifyContexts
}


func (bc *BlockContext) castingInfo() string {
	vctx := bc.currentVerifyContext
	if vctx != nil {
		return fmt.Sprintf("status=%v, castHeight=%v, prevHash=%v, prevTime=%v, casted=%v", vctx.consensusStatus, vctx.castHeight, GetHashPrefix(vctx.prevHash), vctx.prevTime.String(), vctx.castSuccess())
	} else {
		return "not in casting!"
	}
}

func (bc *BlockContext) Reset() {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	bc.reset()
}

//铸块上下文复位，在某个高度轮到当前组铸块时调用。
//to do : 还是索性重新生成。
func (bc *BlockContext) reset() {

	bc.currentVerifyContext = nil
	//bc.Proc.Ticker.StopTickerRoutine(bc.getKingCheckRoutineName())
}

//开始铸块
func (bc *BlockContext) PrepareForProposal(castHeight uint64, expire time.Time, baseBH *types.BlockHeader) {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	var verifyCtx *VerifyContext
	if _, verifyCtx = bc.getVerifyContext(castHeight, baseBH.Hash); verifyCtx != nil {
		//verifyCtx.Rebase(bc, castHeight, preTime, preHash)
		if !verifyCtx.isCasting() {
			verifyCtx.rebase(bc, castHeight, expire, baseBH)
		}
		bc.currentVerifyContext = verifyCtx

	} else {
		verifyCtx = newVerifyContext(bc, castHeight, expire, baseBH)
		bc.verifyContexts = append(bc.verifyContexts, verifyCtx)
		bc.currentVerifyContext = verifyCtx
	}

}

func (bc *BlockContext) getMemIdByIndex(index int) groupsig.ID {
	if index >= bc.groupInfo.MemberCount() {
		return groupsig.ID{}
	}
	return bc.groupInfo.Members[index].ID
}

func (bc *BlockContext) getMinerNonceFromBlockHeader(bh *types.BlockHeader) []model.MinerNonce {
	return GetMinerNonceFromBlockHeader(bh, bc.groupInfo)
}

func (bc *BlockContext) startPowComputation(bh *types.BlockHeader) {
	var (
		baseHash  common.Hash
		startTime time.Time
		worker    = bc.worker
		height 		uint64
	)
	if bh == nil {
		baseHash = bc.groupInfo.Signature.GetHash()
		startTime = time.Now()
		height = 0
	} else {
		baseHash = bh.Hash
		height = bh.Height
		startTime = bh.CurTime
	}
	if worker.Prepare(baseHash, height, startTime, &bc.MinerID, bc.groupInfo.MemberCount()) {
		log.Printf("startPowComputation height=%v, hash=%v, gid=%v\n", height, GetHashPrefix(baseHash), GetIDPrefix(bc.groupInfo.GroupID))
		logStart("POW_COMP", height, GetIDPrefix(bc.MinerID.Uid), "", "")
		worker.Start()
	}
}

func (bc *BlockContext) isPowWorking() bool {
	return bc.worker.IsRunning()
}

type CastBlockContexts struct {
	contexts sync.Map	//string -> *BlockContext
}

func NewCastBlockContexts() *CastBlockContexts {
	return &CastBlockContexts{
		contexts: sync.Map{},
	}
}

func (bctx *CastBlockContexts) addBlockContext(bc *BlockContext) (add bool) {
	_, load := bctx.contexts.LoadOrStore(bc.MinerID.Gid.GetHexString(), bc)
	return !load
}

func (bctx *CastBlockContexts) getBlockContext(gid groupsig.ID) *BlockContext {
	if v, ok := bctx.contexts.Load(gid.GetHexString()); ok {
		return v.(*BlockContext)
	}
	return nil
}

func (bctx *CastBlockContexts) contextSize() int32 {
	size := int32(0)
	bctx.contexts.Range(func(key, value interface{}) bool {
		size ++
		return true
	})
	return size
}

func (bctx *CastBlockContexts) removeContexts(gids []groupsig.ID)  {
	for _, id := range gids {
		log.Println("removeContexts ", GetIDPrefix(id))
		bc := bctx.getBlockContext(id)
		if bc != nil {
			bc.finalize()
			bctx.contexts.Delete(id.GetHexString())
		}
	}
}

func (bctx *CastBlockContexts) forEach(f func(bc *BlockContext) bool) {
	bctx.contexts.Range(func(key, value interface{}) bool {
		v := value.(*BlockContext)
		return f(v)
	})
}
