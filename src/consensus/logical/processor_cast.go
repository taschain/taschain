package logical

import (
	"log"
	"time"
	"middleware/types"
	"consensus/groupsig"
	"common"
	"sync"
	"consensus/model"
	"consensus/base"
	"consensus/net"
	"middleware/statistics"
)

/*
**  Creator: pxf
**  Date: 2018/6/27 上午10:39
**  Description: 
*/

type CastBlockContexts struct {
	contexts sync.Map	//string -> *BlockContext
}

func NewCastBlockContexts() *CastBlockContexts {
	return &CastBlockContexts{
		contexts: sync.Map{},
	}
}

func (bctx *CastBlockContexts) addBlockContext(bc *BlockContext) (add bool) {
    _, load := bctx.contexts.LoadOrStore(bc.MinerID.Gid.GetHexString(), bc)
    return !load
}

func (bctx *CastBlockContexts) getBlockContext(gid groupsig.ID) *BlockContext {
	if v, ok := bctx.contexts.Load(gid.GetHexString()); ok {
		return v.(*BlockContext)
	}
	return nil
}

func (bctx *CastBlockContexts) contextSize() int32 {
	size := int32(0)
	bctx.contexts.Range(func(key, value interface{}) bool {
		size ++
		return true
	})
	return size
}

func (bctx *CastBlockContexts) removeContexts(gids []groupsig.ID)  {
	for _, id := range gids {
		log.Println("removeContexts ", GetIDPrefix(id))
		bc := bctx.getBlockContext(id)
		if bc != nil {
			bc.removeTicker()
			bctx.contexts.Delete(id.GetHexString())
		}
	}
}

func (bctx *CastBlockContexts) forEach(f func(bc *BlockContext) bool) {
    bctx.contexts.Range(func(key, value interface{}) bool {
    	v := value.(*BlockContext)
		return f(v)
	})
}


//增加一个铸块上下文（一个组有一个铸块上下文）
func (p *Processor) AddBlockContext(bc *BlockContext) bool {
	var add = p.blockContexts.addBlockContext(bc)
	log.Printf("AddBlockContext, gid=%v, result=%v\n.", GetIDPrefix(bc.MinerID.Gid), add)
	return add
}

//取得一个铸块上下文
//gid:组ID hex 字符串
func (p *Processor) GetBlockContext(gid groupsig.ID) *BlockContext {
	return p.blockContexts.getBlockContext(gid)
}

func (p *Processor) getReleaseRoutineName() string {
	return "release_routine_" + p.getPrefix()
}


//立即触发一次检查自己是否下个铸块组
func (p *Processor) triggerCastCheck()  {
	//p.Ticker.StartTickerRoutine(p.getCastCheckRoutineName(), true)
	p.Ticker.StartAndTriggerRoutine(p.getCastCheckRoutineName())
}

