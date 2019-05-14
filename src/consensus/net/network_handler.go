package net

import (
	"network"
	"common"
	"log"
	"fmt"
	"runtime/debug"
)

type ConsensusHandler struct {
	processor MessageProcessor
}

var MessageHandler = new(ConsensusHandler)

func (c *ConsensusHandler) Init(proc MessageProcessor) {
	c.processor = proc
	InitStateMachines()
}

func (c *ConsensusHandler) Processor() MessageProcessor {
	return c.processor
}

func (c *ConsensusHandler) ready() bool {
	return c.processor != nil && c.processor.Ready()
}

func (c *ConsensusHandler) Handle(sourceId string, msg network.Message) error {
	code := msg.Code
	body := msg.Body

	defer func() {
		if r := recover(); r != nil {
			common.DefaultLogger.Errorf("error：%v\n", r)
			s := debug.Stack()
			common.DefaultLogger.Errorf(string(s))
		}
	}()

	if !c.ready() {
		log.Printf("message ingored because processor not ready. code=%v\n", code)
		return fmt.Errorf("processor not ready yet")
	}
	switch code {
	case network.GroupInitMsg:
		m, e := unMarshalConsensusGroupRawMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusGroupRawMessage because of unmarshal error:%s", e.Error())
			return e
		}

		//belongGroup := m.GInfo.MemberExists(c.processor.GetMinerID())

		//var machines *StateMachines
		//if belongGroup {
		//	machines = &GroupInsideMachines
		//} else {
		//	machines = &GroupOutsideMachines
		//}
		GroupInsideMachines.GetMachine(m.GInfo.GI.GetHash().Hex(), len(m.GInfo.Mems)).Transform(NewStateMsg(code, m, sourceId))
	case network.KeyPieceMsg:
		m, e := unMarshalConsensusSharePieceMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusSharePieceMessage because of unmarshal error:%s", e.Error())
			return e
		}
		GroupInsideMachines.GetMachine(m.GHash.Hex(), int(m.MemCnt)).Transform(NewStateMsg(code, m, sourceId))
		logger.Infof("SharepieceMsg receive from:%v, gHash:%v", sourceId, m.GHash.Hex())
	case network.SignPubkeyMsg:
		m, e := unMarshalConsensusSignPubKeyMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusSignPubKeyMessage because of unmarshal error:%s", e.Error())
			return e
		}
		GroupInsideMachines.GetMachine(m.GHash.Hex(), int(m.MemCnt)).Transform(NewStateMsg(code, m, sourceId))
		logger.Infof("SignPubKeyMsg receive from:%v, gHash:%v, groupId:%v", sourceId, m.GHash.Hex(), m.GroupID.GetHexString())
	case network.GroupInitDoneMsg:
		m, e := unMarshalConsensusGroupInitedMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusGroupInitedMessage because of unmarshal error%s", e.Error())
			return e
		}
		logger.Infof("Rcv GroupInitDoneMsg from:%s,gHash:%s, groupId:%v", sourceId, m.GHash.Hex(), m.GroupID.GetHexString())

		//belongGroup := c.processor.ExistInGroup(m.GHash)
		//var machines *StateMachines
		//if belongGroup {
		//	machines = &GroupInsideMachines
		//} else {
		//	machines = &GroupOutsideMachines
		//}
		GroupInsideMachines.GetMachine(m.GHash.Hex(), int(m.MemCnt)).Transform(NewStateMsg(code, m, sourceId))

	case network.CurrentGroupCastMsg:

	case network.CastVerifyMsg:
		m, e := unMarshalConsensusCastMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusCastMessage because of unmarshal error%s", e.Error())
			return e
		}
		c.processor.OnMessageCast(m)
	case network.VerifiedCastMsg:
		m, e := unMarshalConsensusVerifyMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusVerifyMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.processor.OnMessageVerify(m)

	case network.CreateGroupaRaw:
		m, e := unMarshalConsensusCreateGroupRawMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusCreateGroupRawMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.processor.OnMessageCreateGroupRaw(m)
		return nil
	case network.CreateGroupSign:
		m, e := unMarshalConsensusCreateGroupSignMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusCreateGroupSignMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.processor.OnMessageCreateGroupSign(m)
		return nil
	case network.CastRewardSignReq:
		m, e := unMarshalCastRewardReqMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard CastRewardSignReqMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.processor.OnMessageCastRewardSignReq(m)
	case network.CastRewardSignGot:
		m, e := unMarshalCastRewardSignMessage(body)
		if e != nil {
			network.Logger.Errorf("[handler]Discard CastRewardSignMessage because of unmarshal error%s", e.Error())
			return e
		}

		c.processor.OnMessageCastRewardSign(m)
	case network.AskSignPkMsg:
		m, e := unMarshalConsensusSignPKReqMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard unMarshalConsensusSignPKReqMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.processor.OnMessageSignPKReq(m)
	case network.AnswerSignPkMsg:
		m, e := unMarshalConsensusSignPubKeyMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard ConsensusSignPubKeyMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.processor.OnMessageSignPK(m)

	case network.GroupPing:
		m, e := unMarshalCreateGroupPingMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard unMarshalCreateGroupPingMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.processor.OnMessageCreateGroupPing(m)
	case network.GroupPong:
		m, e := unMarshalCreateGroupPongMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard unMarshalCreateGroupPongMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.processor.OnMessageCreateGroupPong(m)

	case network.ReqSharePiece:
		m, e := unMarshalSharePieceReqMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard unMarshalSharePieceReqMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.processor.OnMessageSharePieceReq(m)

	case network.ResponseSharePiece:
		m, e := unMarshalSharePieceResponseMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard unMarshalSharePieceResponseMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.processor.OnMessageSharePieceResponse(m)

	case network.BlockSignAggr:
		m, e := unmarshalBlockSignAggrMessage(body)
		if e != nil {
			logger.Errorf("[handler]Discard unmarshalBlockSignAggrMessage because of unmarshal error:%s", e.Error())
			return e
		}
		c.processor.OnMessageBlockSignAggrMessage(m)

	}
	return nil
}

//----------------------------------------------------------------------------------------------------------------------

