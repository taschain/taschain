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
	"fmt"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/middleware/types"
	"gopkg.in/fatih/set.v0"
	"sync/atomic"
)

/*
**  Creator: pxf
**  Date: 2018/5/21 下午5:49
**  Description:
 */

const (
	slIniting       int32 = iota
	slWaiting             //等待签名片段达到阈值
	slSigned              //自己是否签名过
	slRecoverd            //恢复出组签名
	slVerified            //组签名用组公钥验证通过
	slSuccess             //已上链广播
	slFailed              //铸块过程中失败，不可逆
	slRewardSignReq       //分红交易签名请求已发
	slRewardSent          //分红交易已广播
)

//铸块槽结构，和某个KING的共识数据一一对应
type SlotContext struct {
	//验证相关
	BH *types.BlockHeader //出块头详细数据
	//QueueNumber    int64             //铸块槽序号(<0无效)，等同于出块人序号。
	//vrfValue       *big.Int
	gSignGenerator *model.GroupSignGenerator //块签名产生器
	rSignGenerator *model.GroupSignGenerator //随机数签名产生器
	slotStatus     int32
	//lostTxHash     set.Interface

	castor groupsig.ID

	txs []*types.Transaction

	//initLock sync.Mutex

	//奖励相关
	rewardTrans    *types.Transaction
	rewardGSignGen *model.GroupSignGenerator //奖励交易签名产生器

	signedRewardTxHashs set.Interface //已签名的交易hash
}

func createSlotContext(bh *types.BlockHeader, threshold int) *SlotContext {
	return &SlotContext{
		BH: bh,
		//vrfValue:       bh.ProveValue,
		castor:         groupsig.DeserializeID(bh.Castor),
		slotStatus:     slWaiting,
		gSignGenerator: model.NewGroupSignGenerator(threshold),
		rSignGenerator: model.NewGroupSignGenerator(threshold),
		//rewardGSignGen: model.NewGroupSignGenerator(threshold),
		//lostTxHash:     set.New(set.ThreadSafe),
	}
}

//加锁，只要初始化一次（verifyblock）
//func (sc *SlotContext) initIfNeeded() bool {
//	sc.initLock.Lock()
//	defer sc.initLock.Unlock()
//
//	bh := sc.BH
//	if sc.slotStatus == slIniting {
//		slog := taslog.NewSlowLog("InitSlot", 0.1)
//		slog.AddStage("VerifyBlock")
//		lostTxs, ccr := core.BlockChainImpl.VerifyBlock(*bh)
//		slog.EndStage()
//		slog.Log("height=%v, hash=%v, lost trans size %v , ret %v", bh.Height, bh.Hash.ShortS(), len(lostTxs), ccr)
//
//		lostTxsStrings := make([]string, len(lostTxs))
//		for idx, tx := range lostTxs {
//			lostTxsStrings[idx] = tx.ShortS()
//		}
//		sc.addLostTrans(lostTxs)
//		if ccr == -1 {
//			sc.setSlotStatus(slFailed)
//			return false
//		} else {
//			sc.setSlotStatus(slWaiting)
//		}
//	}
//	return true
//}

//func (sc *SlotContext) HasTransLost() bool {
//	return sc.lostTxHash.Size() > 0
//}

func (sc *SlotContext) setSlotStatus(st int32) {
	atomic.StoreInt32(&sc.slotStatus, st)
}

func (sc *SlotContext) IsFailed() bool {
	st := sc.GetSlotStatus()
	return st == slFailed
}
func (sc *SlotContext) IsRewardSent() bool {
	st := sc.GetSlotStatus()
	return st == slRewardSent
}
func (sc *SlotContext) GetSlotStatus() int32 {
	return atomic.LoadInt32(&sc.slotStatus)
}

//func (sc SlotContext) lostTransSize() int {
//	return sc.lostTxHash.Size()
//}

//func (sc *SlotContext) addLostTrans(txs []common.Hash) {
//	if len(txs) == 0 {
//		return
//	}
//	for _, tx := range txs {
//		sc.lostTxHash.Add(tx)
//	}
//}

//用接收到的新交易更新缺失的交易集
//返回接收前以及接收后是否不在缺失交易
//func (sc *SlotContext) AcceptTrans(ths []common.Hash) bool {
//	l := sc.lostTransSize()
//	if l == 0 { //已经无缺失
//		return false
//	}
//	for _, tx := range ths {
//		sc.lostTxHash.Remove(tx)
//	}
//	return l > sc.lostTransSize()
//}

func (sc SlotContext) MessageSize() int {
	return sc.gSignGenerator.WitnessSize()
}

