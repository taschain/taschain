package logical

import (
	"common"
	"consensus/groupsig"
	"fmt"
	"math/big"
	"time"
)

//铸块人身份
type WITNESS_STATUS int

const (
	WS_KING     WITNESS_STATUS = iota //出块人
	WS_MINISTER                       //验证人
)

//组铸块共识状态（针对某个高度而言）
type CAST_BLOCK_CONSENSUS_STATUS int

const (
	CBCS_IDLE           CAST_BLOCK_CONSENSUS_STATUS = iota //非当前组
	CBCS_CURRENT                                           //成为当前铸块组
	CBCS_CASTING                                           //至少收到一块组内共识数据
	CBCS_BLOCKED                                           //组内已有铸块完成（已通知到组外）
	CBCS_MIN_QN_BLOCKED                                    //组内最小铸块完成（已通知到组外），该高度铸块结束
	CBCS_TIMEOUT                                           //组铸块超时
)

//铸块槽状态
type SLOT_STATUS int

const (
	SS_WAITING  SLOT_STATUS = iota //等待签名片段达到阈值
	SS_RECOVERD                    //恢复出组签名
	SS_VERIFIED                    //组签名用组公钥验证通过
	SS_FAILED                      //铸块过程中失败，不可逆
)

//铸块槽结构
type SlotContext struct {
	TimeRev     time.Time                          //插槽被创建的时间（也就是接收到该插槽第一包数据的时间）
	HeaderHash  common.Hash                        //出块头哈希
	QueueNumber int64                              //铸块槽序号(<0无效)，等同于出块人序号。
	King        groupsig.ID                        //出块者ID
	MapWitness  map[groupsig.ID]groupsig.Signature //该铸块槽的见证人验证签名列表
	GroupSign   groupsig.Signature                 //成功输出的组签名
	SlotStatus  SLOT_STATUS
}

