package logical

import (
	"time"
	"common"
	"sync"
		"middleware/types"
		"strconv"
	"consensus/model"
		"consensus/groupsig"
	"consensus/logical/pow"
	"sync/atomic"
)

/*
**  Creator: pxf
**  Date: 2018/5/29 上午10:19
**  Description: 
*/

//组铸块共识状态（针对某个高度而言）
type CAST_BLOCK_CONSENSUS_STATUS int32

const (
	CBCS_IDLE           CAST_BLOCK_CONSENSUS_STATUS = iota //非当前组
	CBCS_CURRENT                                           //成为当前铸块组
	CBCS_CASTING                                           //至少收到一块组内共识数据
	CBCS_BLOCKED                                           //组内已有铸块完成（已通知到组外）
	CBCS_TIMEOUT                                           //组铸块超时
)

type QUERY_SLOT_RESULT int //查找插槽结果枚举

const (
	QUERY_EMPTY_SLOT   QUERY_SLOT_RESULT = iota //找到一个空槽
	QUERY_REPLACE_SLOT                          //找到一个能替换的槽
	QUERY_EXIST_SLOT                            //插槽已存在
)


const (
	TRANS_INVALID_SLOT          int8	= iota //无效验证槽
	TRANS_DENY                                 //拒绝该交易
	TRANS_ACCEPT_NOT_FULL                      //接受交易, 但仍缺失交易
	TRANS_ACCEPT_FULL_RECOVERED                //接受交易, 无缺失, 验证已达到门限
	TRANS_ACCEPT_FULL_PIECE                    //接受交易, 无缺失, 未达到门限
)

func TRANS_ACCEPT_RESULT_DESC(ret int8) string {
	switch ret {
	case TRANS_INVALID_SLOT:
		return "验证槽无效"
	case TRANS_DENY:
		return "不接收该批交易"
	case TRANS_ACCEPT_NOT_FULL:
		return "接收交易,但仍缺失"
	case TRANS_ACCEPT_FULL_PIECE:
		return "交易收齐,等待分片"
	case TRANS_ACCEPT_FULL_RECOVERED:
		return "交易收齐,分片已到门限"
	}
	return strconv.FormatInt(int64(ret), 10)
}

type VerifyContext struct {
	prevTime   time.Time
	prevHash   common.Hash
	prevRand   []byte
	prevSign   []byte
	castHeight uint64

	powResult 	*pow.PreConfirmedPowResult

	expireTime	time.Time			//铸块超时时间

	consensusStatus CAST_BLOCK_CONSENSUS_STATUS //铸块状态
	proposal 	bool		//自己是否已经提案

	slots [model.MAX_CAST_SLOT]*SlotContext

	blockCtx *BlockContext

	lock sync.RWMutex
}

func newVerifyContext(bc *BlockContext, castHeight uint64, expire time.Time, preBH *types.BlockHeader) *VerifyContext {
	ctx := &VerifyContext{}
	ctx.rebase(bc, castHeight, expire, preBH)
	return ctx
}

func (vc *VerifyContext) resetSlotContext() {
	for i := 0; i < model.MAX_CAST_SLOT; i++ {
		sc := new(SlotContext)
		sc.reset(vc.prevRand, vc.blockCtx.threshold())
		vc.slots[i] = sc
	}
}

func (vc *VerifyContext) getStatus() CAST_BLOCK_CONSENSUS_STATUS {
    return (CAST_BLOCK_CONSENSUS_STATUS)(atomic.LoadInt32((*int32)(&vc.consensusStatus)))
}

func (vc *VerifyContext) setStatus(st CAST_BLOCK_CONSENSUS_STATUS) {
	atomic.StoreInt32((*int32)(&vc.consensusStatus), int32(st))
}

func (vc *VerifyContext) isCasting() bool {
	return vc.getStatus() != CBCS_IDLE && !vc.castTimeout()
}

func (vc *VerifyContext) castTimeout() bool {
    return vc.getStatus() == CBCS_TIMEOUT || time.Now().After(vc.expireTime)
}

func (vc *VerifyContext) rebase(bc *BlockContext, castHeight uint64, expire time.Time, preBH *types.BlockHeader)  {
    vc.prevTime = preBH.CurTime
    vc.prevHash = preBH.Hash
    vc.castHeight = castHeight
    vc.prevSign = preBH.Signature
    vc.prevRand = RealRandom(preBH)
    vc.blockCtx = bc
	vc.expireTime = expire
	vc.consensusStatus = CBCS_CURRENT
	vc.powResult = bc.worker.LoadConfirm()
	vc.resetSlotContext()
}

func (vc *VerifyContext) markProposal() {
    vc.proposal = true
}

func (vc *VerifyContext) isProposal() bool {
    return vc.proposal
}

func (vc *VerifyContext) canProposal(id groupsig.ID) bool {
	if _, mn := vc.powResult.GetMinerNonce(id); mn != nil {
		return true
	}
	return false
}

func (vc *VerifyContext) setTimeout() {
	vc.setStatus(CBCS_TIMEOUT)
}

