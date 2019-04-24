package logical

import (
	"common"
	"consensus/groupsig"
	"consensus/model"
	"consensus/net"
	"middleware/types"
	"strings"
	"sync"
	"runtime/debug"
	"monitor"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2018/6/27 上午10:39
**  Description:
 */

type CastBlockContexts struct {
	blockCtxs    sync.Map //string -> *BlockContext
	reservedVctx sync.Map //uint64 -> *VerifyContext 存储已经有签出块的verifyContext，待广播
}

func NewCastBlockContexts() *CastBlockContexts {
	return &CastBlockContexts{
		blockCtxs: sync.Map{},
	}
}

func (bctx *CastBlockContexts) addBlockContext(bc *BlockContext) (add bool) {
	_, load := bctx.blockCtxs.LoadOrStore(bc.MinerID.Gid.GetHexString(), bc)
	return !load
}

func (bctx *CastBlockContexts) getBlockContext(gid groupsig.ID) *BlockContext {
	if v, ok := bctx.blockCtxs.Load(gid.GetHexString()); ok {
		return v.(*BlockContext)
	}
	return nil
}

func (bctx *CastBlockContexts) blockContextSize() int32 {
	size := int32(0)
	bctx.blockCtxs.Range(func(key, value interface{}) bool {
		size++
		return true
	})
	return size
}

func (bctx *CastBlockContexts) removeBlockContexts(gids []groupsig.ID) {
	for _, id := range gids {
		stdLogger.Infof("removeBlockContexts %v", id.ShortS())
		bc := bctx.getBlockContext(id)
		if bc != nil {
			//bc.removeTicker()
			for _, vctx := range bc.SafeGetVerifyContexts() {
				bctx.removeReservedVctx(vctx.castHeight)
			}
			bctx.blockCtxs.Delete(id.GetHexString())
		}
	}
}

func (bctx *CastBlockContexts) forEachBlockContext(f func(bc *BlockContext) bool) {
	bctx.blockCtxs.Range(func(key, value interface{}) bool {
		v := value.(*BlockContext)
		return f(v)
	})
}

func (bctx *CastBlockContexts) removeReservedVctx(height uint64) {
	bctx.reservedVctx.Delete(height)
}

func (bctx *CastBlockContexts) addReservedVctx(vctx *VerifyContext) bool {
	_, load := bctx.reservedVctx.LoadOrStore(vctx.castHeight, vctx)
	return !load
}

func (bctx *CastBlockContexts) forEachReservedVctx(f func(vctx *VerifyContext) bool) {
	bctx.reservedVctx.Range(func(key, value interface{}) bool {
		v := value.(*VerifyContext)
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

//立即触发一次检查自己是否下个铸块组
func (p *Processor) triggerCastCheck() {
	//p.Ticker.StartTickerRoutine(p.getCastCheckRoutineName(), true)
	p.Ticker.StartAndTriggerRoutine(p.getCastCheckRoutineName())
}

func (p *Processor) CalcVerifyGroupFromCache(preBH *types.BlockHeader, height uint64) (*groupsig.ID) {
	var hash = CalcRandomHash(preBH, height)

	selectGroup, err := p.globalGroups.SelectNextGroupFromCache(hash, height)
	if err != nil {
		stdLogger.Errorf("SelectNextGroupFromCache height=%v, err: %v", height, err)
		return nil
	}
	return &selectGroup
}

func (p *Processor) CalcVerifyGroupFromChain(preBH *types.BlockHeader, height uint64) (*groupsig.ID) {
	var hash = CalcRandomHash(preBH, height)

	selectGroup, err := p.globalGroups.SelectNextGroupFromChain(hash, height)
	if err != nil {
		stdLogger.Errorf("SelectNextGroupFromChain height=%v, err:%v", height, err)
		return nil
	}
	return &selectGroup
}

func (p *Processor) spreadGroupBrief(bh *types.BlockHeader, height uint64) *net.GroupBrief {
	nextId := p.CalcVerifyGroupFromCache(bh, height)
	if nextId == nil {
		return nil
	}
	group := p.GetGroup(*nextId)
	g := &net.GroupBrief{
		Gid:    *nextId,
		MemIds: group.GetMembers(),
	}
	return g
}

func (p *Processor) reserveBlock(vctx *VerifyContext, slot *SlotContext) {
	bh := slot.BH
	blog := newBizLog("reserveBLock")
	blog.log("height=%v, totalQN=%v, hash=%v, slotStatus=%v", bh.Height, bh.TotalQN, bh.Hash.ShortS(), slot.GetSlotStatus())
	if slot.IsRecovered() {
		vctx.markCastSuccess() //onBlockAddSuccess方法中也mark了，该处调用是异步的
		p.blockContexts.addReservedVctx(vctx)
		if !p.tryBroadcastBlock(vctx) {
			blog.log("reserved, height=%v", vctx.castHeight)
		}

	}

	return
}

func (p *Processor) tryBroadcastBlock(vctx *VerifyContext) bool {
	if sc := vctx.checkBroadcast(); sc != nil {
		bh := sc.BH
		tlog := newHashTraceLog("tryBroadcastBlock", bh.Hash, p.GetMinerID())
		tlog.log("try broadcast, height=%v, totalQN=%v, 耗时%v秒", bh.Height, bh.TotalQN, p.ts.Since(bh.CurTime).Seconds())

		//异步进行，使得请求快速返回，防止消息积压
		go p.successNewBlock(vctx, sc) //上链和组外广播

		p.blockContexts.removeReservedVctx(vctx.castHeight)
		return true
	}
	return false
}

//在某个区块高度的QN值成功出块，保存上链，向组外广播
//同一个高度，可能会因QN不同而多次调用该函数
//但一旦低的QN出过，就不该出高的QN。即该函数可能被多次调用，但是调用的QN值越来越小
func (p *Processor) successNewBlock(vctx *VerifyContext, slot *SlotContext) {
	defer func() {
		if r := recover(); r != nil {
			common.DefaultLogger.Errorf("error：%v\n", r)
			s := debug.Stack()
			common.DefaultLogger.Errorf(string(s))
		}
	}()

	bh := slot.BH

	blog := newBizLog("successNewBlock—"+bh.Hash.ShortS())

	if slot.IsFailed() {
		blog.log("slot is failed")
		return
	}
	if vctx.broadCasted() {
		blog.log("block broadCasted!")
		return
	}

	if p.blockOnChain(bh.Hash) { //已经上链
		blog.log("block alreayd onchain!")
		return
	}

	block := p.MainChain.GenerateBlock(*bh)

	if block == nil {
		blog.log("core.GenerateBlock is nil! won't broadcast block! height=%v", bh.Height)
		return
	}
	gb := p.spreadGroupBrief(bh, bh.Height+1)
	if gb == nil {
		blog.log("spreadGroupBrief nil, bh=%v, height=%v", bh.Hash.ShortS(), bh.Height)
		return
	}

	gpk := p.getGroupPubKey(groupsig.DeserializeId(bh.GroupId))
	if !slot.VerifyGroupSigns(gpk, vctx.prevBH.Random) { //组签名验证通过
		blog.log("group pub key local check failed, gpk=%v, hash in slot=%v, hash in bh=%v status=%v.",
			gpk.ShortS(), slot.BH.Hash.ShortS(), bh.Hash.ShortS(), slot.GetSlotStatus())
		return
	}

	r := p.doAddOnChain(block)

	if r != int8(types.AddBlockSucc) { //分叉调整或 上链失败都不走下面的逻辑
		if r != int8(types.Forking) {
			slot.setSlotStatus(SS_FAILED)
		}
		return
	}

	tlog := newHashTraceLog("successNewBlock", bh.Hash, p.GetMinerID())

	cbm := &model.ConsensusBlockMessage{
		Block: *block,
	}

	p.NetServer.BroadcastNewBlock(cbm, gb)
	tlog.log("broadcasted height=%v, 耗时%v秒", bh.Height, p.ts.Since(bh.CurTime).Seconds())

	//发送日志
	le := &monitor.LogEntry{
		LogType:  monitor.LogTypeBlockBroadcast,
		Height:   bh.Height,
		Hash:     bh.Hash.Hex(),
		PreHash:  bh.PreHash.Hex(),
		Proposer: slot.castor.GetHexString(),
		Verifier: gb.Gid.GetHexString(),
	}
	monitor.Instance.AddLog(le)

	vctx.broadcastSlot = slot
	vctx.markBroadcast()
	slot.setSlotStatus(SS_SUCCESS)

	//如果是联盟链，则不打分红交易
	if !consensusConfManager.GetBool("league", false) {
		p.reqRewardTransSign(vctx, bh)
	}
	blog.log("After BroadcastNewBlock hash=%v:%v", bh.Hash.ShortS(), p.ts.Now().Format(TIMESTAMP_LAYOUT))
	return
}

func (p *Processor) blockProposal() {
	blog := newBizLog("blockProposal")
	top := p.MainChain.QueryTopBlock()
	worker := p.GetVrfWorker()
	if worker.getBaseBH().Hash != top.Hash {
		blog.log("vrf baseBH differ from top!")
		return
	}
	if worker.isProposed() || worker.isSuccess() {
		blog.log("vrf worker proposed/success, status %v", worker.getStatus())
		return
	}
	height := worker.castHeight

	if !p.ts.NowAfter(worker.baseBH.CurTime) {
		blog.log("not the time!now=%v, pre=%v, height=%v", p.ts.Now(), worker.baseBH.CurTime, height)
		return
	}

	totalStake := p.minerReader.getTotalStake(worker.baseBH.Height, false)
	blog.log("totalStake height=%v, stake=%v", height, totalStake)
	pi, qn, err := worker.Prove(totalStake)
	if err != nil {
		blog.log("vrf prove not ok! %v", err)
		return
	}

	if p.proveChecker.proveExists(pi) {
		blog.log("vrf prove exist, not proposal")
		return
	}

	if worker.timeout() {
		blog.log("vrf worker timeout")
		return
	}

	gb := p.spreadGroupBrief(top, height)
	if gb == nil {
		blog.log("spreadGroupBrief nil, bh=%v, height=%v", top.Hash.ShortS(), height)
		return
	}
	gid := gb.Gid

	block := p.MainChain.CastBlock(uint64(height), pi, qn, p.GetMinerID().Serialize(), gid.Serialize())
	if block == nil {
		blog.log("MainChain::CastingBlock failed, height=%v", height)
		return
	}
	bh := block.Header
	tlog := newHashTraceLog("CASTBLOCK", bh.Hash, p.GetMinerID())
	blog.log("begin proposal, hash=%v, height=%v, qn=%v,, verifyGroup=%v, pi=%v...", bh.Hash.ShortS(), height, qn, gid.ShortS(), pi.ShortS())
	tlog.logStart("height=%v,qn=%v, preHash=%v, verifyGroup=%v", bh.Height, qn, bh.PreHash.ShortS(), gid.ShortS())

	if bh.Height > 0 && bh.Height == height && bh.PreHash == worker.baseBH.Hash {
		skey := p.mi.SK //此处需要用普通私钥，非组相关私钥

		ccm := &model.ConsensusCastMessage{
			BH: *bh,
		}
		//发给每个人的消息hash是相同的，签名也相同
		if !ccm.GenSign(model.NewSecKeyInfo(p.GetMinerID(), skey), ccm) {
			blog.log("sign fail, id=%v, sk=%v", p.GetMinerID().ShortS(), skey.ShortS())
			return
		}
		//生成全量账本hash
		proveHashs := p.proveChecker.genProveHashs(height, worker.getBaseBH().Random, gb.MemIds)
		p.NetServer.SendCastVerify(ccm, gb, proveHashs)

		//ccm.GenRandomSign(skey, worker.baseBH.Random)//castor不能对随机数签名
		tlog.log("铸块成功, SendVerifiedCast, 时间间隔 %v, castor=%v, hash=%v, genHash=%v", bh.CurTime.Sub(bh.PreTime).Seconds(), ccm.SI.GetID().ShortS(), bh.Hash.ShortS(), ccm.SI.DataHash.ShortS())

		//发送日志
		le := &monitor.LogEntry{
			LogType:  monitor.LogTypeProposal,
			Height:   bh.Height,
			Hash:     bh.Hash.Hex(),
			PreHash:  bh.PreHash.Hex(),
			Proposer: p.GetMinerID().GetHexString(),
			Verifier: gb.Gid.GetHexString(),
			Ext:      fmt.Sprintf("qn:%v,totalQN:%v", qn, bh.TotalQN),
		}
		monitor.Instance.AddLog(le)
		p.proveChecker.addProve(pi)
		worker.markProposed()

	} else {
		blog.log("bh/prehash Error or sign Error, bh=%v, real height=%v. bc.prehash=%v, bh.prehash=%v", height, bh.Height, worker.baseBH.Hash, bh.PreHash)
	}

}

//请求组内对奖励交易签名
func (p *Processor) reqRewardTransSign(vctx *VerifyContext, bh *types.BlockHeader) {
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
	//签过了自己就不用再发了
	if slot.hasSignedRewardTx() {
		blog.log("has signed reward tx")
		return
	}

	groupID := groupsig.DeserializeId(bh.GroupId)
	group := p.GetGroup(groupID)

	size := slot.gSignGenerator.WitnessSize()
	targetIdIndexs := make([]int32, size)
	signs := make([]groupsig.Signature, size)
	idHexs := make([]string, size)

	i := 0
	for idx, mem := range group.GetMembers() {
		if sig, ok := slot.gSignGenerator.GetWitness(mem); ok {
			signs[i] = sig
			targetIdIndexs[i] = int32(idx)
			idHexs[i] = mem.ShortS()
			i++
		}
	}


	bonus, tx := p.MainChain.GetBonusManager().GenerateBonus(targetIdIndexs, bh.Hash, bh.GroupId, model.Param.VerifyBonus)
	blog.debug("generate bonus txHash=%v, targetIds=%v, height=%v", bonus.TxHash.ShortS(), bonus.TargetIds, bh.Height)

	tlog := newHashTraceLog("REWARD_REQ", bh.Hash, p.GetMinerID())
	tlog.log("txHash=%v, targetIds=%v", bonus.TxHash.ShortS(), strings.Join(idHexs, ","))

	if slot.SetRewardTrans(tx) {
		msg := &model.CastRewardTransSignReqMessage{
			Reward:       *bonus,
			SignedPieces: signs,
		}
		ski := model.NewSecKeyInfo(p.GetMinerID(), p.getSignKey(groupID))
		if msg.GenSign(ski, msg) {
			p.NetServer.SendCastRewardSignReq(msg)
			blog.log("reward req send height=%v, gid=%v", bh.Height, groupID.ShortS())
		} else {
			blog.debug("genSign fail, id=%v, sk=%v, belong=%v", ski.ID.ShortS(), ski.SK.ShortS(), p.IsMinerGroup(groupID))
		}
	}

}
