package logical

import (
	"middleware/notify"
	"log"
	"middleware/types"
	"consensus/model"
	"consensus/groupsig"
	"common"
)

func (p *Processor) triggerFutureVerifyMsg(hash common.Hash) {
	futures := p.getFutureVerifyMsgs(hash)
	if futures == nil || len(futures) == 0 {
		return
	}
	p.removeFutureVerifyMsgs(hash)
	mtype := "FUTURE_VERIFY"
	for _, msg := range futures {
		tlog := newBlockTraceLog(mtype, msg.BH.Hash, msg.SI.GetID())
		tlog.logStart("size %v", len(futures))
		p.doVerify(mtype, msg, nil, tlog, newBizLog(mtype))
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
		p.signCastRewardReq(msg.(*model.CastRewardTransSignReqMessage), bh, blog)
	}
}

func (p *Processor) triggerFutureBlockMsg(preBH *types.BlockHeader) {
	futureMsgs := p.getFutureBlockMsgs(preBH.Hash)
	if futureMsgs == nil || len(futureMsgs) == 0 {
		return
	}
	log.Printf("handle future blocks, size=%v\n", len(futureMsgs))
	for _, msg := range futureMsgs {
		tbh := msg.Block.Header
		tlog := newBlockTraceLog("OMB-FUTRUE", tbh.Hash, groupsig.DeserializeId(tbh.Castor))
		tlog.log( "%v", "trigger cached future block")
		p.receiveBlock(&msg.Block, preBH)
	}
	p.removeFutureBlockMsgs(preBH.Hash)
}

func (p *Processor) onBlockAddSuccess(message notify.Message) {
	if !p.Ready() {
		return
	}
	block := message.GetData().(types.Block)
	bh := block.Header

	gid := groupsig.DeserializeId(bh.GroupId)
	if p.IsMinerGroup(gid) {
		bc := p.GetBlockContext(gid)
		if bc == nil {
			panic("get blockContext nil")
		}
		bc.AddCastedHeight(bh.Height)
		vctx := bc.GetVerifyContextByHeight(bh.Height)
		if vctx != nil && vctx.prevBH.Hash == bh.PreHash {
			vctx.markCastSuccess()
		}

	}

	vrf := p.getVrfWorker()
	if vrf != nil && vrf.baseBH.Hash == bh.PreHash && vrf.castHeight == bh.Height {
		vrf.markSuccess()
	}
	p.triggerCastCheck()

	p.triggerFutureBlockMsg(bh)
	p.triggerFutureVerifyMsg(bh.Hash)
	p.triggerFutureRewardSign(bh)
	p.groupManager.CreateNextGroupRoutine()

	p.cleanVerifyContext(bh.Height)
}

func (p *Processor) onGroupAddSuccess(message notify.Message) {
	group := message.GetData().(types.Group)
	if group.Id == nil || len(group.Id) == 0 {
		return
	}
	sgi := NewSGIFromCoreGroup(&group)
	log.Printf("groupAddEventHandler receive message, groupId=%v\n", sgi.GroupID.ShortS())
	p.acceptGroup(sgi)
}

func (p *Processor) onNewBlockReceive(message notify.Message)  {
	if !p.Ready() {
		return
	}
    msg := &model.ConsensusBlockMessage{
    	Block: message.GetData().(types.Block),
	}
    p.OnMessageBlock(msg)
}



