//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package logical

import (
	"time"
	"consensus/groupsig"
	"common"
	"sync/atomic"
	"middleware/types"
	"strconv"
	"core"
	"consensus/model"
	"log"
	"gopkg.in/fatih/set.v0"
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

//铸块槽结构，和某个KING的共识数据一一对应
type SlotContext struct {
	TimeRev time.Time //插槽被创建的时间（也就是接收到该插槽第一包数据的时间）
	//HeaderHash   common.Hash                   //出块头哈希(就这个哈希值达成一致)
	BH             types.BlockHeader             //出块头详细数据
	QueueNumber    int64                         //铸块槽序号(<0无效)，等同于出块人序号。
	King           groupsig.ID                   //出块者ID
	gSignGenerator *model.GroupSignGenerator
	slotStatus     int32
	lostTxHash     set.Interface
}

func createSlotContext(threshold int) *SlotContext {
    return &SlotContext{
    	TimeRev: time.Now(),
    	QueueNumber: model.INVALID_QN,
    	slotStatus: SS_INVALID,
    	gSignGenerator: model.NewGroupSignGenerator(threshold),
    	lostTxHash: set.New(set.ThreadSafe),
	}
}

func (sc *SlotContext) HasTransLost() bool {
	return sc.lostTxHash.Size() > 0
}

func (sc *SlotContext) setSlotStatus(st int32) {
	atomic.StoreInt32(&sc.slotStatus, st)
}

func (sc *SlotContext) IsFailed() bool {
	st := sc.GetSlotStatus()
	return st == SS_FAILED
}

func (sc *SlotContext) GetSlotStatus() int32 {
	return atomic.LoadInt32(&sc.slotStatus)
}

func (sc SlotContext) lostTransSize() int {
	return sc.lostTxHash.Size()
}

func (sc *SlotContext) addLostTrans(txs []common.Hash) {
	if len(txs) == 0 {
		return
	}
	for _, tx := range txs {
		sc.lostTxHash.Add(tx)
	}
}

//用接收到的新交易更新缺失的交易集
//返回接收前以及接收后是否不在缺失交易
func (sc *SlotContext) AcceptTrans(ths []common.Hash) (bool) {
	l := sc.lostTransSize()
	if l == 0 { //已经无缺失
		return false
	}
	sc.lostTxHash.Remove(ths)
	return l > sc.lostTransSize()
}

func (sc SlotContext) MessageSize() int {
	return sc.gSignGenerator.WitnessSize()
}

//验证组签名
//pk：组公钥
//返回true验证通过，返回false失败。
func (sc *SlotContext) VerifyGroupSign(pk groupsig.Pubkey) bool {
	st := sc.GetSlotStatus()
	if st == SS_SUCCESS || st == SS_VERIFIED { //已经验证过组签名
		return true
	}
	if st != SS_RECOVERD {
		return false
	}
	b := sc.gSignGenerator.VerifyGroupSign(pk, sc.BH.Hash.Bytes())
	if b {
		sc.setSlotStatus(SS_VERIFIED) //组签名验证通过
	} else {
		sc.setSlotStatus(SS_FAILED)
	}
	return b
}

func (sc SlotContext) GetGroupSign() groupsig.Signature {
	return sc.gSignGenerator.GetGroupSign()
}

func (sc *SlotContext) IsVerified() bool {
	return sc.GetSlotStatus() == SS_VERIFIED
}

func (sc *SlotContext) IsRecovered() bool {
	return sc.GetSlotStatus() == SS_RECOVERD
}

func (sc *SlotContext) IsSuccess() bool {
	return sc.GetSlotStatus() == SS_SUCCESS
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

//收到一个组内验证签名片段
//返回：=0, 验证请求被接受，阈值达到组签名数量。=1，验证请求被接受，阈值尚未达到组签名数量。=2，重复的验签。=3，数据异常。
func (sc *SlotContext) AcceptPiece(bh types.BlockHeader, si model.SignData) CAST_BLOCK_MESSAGE_RESULT {
	if si.DataHash != sc.BH.Hash {
		panic("SlotContext::AcceptPiece failed, hash diff.")
	}
	add, generate := sc.gSignGenerator.AddWitness(si.SignMember, si.DataSign)
	log.Printf("AcceptPiece %v %v\n", add, generate)
	if !add { //已经收到过该成员的验签
		//忽略
		return CBMR_IGNORE_REPEAT
	} else { //没有收到过该用户的签名
		if generate { //达到组签名条件; (不一定需要收到king的消息 ? : by wenqin 2018/5/21)
			sc.setSlotStatus(SS_RECOVERD)
			sc.BH.Signature = sc.gSignGenerator.GetGroupSign().Serialize()
			return CBMR_THRESHOLD_SUCCESS
		} else {
			return CBMR_PIECE_NORMAL
		}
	}
	return CBMR_ERROR_UNKNOWN
}


//根据（某个QN值）接收到的第一包数据生成一个新的插槽
func initSlotContext(bh *types.BlockHeader, threshold int) *SlotContext {

	sc := createSlotContext(threshold)

	sc.BH = *bh
	sc.QueueNumber = int64(bh.QueueNumber)
	sc.King.Deserialize(bh.Castor)
	sc.setSlotStatus(SS_WAITING)
	ltl, ccr, _, _ := core.BlockChainImpl.VerifyCastingBlock(*bh)
	log.Printf("initSlotContext verifyCastingBlock lost trans size %v, ret %v\n", len(ltl), ccr)
	sc.addLostTrans(ltl)
	if ccr == -1 {
		sc.setSlotStatus(SS_FAILED)
	}

	return sc
}

func (sc SlotContext) IsValid() bool {
	return sc.QueueNumber > model.INVALID_QN
}

func (sc *SlotContext) StatusTransform(from int32, to int32) bool {
    return atomic.CompareAndSwapInt32(&sc.slotStatus, from, to)
}