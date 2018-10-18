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
	"common"
	"consensus/groupsig"
	"consensus/model"
	"core"
	"fmt"
	"gopkg.in/fatih/set.v0"
	"log"
	"math/big"
	"middleware/types"
	"sync/atomic"
)

/*
**  Creator: pxf
**  Date: 2018/5/21 下午5:49
**  Description:
 */

const (
	SS_INITING     int32 = iota
	SS_WAITING           //等待签名片段达到阈值
	SS_SIGNED            //自己是否签名过
	SS_RECOVERD          //恢复出组签名
	SS_VERIFIED          //组签名用组公钥验证通过
	SS_SUCCESS           //已上链广播
	SS_FAILED            //铸块过程中失败，不可逆
	SS_REWARD_REQ        //分红交易签名请求已发
	SS_REWARD_SEND       //分红交易已广播
)

//铸块槽结构，和某个KING的共识数据一一对应
type SlotContext struct {
	//验证相关
	BH types.BlockHeader //出块头详细数据
	//QueueNumber    int64             //铸块槽序号(<0无效)，等同于出块人序号。
	vrfValue       *big.Int
	gSignGenerator *model.GroupSignGenerator //块签名产生器
	rSignGenerator *model.GroupSignGenerator //随机数签名产生器
	slotStatus     int32
	lostTxHash     set.Interface

	castor groupsig.ID

	//奖励相关
	rewardTrans    *types.Transaction
	rewardGSignGen *model.GroupSignGenerator //奖励交易签名产生器
}

func createSlotContext(bh *types.BlockHeader, threshold int) *SlotContext {
	return &SlotContext{
		BH:             *bh,
		vrfValue:       bh.ProveValue,
		castor:         groupsig.DeserializeId(bh.Castor),
		slotStatus:     SS_INITING,
		gSignGenerator: model.NewGroupSignGenerator(threshold),
		rSignGenerator: model.NewGroupSignGenerator(threshold),
		rewardGSignGen: model.NewGroupSignGenerator(threshold),
		lostTxHash:     set.New(set.ThreadSafe),
	}
}

func (sc *SlotContext) init(bh *types.BlockHeader) bool {
	log.Printf("start verifyblock, height=%v, hash=%v", bh.Height, bh.Hash.ShortS())

	lostTxs, ccr, _, _ := core.BlockChainImpl.VerifyBlock(*bh)
	lostTxsStrings := make([]string, len(lostTxs))
	for idx, tx := range lostTxs {
		lostTxsStrings[idx] = tx.ShortS()
	}
	log.Printf("initSlotContext verifyCastingBlock height=%v, hash=%v, lost trans size %v %v, ret %v\n",  bh.Height, bh.Hash.ShortS(), len(lostTxs), lostTxsStrings, ccr)
	sc.addLostTrans(lostTxs)
	if ccr == -1 {
		sc.setSlotStatus(SS_FAILED)
		return false
	} else {
		sc.setSlotStatus(SS_WAITING)
		return true
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
func (sc *SlotContext) IsRewardSent() bool {
	st := sc.GetSlotStatus()
	return st == SS_REWARD_SEND
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
func (sc *SlotContext) AcceptTrans(ths []common.Hash) bool {
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
		if !rsign.IsValid() {
			panic(fmt.Sprintf("rsign is invalid, bhHash=%v, height=%v, random=%v", bh.Hash.ShortS(), bh.Height, bh.Random))
		}
		radd, rgen := sc.rSignGenerator.AddWitness(si.SignMember, *rsign)

		if radd && generate && rgen { //达到组签名条件
			sc.setSlotStatus(SS_RECOVERD)
			sc.BH.Signature = sc.gSignGenerator.GetGroupSign().Serialize()
			sc.BH.Random = sc.rSignGenerator.GetGroupSign().Serialize()
			if len(sc.BH.Signature) == 0 {
				newBizLog("AcceptVerifyPiece").log("slot bh sign is empty hash=%v, sign=%v", sc.BH.Hash.ShortS(), sc.gSignGenerator.GetGroupSign().ShortS())
			}
			return CBMR_THRESHOLD_SUCCESS
		} else {
			return CBMR_PIECE_NORMAL
		}
	}
	return CBMR_ERROR_UNKNOWN
}

func (sc *SlotContext) IsValid() bool {
	return sc.GetSlotStatus() != SS_INITING
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
