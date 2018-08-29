package logical

import (
	"time"
	"consensus/groupsig"
	"common"
	"sync/atomic"
	"middleware/types"
	"strconv"
	"core"
	"unsafe"
	"consensus/model"
	"sync"
)

/*
**  Creator: pxf
**  Date: 2018/5/21 下午5:49
**  Description: 
*/

//铸块槽状态
type SLOT_STATUS int

const (
	SS_WAITING      int32 = iota //等待签名片段达到阈值
	SS_BRAODCASTED               //是否已经广播验证过
	SS_RECOVERD                  //恢复出组签名
	SS_VERIFIED                  //组签名用组公钥验证通过
	SS_ONCHAIN                   //已上链
	SS_FAILED_CHAIN              //链反馈失败，不可逆
	SS_FAILED                    //铸块过程中失败，不可逆
)

//铸块槽结构，和某个KING的共识数据一一对应
type SlotContext struct {
	TimeRev time.Time //插槽被创建的时间（也就是接收到该插槽第一包数据的时间）
	//HeaderHash   common.Hash                   //出块头哈希(就这个哈希值达成一致)
	BH          types.BlockHeader             //出块头详细数据
	QueueNumber int64                         //铸块槽序号(<0无效)，等同于出块人序号。
	King        groupsig.ID                   //出块者ID
	MapWitness  map[string]groupsig.Signature //该铸块槽的见证人验证签名列表
	lock 		sync.RWMutex
	GroupSign   groupsig.Signature            //成功输出的组签名
	SlotStatus  int32
	LosingTrans map[common.Hash]int //本地缺失的交易集
	transFulled *bool                //针对该区块头的交易集在本地链上已全部存在
	threshold   int
}

func (sc *SlotContext) IsTransFull() bool {
	unsafePtr := unsafe.Pointer(&sc.transFulled)
	return *(*bool)(atomic.LoadPointer(&unsafePtr))
}

func (sc *SlotContext) setTransFull(full bool) {
	unsafePtr := unsafe.Pointer(&sc.transFulled)
	atomic.StorePointer(&unsafePtr, unsafe.Pointer(&full))
}

func (sc *SlotContext) setSlotStatus(st int32) {
	atomic.StoreInt32(&sc.SlotStatus, st)
}

func (sc *SlotContext) IsFailed() bool {
	st := sc.slotStatus()
	return st == SS_FAILED_CHAIN || st == SS_FAILED
}

func (sc *SlotContext) slotStatus() int32 {
	return atomic.LoadInt32(&sc.SlotStatus)
}

func (sc SlotContext) lostTransSize() int {
	return len(sc.LosingTrans)
}

func (sc *SlotContext) InitLostingTrans(ths []common.Hash) {
	for _, v := range ths {
		sc.LosingTrans[v] = 0
	}
	sc.setTransFull(len(sc.LosingTrans) == 0)
	return
}

//用接收到的新交易更新缺失的交易集
//返回接收前以及接收后是否不在缺失交易
func (sc *SlotContext) AcceptTrans(ths []common.Hash) (bool) {
	if len(sc.LosingTrans) == 0 { //已经无缺失
		return false
	}
	accept := false
	for _, th := range ths {
		if _, ok := sc.LosingTrans[th]; ok {
			accept = true
			break
		}
	}
	if accept {
		for _, th := range ths {
			delete(sc.LosingTrans, th)
		}
	}
	sc.setTransFull(len(sc.LosingTrans) == 0)
	return accept
}

func (sc SlotContext) MessageSize() int {
	return len(sc.MapWitness)
}

//验证组签名
//pk：组公钥
//返回true验证通过，返回false失败。
func (sc *SlotContext) VerifyGroupSign(pk groupsig.Pubkey) bool {
	st := sc.slotStatus()
	if st == SS_VERIFIED { //已经验证过组签名
		return true
	}
	if st != SS_RECOVERD || !sc.GroupSign.IsValid() {
		return false
	}
	b := groupsig.VerifySig(pk, sc.BH.Hash.Bytes(), sc.GroupSign)
	if b {
		sc.setSlotStatus(SS_VERIFIED) //组签名验证通过
	}
	return b
}

