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
	"consensus/model"
	"middleware/types"
	"sync"
	"time"
	"common"
)

type castedBlock struct {
	height uint64
	preHash common.Hash
}

///////////////////////////////////////////////////////////////////////////////
//组铸块共识上下文结构（一个高度有一个上下文，一个组的不同铸块高度不重用）
type BlockContext struct {
	Version      uint
	GroupMembers int                 //组成员数量
	Proc         *Processor          //处理器
	MinerID      *model.GroupMinerID //矿工ID和所属组ID
	pos          int                 //矿工在组内的排位

	//变化
	vctxs map[uint64]*VerifyContext //height -> *VerifyContext
	//currentVCtx    atomic.Value 	//当前铸块的verifycontext

	recentCasted [40]*castedBlock
	curr          int

	lock sync.RWMutex
}

func NewBlockContext(p *Processor, sgi *StaticGroupInfo) *BlockContext {
	bc := &BlockContext{
		Proc:               p,
		MinerID:            model.NewGroupMinerID(sgi.GroupID, p.GetMinerID()),
		GroupMembers:       sgi.GetMemberCount(),
		vctxs:              make(map[uint64]*VerifyContext),
		Version:            model.CONSENSUS_VERSION,
		curr:          		0,
	}

	return bc
}

func (bc *BlockContext) threshold() int {
	return model.Param.GetGroupK(bc.GroupMembers)
}

func (bc *BlockContext) GetVerifyContextByHeight(height uint64) *VerifyContext {
	bc.lock.RLock()
	defer bc.lock.RUnlock()

	return bc.getVctxByHeight(height)
}

func (bc *BlockContext) getVctxByHeight(height uint64) *VerifyContext {
	if v, ok := bc.vctxs[height]; ok {
		return v
	}
	return nil
}

func (bc *BlockContext) replaceVerifyCtx(height uint64, expireTime time.Time, preBH *types.BlockHeader) *VerifyContext {
	vctx := newVerifyContext(bc, height, expireTime, preBH)
	bc.vctxs[vctx.castHeight] = vctx
	return vctx
}

func (bc *BlockContext) getOrNewVctx(height uint64, expireTime time.Time, preBH *types.BlockHeader) *VerifyContext {
	var vctx *VerifyContext
	blog := newBizLog("getOrNewVctx")

	//若该高度还没有verifyContext， 则创建一个
	if vctx = bc.getVctxByHeight(height); vctx == nil {
		vctx = newVerifyContext(bc, height, expireTime, preBH)
		bc.vctxs[vctx.castHeight] = vctx
		blog.log("add vctx expire %v", expireTime)
	} else {
		// hash不一致的情况下，
		if vctx.prevBH.Hash != preBH.Hash {
			blog.log("vctx pre hash diff, height=%v, existHash=%v, commingHash=%v", height, vctx.prevBH.Hash.ShortS(), preBH.Hash.ShortS())
			preOld := bc.Proc.getBlockHeaderByHash(vctx.prevBH.Hash)
			//原来的preBH可能被分叉调整干掉了，则此vctx已无效， 重新用新的preBH
			if preOld == nil {
				vctx = bc.replaceVerifyCtx(height, expireTime, preBH)
				return vctx
			}
			preNew := bc.Proc.getBlockHeaderByHash(preBH.Hash)
			//新的preBH不存在了，也可能被分叉干掉了，此处直接返回nil
			if preNew == nil {
				return nil
			}
			//新旧preBH都非空， 取高度高的preBH？
			if preOld.Height < preNew.Height {
				vctx = bc.replaceVerifyCtx(height, expireTime, preNew)
			}
		} else {
			if height == 1 && expireTime.After(vctx.expireTime) {
				vctx.expireTime = expireTime
			}
			blog.log("get exist vctx height %v, expire %v", height, vctx.expireTime)
		}
	}
	return vctx
}

func (bc *BlockContext) SafeGetVerifyContexts() []*VerifyContext {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	vctx := make([]*VerifyContext, len(bc.vctxs))
	i := 0
	for _, vc := range bc.vctxs {
		vctx[i] = vc
		i++
	}
	return vctx
}

func (bc *BlockContext) GetOrNewVerifyContext(bh *types.BlockHeader, preBH *types.BlockHeader) *VerifyContext {
	deltaHeightByTime := DeltaHeightByTime(bh, preBH)

	expireTime := GetCastExpireTime(preBH.CurTime, deltaHeightByTime, bh.Height)

	bc.lock.Lock()
	defer bc.lock.Unlock()

	vctx := bc.getOrNewVctx(bh.Height, expireTime, preBH)
	return vctx
}

func (bc *BlockContext) CleanVerifyContext(height uint64) {
	newCtxs := make(map[uint64]*VerifyContext, 0)
	for _, ctx := range bc.SafeGetVerifyContexts() {
		bRemove := ctx.shouldRemove(height)
		if !bRemove {
			newCtxs[ctx.castHeight] = ctx
		} else {
			ctx.Clear()
			bc.Proc.blockContexts.removeReservedVctx(ctx.castHeight)
			stdLogger.Debugf("CleanVerifyContext: ctx.castHeight=%v, ctx.prevHash=%v, signedMaxQN=%v, signedNum=%v\n", ctx.castHeight, ctx.prevBH.Hash.ShortS(), ctx.signedMaxWeight, ctx.signedNum)
		}
	}

	bc.lock.Lock()
	defer bc.lock.Unlock()
	bc.vctxs = newCtxs
}
//
func (bc *BlockContext) IsHeightCasted(height uint64, pre common.Hash) (cb *castedBlock, casted bool) {
	bc.lock.RLock()
	defer bc.lock.RUnlock()
	for i, h := range bc.recentCasted {
		if h != nil && h.height == height {
			cb = bc.recentCasted[i]
			casted = h.preHash == pre
			return
		}
	}
	return
}

func (bc *BlockContext) AddCastedHeight(height uint64, pre common.Hash) {
	if cb, same := bc.IsHeightCasted(height, pre); same {
		return
	} else {
		bc.lock.Lock()
		defer bc.lock.Unlock()

		if cb != nil {
			cb.preHash = pre
		} else {
			bc.recentCasted[bc.curr] = &castedBlock{height: height, preHash: pre}
			bc.curr = (bc.curr+1)%len(bc.recentCasted)
		}
	}
}
