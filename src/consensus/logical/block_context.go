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
	"log"
	"time"
	"fmt"
	"sync"
	"middleware/types"
	"consensus/model"
)

///////////////////////////////////////////////////////////////////////////////
//组铸块共识上下文结构（一个高度有一个上下文，一个组的不同铸块高度不重用）
type BlockContext struct {
	Version         uint
	GroupMembers    int                        //组成员数量
	verifyContexts	[]*VerifyContext
	currentVerifyContext *VerifyContext //当前铸块的verifycontext

	recentCastedHeight []uint64
	castedCount		uint64

	lock sync.RWMutex

	Proc    *Processor   //处理器
	MinerID *model.GroupMinerID //矿工ID和所属组ID
	pos     int          //矿工在组内的排位
}

func NewBlockContext(p *Processor, sgi *StaticGroupInfo) *BlockContext {
	bc := &BlockContext{
		Proc: p,
		MinerID: model.NewGroupMinerID(sgi.GroupID, p.GetMinerID()),
		verifyContexts: make([]*VerifyContext, 0),
		GroupMembers: len(sgi.Members),
		Version: model.CONSENSUS_VERSION,
		castedCount: 0,
		recentCastedHeight: make([]uint64, 100),
	}
	bc.reset()
	return bc
}

func (bc *BlockContext) threshold() int {
    return model.Param.GetGroupK(bc.GroupMembers)
}

func (bc *BlockContext) getKingCheckRoutineName() string {
	return "king_check_routine_" + GetIDPrefix(bc.MinerID.Gid)
}

func (bc *BlockContext) alreadyInCasting(height uint64, preHash common.Hash) bool {
	vctx := bc.GetCurrentVerifyContext()
	if vctx != nil {
		vctx.lock.Lock()
		defer vctx.lock.Unlock()
		return vctx.isCasting() && !vctx.castSuccess() && vctx.castHeight == height && vctx.prevBH.Hash == preHash
	} else {
		return false
	}
}

func (bc *BlockContext) GetVerifyContextByHeight(height uint64) (int, *VerifyContext) {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	for idx, ctx := range bc.verifyContexts {
		if ctx.castHeight == height {
			return idx, ctx
		}
	}
	return -1, nil
}

func (bc *BlockContext) GetCurrentVerifyContext() *VerifyContext {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	return bc.currentVerifyContext
}

func (bc *BlockContext) AddVerifyContext(vctx *VerifyContext) bool {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	bc.verifyContexts = append(bc.verifyContexts, vctx)
	return true
}

func (bc *BlockContext) getOrNewVctx(height uint64, expireTime time.Time, preBH *types.BlockHeader) *VerifyContext {
	var vctx *VerifyContext
	//若该高度还没有verifyContext， 则创建一个
	if _, vctx = bc.GetVerifyContextByHeight(height); vctx == nil {
		vctx = newVerifyContext(bc, height, expireTime, preBH)
		bc.AddVerifyContext(vctx)
	} else {
		vctx.lock.Lock()
		defer vctx.lock.Unlock()

		blog := newBizLog("getOrNewVctx")
		if vctx.prevBH.Hash != preBH.Hash {
			blog.log("vctx pre hash diff, height=%v, existHash=%v, commingHash=%v", height, GetHashPrefix(vctx.prevBH.Hash), GetHashPrefix(preBH.Hash))
			// hash不一致的情况下，
			// 1. 先取前项hash已存在的
			// 2. 若前项hash都存在，则取前项hash高度高的
			preBH1 := bc.Proc.getBlockHeaderByHash(vctx.prevBH.Hash)
			preBH2 := bc.Proc.getBlockHeaderByHash(preBH.Hash)
			if preBH1 != nil && preBH2 == nil {
				return vctx
			} else if preBH1 == nil && preBH2 != nil {
				vctx.rebase(bc, height, expireTime, preBH2)
			} else if preBH1 != nil && preBH2 != nil {
				if preBH1.Height < preBH2.Height {
					vctx.rebase(bc, height, expireTime, preBH2)
				}
			} else {
				blog.log("both preBH is nil!height=%v, preHash1=%v, preHash2=%v", height, GetHashPrefix(vctx.prevBH.Hash), GetHashPrefix(preBH.Hash))
				panic("verifycontext error")
			}
		}
	}
	return vctx
}

func (bc *BlockContext) GetOrNewVerifyContext(bh *types.BlockHeader, preBH *types.BlockHeader) *VerifyContext {
	expireTime := GetCastExpireTime(bh.PreTime, bh.Height - preBH.Height)
	return bc.getOrNewVctx(bh.Height, expireTime, preBH)
}

