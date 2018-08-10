package logical

import (
			"consensus/model"
	"log"
	"time"
	"consensus/logical/pow"
	"middleware/types"
)

/*
**  Creator: pxf
**  Date: 2018/8/9 下午4:06
**  Description: 
*/

//出块完成后，启动pow计算
func (p *Processor) startPowComputation(bh *types.BlockHeader, gminer *model.GroupMinerID)  {
	group := p.getGroup(gminer.Gid)
	if group == nil {
		return
	}
	p.worker.Prepare(bh, gminer, model.Param.GetGroupK(len(group.Members)))
	log.Printf("startPowComputation bh=%v\n", p.blockPreview(bh))
	logStart("POW_COMP", p.worker.BH.Height, p.worker.BH.QueueNumber, p.getPrefix(), "", "")
	p.worker.Start()
}


func (p *Processor) powWorkerLoop()  {
    for {
		select {
		case cmd := <- p.worker.CmdCh:
			switch cmd {
			case pow.CMD_POW_RESULT:
				p.onPowResult()
			case pow.CMD_POW_CONFIRM:
				p.onPowConfirm()
			}
		}
	}
}

//算出pow结果，发送给其他组员
func (p *Processor) onPowResult()  {
	if !p.worker.Success() {
		return
	}

	msg := &model.ConsensusPowResultMessage{
		BlockHash: p.worker.BH.Hash,
		Nonce:     p.worker.Nonce,
		GroupID:   p.worker.GroupMiner.Gid,
	}

	msg.GenSign(model.NewSecKeyInfo(p.GetMinerID(), p.getMinerGroupSignKey(p.worker.GroupMiner.Gid)), msg)

	logHalfway("POW_Result", p.worker.BH.Height, p.worker.BH.QueueNumber, p.getPrefix(), "nonce %v, cost %v", p.worker.Nonce, time.Since(p.worker.StartTime).String())

	log.Printf("send ConsensusPowResultMessage ...")
	p.NetServer.SendPowResultMessage(msg)
}

func (p *Processor) onPowConfirm()  {
	if !p.worker.Confirmed() {
		return
	}

	msg := &model.ConsensusPowConfirmMessage{
		GroupID: p.worker.GroupMiner.Gid,
		BlockHash: p.worker.BH.Hash,
		NonceSeq: p.worker.GetNonceSeq(),
	}
	msg.GenSign(model.NewSecKeyInfo(p.GetMinerID(), p.getMinerGroupSignKey(p.worker.GroupMiner.Gid)), msg)
	logHalfway("POW_Confirm", p.worker.BH.Height, p.worker.BH.QueueNumber, p.getPrefix(), "nonce %v, cost %v", p.worker.Nonce, time.Since(p.worker.StartTime).String())

	log.Printf("send ConsensusPowConfirmMessage ...")
	p.NetServer.SendPowConfirmMessage(msg)
}