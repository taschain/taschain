package logical

import (
	"common"
	"consensus/groupsig"
	"fmt"
	"time"
)

//见证人身份
type WitnessStatus int

const (
	WS_KING     WitnessStatus = iota //铸块人
	WS_MINISTER                      //验证人
)

//共识状态
type ConsensusStatus int

const (
	CS_IDLE      ConsensusStatus = iota //非当前组
	CS_CURRENT                          //成为当前铸块组
	CS_CASTING                          //铸块中
	CS_CASTED                           //铸块完成
	CS_VERIFYING                        //验证中
	CS_VERIFIED                         //验证完成
	CS_BLOCKED                          //铸块完成（已通知到组外）
)

const CONSENSUS_VERSION = 1  //共识版本号
const SSSS_THRESHOLD = 51    //1-100
const GROUP_MAX_MEMBERS = 10 //一个组最大的成员数量
const MAX_UNKNOWN_BLOCKS = 5 //内存保存最大不能上链的未来块（中间块没有收到）
const MAX_SYNC_CASTORS = 3   //最多同时支持几个铸块验证
const INVALID_QN = -1        //无效的队列序号

//铸块上下文，和当前谁铸块（QueueNumber）有关
type cast_context struct {
	TimeRev     time.Time                          //接收到的时间
	QueueNumber int64                              //铸块者序号(<0无效)
	castor      groupsig.ID                        //铸块者ID
	MapWitness  map[groupsig.ID]groupsig.Signature //组签名列表
}

//收到一个（组内分片）验证通过请求
//返回：=0, 验证请求被接受，阈值达到组签名数量。=1，验证请求被接受，阈值尚未达到组签名数量。=2，重复的验签。=3，数据异常。
func (cc cast_context) Verified(uid groupsig.ID, si groupsig.Signature) uint8 {
	if len(cc.MapWitness) > GROUP_MAX_MEMBERS {
		panic("cast_context::Verified failed, too many members.")
	}
	v, ok := cc.MapWitness[uid]
	if ok { //已经收到过该成员的验签
		if v != si {
			panic("cast_context::Verified failed, one member's two sign diff.")
		}
		//忽略
		return 2
	} else {
		cc.MapWitness[uid] = si
		if len(cc.MapWitness) > (GROUP_MAX_MEMBERS * SSSS_THRESHOLD / 100) {
			return 0
		} else {
			return 1
		}
	}
	return 3
}

//判断当前节点当前时间是否为铸块人
func (cc cast_context) IsKing(self groupsig.ID) bool {
	return cc.castor == self
}

func BlankCastContext() cast_context {
	var cc cast_context
	cc.QueueNumber = INVALID_QN
	return cc
}

func CreateCastContext(uid groupsig.ID, qn int64) cast_context {
	var cc cast_context
	cc.TimeRev = time.Now()
	cc.QueueNumber = qn
	cc.castor = uid
	return cc
}

func (cc cast_context) IsValid() bool {
	return cc.QueueNumber > INVALID_QN
}

//取得铸块权重
//第一顺为权重1，第二顺位权重2，第三顺位权重4...，即权重越低越好（但0为无效）
func (cc cast_context) GetWeight() uint64 {
	const MAX_QN int64 = 63
	if cc.QueueNumber <= MAX_QN {
		return uint64(cc.QueueNumber) << 1
	} else {
		return 0
	}
}

///////////////////////////////////////////////////////////////////////////////
//共识上下文，和块高度有关
type block_context struct {
	Version       uint
	PreTime       time.Time                      //所属组的当前铸块起始时间戳(组内必须一致，不然时间片会异常，所以直接取上个铸块完成时间)
	CCTimer       time.Timer                     //共识定时器
	VerifiedMaxQN int64                          //已验证过的最大QN
	CS            ConsensusStatus                //铸块状态
	PrevHash      common.Hash                    //上一块哈希值
	CastHeight    uint64                         //待铸块高度
	GroupMembers  uint                           //组成员数量
	Threshold     uint                           //百分比（0-100）
	Castors       [MAX_SYNC_CASTORS]cast_context //并行铸块人
}