func (sc SlotContext) GetGroupSign() groupsig.Signature {
	return sc.GroupSign
}

//（达到超过阈值的签名片段后）生成组签名
//如成功，则置位成员变量GroupSign和GSStatus，返回true。
func (sc *SlotContext) GenGroupSign() bool {
	st := sc.slotStatus()
	if st == SS_RECOVERD || st == SS_VERIFIED {
		return true
	}
	if st == SS_FAILED {
		return false
	}
	if sc.thresholdWitnessGot() /* && sc.HasKingMessage() */ { //达到组签名恢复阈值，且当前节点收到了出块人消息
		sc.lock.RLock()
		defer sc.lock.RUnlock()
		gs := groupsig.RecoverSignatureByMapI(sc.MapWitness, sc.threshold)
		if gs != nil {
			sc.GroupSign = *gs
			sc.setSlotStatus(SS_RECOVERD)
			return true
		} else {
			sc.setSlotStatus(SS_FAILED)
			panic("CastContext::GenGroupSign failed, groupsig.RecoverSign return nil.")
		}
	}
	return false
}

func (sc *SlotContext) IsVerified() bool {
	return sc.slotStatus() == SS_VERIFIED
}

type CAST_BLOCK_MESSAGE_RESULT int8 //出块和验证消息处理结果枚举

const (
	CBMR_PIECE_NORMAL         CAST_BLOCK_MESSAGE_RESULT = iota //收到一个分片，接收正常
	CBMR_PIECE_LOSINGTRANS                                     //收到一个分片, 缺失交易
	CBMR_THRESHOLD_SUCCESS                                     //收到一个分片且达到阈值，组签名成功
	CBMR_THRESHOLD_FAILED                                      //收到一个分片且达到阈值，组签名失败
	CBMR_IGNORE_REPEAT                                         //丢弃：重复收到该消息
	CBMR_IGNORE_QN_BIG_QN                                      //丢弃：QN太大
	CBMR_IGNORE_QN_ERROR                                       //丢弃：qn错误
	CBMR_IGNORE_KING_ERROR                                     //丢弃：king错误
	CBMR_IGNORE_MAX_QN_SIGNED                                  //丢弃：该节点已向组外广播出更低QN值的块
	CBMR_STATUS_FAIL                                           //已经失败的
	CBMR_ERROR_UNKNOWN                                         //异常：未知异常
)

func CBMR_RESULT_DESC(ret CAST_BLOCK_MESSAGE_RESULT) string {
	switch ret {
	case CBMR_PIECE_NORMAL:
		return "正常分片"
	case CBMR_PIECE_LOSINGTRANS:
		return "交易缺失"
	case CBMR_THRESHOLD_SUCCESS:
		return "达到门限值组签名成功"
	case CBMR_THRESHOLD_FAILED:
		return "达到门限值但组签名失败"
	case CBMR_IGNORE_QN_BIG_QN, CBMR_IGNORE_QN_ERROR:
		return "qn错误"
	case CBMR_IGNORE_KING_ERROR:
		return "king错误"
	case CBMR_IGNORE_MAX_QN_SIGNED:
		return "已出更大qn"
	case CBMR_STATUS_FAIL:
		return "失败状态"
	case CBMR_IGNORE_REPEAT:
		return "重复消息"
	}
	return strconv.FormatInt(int64(ret), 10)
}

func (sc *SlotContext) addSign(id groupsig.ID, sign groupsig.Signature)  {
    sc.lock.Lock()
    defer sc.lock.Unlock()
	sc.MapWitness[id.GetHexString()] = sign
}

func (sc *SlotContext) getSign(id groupsig.ID) (groupsig.Signature, bool) {
	sc.lock.RLock()
	defer sc.lock.RUnlock()
    v, ok := sc.MapWitness[id.GetHexString()]
    return v, ok
}

