package logical

import (
	"core"
	"log"
	"time"
	"consensus/groupsig"
	"common"
	"sync/atomic"
	"middleware/types"
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
	GroupSign   groupsig.Signature            //成功输出的组签名
	SlotStatus  int32
	LosingTrans map[common.Hash]int //本地缺失的交易集
	TransFulled bool                //针对该区块头的交易集在本地链上已全部存在
}

func (sc *SlotContext) isAllTransExist() bool {
	return sc.TransFulled
}

func (sc *SlotContext) statusChainFailed() {
	sc.SlotStatus = SS_FAILED_CHAIN
}

func (sc *SlotContext) IsFailed() bool {
	return sc.SlotStatus == SS_FAILED_CHAIN || sc.SlotStatus == SS_FAILED
}

func (sc *SlotContext) InitLostingTrans(ths []common.Hash) {
	log.Printf("slot begin InitLostingTrans, cur_count=%v, input_count=%v...\n", len(sc.LosingTrans), len(ths))
	if sc.TransFulled {
		panic("SlotContext::InitLostingTrans failed, transFulled=true")
	}
	sc.LosingTrans = make(map[common.Hash]int)
	for _, v := range ths {
		sc.LosingTrans[v] = 0
	}
	sc.TransFulled = len(sc.LosingTrans) == 0
	log.Printf("slot end InitLostingTrans, cur_count=%v, fulled=%v.\n", len(sc.LosingTrans), sc.TransFulled)
	return
}

//用接收到的新交易更新缺失的交易集
//返回接收前以及接收后是否不在缺失交易
func (sc *SlotContext) ReceTrans(ths []common.Hash) (bool) {
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
	sc.TransFulled = len(sc.LosingTrans) == 0
	return accept
}

func (sc SlotContext) MessageSize() int {
	return len(sc.MapWitness)
}

//验证组签名
//pk：组公钥
//返回true验证通过，返回false失败。
func (sc *SlotContext) VerifyGroupSign(pk groupsig.Pubkey) bool {
	if sc.SlotStatus == SS_VERIFIED { //已经验证过组签名
		return true
	}
	if sc.SlotStatus != SS_RECOVERD || !sc.GroupSign.IsValid() {
		return false
	}
	b := groupsig.VerifySig(pk, sc.BH.Hash.Bytes(), sc.GroupSign)
	if b {
		sc.SlotStatus = SS_VERIFIED //组签名验证通过
	}
	return b
}

func (sc SlotContext) GetGroupSign() groupsig.Signature {
	return sc.GroupSign
}

//（达到超过阈值的签名片段后）生成组签名
//如成功，则置位成员变量GroupSign和GSStatus，返回true。
func (sc *SlotContext) GenGroupSign() bool {
	if sc.SlotStatus == SS_RECOVERD || sc.SlotStatus == SS_VERIFIED {
		return true
	}
	if sc.SlotStatus == SS_FAILED {
		return false
	}
	if len(sc.MapWitness) >= GetGroupK() /* && sc.HasKingMessage() */ { //达到组签名恢复阈值，且当前节点收到了出块人消息
		gs := groupsig.RecoverSignatureByMapI(sc.MapWitness, GetGroupK())
		if gs != nil {
			sc.GroupSign = *gs
			sc.SlotStatus = SS_RECOVERD
			return true
		} else {
			sc.SlotStatus = SS_FAILED
			panic("CastContext::GenGroupSign failed, groupsig.RecoverSign return nil.")
		}
	}
	return false
}

func (sc *SlotContext) IsVerified() bool {
	return atomic.LoadInt32(&sc.SlotStatus) == SS_VERIFIED
}

type CAST_BLOCK_MESSAGE_RESULT int8 //出块和验证消息处理结果枚举

const (
	CBMR_PIECE_NORMAL         CAST_BLOCK_MESSAGE_RESULT = iota //收到一个分片，接收正常
	CBMR_PIECE_LOSINGTRANS                                     //收到一个分片, 缺失交易
	CBMR_THRESHOLD_SUCCESS                                     //收到一个分片且达到阈值，组签名成功
	CBMR_THRESHOLD_FAILED                                      //收到一个分片且达到阈值，组签名失败
	CBMR_IGNORE_REPEAT                                         //丢弃：重复收到该消息
	CMBR_IGNORE_QN_BIG_QN                                      //丢弃：QN太大
	CMBR_IGNORE_QN_FUTURE                                      //丢弃：未轮到该QN
	CMBR_IGNORE_QN_ERROR                                       //丢弃：qn错误
	CMBR_IGNORE_KING_ERROR                                     //丢弃：king错误
	CMBR_IGNORE_MAX_QN_SIGNED                                  //丢弃：该节点已向组外广播出更低QN值的块
	CMBR_IGNORE_NOT_CASTING                                    //丢弃：未启动当前组铸块共识
	CBMR_ERROR_ARG                                             //异常：参数异常
	CBMR_ERROR_SIGN                                            //异常：签名验证异常
	CBMR_STATUS_FAIL                                           //已经失败的
	CMBR_ERROR_UNKNOWN                                         //异常：未知异常
)

