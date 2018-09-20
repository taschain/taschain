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

	for _, msg := range futures {
		logStart("FUTURE_VERIFY", msg.BH.Height, msg.BH.QueueNumber, GetIDPrefix(msg.SI.SignMember), "size %v", len(futures))
		p.doVerify("FUTURE_VERIFY", msg, nil)
	}

}

func (p *Processor) onBlockAddSuccess(message notify.Message) {
	if !p.Ready() {
		return
	}
	block := message.GetData().(types.Block)
	preHeader := block.Header

	gid := *groupsig.DeserializeId(block.Header.GroupId)
	if p.IsMinerGroup(gid) {
		bc := p.GetBlockContext(gid)
		if bc == nil {
			panic("get blockContext nil")
		}
		bc.AddCastedHeight(block.Header.Height)
		_, vctx := bc.GetVerifyContextByHeight(block.Header.Height)
		if vctx != nil && vctx.prevBH.Hash == block.Header.PreHash {
			vctx.markCastSuccess()
		}
	}

	for {
		futureMsgs := p.getFutureBlockMsgs(preHeader.Hash)
		if futureMsgs == nil || len(futureMsgs) == 0 {
			break
		}
		log.Printf("handle future blocks, size=%v\n", len(futureMsgs))
		for _, msg := range futureMsgs {
			tbh := msg.Block.Header
			logHalfway("OMB", tbh.Height, tbh.QueueNumber, "", "trigger cached future block")
			p.receiveBlock(&msg.Block, preHeader)
		}
		p.removeFutureBlockMsgs(preHeader.Hash)
		preHeader = p.MainChain.QueryTopBlock()
	}
	p.triggerCastCheck()

	p.triggerFutureVerifyMsg(block.Header.Hash)
	p.groupManager.CreateNextGroupRoutine()
	p.cleanVerifyContext(preHeader.Height)


}

func (p *Processor) onGroupAddSuccess(message notify.Message) {
	group := message.GetData().(types.Group)
	sgi := NewSGIFromCoreGroup(&group)
	log.Printf("groupAddEventHandler receive message, groupId=%v\n", GetIDPrefix(sgi.GroupID))
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