//收到一个组内验证签名片段
//返回：=0, 验证请求被接受，阈值达到组签名数量。=1，验证请求被接受，阈值尚未达到组签名数量。=2，重复的验签。=3，数据异常。
func (sc *SlotContext) AcceptPiece(bh types.BlockHeader, si model.SignData) CAST_BLOCK_MESSAGE_RESULT {
	if bh.GenHash() != si.DataHash {
		panic("SlotContext::AcceptPiece arg failed, hash not samed 1.")
	}
	if bh.Hash != si.DataHash {
		panic("SlotContext::AcceptPiece arg failed, hash not samed 2.")
	}

	if si.DataHash != sc.BH.Hash {
		panic("SlotContext::AcceptPiece failed, hash diff.")
	}
	v, ok := sc.getSign(si.SignMember)
	if ok { //已经收到过该成员的验签
		if !v.IsEqual(si.DataSign) {
			panic("CastContext::Verified failed, one member's two sign diff.")
		}
		//忽略
		return CBMR_IGNORE_REPEAT
	} else { //没有收到过该用户的签名
		//sc.MapWitness[si.GetID().GetHexString()] = si.DataSign
		sc.addSign(si.SignMember, si.DataSign)
		//if !sc.transFulled {
		//	return CBMR_PIECE_LOSINGTRANS
		//}
		if sc.thresholdWitnessGot() /* && sc.HasKingMessage() */ { //达到组签名条件; (不一定需要收到king的消息 ? : by wenqin 2018/5/21)
			if sc.GenGroupSign() {
				return CBMR_THRESHOLD_SUCCESS
			} else {
				return CBMR_THRESHOLD_FAILED
			}
		} else {
			return CBMR_PIECE_NORMAL
		}
	}
	return CBMR_ERROR_UNKNOWN
}

func (sc *SlotContext) thresholdWitnessGot() bool {
	sc.lock.RLock()
	defer sc.lock.RUnlock()
    return len(sc.MapWitness) >= sc.threshold
}


//根据（某个QN值）接收到的第一包数据生成一个新的插槽
func newSlotContext(bh *types.BlockHeader, si *model.SignData, threshold int) *SlotContext {
	if bh.GenHash() != si.DataHash {
		panic("newSlotContext arg failed, hash not samed 1.")
	}
	if bh.Hash != si.DataHash {
		//log.Printf("King=%v, sender=%v.\n", bh.Castor)
		panic("newSlotContext arg failed, hash not samed 2")
	}
	sc := new(SlotContext)
	sc.reset(threshold)

	sc.BH = *bh
	sc.QueueNumber = int64(bh.QueueNumber)
	sc.King.Deserialize(bh.Castor)
	sc.addSign(si.SignMember, si.DataSign)

	//if !PROC_TEST_MODE {
	ltl, ccr, _, _ := core.BlockChainImpl.VerifyCastingBlock(*bh)
	sc.InitLostingTrans(ltl)
	if ccr == -1 {
		sc.SlotStatus = SS_FAILED_CHAIN
	}
	//}
	return sc
}

func (sc *SlotContext) reset(threshold int) {
	sc.TimeRev = time.Time{}
	//sc.HeaderHash = *new(common.Hash)
	sc.QueueNumber = model.INVALID_QN
	sc.transFulled = new(bool)
	sc.SlotStatus = SS_WAITING
	sc.MapWitness = make(map[string]groupsig.Signature)
	sc.LosingTrans = make(map[common.Hash]int)
	sc.threshold = threshold
	return
}

func (sc SlotContext) IsValid() bool {
	return sc.QueueNumber > model.INVALID_QN
}

/*
//取得铸块权重
//第一顺为权重1，第二顺位权重2，第三顺位权重4...，即权重越低越好（但0为无效）
func (sc SlotContext) GetWeight() uint64 {
	if sc.QueueNumber <= int64(MAX_QN) {
		return uint64(sc.QueueNumber) << 1
	} else {
		return 0
	}
}
*/
