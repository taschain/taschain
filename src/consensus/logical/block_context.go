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
	"sync/atomic"
)

///////////////////////////////////////////////////////////////////////////////
//组铸块共识上下文结构（一个高度有一个上下文，一个组的不同铸块高度不重用）
type BlockContext struct {
	Version         uint
	GroupMembers    int                        //组成员数量
	Proc    *Processor   //处理器
	MinerID *model.GroupMinerID //矿工ID和所属组ID
	pos     int          //矿工在组内的排位

	//变化
	verifyContexts []*VerifyContext
	currentVCtx    atomic.Value 	//当前铸块的verifycontext

	recentCastedHeight []uint64
	castedCount		uint64

	lock sync.RWMutex

}

func NewBlockContext(p *Processor, sgi *StaticGroupInfo) *BlockContext {
	bc := &BlockContext{
		Proc: p,
		MinerID: model.NewGroupMinerID(sgi.GroupID, p.GetMinerID()),
		verifyContexts: make([]*VerifyContext, 0),
		GroupMembers: len(sgi.Members),
		Version: model.CONSENSUS_VERSION,
		castedCount: 0,
		recentCastedHeight: make([]uint64, 20),
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
	vctx := bc.getCurrentVerifyCtx()
	if vctx != nil {
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

func (bc *BlockContext) getCurrentVerifyCtx() *VerifyContext {
	return bc.currentVCtx.Load().(*VerifyContext)
}

func (bc *BlockContext) setCurrentVerifyCtx(vctx *VerifyContext)  {
	bc.currentVCtx.Store(vctx)
}

func (bc *BlockContext) AddVerifyContext(vctx *VerifyContext) bool {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	bc.verifyContexts = append(bc.verifyContexts, vctx)
	return true
}

func (bc *BlockContext) replaceVerifyCtx(idx int, height uint64, expireTime time.Time, preBH *types.BlockHeader) *VerifyContext {
	bc.lock.Lock()
	defer bc.lock.Unlock()
	vctx := newVerifyContext(bc, height, expireTime, preBH)
	bc.verifyContexts[idx] = vctx
	return vctx
}

func (bc *BlockContext) getOrNewVctx(height uint64, expireTime time.Time, preBH *types.BlockHeader) (int, *VerifyContext) {
	var (
		vctx *VerifyContext
		idx int
	)

	//若该高度还没有verifyContext， 则创建一个
	if idx, vctx = bc.GetVerifyContextByHeight(height); vctx == nil {
		vctx = newVerifyContext(bc, height, expireTime, preBH)
		bc.AddVerifyContext(vctx)
		idx = len(bc.verifyContexts) -1
	} else {
		blog := newBizLog("getOrNewVctx")
		// hash不一致的情况下，
		if vctx.prevBH.Hash != preBH.Hash {
			blog.log("vctx pre hash diff, height=%v, existHash=%v, commingHash=%v", height, GetHashPrefix(vctx.prevBH.Hash), GetHashPrefix(preBH.Hash))
			preOld := bc.Proc.getBlockHeaderByHash(vctx.prevBH.Hash)
			//原来的preBH可能被分叉调整干掉了，则此vctx已无效， 重新用新的preBH
			if preOld == nil {
				vctx = bc.replaceVerifyCtx(idx, height, expireTime, preBH)
				return idx, vctx
			}
			preNew := bc.Proc.getBlockHeaderByHash(preBH.Hash)
			//新的preBH不存在了，也可能被分叉干掉了，此处直接返回nil
			if preNew == nil {
				return -1, nil
			}
			//新旧preBH都非空， 取高度高的preBH？
			if preOld.Height < preNew.Height {
				vctx = bc.replaceVerifyCtx(idx, height, expireTime, preNew)
			}
		}
	}
	return idx, vctx
}

func (bc *BlockContext) GetOrNewVerifyContext(bh *types.BlockHeader, preBH *types.BlockHeader) *VerifyContext {
	expireTime := GetCastExpireTime(bh.PreTime, bh.Height - preBH.Height)
	_, vctx := bc.getOrNewVctx(bh.Height, expireTime, preBH)
	return vctx
}

func (bc *BlockContext) CleanVerifyContext(height uint64)  {
	newCtxs := make([]*VerifyContext, 0)
	for _, ctx := range bc.SafeGetVerifyContexts() {
		bRemove := ctx.shouldRemove(height)
		if !bRemove  {
			newCtxs = append(newCtxs, ctx)
		} else {
			if bc.getCurrentVerifyCtx() == ctx {
				bc.reset()
			}
			log.Printf("CleanVerifyContext: ctx.castHeight=%v, ctx.prevHash=%v\n", ctx.castHeight, GetHashPrefix(ctx.prevBH.Hash))
		}
	}

	bc.lock.Lock()
	defer bc.lock.Unlock()
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
	vctx := bc.getCurrentVerifyCtx()
	if vctx != nil {
		return fmt.Sprintf("status=%v, castHeight=%v, prevHash=%v, prevTime=%v", vctx.consensusStatus, vctx.castHeight, GetHashPrefix(vctx.prevBH.Hash), vctx.prevBH.CurTime.String())
	} else {
		return "not in casting!"
	}
}

func (bc *BlockContext) Reset() {
	bc.reset()
}

//铸块上下文复位，在某个高度轮到当前组铸块时调用。
//to do : 还是索性重新生成。
func (bc *BlockContext) reset() {
	bc.setCurrentVerifyCtx(nil)
	bc.Proc.Ticker.StopTickerRoutine(bc.getKingCheckRoutineName())
}

//开始铸块
func (bc *BlockContext) StartCast(castHeight uint64, expire time.Time, baseBH *types.BlockHeader) bool {
	idx, vctx := bc.getOrNewVctx(castHeight, expire, baseBH)
	if vctx == nil {
		return false
	}
	if !vctx.isCasting() {
		bc.replaceVerifyCtx(idx, castHeight, expire, baseBH)
	}

	bc.setCurrentVerifyCtx(vctx)
	bc.Proc.Ticker.StartAndTriggerRoutine(bc.getKingCheckRoutineName())
	return true
}

//定时器例行处理
//如果返回false, 则关闭定时器
func (bc *BlockContext) kingTickerRoutine() bool {
	if !bc.Proc.Ready() {
		return false
	}
	//log.Printf("proc(%v) begin kingTickerRoutine, time=%v...\n", bc.Proc.getPrefix(), time.Now().Format(time.Stamp))

	vctx := bc.getCurrentVerifyCtx()
	if vctx == nil {
		log.Printf("kingTickerRoutine: verifyContext is nil, return!\n")
		return false
	}

	if !vctx.isCasting() || vctx.castSuccess() { //没有在组铸块共识中或已经出最高qn块
		log.Printf("proc(%v) not in casting, reset and direct return. castingInfo=%v.\n", bc.Proc.getPrefix(), bc.castingInfo())
		bc.reset() //提前出块完成
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