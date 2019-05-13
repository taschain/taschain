package logical

import (
	"common"
	"consensus/groupsig"
	"consensus/model"
	"consensus/net"
	"middleware/types"
	"strings"
	"monitor"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2018/6/27 上午10:39
**  Description:
 */


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
		if !p.tryNotify(vctx) {
			blog.log("reserved, height=%v", vctx.castHeight)
		}
	}

	return
}

func (p *Processor) tryNotify(vctx *VerifyContext) bool {
	if sc := vctx.checkNotify(); sc != nil {
		bh := sc.BH
		tlog := newHashTraceLog("tryNotify", bh.Hash, p.GetMinerID())
		tlog.log("try broadcast, height=%v, totalQN=%v, 耗时%v秒", bh.Height, bh.TotalQN, p.ts.Since(bh.CurTime))

		p.consensusFinalize(vctx, sc) //上链和组外广播

		p.blockContexts.removeReservedVctx(vctx.castHeight)
		return true
	}
	return false
}

func (p *Processor) onBlockSignAggregation(hash common.Hash, sign groupsig.Signature, random groupsig.Signature) error {
	block := p.blockContexts.getProposed(hash)
	if block == nil {
		return fmt.Errorf("block is nil hash=%v", hash.ShortS())
	}
	block.Header.Signature = sign.Serialize()
	block.Header.Random = random.Serialize()

	r := p.doAddOnChain(block)

	if r != int8(types.AddBlockSucc) { //分叉调整或 上链失败都不走下面的逻辑
		return fmt.Errorf("onchain result %v", r)
	}

	bh := block.Header
	tlog := newHashTraceLog("onBlockSignAggregation", bh.Hash, p.GetMinerID())

	cbm := &model.ConsensusBlockMessage{
		Block: *block,
	}
	gb := p.spreadGroupBrief(bh, bh.Height+1)
	if gb == nil {
		return fmt.Errorf("next group is nil")
	}
	p.NetServer.BroadcastNewBlock(cbm, gb)
	tlog.log("broadcasted height=%v, 耗时%v秒", bh.Height, p.ts.Since(bh.CurTime))

	//发送日志
	le := &monitor.LogEntry{
		LogType:  monitor.LogTypeBlockBroadcast,
		Height:   bh.Height,
		Hash:     bh.Hash.Hex(),
		PreHash:  bh.PreHash.Hex(),
		Proposer: groupsig.DeserializeId(bh.Castor).GetHexString(),
		Verifier: gb.Gid.GetHexString(),
	}
	monitor.Instance.AddLog(le)
	p.blockContexts.removeProposed(hash)
	return nil
}

//在某个区块高度的QN值成功出块，保存上链，向组外广播
//同一个高度，可能会因QN不同而多次调用该函数
//但一旦低的QN出过，就不该出高的QN。即该函数可能被多次调用，但是调用的QN值越来越小
func (p *Processor) consensusFinalize(vctx *VerifyContext, slot *SlotContext) {
	bh := slot.BH

	blog := newBizLog("consensusFinalize—"+bh.Hash.ShortS())

	if p.blockOnChain(bh.Hash) { //已经上链
		blog.log("block alreayd onchain!")
		return
	}

	gpk := p.getGroupPubKey(groupsig.DeserializeId(bh.GroupId))
	if !slot.VerifyGroupSigns(gpk, vctx.prevBH.Random) { //组签名验证通过
		blog.log("group pub key local check failed, gpk=%v, hash in slot=%v, hash in bh=%v status=%v.",
			gpk.ShortS(), slot.BH.Hash.ShortS(), bh.Hash.ShortS(), slot.GetSlotStatus())
		return
	}

	//如果自己是提案者，则直接上链再广播
	if false {	//会导致提案者分布不均衡
		err := p.onBlockSignAggregation(bh.Hash, slot.gSignGenerator.GetGroupSign(), slot.rSignGenerator.GetGroupSign())
		if err != nil {
			blog.log("onBlockSignAggregation fail: %v", err)
			slot.setSlotStatus(slFailed)
			return
		}
	} else { //通知提案者
		aggrMsg := &model.BlockSignAggrMessage{
			Hash: bh.Hash,
			Sign: slot.gSignGenerator.GetGroupSign(),
			Random: slot.rSignGenerator.GetGroupSign(),
		}
		tlog := newHashTraceLog("consensusFinalize", bh.Hash, p.GetMinerID())
		tlog.log("send sign aggr msg to %v", slot.castor.ShortS())
		p.NetServer.SendBlockSignAggrMessage(aggrMsg, slot.castor)

	}
	slot.setSlotStatus(slSuccess)
	vctx.markNotified()
	vctx.successSlot = slot
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

	if height > 1 && p.proveChecker.proveExists(pi) {
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
		tlog.log("铸块成功, SendVerifiedCast, 时间间隔 %v, castor=%v, hash=%v, genHash=%v", bh.Elapsed, ccm.SI.GetID().ShortS(), bh.Hash.ShortS(), ccm.SI.DataHash.ShortS())

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

		p.blockContexts.addProposed(block)

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

	targetIdIndexs := make([]int32, 0)
	signs := make([]groupsig.Signature, 0)
	idHexs := make([]string, 0)

	threshold := model.Param.GetGroupK(group.GetMemberCount())
	for idx, mem := range group.GetMembers() {
		if sig, ok := slot.gSignGenerator.GetWitness(mem); ok {
			signs = append(signs, sig)
			targetIdIndexs = append(targetIdIndexs, int32(idx))
			idHexs = append(idHexs, mem.ShortS())
			if len(signs) >= threshold {
				break
			}
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
