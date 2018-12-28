package logical

import (
	"common"
	"consensus/groupsig"
	"consensus/model"
	"middleware/notify"
	"middleware/types"
)

func (p *Processor) triggerFutureVerifyMsg(hash common.Hash) {
	futures := p.getFutureVerifyMsgs(hash)
	if futures == nil || len(futures) == 0 {
		return
	}
	p.removeFutureVerifyMsgs(hash)
	mtype := "FUTURE_VERIFY"
	for _, msg := range futures {
		tlog := newHashTraceLog(mtype, msg.BH.Hash, msg.SI.GetID())
		tlog.logStart("size %v", len(futures))
		err := p.doVerify(mtype, msg, tlog, newBizLog(mtype))
		if err != nil {
			tlog.logEnd("result=%v", err.Error())
		}
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
		send, err := p.signCastRewardReq(msg.(*model.CastRewardTransSignReqMessage), bh)
		blog.log("send %v, result %v", send, err)
	}
}

//func (p *Processor) triggerFutureBlockMsg(preBH *types.BlockHeader) {
//	futureMsgs := p.getFutureBlockMsgs(preBH.Hash)
//	if futureMsgs == nil || len(futureMsgs) == 0 {
//		return
//	}
//	log.Printf("handle future blocks, size=%v\n", len(futureMsgs))
//	for _, msg := range futureMsgs {
//		tbh := msg.Block.Header
//		tlog := newHashTraceLog("OMB-FUTRUE", tbh.Hash, groupsig.DeserializeId(tbh.Castor))
//		tlog.log( "%v", "trigger cached future block")
//		p.receiveBlock(&msg.Block, preBH)
//	}
//	p.removeFutureBlockMsgs(preBH.Hash)
//}

func (p *Processor) onBlockAddSuccess(message notify.Message) {
	if !p.Ready() {
		return
	}
	block := message.GetData().(types.Block)
	bh := block.Header

	tlog := newMsgTraceLog("OnBlockAddSuccess", bh.Hash.ShortS(), "")
	tlog.log("preHash=%v, height=%v", bh.PreHash.ShortS(), bh.Height)

	gid := groupsig.DeserializeId(bh.GroupId)
	if p.IsMinerGroup(gid) {
		bc := p.GetBlockContext(gid)
		if bc == nil {
			panic("get blockContext nil")
		}
		bc.AddCastedHeight(bh.Height)
		vctx := bc.GetVerifyContextByHeight(bh.Height)
		if vctx != nil && vctx.prevBH.Hash == bh.PreHash {
			vctx.markBroadcast()
		}

	}

	vrf := p.getVrfWorker()
	if vrf != nil && vrf.baseBH.Hash == bh.PreHash && vrf.castHeight == bh.Height {
		vrf.markSuccess()
	}
	//p.triggerCastCheck()

	//p.triggerFutureBlockMsg(bh)
	p.triggerFutureVerifyMsg(bh.Hash)
	p.triggerFutureRewardSign(bh)
	p.groupManager.CreateNextGroupRoutine()

	p.cleanVerifyContext(bh.Height)
	notify.BUS.Publish(notify.BlockAddSuccConsensusUpdate, nil)
}

func (p *Processor) onGroupAddSuccess(message notify.Message) {
	group := message.GetData().(types.Group)
	stdLogger.Infof("groupAddEventHandler receive message, groupId=%v, workheight=%v\n", group.Id, group.Header.WorkHeight)
	if group.Id == nil || len(group.Id) == 0 {
		return
	}
	sgi := NewSGIFromCoreGroup(&group)
	p.acceptGroup(sgi)
	notify.BUS.Publish(notify.GroupAddSuccConsensusUpdate, nil)
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

func (p *Processor) onMissTxAddSucc(message notify.Message) {
	if !p.Ready() {
		return
	}
	tgam, ok := message.(*notify.TransactionGotAddSuccMessage)
	if !ok {
		stdLogger.Infof("minerTransactionHandler Message assert not ok!")
		return
	}
	transactions := tgam.Transactions
	var txHashes []common.Hash
	for _, tx := range transactions {
		txHashes = append(txHashes, tx.Hash)
	}
	p.OnMessageNewTransactions(txHashes)
}