func (bc *BlockContext) CleanVerifyContext(height uint64)  {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	newCtxs := make([]*VerifyContext, 0)
	for _, ctx := range bc.verifyContexts {
		if !ctx.ShouldRemove(height) {
			newCtxs = append(newCtxs, ctx)
		} else {
			if bc.currentVerifyContext == ctx {
				bc.reset()
			}
			log.Printf("CleanVerifyContext: ctx.castHeight=%v, ctx.prevHash=%v\n", ctx.castHeight, GetHashPrefix(ctx.prevBH.Hash))
		}
	}
	bc.verifyContexts = newCtxs
}

func (bc *BlockContext) SafeGetVerifyContexts() []*VerifyContext {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	ctxs := make([]*VerifyContext, len(bc.verifyContexts))
	copy(ctxs, bc.verifyContexts)
	return ctxs
}

func (bc *BlockContext) castingInfo() string {
	vctx := bc.currentVerifyContext
	if vctx != nil {
		return fmt.Sprintf("status=%v, castHeight=%v, prevHash=%v, prevTime=%v", vctx.consensusStatus, vctx.castHeight, GetHashPrefix(vctx.prevBH.Hash), vctx.prevBH.CurTime.String())
	} else {
		return "not in casting!"
	}
}

func (bc *BlockContext) Reset() {
	bc.lock.Lock()
	defer bc.lock.Unlock()

	bc.reset()
}

//铸块上下文复位，在某个高度轮到当前组铸块时调用。
//to do : 还是索性重新生成。
func (bc *BlockContext) reset() {
	bc.currentVerifyContext = nil
	bc.Proc.Ticker.StopTickerRoutine(bc.getKingCheckRoutineName())
	//log.Printf("end BlockContext::Reset.\n")
}

//开始铸块
func (bc *BlockContext) StartCast(castHeight uint64, expire time.Time, baseBH *types.BlockHeader) {
	vctx := bc.getOrNewVctx(castHeight, expire, baseBH)
	vctx.lock.Lock()
	if !vctx.isCasting() {
		vctx.rebase(bc, castHeight, expire, baseBH)
	}
	vctx.lock.Unlock()

	bc.lock.Lock()
	bc.currentVerifyContext = vctx
	bc.lock.Unlock()

	bc.Proc.Ticker.StartAndTriggerRoutine(bc.getKingCheckRoutineName())
	return
}

//定时器例行处理
//如果返回false, 则关闭定时器
func (bc *BlockContext) kingTickerRoutine() bool {
	if !bc.Proc.Ready() {
		return false
	}
	//log.Printf("proc(%v) begin kingTickerRoutine, time=%v...\n", bc.Proc.getPrefix(), time.Now().Format(time.Stamp))

	vctx := bc.GetCurrentVerifyContext()
	if vctx == nil {
		log.Printf("kingTickerRoutine: verifyContext is nil, return!\n")
		return false
	}

	vctx.lock.Lock()
	defer vctx.lock.Unlock()

	if !vctx.isCasting() || vctx.castSuccess() { //没有在组铸块共识中或已经出最高qn块
		log.Printf("proc(%v) not in casting, reset and direct return. castingInfo=%v.\n", bc.Proc.getPrefix(), bc.castingInfo())
		bc.Reset() //提前出块完成
		return false
	}

	d := time.Since(vctx.prevBH.CurTime) //上个铸块完成到现在的时间
	//max := vctx.getMaxCastTime()

	if vctx.castExpire() { //超过了组最大铸块时间
		log.Printf("proc(%v) end kingTickerRoutine, out of max group cast time, time=%v secs, castInfo=%v.\n", bc.Proc.getPrefix(), d.Seconds(), bc.castingInfo())
		//bc.reset()
		vctx.markTimeout()
		return false
	} else {
		//当前组仍在有效铸块共识时间内
		//检查自己是否成为铸块人
		index, qn := vctx.calcCastor() //当前铸块人（KING）和QN值
		if index < 0 {
			log.Printf("kingTickerRoutine: calcCastor index =%v\n", index)
			return false
		}
		bc.Proc.kingCheckAndCast(bc, vctx, index, qn)
		return true
	}
	return true
}

func (bc *BlockContext) getGroupSecret() *GroupSecret {
    return bc.Proc.getGroupSecret(bc.MinerID.Gid)
}

func (bc *BlockContext) registerTicker()  {
	bc.Proc.Ticker.RegisterRoutine(bc.getKingCheckRoutineName(), bc.kingTickerRoutine, uint32(model.Param.MaxUserCastTime))
}

func (bc *BlockContext) removeTicker()  {
	bc.Proc.Ticker.RemoveRoutine(bc.getKingCheckRoutineName())
}

func (bc *BlockContext) IsHeightCasted(height uint64) bool {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	for _, h := range bc.recentCastedHeight {
		if h == height {
			return true
		}
	}
	return false
}

func (bc *BlockContext) AddCastedHeight(height uint64)  {
	if bc.IsHeightCasted(height) {
		return
	}
	bc.lock.Lock()
	defer bc.lock.Unlock()

    bc.castedCount++
    idx := bc.castedCount % uint64(len(bc.recentCastedHeight))
    bc.recentCastedHeight[idx] = height
}