//收到一个组内验证签名片段
//返回：=0, 验证请求被接受，阈值达到组签名数量。=1，验证请求被接受，阈值尚未达到组签名数量。=2，重复的验签。=3，数据异常。
func (sc *SlotContext) AcceptPiece(bh types.BlockHeader, si SignData) CAST_BLOCK_MESSAGE_RESULT {
	if bh.GenHash() != si.DataHash {
		panic("SlotContext::AcceptPiece arg failed, hash not samed 1.")
	}
	if bh.Hash != si.DataHash {
		panic("SlotContext::AcceptPiece arg failed, hash not samed 2.")
	}

	if len(sc.MapWitness) > GROUP_MAX_MEMBERS || sc.MapWitness == nil {
		panic("CastContext::Verified failed, too many members or map nil.")
	}
	if si.DataHash != sc.BH.Hash {
		log.Printf("SlotContext::AcceptPiece failed, hash diff.\n")
		log.Printf("exist hash=%v.\n", GetHashPrefix(sc.BH.Hash))
		log.Printf("recv hash=%v.\n", GetHashPrefix(si.DataHash))
		panic("SlotContext::AcceptPiece failed, hash diff.")
	}
	v, ok := sc.MapWitness[si.GetID().GetHexString()]
	if ok { //已经收到过该成员的验签
		if !v.IsEqual(si.DataSign) {
			log.Printf("DIFF ERROR: sender=%v, exist_sign=%v, new_sign=%v.\n", GetIDPrefix(si.GetID()), v.GetHexString(), si.DataSign.GetHexString())
			panic("CastContext::Verified failed, one member's two sign diff.")
		}
		//忽略
		return CBMR_IGNORE_REPEAT
	} else { //没有收到过该用户的签名
		sc.MapWitness[si.GetID().GetHexString()] = si.DataSign
		if len(sc.MapWitness) >= GetGroupK() /* && sc.HasKingMessage() */ { //达到组签名条件; (不一定需要收到king的消息 ? : by wenqin 2018/5/21)
			if sc.GenGroupSign() {
				return CBMR_THRESHOLD_SUCCESS
			} else {
				return CBMR_THRESHOLD_FAILED
			}
		} else {
			return CBMR_PIECE_NORMAL
		}
	}
	return CMBR_ERROR_UNKNOWN
}

//判断某个成员是否为插槽的出块人
func (sc SlotContext) IsKing(member groupsig.ID) bool {
	return sc.King == member
}

//根据（某个QN值）接收到的第一包数据生成一个新的插槽
func newSlotContext(bh *types.BlockHeader, si *SignData) *SlotContext {
	if bh.GenHash() != si.DataHash {
		log.Printf("newSlotContext arg failed 1, bh.Gen()=%v, si_hash=%v.\n", GetHashPrefix(bh.GenHash()), GetHashPrefix(si.DataHash))
		panic("newSlotContext arg failed, hash not samed 1.")
	}
	if bh.Hash != si.DataHash {
		log.Printf("newSlotContext arg failed 2, bh_hash=%v, si_hash=%v.\n", GetHashPrefix(bh.Hash), GetHashPrefix(si.DataHash))
		//log.Printf("King=%v, sender=%v.\n", bh.Castor)
		panic("newSlotContext arg failed, hash not samed 2")
	}
	sc := new(SlotContext)
	sc.TimeRev = time.Now()
	sc.SlotStatus = SS_WAITING
	sc.BH = *bh
	//sc.HeaderHash = si.DataHash
	log.Printf("create new slot, hash=%v.\n", GetHashPrefix(sc.BH.Hash))
	sc.QueueNumber = int64(bh.QueueNumber)
	sc.King.Deserialize(bh.Castor)
	sc.MapWitness = make(map[string]groupsig.Signature)
	sc.MapWitness[si.GetID().GetHexString()] = si.DataSign
	sc.LosingTrans = make(map[common.Hash]int)

	if !PROC_TEST_MODE {
		ltl, ccr, _, _ := core.BlockChainImpl.VerifyCastingBlock(*bh)
		log.Printf("BlockChainImpl.VerifyCastingBlock result=%v.", ccr)
		sc.InitLostingTrans(ltl)
		if ccr == -1 {
			sc.SlotStatus = SS_FAILED_CHAIN
		}
	}
	return sc
}

func (sc *SlotContext) reset() {
	sc.TimeRev = *new(time.Time)
	//sc.HeaderHash = *new(common.Hash)
	sc.BH = types.BlockHeader{}
	sc.QueueNumber = INVALID_QN
	sc.SlotStatus = SS_WAITING
	sc.King = *groupsig.NewIDFromInt(0)
	sc.MapWitness = make(map[string]groupsig.Signature)
	sc.LosingTrans = make(map[common.Hash]int)
	return
}

func (sc SlotContext) IsValid() bool {
	return sc.QueueNumber > INVALID_QN
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
