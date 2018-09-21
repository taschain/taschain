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
	"consensus/groupsig"
	"common"
	"sync/atomic"
	"middleware/types"
	"core"
	"consensus/model"
	"log"
	"gopkg.in/fatih/set.v0"
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
	SS_REWARD_REQ //分红交易签名请求已发
	SS_REWARD_SEND //分红交易已广播
)

//铸块槽结构，和某个KING的共识数据一一对应
type SlotContext struct {
	//验证相关
	BH             types.BlockHeader //出块头详细数据
	//QueueNumber    int64             //铸块槽序号(<0无效)，等同于出块人序号。
	vrfValue 		uint64
	gSignGenerator *model.GroupSignGenerator	//块签名产生器
	rSignGenerator *model.GroupSignGenerator	//随机数签名产生器
	slotStatus     int32
	lostTxHash     set.Interface

	//奖励相关
	rewardTrans 		*types.Transaction
	rewardGSignGen	*model.GroupSignGenerator	//奖励交易签名产生器
}

func createSlotContext(threshold int) *SlotContext {
	return &SlotContext{
		//QueueNumber:    model.INVALID_QN,
		slotStatus:     SS_INVALID,
		gSignGenerator: model.NewGroupSignGenerator(threshold),
		rSignGenerator: model.NewGroupSignGenerator(threshold),
		lostTxHash:     set.New(set.ThreadSafe),
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
	for _, tx := range ths {
		sc.lostTxHash.Remove(tx)
	}
	return l > sc.lostTransSize()
}

func (sc SlotContext) MessageSize() int {
	return sc.gSignGenerator.WitnessSize()
}

//验证组签名
//pk：组公钥
//返回true验证通过，返回false失败。
func (sc *SlotContext) VerifyGroupSigns(pk groupsig.Pubkey, preRandom []byte) bool {
	st := sc.GetSlotStatus()
	if st == SS_SUCCESS || st == SS_VERIFIED { //已经验证过组签名
		return true
	}
	if st != SS_RECOVERD {
		return false
	}
	b := sc.gSignGenerator.VerifyGroupSign(pk, sc.BH.Hash.Bytes())
	if b {
		b = sc.rSignGenerator.VerifyGroupSign(pk, preRandom)
		if b {
			sc.setSlotStatus(SS_VERIFIED) //组签名验证通过
		}
	}
	if !b {
		sc.setSlotStatus(SS_FAILED)
	}
	return b
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


//收到一个组内验证签名片段
//返回：=0, 验证请求被接受，阈值达到组签名数量。=1，验证请求被接受，阈值尚未达到组签名数量。=2，重复的验签。=3，数据异常。
func (sc *SlotContext) AcceptVerifyPiece(bh *types.BlockHeader, si *model.SignData) CAST_BLOCK_MESSAGE_RESULT {
	if si.DataHash != sc.BH.Hash {
		return CBMR_BH_HASH_DIFF
	}
	add, generate := sc.gSignGenerator.AddWitness(si.SignMember, si.DataSign)

	if !add { //已经收到过该成员的验签
		//忽略
		return CBMR_IGNORE_REPEAT
	} else { //没有收到过该用户的签名
		rsign := groupsig.DeserializeSign(bh.Random)
		if rsign == nil {
			panic("SlotContext:randSign deserialize nil")
		}
		radd, rgen := sc.rSignGenerator.AddWitness(si.SignMember, *rsign)

		if radd && generate && rgen { //达到组签名条件
			sc.setSlotStatus(SS_RECOVERD)
			sc.BH.Signature = sc.gSignGenerator.GetGroupSign().Serialize()
			sc.BH.Random = sc.rSignGenerator.GetGroupSign().Serialize()
			return CBMR_THRESHOLD_SUCCESS
		} else {
			return CBMR_PIECE_NORMAL
		}
	}
	return CBMR_ERROR_UNKNOWN
}

//根据（某个QN值）接收到的第一包数据生成一个新的插槽
func (sc *SlotContext) init(bh *types.BlockHeader) bool {
	if sc.StatusTransform(SS_INVALID, SS_WAITING) {
		sc.BH = *bh
		sc.vrfValue = bh.Nonce
		log.Printf("start verifyblock, height=%v, qn=%v", bh.Height, bh.QueueNumber)
		ltl, ccr, _, _ := core.BlockChainImpl.VerifyCastingBlock(*bh)
		log.Printf("initSlotContext verifyCastingBlock height=%v, qn=%v, lost trans size %v, ret %v\n",  bh.Height, bh.QueueNumber, len(ltl), ccr)
		sc.addLostTrans(ltl)
		if ccr == -1 {
			sc.setSlotStatus(SS_FAILED)
		}
		return true
	}
	return false
}

func (sc *SlotContext) IsValid() bool {
	return sc.GetSlotStatus() != SS_INVALID
}

func (sc *SlotContext) StatusTransform(from int32, to int32) bool {
	return atomic.CompareAndSwapInt32(&sc.slotStatus, from, to)
}

func (sc *SlotContext) TransBrief() string {
    return fmt.Sprintf("总交易数%v，缺失%v", len(sc.BH.Transactions), sc.lostTransSize())
}

func (sc *SlotContext) SetRewardTrans(tx *types.Transaction) bool {
	if sc.StatusTransform(SS_SUCCESS, SS_REWARD_REQ) {
		sc.rewardTrans = tx
		return true
	}
	return false
}

func (sc *SlotContext) AcceptRewardPiece(sd *model.SignData) (accept, recover bool) {
	if sc.rewardTrans != nil && sc.rewardTrans.Hash != sd.DataHash {
		return
	}
	accept, recover = sc.rewardGSignGen.AddWitness(sd.GetID(), sd.DataSign)
	if accept && recover {
		//交易设置签名
		//sc.rewardTrans.
	}
	return
}