//验证组签名
//pk：组公钥
//返回true验证通过，返回false失败。
func (sc *SlotContext) VerifyGroupSigns(pk groupsig.Pubkey, preRandom []byte) bool {
	if sc.IsVerified() || sc.IsSuccess() {
		return true
	}
	b := sc.gSignGenerator.VerifyGroupSign(pk, sc.BH.Hash.Bytes())
	if b {
		b = sc.rSignGenerator.VerifyGroupSign(pk, preRandom)
		if b {
			sc.setSlotStatus(slVerified) //组签名验证通过
		}
	}
	if !b {
		sc.setSlotStatus(slFailed)
	}
	return b
}

func (sc *SlotContext) IsVerified() bool {
	return sc.GetSlotStatus() == slVerified
}

func (sc *SlotContext) IsRecovered() bool {
	return sc.GetSlotStatus() == slRecoverd
}

func (sc *SlotContext) IsSuccess() bool {
	return sc.GetSlotStatus() == slSuccess
}

func (sc *SlotContext) IsWaiting() bool {
	return sc.GetSlotStatus() == slWaiting
}

//收到一个组内验证签名片段
//返回：=0, 验证请求被接受，阈值达到组签名数量。=1，验证请求被接受，阈值尚未达到组签名数量。=2，重复的验签。=3，数据异常。
func (sc *SlotContext) AcceptVerifyPiece(signer groupsig.ID, sign groupsig.Signature, randomSign groupsig.Signature) (ret int8, err error) {
	var (
		add      bool
		generate bool
	)

	add, generate = sc.gSignGenerator.AddWitness(signer, sign)

	//已经收到过该成员的验签
	if !add {
		//忽略
		return pieceFail, fmt.Errorf("CBMR_IGNORE_REPEAT")
	}

	//没有收到过该用户的签名
	radd, rgen := sc.rSignGenerator.AddWitness(signer, randomSign)
	if radd && generate && rgen { //达到组签名条件
		sc.setSlotStatus(slRecoverd)
		return pieceThreshold, nil
	}
	return pieceNormal, nil
}

func (sc *SlotContext) IsValid() bool {
	return sc.GetSlotStatus() != slIniting
}

func (sc *SlotContext) StatusTransform(from int32, to int32) bool {
	return atomic.CompareAndSwapInt32(&sc.slotStatus, from, to)
}

//func (sc *SlotContext) TransBrief() string {
//	return fmt.Sprintf("总交易数%v，缺失%v", len(sc.BH.Transactions), sc.lostTransSize())
//}

func (sc *SlotContext) SetRewardTrans(tx *types.Transaction) bool {
	if !sc.hasSignedRewardTx() && sc.StatusTransform(slSuccess, slRewardSignReq) {
		sc.rewardTrans = tx
		return true
	}
	return false
}

func (sc *SlotContext) AcceptRewardPiece(sd *model.SignData) (accept, recover bool) {
	if sc.rewardTrans != nil && sc.rewardTrans.Hash != sd.DataHash {
		return
	}
	if sc.rewardTrans == nil {
		return
	}
	if sc.rewardGSignGen == nil {
		sc.rewardGSignGen = model.NewGroupSignGenerator(sc.gSignGenerator.Threshold())
	}
	accept, recover = sc.rewardGSignGen.AddWitness(sd.GetID(), sd.DataSign)
	if accept && recover {
		//铸块分红交易使用组签名
		if sc.rewardTrans.Sign == nil {
			signBytes := sc.rewardGSignGen.GetGroupSign().Serialize()
			tmpBytes := make([]byte, common.SignLength)
			//group signature length = 33, common signature length = 65.  VerifyBonusTransaction() will recover common sig to groupsig
			copy(tmpBytes[0:len(signBytes)], signBytes)
			sign := common.BytesToSign(tmpBytes)
			sc.rewardTrans.Sign = sign.Bytes()
			//fmt.Printf("Bonus sign 1: hash=%v , gsign=%v\n", sd.DataHash.Hex(), sc.rewardGSignGen.GetGroupSign().GetHexString())
		}
	}
	return
}

func (sc *SlotContext) addSignedTxHash(hash common.Hash) {
	if sc.signedRewardTxHashs == nil {
		sc.signedRewardTxHashs = set.New(set.ThreadSafe)
	}
	sc.signedRewardTxHashs.Add(hash)
}

func (sc *SlotContext) hasSignedTxHash(hash common.Hash) bool {
	if sc.signedRewardTxHashs == nil {
		return false
	}
	return sc.signedRewardTxHashs.Has(hash)
}

//是否签过分红交易
func (sc *SlotContext) hasSignedRewardTx() bool {
	if sc.signedRewardTxHashs == nil {
		return false
	}
	return sc.signedRewardTxHashs.Size() > 0
}