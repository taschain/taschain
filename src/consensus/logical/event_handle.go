package logical

import (
	"middleware/notify"
	"log"
	"middleware/types"
)

type blockAddEventHandler struct {
	p *Processor
}

func (h *blockAddEventHandler) Handle(message notify.Message) {

	block := message.GetData().(types.Block)
	topBH := h.p.MainChain.QueryTopBlock()
	preHeader := block.Header

	for {
		futureMsgs := h.p.getFutureBlockMsgs(preHeader.Hash)
		if futureMsgs == nil || len(futureMsgs) == 0 {
			break
		}
		log.Printf("handle future blocks, size=%v\n", len(futureMsgs))
		for _, msg := range futureMsgs {
			tbh := msg.Block.Header
			logHalfway("OMB", tbh.Height, tbh.QueueNumber, GetIDPrefix(msg.SI.SignMember), "trigger cached future block")
			h.p.receiveBlock(msg, preHeader)
		}
		h.p.removeFutureBlockMsgs(preHeader.Hash)
		preHeader = h.p.MainChain.QueryTopBlock()
	}

	nowTop := h.p.MainChain.QueryTopBlock()
	if topBH.Hash != nowTop.Hash {
		h.p.triggerCastCheck()
	}

}