//检查是否当前组铸块
func (p *Processor) checkSelfCastRoutine() bool {
	if !p.Ready() {
		return false
	}

	blog := newBizLog("checkSelfCastRoutine")

	if p.MainChain.IsAdujsting() {
		blog.log("isAdjusting, return...")
		p.triggerCastCheck()
		return false
	}

	top := p.MainChain.QueryTopBlock()

	var (
		expireTime time.Time
		castHeight uint64
		deltaHeight uint64
	)
	d := time.Since(top.CurTime)
	if d < 0 {
		return false
	}

	deltaHeight = uint64(d.Seconds()) / uint64(model.Param.MaxGroupCastTime) + 1
	expireTime = GetCastExpireTime(top.CurTime, deltaHeight)

	if top.Height > 0 {
		castHeight = top.Height + deltaHeight
	} else {
		castHeight = uint64(1)
	}
	if !p.canProposalAt(castHeight) {
		return false
	}

	blog.log("topHeight=%v, topHash=%v, topCurTime=%v, castHeight=%v, expireTime=%v", top.Height, GetHashPrefix(top.Hash), top.CurTime, castHeight, expireTime)
	worker := p.getVrfWorker()

	if worker != nil && worker.workingOn(top, castHeight) {
		blog.log("already working on that block, status=%v", worker.getStatus())
		return false
	} else {
		worker = newVRFWorker(top, castHeight, expireTime)
		p.setVrfWorker(worker)
		p.blockProposal()
	}
	return true

	//selectGroup := p.calcCastGroup(top, castHeight)
	//if selectGroup == nil {
	//	return false
	//}
	//
	//blog.log("NEXT CAST GROUP is %v, castHeight=%v, expire=%v", GetIDPrefix(*selectGroup), castHeight, expireTime)
	//
	////自己属于下一个铸块组
	//if p.IsMinerGroup(*selectGroup) {
	//	bc := p.GetBlockContext(*selectGroup)
	//	if bc == nil {
	//		blog.log("[ERROR]checkSelfCastRoutine: get nil blockcontext!, gid=%v", GetIDPrefix(*selectGroup))
	//		return false
	//	}
	//
	//	blog.log("BECOME NEXT CAST GROUP! castHeight=%v, uid=%v, gid=%v, vctxcnt=%v, castCnt=%v, rHeights=%v", castHeight, GetIDPrefix(p.GetMinerID()), GetIDPrefix(*selectGroup), len(bc.verifyContexts), bc.castedCount, bc.recentCastedHeight)
	//
	//	if !bc.StartCast(castHeight, expireTime, top) {
	//		blog.log("startCast fail, castHeight=%v, expire=%v,topBH=%v", castHeight, expireTime, bc.Proc.blockPreview(top))
	//	}
	//
	//	return true
	//} else { //自己不是下一个铸块组, 但是当前在铸块
	//	p.blockContexts.forEach(func(_bc *BlockContext) bool {
	//		blog.log("reset casting blockcontext: castingInfo=%v", _bc.castingInfo())
	//		_bc.Reset()
	//		return true
	//	})
	//}
	//
	//return false
}

func (p *Processor) getCastCheckRoutineName() string {
	return "self_cast_check_" + p.getPrefix()
}

func (p *Processor) calcVerifyGroup(preBH *types.BlockHeader, height uint64) *groupsig.ID {
	var hash common.Hash
	data := preBH.Random

	deltaHeight := height - preBH.Height
	for ; deltaHeight > 0; deltaHeight -- {
		hash = base.Data2CommonHash(data)
		data = hash.Bytes()
	}

	selectGroup, err := p.globalGroups.SelectNextGroup(hash, height)
	if err != nil {
		log.Println("calcCastGroup err:", err)
		return nil
	}
	return &selectGroup
}


//在某个区块高度的QN值成功出块，保存上链，向组外广播
//同一个高度，可能会因QN不同而多次调用该函数
//但一旦低的QN出过，就不该出高的QN。即该函数可能被多次调用，但是调用的QN值越来越小
func (p *Processor) SuccessNewBlock(bh *types.BlockHeader, vctx *VerifyContext, slot *SlotContext, gid groupsig.ID) {

	if bh == nil {
		panic("SuccessNewBlock arg failed.")
	}

	if p.blockOnChain(bh) { //已经上链
		log.Printf("SuccessNewBlock core.GenerateBlock is nil! block alreayd onchain!")
		return
	}

	block := p.MainChain.GenerateBlock(*bh)

	if block == nil {
		log.Printf("SuccessNewBlock core.GenerateBlock is nil! won't broadcast block!")
		return
	}

	r := p.doAddOnChain(block)

	if r != 0 && r != 1 { //分叉调整或 上链失败都不走下面的逻辑
		return
	}

	cbm := &model.ConsensusBlockMessage{
		Block: *block,
	}

	nextId := p.calcCastGroup(bh, bh.Height+1)
	group := p.getGroup(*nextId)
	mems := make([]groupsig.ID, len(group.Members))
	for idx, mem := range group.Members {
		mems[idx] = mem.ID
	}
	next := &net.NextGroup{
		Gid: *nextId,
		MemIds: mems,
	}
	if slot.StatusTransform(SS_VERIFIED, SS_SUCCESS) {
		logHalfway("SuccessNewBlock", bh.Height, bh.QueueNumber, p.getPrefix(), "SuccessNewBlock, hash %v, 耗时%v秒", GetHashPrefix(bh.Hash), time.Since(bh.CurTime).Seconds())
		p.NetServer.BroadcastNewBlock(cbm, next)
	}

	return
}


