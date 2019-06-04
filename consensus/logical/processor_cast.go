package logical

import (
	"fmt"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/consensus/net"
	"github.com/taschain/taschain/middleware/types"
	"github.com/taschain/taschain/monitor"
	"strings"
)

// triggerCastCheck trigger once to check if you are next ingot group
func (p *Processor) triggerCastCheck() {
	p.Ticker.StartAndTriggerRoutine(p.getCastCheckRoutineName())
}

func (p *Processor) CalcVerifyGroupFromCache(preBH *types.BlockHeader, height uint64) *groupsig.ID {
	var hash = calcRandomHash(preBH, height)

	selectGroup, err := p.globalGroups.SelectNextGroupFromCache(hash, height)
	if err != nil {
		stdLogger.Errorf("SelectNextGroupFromCache height=%v, err: %v", height, err)
		return nil
	}
	return &selectGroup
}

func (p *Processor) CalcVerifyGroupFromChain(preBH *types.BlockHeader, height uint64) *groupsig.ID {
	var hash = calcRandomHash(preBH, height)

	selectGroup, err := p.globalGroups.SelectNextGroupFromChain(hash, height)
	if err != nil {
		stdLogger.Errorf("SelectNextGroupFromChain height=%v, err:%v", height, err)
		return nil
	}
	return &selectGroup
}

func (p *Processor) spreadGroupBrief(bh *types.BlockHeader, height uint64) *net.GroupBrief {
	nextID := p.CalcVerifyGroupFromCache(bh, height)
	if nextID == nil {
		return nil
	}
	group := p.GetGroup(*nextID)
	g := &net.GroupBrief{
		Gid:    *nextID,
		MemIds: group.GetMembers(),
	}
	return g
}

func (p *Processor) reserveBlock(vctx *VerifyContext, slot *SlotContext) {
	bh := slot.BH
	blog := newBizLog("reserveBLock")
	blog.log("height=%v, totalQN=%v, hash=%v, slotStatus=%v", bh.Height, bh.TotalQN, bh.Hash.ShortS(), slot.GetSlotStatus())
	if slot.IsRecovered() {
		// The onBlockAddSuccess method is also marked, the call is asynchronous
		vctx.markCastSuccess()
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

		// Add on chain and out-of-group broadcasting
		p.consensusFinalize(vctx, sc)

		p.blockContexts.removeReservedVctx(vctx.castHeight)
		return true
	}
	return false
}

func (p *Processor) onBlockSignAggregation(block *types.Block, sign groupsig.Signature, random groupsig.Signature) error {

	if block == nil {
		return fmt.Errorf("block is nil")
	}
	block.Header.Signature = sign.Serialize()
	block.Header.Random = random.Serialize()

	r := p.doAddOnChain(block)

	// Fork adjustment or add on chain failure does not take the logic below
	if r != int8(types.AddBlockSucc) {
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

	// Send log
	le := &monitor.LogEntry{
		LogType:  monitor.LogTypeBlockBroadcast,
		Height:   bh.Height,
		Hash:     bh.Hash.Hex(),
		PreHash:  bh.PreHash.Hex(),
		Proposer: groupsig.DeserializeID(bh.Castor).GetHexString(),
		Verifier: gb.Gid.GetHexString(),
	}
	monitor.Instance.AddLog(le)
	return nil
}

// consensusFinalize means The QN value at a certain block height is successfully popped,
// saved in the uplink, and broadcast to the outside of the group.
// The same height, may call this function multiple times due to different QN
// But once the low QN has passed, it should not be a high QN. That is, the function may be
// called multiple times, but the QN value of the call is getting smaller and smaller.
func (p *Processor) consensusFinalize(vctx *VerifyContext, slot *SlotContext) {
	bh := slot.BH

	blog := newBizLog("consensusFinalize—" + bh.Hash.ShortS())

	if p.blockOnChain(bh.Hash) { //已经上链
		blog.log("block alreayd onchain!")
		return
	}

	gpk := p.getGroupPubKey(groupsig.DeserializeID(bh.GroupID))
	if !slot.VerifyGroupSigns(gpk, vctx.prevBH.Random) { //组签名验证通过
		blog.log("group pub key local check failed, gpk=%v, hash in slot=%v, hash in bh=%v status=%v.",
			gpk.ShortS(), slot.BH.Hash.ShortS(), bh.Hash.ShortS(), slot.GetSlotStatus())
		return
	}

	// Ask the proposer for a complete block
	msg := &model.ReqProposalBlock{
		Hash: bh.Hash,
	}
	tlog := newHashTraceLog("consensusFinalize", bh.Hash, p.GetMinerID())
	tlog.log("send ReqProposalBlock msg to %v", slot.castor.ShortS())
	p.NetServer.ReqProposalBlock(msg, slot.castor.GetHexString())

	slot.setSlotStatus(slSuccess)
	vctx.markNotified()
	vctx.successSlot = slot
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
		// Here you need to use a normal private key, a non-group related private key.
		skey := p.mi.SK

		ccm := &model.ConsensusCastMessage{
			BH: *bh,
		}
		// The message hash sent to everyone is the same, the signature is the same
		if !ccm.GenSign(model.NewSecKeyInfo(p.GetMinerID(), skey), ccm) {
			blog.log("sign fail, id=%v, sk=%v", p.GetMinerID().ShortS(), skey.ShortS())
			return
		}
		// Generate full account book hash
		proveHashs := p.proveChecker.genProveHashs(height, worker.getBaseBH().Random, gb.MemIds)
		p.NetServer.SendCastVerify(ccm, gb, proveHashs)

		// ccm.GenRandomSign(skey, worker.baseBH.Random)
		// Castor cannot sign random numbers
		tlog.log("铸块成功, SendVerifiedCast, 时间间隔 %v, castor=%v, hash=%v, genHash=%v", bh.Elapsed, ccm.SI.GetID().ShortS(), bh.Hash.ShortS(), ccm.SI.DataHash.ShortS())

		// Send log
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

// reqRewardTransSign sign the reward transaction within the request group
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
	// If you sign yourself, you don’t have to send it again
	if slot.hasSignedRewardTx() {
		blog.log("has signed reward tx")
		return
	}

	groupID := groupsig.DeserializeID(bh.GroupID)
	group := p.GetGroup(groupID)

	targetIDIndexs := make([]int32, 0)
	signs := make([]groupsig.Signature, 0)
	idHexs := make([]string, 0)

	threshold := model.Param.GetGroupK(group.GetMemberCount())
	for idx, mem := range group.GetMembers() {
		if sig, ok := slot.gSignGenerator.GetWitness(mem); ok {
			signs = append(signs, sig)
			targetIDIndexs = append(targetIDIndexs, int32(idx))
			idHexs = append(idHexs, mem.ShortS())
			if len(signs) >= threshold {
				break
			}
		}
	}

	bonus, tx := p.MainChain.GetBonusManager().GenerateBonus(targetIDIndexs, bh.Hash, bh.GroupID, model.Param.VerifyBonus)
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
