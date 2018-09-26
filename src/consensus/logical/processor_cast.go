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
	"math/big"
	"strings"
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
		log.Println("removeContexts ", id.ShortS())
		bc := bctx.getBlockContext(id)
		if bc != nil {
			//bc.removeTicker()
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
	newBizLog("AddBlockContext").log("gid=%v, result=%v\n.", bc.MinerID.Gid.ShortS(), add)
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

	blog.log("topHeight=%v, topHash=%v, topCurTime=%v, castHeight=%v, expireTime=%v", top.Height, top.Hash.ShortS(), top.CurTime, castHeight, expireTime)
	worker := p.getVrfWorker()

	if worker != nil && worker.workingOn(top, castHeight) {
		blog.log("already working on that block, status=%v", worker.getStatus())
		return false
	} else {
		worker = newVRFWorker(p.getSelfMinerDO(), top, castHeight, expireTime)
		p.setVrfWorker(worker)
		p.blockProposal()
	}
	return true
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
	blog := newBizLog("SuccessNewBlock")
	if p.blockOnChain(bh) { //已经上链
		blog.log("core.GenerateBlock is nil! block alreayd onchain!")
		return
	}

	block := p.MainChain.GenerateBlock(*bh)

	if block == nil {
		blog.log("core.GenerateBlock is nil! won't broadcast block!")
		return
	}

	r := p.doAddOnChain(block)

	if r != 0 && r != 1 { //分叉调整或 上链失败都不走下面的逻辑
		return
	}

	cbm := &model.ConsensusBlockMessage{
		Block: *block,
	}

	nextId := p.calcVerifyGroup(bh, bh.Height+1)
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
		newBlockTraceLog("SuccessNewBlock", bh.Hash, p.GetMinerID()).log( "height=%v, 耗时%v秒", bh.Height, time.Since(bh.CurTime).Seconds())
		p.NetServer.BroadcastNewBlock(cbm, next)
		blog.log("After BroadcastNewBlock:%v",time.Now().Format(TIMESTAMP_LAYOUT))
	}

	return
}

func (p *Processor) blockProposal() {
	blog := newBizLog("blockProposal")
	top := p.MainChain.QueryTopBlock()
	worker := p.getVrfWorker()
	if worker.getBaseBH().Hash != top.Hash {
		blog.log("vrf baseBH differ from top!")
		return
	}
	if worker.isProposed() || worker.isSuccess() {
		blog.log("vrf worker proposed/success, status %v", worker.getStatus())
		return
	}
	height := worker.castHeight

	totalStake := p.minerReader.getTotalStake(top.Height)
	blog.log("totalStake height=%v, stake=%v", top.Height, totalStake)
	pi, err := worker.prove(totalStake)
	if err != nil {
		blog.log("vrf prove not ok! %v", err)
		return
	}

	if worker.timeout() {
		blog.log("vrf worker timeout")
		return
	}

	gid := p.calcVerifyGroup(top, height)
	if gid == nil {
		blog.log("calc verify group is nil!height=%v", height)
		return
	}

	block := p.MainChain.CastingBlock(uint64(height), 0, new(big.Int).SetBytes(pi), p.GetMinerID().Serialize(), gid.Serialize())
	if block == nil {
		blog.log("MainChain::CastingBlock failed, height=%v", height)
		return
	}
	bh := block.Header
	tlog := newBlockTraceLog("CASTBLOCK", bh.Hash, p.GetMinerID())
	blog.log("begin proposal, hash=%v, height=%v, pi=%v...", bh.Hash.ShortS(), height, pi.ShortS())
	tlog.logStart("height=%v,pi=%v", bh.Height, pi.ShortS())

	if bh.Height > 0 && bh.Height == height && bh.PreHash == worker.baseBH.Hash {
		skey := p.getSignKey(*gid)
		//发送该出块消息
		var ccm model.ConsensusCastMessage
		ccm.BH = *bh
		//ccm.GroupID = gid
		ccm.GenSign(model.NewSecKeyInfo(p.GetMinerID(), skey), &ccm)
		ccm.GenRandomSign(skey, worker.baseBH.Random)
		tlog.log( "铸块成功, SendVerifiedCast, 时间间隔 %v", bh.CurTime.Sub(bh.PreTime).Seconds())
		p.NetServer.SendCastVerify(&ccm)

		worker.markProposed()

		statistics.AddBlockLog(common.BootId,statistics.SendCast,ccm.BH.Height,ccm.BH.ProveValue.Uint64(),-1,-1,
			time.Now().UnixNano(),p.GetMinerID().ShortS(),gid.ShortS(),common.InstanceIndex,ccm.BH.CurTime.UnixNano())
	} else {
		blog.log("bh/prehash Error or sign Error, bh=%v, real height=%v. bc.prehash=%v, bh.prehash=%v\n", height, bh.Height, worker.baseBH.Hash, bh.PreHash)
	}

}

//请求组内对奖励交易签名
func (p *Processor) reqRewardTransSign(vctx *VerifyContext, bh *types.BlockHeader)  {
	blog := newBizLog("reqRewardTransSign")
	blog.log("start, bh=%v", p.blockPreview(bh))
	slot := vctx.GetSlotByHash(bh.Hash)
	if slot == nil {
		blog.log("slot is nil")
		return
	}
	if !slot.gSignGenerator.Recovered() {
		blog.log("slot not recovered")
		return
	}
	if !slot.IsSuccess() && !slot.IsVerified() {
		blog.log("slot not verified or success,status=%v", slot.GetSlotStatus())
		return
	}

	groupID := groupsig.DeserializeId(bh.GroupId)
	group := p.getGroup(groupID)

	witnesses := slot.gSignGenerator.GetWitnesses()
	size := len(witnesses)
	targetIdIndexs := make([]int32, size)
	signs := make([]groupsig.Signature, size)
	idHexs := make([]string, size)

	i := 0
	for idStr, piece := range witnesses {
		signs[i] = piece
		var id groupsig.ID
		id.SetHexString(idStr)
		idHexs[i] = id.ShortS()
		targetIdIndexs[i] = int32(group.GetMinerPos(id))
		i++
	}

	bonus, tx := p.MainChain.GenerateBonus(targetIdIndexs, bh.Hash, bh.GroupId, model.Param.GetVerifierBonus())
	blog.log("generate bonus txHash=%v, targetIds=%v, height=%v", bonus.TxHash, bonus.TargetIds, bh.Height)

	tlog := newBlockTraceLog("REWARD_REQ", bh.Hash, p.GetMinerID())
	tlog.log("txHash=%v, targetIds=%v", bonus.TxHash.ShortS(), strings.Join(idHexs, ","))

	if slot.SetRewardTrans(tx) {
		msg := &model.CastRewardTransSignReqMessage{
			Reward: *bonus,
			SignedPieces: signs,
		}
		msg.GenSign(model.NewSecKeyInfo(p.GetMinerID(), p.getSignKey(groupID)), msg)
		p.NetServer.SendCastRewardSignReq(msg)
		blog.log("reward req send height=%v, gid=%v", bh.Height, groupID.ShortS())
	}

}