//检查目前slot中排名最后的
func (vc *VerifyContext) findLastRankSlot() *SlotContext {
	var (
		lastRank = -1
		sc       *SlotContext
	)
	
	for _, v := range vc.slots {
		if v.rank > lastRank {
			lastRank = v.rank
			sc = v
		}
	}
	return sc
}

//根据出块者尝试找到有效的插槽
func (vc *VerifyContext) findSlot(proposer groupsig.ID) (*SlotContext, QUERY_SLOT_RESULT) {

	//寻找已存在的槽
	for _, slot := range vc.slots {
		if slot.proposerNonce != nil && slot.proposerNonce.MinerID.IsEqual(proposer) {
			return slot, QUERY_EXIST_SLOT
		}
	}

	//寻找空槽
	for _, slot := range vc.slots {
		if slot.IsInValid() {
			return slot, QUERY_EMPTY_SLOT
		}
	}

	//寻找失败的非空槽
	for _, slot := range vc.slots {
		if slot.IsFailed() {
			return slot, QUERY_REPLACE_SLOT
		}
	}

	//找到排名最后且未完成签名的slot， 替换之
	slot := vc.findLastRankSlot()
	return slot, QUERY_REPLACE_SLOT

}

func (vc *VerifyContext) castSuccess() bool {
    return vc.getStatus() == CBCS_BLOCKED
}

func (vc *VerifyContext) acceptCV(bh *types.BlockHeader, si *model.SignData) (CAST_BLOCK_MESSAGE_RESULT, *SlotContext) {
	if vc.castTimeout() {
		return CBMR_TIMEOUT, nil
	}
	if bh.GenHash() != si.DataHash {
		panic("SlotContext::AcceptPiece arg failed, hash not samed 1.")
	}
	if bh.Hash != si.DataHash {
		panic("SlotContext::AcceptPiece arg failed, hash not samed 2.")
	}

	if vc.powResult == nil {
		return CBMR_GROUP_POW_RESULT_NOTFOUND, nil
	}
	if vc.castSuccess() {
		return CBMR_CAST_BROADCAST, nil
	}

	proposer := *groupsig.DeserializeId(bh.Castor)

	slot, ret := vc.findSlot(proposer)
	if slot == nil { //没有找到有效的插槽
		return CBMR_NO_AVAIL_SLOT, nil
	}

	switch ret {
	case QUERY_EXIST_SLOT:
		if slot.IsFailed() {
			return CBMR_STATUS_FAIL, nil
		}
	case QUERY_EMPTY_SLOT, QUERY_REPLACE_SLOT:
		rank, nonce := vc.powResult.GetMinerNonce(proposer)
		if nonce == nil {
			return CBMR_PROPOSER_POW_RESULT_NOTFOUND, nil
		}
		minerNonces := vc.blockCtx.getMinerNonceFromBlockHeader(bh)
		if !vc.powResult.CheckEqual(minerNonces) {	//校验pow结果与本地是否一致
			return CBMR_POW_RESULT_DIFF, nil
		}
		slot.prepare(bh, nonce, rank, vc.prevRand, vc.blockCtx.threshold())
	}
	return slot.AcceptPiece(bh, si), slot
}

//完成某个铸块槽的铸块（上链，组外广播）后，更新组的当前高度铸块状态
func (vc *VerifyContext) setCastBroadcasted() {
	vc.setStatus(CBCS_BLOCKED)
}

//收到某个验证人的验证完成消息（可能会比铸块完成消息先收到）
func (vc *VerifyContext) UserVerified(bh *types.BlockHeader, sd *model.SignData) (CAST_BLOCK_MESSAGE_RESULT, *SlotContext) {
	vc.lock.Lock()
	defer vc.lock.Unlock()

	return vc.acceptCV(bh, sd) //>=0为消息正确接收
}

//（网络接收）新到交易集通知
//返回不再缺失交易的QN槽列表
func (vc *VerifyContext) AcceptTrans(slot *SlotContext, ths []common.Hash) int8 {
	vc.lock.Lock()
	defer vc.lock.Unlock()

	if slot.IsInValid() {
		return TRANS_INVALID_SLOT
	}
	accept := slot.AcceptTrans(ths)
	if !accept {
		return TRANS_DENY
	}
	if !slot.IsTransFull() {
		return TRANS_ACCEPT_NOT_FULL
	}
	if slot.IsRecovered() {
		return TRANS_ACCEPT_FULL_RECOVERED
	} else {
		return TRANS_ACCEPT_FULL_PIECE
	}
}

//判断该context是否可以删除
func (vc *VerifyContext) ShouldRemove(topHeight uint64) bool {
	vc.lock.RLock()
	defer vc.lock.RUnlock()

	//不在铸块或者已出最大块的, 可以删除
	if !vc.isCasting() || vc.castSuccess() {
		return true
	}
	//有槽已验证的, 可以删除
	for _, slt := range vc.slots {
		if slt.IsCastSuccess() {
			return true
		}
	}

	if vc.castHeight+20 < topHeight {
		return true
	}

	return false
}
