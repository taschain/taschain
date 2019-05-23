package logical

import (
	"consensus/groupsig"
	"consensus/model"
	"middleware/notify"
	"middleware/types"
	"bytes"
	"fmt"
	"taslog"
	"monitor"
)

func (p *Processor) triggerFutureVerifyMsg(bh *types.BlockHeader) {
	futures := p.getFutureVerifyMsgs(bh.Hash)
	if futures == nil || len(futures) == 0 {
		return
	}
	p.removeFutureVerifyMsgs(bh.Hash)
	mtype := "FUTURE_VERIFY"
	for _, msg := range futures {
		tlog := newHashTraceLog(mtype, msg.BH.Hash, msg.SI.GetID())
		tlog.logStart("size %v", len(futures))
		ok, err := p.verifyCastMessage(mtype, msg, bh)
		tlog.logEnd("result=%v %v", ok, err)
	}

}

func (p *Processor) triggerFutureRewardSign(bh *types.BlockHeader) {
	futures := p.futureRewardReqs.getMessages(bh.Hash)
	if futures == nil || len(futures) == 0 {
		return
	}
	p.futureRewardReqs.remove(bh.Hash)
	mtype := "CMCRSR-Future"
	for _, msg := range futures {
		blog := newBizLog(mtype)
		slog := taslog.NewSlowLog(mtype, 0.5)
		send, err := p.signCastRewardReq(msg.(*model.CastRewardTransSignReqMessage), bh, slog)
		blog.log("send %v, result %v", send, err)
	}
}


func (p *Processor) onBlockAddSuccess(message notify.Message) {
	if !p.Ready() {
		return
	}
	block := message.GetData().(*types.Block)
	bh := block.Header

	tlog := newMsgTraceLog("OnBlockAddSuccess", bh.Hash.ShortS(), "")
	tlog.log("preHash=%v, height=%v", bh.PreHash.ShortS(), bh.Height)

	gid := groupsig.DeserializeId(bh.GroupId)
	if p.IsMinerGroup(gid) {
		p.blockContexts.addCastedHeight(bh.Height, bh.PreHash)
		vctx := p.blockContexts.getVctxByHeight(bh.Height)
		if vctx != nil && vctx.prevBH.Hash == bh.PreHash {
			if vctx.isWorking() {
				vctx.markCastSuccess()
			}
			if !p.conf.GetBool("consensus", "league", false) {
				p.reqRewardTransSign(vctx, bh)
			}
		}
	}

	vrf := p.GetVrfWorker()
	if vrf != nil && vrf.baseBH.Hash == bh.PreHash && vrf.castHeight == bh.Height {
		vrf.markSuccess()
	}

	traceLog := monitor.NewPerformTraceLogger("OnBlockAddSuccess", bh.Hash, bh.Height)
	go p.checkSelfCastRoutine()

	traceLog.Log("start check proposal")

	//p.triggerFutureBlockMsg(bh)
	p.triggerFutureVerifyMsg(bh)
	p.triggerFutureRewardSign(bh)
	p.groupManager.CreateNextGroupRoutine()
	p.blockContexts.removeProposed(bh.Hash)
}

func (p *Processor) onGroupAddSuccess(message notify.Message) {
	group := message.GetData().(*types.Group)
	stdLogger.Infof("groupAddEventHandler receive message, groupId=%v, workheight=%v\n", groupsig.DeserializeId(group.Id).GetHexString(), group.Header.WorkHeight)
	if group.Id == nil || len(group.Id) == 0 {
		return
	}
	sgi := NewSGIFromCoreGroup(group)
	p.acceptGroup(sgi)

	p.groupManager.onGroupAddSuccess(sgi)
	p.joiningGroups.Clean(sgi.GInfo.GroupHash())
	p.globalGroups.removeInitedGroup(sgi.GInfo.GroupHash())

	beginHeight := group.Header.WorkHeight
	topHeight := p.MainChain.Height()

	//当前块高已经超过生效高度了,组可能有点问题
	if beginHeight > 0 && beginHeight <= topHeight {
		stdLogger.Errorf("group add after can work! gid=%v, gheight=%v, beginHeight=%v, currentHeight=%v", sgi.GroupID.ShortS(), group.GroupHeight, beginHeight, topHeight)
		pre := p.MainChain.QueryBlockHeaderFloor(beginHeight-1)
		if pre == nil {
			panic(fmt.Sprintf("block nil at height %v", beginHeight-1))
		}
		for h := beginHeight; h <= topHeight; {
			bh := p.MainChain.QueryBlockHeaderCeil(h)
			if bh == nil {
				break
			}
			if bh.PreHash != pre.Hash {
				panic(fmt.Sprintf("pre error:bh %v, prehash %v, height %v, real pre hash %v height %v", bh.Hash.String(), bh.PreHash.String(), bh.Height, pre.Hash.String(), pre.Height))
			}
			gid := p.CalcVerifyGroupFromChain(pre, bh.Height)
			if !bytes.Equal(gid.Serialize(), bh.GroupId) {
				old := p.MainChain.QueryTopBlock()
				stdLogger.Errorf("adjust top block: old %v %v %v, new %v %v %v", old.Hash.String(), old.PreHash.String(), old.Height, pre.Hash.String(), pre.PreHash.String(), pre.Height)
				p.MainChain.ResetTop(pre)
				break
			}
			pre = bh
			h = bh.Height+1
		}
	}
}

func (p *Processor) onNewBlockReceive(message notify.Message) {
	if !p.Ready() {
		return
	}
	msg := &model.ConsensusBlockMessage{
		Block: message.GetData().(types.Block),
	}
	p.OnMessageBlock(msg)
}

