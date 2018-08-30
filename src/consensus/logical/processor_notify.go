package logical

import (
	"middleware/notify"
	"log"
	"middleware/types"
	"consensus/model"
)

func (p *Processor) onBlockAddSuccess(message notify.Message) {
	if !p.Ready() {
		return
	}
	block := message.GetData().(types.Block)
	preHeader := block.Header

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
	if !p.Ready() {
		return
	}
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



