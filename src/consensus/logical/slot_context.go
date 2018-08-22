package logical

import (
		"consensus/groupsig"
	"common"
	"sync/atomic"
	"middleware/types"
	"strconv"
	"core"
	"unsafe"
	"consensus/model"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2018/5/21 下午5:49
**  Description: 
*/


const (
	SS_INVALID  int32 = iota
	SS_WAITING   //等待签名片段达到阈值
	SS_SIGNED    //自己是否签名过
	SS_RECOVERD  //恢复出组签名
	SS_VERIFIED  //组签名用组公钥验证通过
	SS_SUCCESS   //已上链广播
	SS_FAILED    //铸块过程中失败，不可逆
)

type CAST_BLOCK_MESSAGE_RESULT int8 //出块和验证消息处理结果枚举

const (
	CBMR_PIECE_NORMAL         CAST_BLOCK_MESSAGE_RESULT = iota //收到一个分片，接收正常
	CBMR_THRESHOLD_SUCCESS                                     //收到一个分片且达到阈值，组签名成功
	CBMR_STATUS_FAIL                                           //已经失败的
	CBMR_GROUP_POW_RESULT_NOTFOUND								//未发现组pow结果
	CBMR_PROPOSER_POW_RESULT_NOTFOUND							//未发现该提案者的pow结果
	CBMR_CAST_BROADCAST											//已经广播块
	CBMR_NO_AVAIL_SLOT											//未找到合适slot
	CBMR_POW_RESULT_DIFF										//pow结果与本地不一致
	CBMR_TIMEOUT												//超时
)

func CBMR_RESULT_DESC(ret CAST_BLOCK_MESSAGE_RESULT) string {
	switch ret {
	case CBMR_PIECE_NORMAL:
		return "正常分片"
	case CBMR_THRESHOLD_SUCCESS:
		return "达到门限值组签名成功"
	case CBMR_PROPOSER_POW_RESULT_NOTFOUND:
		return "未找到提案者的pow结果"
	case CBMR_GROUP_POW_RESULT_NOTFOUND:
		return "未找到改组的pow结果"
	case CBMR_CAST_BROADCAST:
		return "块已广播"
	case CBMR_NO_AVAIL_SLOT:
		return "找不到有效插槽"
	case CBMR_STATUS_FAIL:
		return "失败状态"
	case CBMR_TIMEOUT:
		return "铸块共识超时"
	}
	return strconv.FormatInt(int64(ret), 10)
}


//铸块槽结构，和某个KING的共识数据一一对应
type SlotContext struct {
	bh                  types.BlockHeader         //出块头详细数据
	preRands            []byte                    //上一个真随机数
	proposerNonce       *model.MinerNonce         //提案者nonce
	rank                int                       //提案者nonce排名

	blockGSignGenerator *model.GroupSignGenerator //块组签名产生器
	randGSignGenerator  *model.GroupSignGenerator //随机数组签名产生器
	slotStatus          int32
	losingTrans         map[common.Hash]bool //本地缺失的交易集
	transFulled         *bool                //针对该区块头的交易集在本地链上已全部存在
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
	atomic.StoreInt32(&sc.slotStatus, st)
}

func (sc *SlotContext) IsFailed() bool {
	return sc.getSlotStatus() == SS_FAILED
}

func (sc *SlotContext) IsInValid() bool {
	return sc.getSlotStatus() == SS_INVALID
}

func (sc *SlotContext) IsVerified() bool {
	return sc.getSlotStatus() == SS_VERIFIED || sc.getSlotStatus() == SS_SUCCESS
}

func (sc *SlotContext) IsRecovered() bool {
	return sc.getSlotStatus() == SS_RECOVERD || sc.IsVerified()
}

func (sc *SlotContext) IsCastSuccess() bool {
	return sc.getSlotStatus() == SS_SUCCESS
}


func (sc *SlotContext) getSlotStatus() int32 {
	return atomic.LoadInt32(&sc.slotStatus)
}

func (sc SlotContext) lostTransSize() int {
	return len(sc.losingTrans)
}

func (sc *SlotContext) InitLostingTrans(ths []common.Hash) {
	for _, v := range ths {
		sc.losingTrans[v] = true
	}
	sc.setTransFull(len(sc.losingTrans) == 0)
	return
}

func (sc *SlotContext) SignedOnce() bool {
    return atomic.CompareAndSwapInt32(&sc.slotStatus, SS_WAITING, SS_SIGNED)
}

func (sc *SlotContext) BroadcastOnce() bool {
	return atomic.CompareAndSwapInt32(&sc.slotStatus, SS_VERIFIED, SS_SUCCESS)
}

//用接收到的新交易更新缺失的交易集
//返回接收前以及接收后是否不在缺失交易
func (sc *SlotContext) AcceptTrans(ths []common.Hash) (bool) {
	if len(sc.losingTrans) == 0 { //已经无缺失
		return false
	}
	accept := false
	for _, th := range ths {
		if _, ok := sc.losingTrans[th]; ok {
			accept = true
			break
		}
	}
	if accept {
		for _, th := range ths {
			delete(sc.losingTrans, th)
		}
	}
	sc.setTransFull(len(sc.losingTrans) == 0)
	return accept
}


//验证组签名
//pk：组公钥
//返回true验证通过，返回false失败。
func (sc *SlotContext) VerifyGroupSign(pk groupsig.Pubkey) bool {
	st := sc.getSlotStatus()
	if st == SS_VERIFIED { //已经验证过组签名
		return true
	}
	if st != SS_RECOVERD {
		return false
	}
	if sc.blockGSignGenerator.VerifyGroupSign(pk, sc.bh.Hash.Bytes()) &&
		sc.randGSignGenerator.VerifyGroupSign(pk, sc.preRands) {
		return atomic.CompareAndSwapInt32(&sc.slotStatus, SS_RECOVERD, SS_VERIFIED)
	}

	return false
}

//收到一个组内验证签名片段
//返回：=0, 验证请求被接受，阈值达到组签名数量。=1，验证请求被接受，阈值尚未达到组签名数量。=2，重复的验签。=3，数据异常。
func (sc *SlotContext) AcceptPiece(bh *types.BlockHeader, si *model.SignData) CAST_BLOCK_MESSAGE_RESULT {
	if si.DataHash != sc.bh.Hash {
		panic("SlotContext::AcceptPiece failed, hash diff.")
	}
	add, gen := sc.blockGSignGenerator.AddWitness(si.SignMember, si.DataSign)
	add1, gen1 := sc.randGSignGenerator.AddWitness(si.SignMember, *groupsig.DeserializeSign(bh.RandSig))

	if add && gen && add1 && gen1 {
		sc.setSlotStatus(SS_RECOVERD)
		sc.setGSign()
		return CBMR_THRESHOLD_SUCCESS
	}
	return CBMR_PIECE_NORMAL
}


func (sc *SlotContext) prepare(bh *types.BlockHeader, nonce *model.MinerNonce, rank int, preRand []byte, threshold int) {
	sc.reset(preRand, threshold)

	sc.bh = *bh
	sc.preRands = preRand
	sc.rank = rank
	sc.blockGSignGenerator = model.NewGroupSignGenerator(threshold)
	sc.randGSignGenerator = model.NewGroupSignGenerator(threshold)
	sc.proposerNonce = nonce
	sc.setSlotStatus(SS_WAITING)

	ltl, ccr, _, _ := core.BlockChainImpl.VerifyCastingBlock(*bh)
	sc.InitLostingTrans(ltl)
	if ccr == -1 {
		sc.setSlotStatus(SS_FAILED)
	}
}

func (sc *SlotContext) reset(preRand []byte, threshold int) {
	sc.bh = types.BlockHeader{}
	sc.preRands = preRand
	sc.rank = -1
	sc.blockGSignGenerator = nil
	sc.randGSignGenerator = nil
	sc.proposerNonce = nil
	sc.transFulled = new(bool)
	sc.losingTrans = make(map[common.Hash]bool)
	sc.setSlotStatus(SS_INVALID)
	return
}

func (sc *SlotContext) setGSign() {
    sc.bh.RandSig = sc.randGSignGenerator.GetGroupSign().Serialize()
    sc.bh.Signature = sc.blockGSignGenerator.GetGroupSign().Serialize()
}

func (sc *SlotContext) getBH() *types.BlockHeader {
    return &sc.bh
}

func (sc *SlotContext) TransBrief() string {
    return fmt.Sprintf("总交易数%v, 缺失数%v", len(sc.bh.Transactions), len(sc.losingTrans))
}