//检查是否轮到自己出块
//func (p *Processor) kingCheckAndCast(bc *BlockContext, vctx *VerifyContext, kingIndex int32, qn int64) {
//	//p.castLock.Lock()
//	//defer p.castLock.Unlock()
//	gid := bc.MinerID.Gid
//	height := vctx.castHeight
//
//	//log.Printf("prov(%v) begin kingCheckAndCast, gid=%v, kingIndex=%v, qn=%v, height=%v.\n", p.getPrefix(), GetIDPrefix(gid), kingIndex, qn, height)
//	if kingIndex < 0 || qn < 0 {
//		return
//	}
//
//	sgi := p.getGroup(gid)
//
//	log.Printf("time=%v, Current kingIndex=%v, KING=%v, qn=%v.\n", time.Now().Format(time.Stamp), kingIndex, GetIDPrefix(sgi.GetCastor(int(kingIndex))), qn)
//	if sgi.GetCastor(int(kingIndex)).GetHexString() == p.GetMinerID().GetHexString() { //轮到自己铸块
//		log.Printf("curent node IS KING!\n")
//		if !vctx.isQNCasted(qn) { //在该高度该QN，自己还没铸过快
//			head := p.castBlock(bc, vctx, qn) //铸块
//			if head != nil {
//				vctx.addCastedQN(qn)
//			}
//		} else {
//			log.Printf("In height=%v, qn=%v current node already casted.\n", height, qn)
//		}
//	}
//	return
//}

func (p *Processor) blockProposal() {
	blog := newBizLog("blockProposal")
	top := p.MainChain.QueryTopBlock()
	worker := p.getVrfWorker()
	if worker.getBaseBH().Hash != top.Hash {
		blog.log("vrf baseBH differ from top!")
		return
	}

	ok, nonce := worker.prove()
	if !ok {
		blog.log("vrf prove not ok! nonce=%v", nonce)
		return
	}

	height := worker.castHeight

	blog.log("begin proposal, height=%v, qn=%v...\n", height, nonce)

	gid := p.calcVerifyGroup(top, height)
	if gid == nil {
		blog.log("calc verify group is nil!height=%v", height)
		return
	}

	logStart("CASTBLOCK", height, uint64(nonce), p.getPrefix(), "开始铸块")

	//调用鸠兹的铸块处理
	block := p.MainChain.CastingBlock(uint64(height), uint64(nonce), uint64(1), p.GetMinerID().Serialize(), gid.Serialize())
	if block == nil {
		blog.log("MainChain::CastingBlock failed, height=%v, nonce=%v, gid=%v, mid=%v.\n", height, nonce, GetIDPrefix(*gid), GetIDPrefix(p.GetMinerID()))
		//panic("MainChain::CastingBlock failed, jiuci return nil.\n")
		logHalfway("CASTBLOCK", height, uint64(nonce), p.getPrefix(), "铸块失败, block为空")
		return
	}

	bh := block.Header

	blog.log("bh %v, top bh %v\n", p.blockPreview(bh), p.blockPreview(p.MainChain.QueryTopBlock()))
	if bh.Height > 0 && bh.Height == height && bh.PreHash == worker.baseBH.Hash {
		skey := p.getSignKey(*gid)
		//发送该出块消息
		var ccm model.ConsensusCastMessage
		ccm.BH = *bh
		//ccm.GroupID = gid
		ccm.GenSign(model.NewSecKeyInfo(p.GetMinerID(), skey), &ccm)
		ccm.GenRandomSign(skey, worker.baseBH.Random)
		logHalfway("CASTBLOCK", height, uint64(nonce), p.getPrefix(), "铸块成功, SendVerifiedCast, hash %v, 时间间隔 %v", GetHashPrefix(bh.Hash), bh.CurTime.Sub(bh.PreTime).Seconds())

		p.NetServer.SendCastVerify(&ccm)

		worker.markProposed()

		statistics.AddBlockLog(common.BootId,statistics.SendCast,ccm.BH.Height,ccm.BH.QueueNumber,-1,-1,
			time.Now().UnixNano(),GetIDPrefix(p.GetMinerID()),GetIDPrefix(*gid),common.InstanceIndex,ccm.BH.CurTime.UnixNano())
	} else {
		log.Printf("bh/prehash Error or sign Error, bh=%v, real height=%v. bc.prehash=%v, bh.prehash=%v\n", height, bh.Height, worker.baseBH.Hash, bh.PreHash)
	}

}