//是否接受一个铸块人的铸块
//pos：铸块人在组内的排位
//=0, 接受; >0, 不接受。
func (bc block_context) AccpetCastor(cc cast_context) uint8 {
	count := getCastTimeWindow(bc.PreTime)
	if count >= 0 {
		if int32(cc.QueueNumber) >= count { //铸块窗口合法
			for i, v := range bc.Castors {
				if !v.IsValid() || bc.VerifiedMaxQN > cc.QueueNumber { //发现一个空槽，或者存在一个QN更大的铸块
					bc.Castors[i] = cc //替换
					if bc.VerifiedMaxQN == -1 || bc.VerifiedMaxQN > cc.QueueNumber {
						bc.VerifiedMaxQN = cc.QueueNumber
					}
					return 0
				}
			}
			//遍历结束
			return 3 //当前存在的铸块人QN都比新的铸块QN小
		} else {
			return 2
		}
	} else {
		return 1
	}
}

//判断当前节点当前是否在铸块中（所属组是待铸块高度的对应组）
func (bc block_context) IsCasting() bool {
	if bc.CS == CS_IDLE || bc.CS == CS_BLOCKED {
		return false
	} else {
		return true
	}
}

//以bh为最大成功完成高度，tc为铸块完成时间，初始化上下文
func (bc block_context) Reset(bh uint64, tc time.Time, h common.Hash) {
	bc.Version = CONSENSUS_VERSION
	bc.PreTime = tc //上一块的铸块成功时间
	bc.CCTimer.Stop()
	//bc.WS = WS_MINISTER
	bc.CS = CS_IDLE
	bc.VerifiedMaxQN = INVALID_QN
	bc.PrevHash = h
	bc.CastHeight = bh + 1
	bc.GroupMembers = GROUP_MAX_MEMBERS
	bc.Threshold = SSSS_THRESHOLD
	for i, _ := range bc.Castors {
		bc.Castors[i] = BlankCastContext()
	}
}

//节点所在组成为当前铸块组
//bn: 完成块高度
//tc: 完成块出块时间
//h:  完成块哈希
//该函数会被多次重入，需要做容错处理。
func (bc *block_context) Begin_Cast(bh uint64, tc time.Time, h common.Hash) bool {
	var max_height uint64 = 0
	//to do : 鸠兹从链上取得最高有效块
	if (bh <= max_height) || (bh > max_height+MAX_UNKNOWN_BLOCKS) {
		//不在合法的铸块高度内
		return false
	}
	if bc.IsCasting() { //已经在铸块中
		if bc.CastHeight == (bh + 1) { //已经在铸消息通知的块
			if bc.PreTime != tc || bc.PrevHash != h {
				panic("block_context:Begin_Cast failed, arg error.\n")
			} else {
				//忽略该消息
				fmt.Printf("block_context:Begin_Cast ignored, already in casting.\n")
			}
		}
	} else { //没有在铸块中
		bc.Reset(bh, tc, h)
		bc.CS = CS_CURRENT
	}
	return true
}

//个人铸块完成（非组出块完成）, 同时也代表个人验证完成（个人出块即验证）
//铸块者更新自己的上下文（注：可能同时他也是更低QN铸块人的验证人）
//uid : 铸块人ID
//qn : 铸块的qn值
func (bc block_context) UserCasted(uid groupsig.ID, qn int64) {
	if bc.CS == CS_IDLE {
		panic("UserCasted failed, CS value illegal.")
	} else {
		fmt.Printf("UserCasted calling, CS value = %v.\n", bc.CS)
	}
	cc := CreateCastContext(uid, qn)
	if bc.AccpetCastor(cc) == 0 { //铸块被接受
		bc.CS = CS_CASTED //
	}
	return
}

//个人验证完成
//uid ： 验证人ID
//qn : 验证块的qn值
//si : 组签名值
func (bc block_context) UserVerified(uid groupsig.ID, qn int64, si groupsig.Signature) {

}