//是否已收到出块人的消息
func (sc SlotContext) HasKingMessage() bool {
	if sc.King.IsValid() {
		_, ok := sc.MapWitness[sc.King]
		return ok
	}
	return false
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
	b := groupsig.VerifySig(pk, sc.HeaderHash.Bytes(), sc.GroupSign)
	if b {
		sc.SlotStatus = SS_VERIFIED //组签名验证通过
	}
	return b
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
	if len(sc.MapWitness) >= GROUP_MIN_WITNESSES && sc.HasKingMessage() { //达到组签名恢复阈值，且当前节点收到了出块人消息
		gs := groupsig.RecoverSignatureByMapI(sc.MapWitness, GROUP_MIN_WITNESSES)
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

type CAST_BLOCK_MESSAGE_RESULT int8 //出块和验证消息处理结果枚举

const (
	CBMR_PIECE                CAST_BLOCK_MESSAGE_RESULT = iota //收到一个分片，接收正常
	CBMR_THRESHOLD_SUCCESS                                     //收到一个分片且达到阈值，组签名成功
	CBMR_THRESHOLD_FAILED                                      //收到一个分片且达到阈值，组签名失败
	CBMR_IGNORE_REPEAT                                         //丢弃：重复收到该消息
	CMBR_IGNORE_QN_BIG_QN                                      //丢弃：QN太大
	CMBR_IGNORE_QN_FUTURE                                      //丢弃：未轮到该QN
	CMBR_IGNORE_CASTED                                         //丢弃：该高度出块已完成
	CMBR_IGNORE_TIMEOUT                                        //丢弃：该高度出块时间已过
	CMBR_IGNORE_MIN_QN_SIGNED                                  //丢弃：该节点已向组外广播出更低QN值的块
	CMBR_IGNORE_NOT_CASTING                                    //丢弃：未启动当前组铸块共识
	CBMR_ERROR_ARG                                             //异常：参数异常
	CBMR_ERROR_SIGN                                            //异常：签名验证异常
	CMBR_ERROR_UNKNOWN                                         //异常：未知异常
)

//收到一个组内验证签名片段
//返回：=0, 验证请求被接受，阈值达到组签名数量。=1，验证请求被接受，阈值尚未达到组签名数量。=2，重复的验签。=3，数据异常。
func (sc *SlotContext) AcceptPiece(cs ConsensusBlockSummary, si SignData) CAST_BLOCK_MESSAGE_RESULT {
	if len(sc.MapWitness) > GROUP_MAX_MEMBERS {
		panic("CastContext::Verified failed, too many members.")
	}
	v, ok := sc.MapWitness[si.GetID()]
	if ok { //已经收到过该成员的验签
		if v != si.DataSign {
			panic("CastContext::Verified failed, one member's two sign diff.")
		}
		//忽略
		return CBMR_IGNORE_REPEAT
	} else { //没有收到过该用户的签名
		sc.MapWitness[si.GetID()] = si.DataSign
		if len(sc.MapWitness) > (GROUP_MAX_MEMBERS*SSSS_THRESHOLD/100) && sc.HasKingMessage() { //达到组签名条件
			if sc.GenGroupSign() {
				return CBMR_THRESHOLD_SUCCESS
			} else {
				return CBMR_THRESHOLD_FAILED
			}
		} else {
			return CBMR_PIECE
		}
	}
	return CMBR_ERROR_UNKNOWN
}

//判断某个成员是否为插槽的出块人
func (sc SlotContext) IsKing(member groupsig.ID) bool {
	return sc.King == member
}

//根据（某个QN值）接收到的第一包数据生成一个新的插槽
func newSlotContext(cs ConsensusBlockSummary, si SignData) SlotContext {
	var sc SlotContext
	sc.TimeRev = time.Now()
	sc.HeaderHash = cs.DataHash
	sc.QueueNumber = cs.QueueNumber
	sc.King = cs.Castor
	sc.MapWitness = make(map[groupsig.ID]groupsig.Signature)
	sc.MapWitness[si.GetID()] = si.DataSign
	return sc
}

func (sc *SlotContext) Reset() {
	sc.QueueNumber = INVALID_QN
	sc.King = *groupsig.NewIDFromInt(0)
	sc.MapWitness = make(map[groupsig.ID]groupsig.Signature)
	sc.HeaderHash = *new(common.Hash)
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
///////////////////////////////////////////////////////////////////////////////
//组铸块共识上下文结构（一个高度有一个上下文）
type BlockContext struct {
	Version         uint
	PreTime         time.Time                     //所属组的当前铸块起始时间戳(组内必须一致，不然时间片会异常，所以直接取上个铸块完成时间)
	CCTimer         time.Ticker                   //共识定时器
	SignedMinQN     int64                         //组内已铸出的最小QN值的块
	ConsensusStatus CAST_BLOCK_CONSENSUS_STATUS   //铸块状态
	PrevHash        common.Hash                   //上一块哈希值
	CastHeight      uint64                        //待铸块高度
	GroupMembers    uint                          //组成员数量
	Threshold       uint                          //百分比（0-100）
	Slots           [MAX_SYNC_CASTORS]SlotContext //铸块槽列表

	Proc    *Processer  //处理器
	GroupID groupsig.ID //所属组
}

//检查是否要处理某个铸块槽
//返回true需要处理，返回false可以丢弃。
func (bc *BlockContext) NeedHandleQN(qn uint) bool {
	if bc.SignedMinQN == INVALID_QN { //当前该组还没有铸出过块
		return true
	} else { //当前该组已经有成功的铸块（来自某个铸块槽）
		return qn < uint(bc.SignedMinQN)
	}
}

//完成（上链，向组外广播）某个铸块槽后更新当前高度的最小QN值
func (bc *BlockContext) SignedUpdateMinQN(qn uint) bool {
	b := bc.NeedHandleQN(qn)
	if b {
		bc.SignedMinQN = int64(qn)
	}
	return b
}

//完成某个铸块槽的铸块（上链，组外广播）后，更新组的当前高度铸块状态
func (bc *BlockContext) CastedUpdateStatus(qn uint) bool {
	if bc.ConsensusStatus == CBCS_CASTING || bc.ConsensusStatus == CBCS_BLOCKED || bc.ConsensusStatus == CBCS_MIN_QN_BLOCKED {
		if bc.ConsensusStatus == CBCS_MIN_QN_BLOCKED { //已经铸出过QN=1的块
			return false //该高度不用再铸块了
		} else {
			if bc.ConsensusStatus == CBCS_CASTING {
				if qn == 1 {
					bc.ConsensusStatus = CBCS_MIN_QN_BLOCKED
				} else {
					bc.ConsensusStatus = CBCS_BLOCKED
				}
				return true
			} else {
				//已经铸出过块
				if qn == 1 { //收到最小QN块消息
					bc.ConsensusStatus = CBCS_MIN_QN_BLOCKED
				}
				return true
			}
		}
	} else { //不在铸块周期内
		return false
	}
}

//检查是否有空槽可以接纳一个铸块槽
//如果还有空槽，返回空槽序号。如果没有空槽，返回-1.
func (bc *BlockContext) findEmptySlot() int32 {
	for i, v := range bc.Slots {
		if v.QueueNumber == INVALID_QN {
			return int32(i)
		}
	}
	return -1
}

//检查目前在处理中的QN值最高的铸块槽。
//返回QN值最高的铸块槽的序号和QN值。如果当前全部是空槽，序号和QN值都返回-1.
func (bc *BlockContext) findMaxQNSlot() (int32, int64) {
	var index int32 = -1
	var max_qn int64 = -1
	for i, v := range bc.Slots {
		if v.QueueNumber > max_qn {
			max_qn = v.QueueNumber
			index = int32(i)
		}
	}
	return index, max_qn
}

//检查是否有指定QN值的铸块槽
//返回：int32:铸块槽序号（没找到返回-1），bool：该铸块槽是否收到出块人消息（在铸块槽序号>=0时有意义）
func (bc *BlockContext) findCastSlot(qn int64) (int32, bool) {
	for i, v := range bc.Slots {
		if v.QueueNumber == qn {
			return int32(i), v.HasKingMessage()
		}
	}
	return -1, false
}

type QN_QUERY_SLOT_RESULT int //根据QN查找插槽结果枚举

const (
	QQSR_EMPTY_SLOT                     QN_QUERY_SLOT_RESULT = iota //找到一个空槽
	QQSR_REPLACE_SLOT                                               //找到一个能替换（QN值更低）的槽
	QQSR_EXIST_SLOT_WITH_KINGMESSAGE                                //该QN对应的插槽已存在，且已收到铸块人消息
	QQSR_EXIST_SLOT_WITHOUT_KINGMESSAGE                             //该QN对应的插槽已存在，但尚未收到铸块人消息
	QQSR_NO_UNKNOWN                                                 //未知结果
)

//根据QN优先级规则，尝试找到有效的插槽
func (bc *BlockContext) ConsensusFindSlot(qn int64) (int32, QN_QUERY_SLOT_RESULT) {
	var info QN_QUERY_SLOT_RESULT = QQSR_NO_UNKNOWN
	var max_qn int64 = -1
	i, km := bc.findCastSlot(qn)
	if i >= 0 { //该qn的槽已存在
		if km {
			info = QQSR_EXIST_SLOT_WITH_KINGMESSAGE
		} else {
			info = QQSR_EXIST_SLOT_WITHOUT_KINGMESSAGE
		}
		return i, info
	} else {
		i = bc.findEmptySlot()
		if i >= 0 { //找到空槽
			return i, QQSR_EMPTY_SLOT
		} else {
			i, max_qn = bc.findMaxQNSlot() //取得最大槽
			if qn < max_qn {               //最大槽的QN被新的QN大
				return i, QQSR_REPLACE_SLOT
			}
		}
	}
	return -1, QQSR_NO_UNKNOWN
}

//铸块共识消息处理函数
//cv：铸块共识数据，出块消息或验块消息生成的ConsensusBlockSummary.
//=0, 接受; =1,接受，达到阈值；<0, 不接受。
func (bc *BlockContext) accpetCV(cs ConsensusBlockSummary, si SignData) CAST_BLOCK_MESSAGE_RESULT {
	count := getCastTimeWindow(bc.PreTime)
	if count < 0 || cs.QueueNumber <= 0 { //时间窗口异常
		return CBMR_ERROR_ARG
	}
	if int32(cs.QueueNumber) >= count { //未轮到该QN出块
		return CMBR_IGNORE_QN_FUTURE
	}

	if !bc.NeedHandleQN(uint(cs.QueueNumber)) { //该组已经铸出过QN值更低的块
		return CMBR_IGNORE_MIN_QN_SIGNED
	}

	i, info := bc.ConsensusFindSlot(cs.QueueNumber)
	if i < 0 { //没有找到有效的插槽
		return CMBR_IGNORE_QN_BIG_QN
	}
	//找到有效的插槽
	if info == QQSR_EMPTY_SLOT || info == QQSR_REPLACE_SLOT {
		bc.Slots[i] = newSlotContext(cs, si)
	} else {
		result := bc.Slots[i].AcceptPiece(cs, si)
		return result
	}
	return CMBR_ERROR_UNKNOWN
}

//判断当前节点所在组当前是否在铸块共识中
func (bc BlockContext) IsCasting() bool {
	if bc.ConsensusStatus == CBCS_IDLE || bc.ConsensusStatus == CBCS_MIN_QN_BLOCKED || bc.ConsensusStatus == CBCS_TIMEOUT {
		//空闲，已出权重最高的块，超时
		return false
	} else {
		return true
	}
}

//铸块上下文复位，在某个高度轮到当前组铸块时调用。
//to do : 还是索性重新生成。
func (bc *BlockContext) Reset() {
	fmt.Printf("BlockContext::Reset...\n")
	bc.Version = CONSENSUS_VERSION
	bc.PreTime = *new(time.Time)
	bc.CCTimer.Stop() //关闭定时器
	bc.ConsensusStatus = CBCS_IDLE
	bc.SignedMinQN = INVALID_QN
	bc.PrevHash = *new(common.Hash)
	bc.CastHeight = 0
	bc.GroupMembers = GROUP_MAX_MEMBERS
	bc.Threshold = SSSS_THRESHOLD
	bc.Slots = *new([MAX_SYNC_CASTORS]SlotContext)
}

//组铸块共识初始化
//bh : 上一块完成高度，tc：上一块完成时间；h：上一块哈希值
func (bc *BlockContext) beginConsensus(bh uint64, tc time.Time, h common.Hash) {
	fmt.Printf("BlockContext::BeginConsensus...\n")
	bc.PreTime = tc //上一块的铸块成功时间
	bc.ConsensusStatus = CBCS_CURRENT
	bc.SignedMinQN = INVALID_QN //等待第一个出块者
	bc.PrevHash = h
	bc.CastHeight = bh + 1
	bc.Slots = *new([MAX_SYNC_CASTORS]SlotContext)
	bc.StartTimer() //启动定时器
	return
}

//节点所在组成为当前铸块组
//bn: 完成块高度
//tc: 完成块出块时间
//h:  完成块哈希
//该函数会被多次重入，需要做容错处理。
//在某个高度第一次进入时会启动定时器
func (bc *BlockContext) BeingCastGroup(bh uint64, tc time.Time, h common.Hash) bool {
	var max_height uint64 = 0
	//to do : 鸠兹从链上取得最高有效块
	if (bh <= max_height) || (bh > max_height+MAX_UNKNOWN_BLOCKS) {
		//不在合法的铸块高度内
		return false
	}
	if bc.IsCasting() { //已经在铸块共识中
		if bc.CastHeight == (bh + 1) { //已经在铸消息通知的块
			if bc.PreTime != tc || bc.PrevHash != h {
				panic("block_context:Begin_Cast failed, arg error.\n")
			} else {
				//忽略该消息
				fmt.Printf("block_context:Begin_Cast ignored, already in casting.\n")
			}
		}
	} else { //没有在铸块中
		bc.beginConsensus(bh, tc, h)
	}
	return true
}

//收到某个铸块人的铸块完成消息（个人铸块完成消息也是个人验证完成消息）
func (bc *BlockContext) UserCasted(cs ConsensusBlockSummary, sd SignData) CAST_BLOCK_MESSAGE_RESULT {
	if !bc.IsCasting() {
		return CMBR_IGNORE_NOT_CASTING
	}
	result := bc.accpetCV(cs, sd)
	return result
}

//收到某个验证人的验证完成消息（可能会比铸块完成消息先收到）
func (bc *BlockContext) UserVerified(cs ConsensusBlockSummary, sd SignData) CAST_BLOCK_MESSAGE_RESULT {
	if !bc.IsCasting() { //没有在组铸块共识窗口
		return CMBR_IGNORE_NOT_CASTING
	}
	result := bc.accpetCV(cs, sd) //>=0为消息正确接收
	return result
}

func (bc BlockContext) VerifyGroupSign(cs ConsensusBlockSummary, pk groupsig.Pubkey) bool {
	//找到cs对应的槽
	i, king := bc.findCastSlot(cs.QueueNumber)
	if i >= 0 && king {
		b := bc.Slots[i].VerifyGroupSign(pk)
		return b
	}
	return false
}

//计算当前铸块人位置和QN
func (bc *BlockContext) CalcCastor() (int32, int64) {
	var index int32 = -1
	var qn int64 = -1
	d := time.Since(bc.PreTime)
	var secs uint64 = uint64(d.Seconds())
	if secs < uint64(MAX_GROUP_BLOCK_TIME) { //在组铸块共识时间窗口内
		qn = int64(secs / uint64(MAX_USER_CAST_TIME))
		first_i := bc.getFirstCastor() //取得第一个铸块人位置
		if first_i >= 0 && bc.GroupMembers > 0 {
			index = (int32(qn) + first_i) % int32(bc.GroupMembers)
		} else {
			qn = -1
		}
	}
	return index, qn
}

//取得第一个铸块人在组内的位置
func (bc *BlockContext) getFirstCastor() int32 {
	var index int32 = -1
	bi_hash := bc.PrevHash.Big()
	if bi_hash.BitLen() > 0 && bc.GroupMembers > 0 {
		index = int32(bi_hash.Mod(bi_hash, big.NewInt(int64(bc.GroupMembers))).Int64())
	}
	return index
}

func (bc *BlockContext) StartTimer() {
	bc.CCTimer.Stop()
	bc.CCTimer = *time.NewTicker(TIMER_INTEVAL_SECONDS)
	var count int
	for _ = range bc.CCTimer.C {
		count++
		fmt.Printf("block_context::StartTicker, count=%v.\n", count)
		go bc.TickerRoutine()
	}
	return
	//<-bc.CCTimer.C
}

//定时器例行处理
func (bc *BlockContext) TickerRoutine() {
	if !bc.IsCasting() { //没有在组铸块共识中
		bc.Reset() //提前出块完成
		return
	}
	d := time.Since(bc.PreTime)                    //上个铸块完成到现在的时间
	if int32(d.Seconds()) > MAX_GROUP_BLOCK_TIME { //超过了组最大铸块时间
		fmt.Printf("block_context::TickerRoutine, out of max group cast time, time=%v secs, status=%v.\n", d.Seconds(), bc.ConsensusStatus)
		bc.Reset()
	} else {
		//当前组仍在有效铸块共识时间内
		//检查自己是否成为铸块人
		if bc.Proc != nil {
			index, qn := bc.CalcCastor() //当前铸块人（KING）和QN值
			bc.Proc.CheckCastRoutine(bc.GroupID, index, qn, uint(bc.CastHeight))
		}
	}
	return
}
