package net

import (
	"github.com/gogo/protobuf/proto"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/core"
	"github.com/taschain/taschain/middleware/pb"
	"github.com/taschain/taschain/middleware/types"
	"github.com/taschain/taschain/network"
	"time"
)

type NetworkServerImpl struct {
	net network.Network
}

func NewNetworkServer() NetworkServer {
	return &NetworkServerImpl{
		net: network.GetNetInstance(),
	}
}

func id2String(ids []groupsig.ID) []string {
	idStrs := make([]string, len(ids))
	for idx, id := range ids {
		idStrs[idx] = id.GetHexString()
	}
	return idStrs
}

/*
Group network management
*/

func (ns *NetworkServerImpl) BuildGroupNet(gid string, mems []groupsig.ID) {
	memStrs := id2String(mems)
	ns.net.BuildGroupNet(gid, memStrs)
}

func (ns *NetworkServerImpl) ReleaseGroupNet(gid string) {
	ns.net.DissolveGroupNet(gid)
}

func (ns *NetworkServerImpl) send2Self(self groupsig.ID, m network.Message) {
	go MessageHandler.Handle(self.GetHexString(), m)
}

/*
Group initialization
*/

// SendGroupInitMessage broadcast group initialization message
func (ns *NetworkServerImpl) SendGroupInitMessage(grm *model.ConsensusGroupRawMessage) {
	body, e := marshalConsensusGroupRawMessage(grm)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusGroupRawMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.GroupInitMsg, Body: body}
	// The target group has not been built and needs to be sent point to point.
	for _, mem := range grm.GInfo.Mems {
		logger.Debugf("%v SendGroupInitMessage gHash %v to %v", grm.SI.GetID().GetHexString(), grm.GInfo.GroupHash().Hex(), mem.GetHexString())
		ns.net.Send(mem.GetHexString(), m)
	}
}

// SendKeySharePiece means intra-group broadcast key, for each
// directed transmission, intra-group broadcast
func (ns *NetworkServerImpl) SendKeySharePiece(spm *model.ConsensusSharePieceMessage) {

	body, e := marshalConsensusSharePieceMessage(spm)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusSharePieceMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.KeyPieceMsg, Body: body}
	if spm.SI.SignMember.IsEqual(spm.Dest) {
		go ns.send2Self(spm.SI.GetID(), m)
		return
	}

	begin := time.Now()
	go ns.net.SendWithGroupRelay(spm.Dest.GetHexString(), spm.GHash.Hex(), m)
	logger.Debugf("SendKeySharePiece to id:%s,hash:%s, gHash:%v, cost time:%v", spm.Dest.GetHexString(), m.Hash(), spm.GHash.Hex(), time.Since(begin))
}

// SendSignPubKey means group broadcast signature public key
func (ns *NetworkServerImpl) SendSignPubKey(spkm *model.ConsensusSignPubKeyMessage) {
	body, e := marshalConsensusSignPubKeyMessage(spkm)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusSignPubKeyMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.SignPubkeyMsg, Body: body}
	// Send to yourself
	ns.send2Self(spkm.SI.GetID(), m)

	begin := time.Now()
	go ns.net.SpreadAmongGroup(spkm.GHash.Hex(), m)
	logger.Debugf("SendSignPubKey hash:%s, dummyId:%v, cost time:%v", m.Hash(), spkm.GHash.Hex(), time.Since(begin))
}

// BroadcastGroupInfo means group initialization completed,
// network broadcast group information
func (ns *NetworkServerImpl) BroadcastGroupInfo(cgm *model.ConsensusGroupInitedMessage) {
	body, e := marshalConsensusGroupInitedMessage(cgm)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusGroupInitedMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.GroupInitDoneMsg, Body: body}
	// Send to yourself
	ns.send2Self(cgm.SI.GetID(), m)

	go ns.net.Broadcast(m)
	logger.Debugf("Broadcast GROUP_INIT_DONE_MSG, hash:%s, gHash:%v", m.Hash(), cgm.GHash.Hex())

}

/*
Group coinage
*/

// SendCastVerify means the coinage node completes the coinage,
// sends the blockheader signature, and sends it to other nodes
// in the group for verification. Broadcast within a group
func (ns *NetworkServerImpl) SendCastVerify(ccm *model.ConsensusCastMessage, gb *GroupBrief, proveHashs []common.Hash) {
	bh := types.BlockHeaderToPb(&ccm.BH)
	si := signDataToPb(&ccm.SI)

	for idx, mem := range gb.MemIds {
		message := &tas_middleware_pb.ConsensusCastMessage{Bh: bh, Sign: si, ProveHash: proveHashs[idx].Bytes()}
		body, err := proto.Marshal(message)
		if err != nil {
			logger.Errorf("marshalConsensusCastMessage error:%v %v", err, mem.GetHexString())
			continue
		}
		m := network.Message{Code: network.CastVerifyMsg, Body: body}
		go ns.net.Send(mem.GetHexString(), m)
	}
}

// SendVerifiedCast means the intra-group node starts its own signature
// after the verification is passed, broadcasts the verification block
// and broadcasts within the group. Verify not pass, keep silent
func (ns *NetworkServerImpl) SendVerifiedCast(cvm *model.ConsensusVerifyMessage, receiver groupsig.ID) {
	body, e := marshalConsensusVerifyMessage(cvm)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusVerifyMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.VerifiedCastMsg, Body: body}

	// The verification message needs to be sent to itself, otherwise
	// it will not contain its own signature in its own fragment,
	// resulting in no rewards.
	go ns.send2Self(cvm.SI.GetID(), m)

	go ns.net.SpreadAmongGroup(receiver.GetHexString(), m)
	logger.Debugf("[peer]send VARIFIED_CAST_MSG,hash:%s", cvm.BlockHash.Hex())
}

// BroadcastNewBlock means the whole network broadcasts the block
// signed by the group.
func (ns *NetworkServerImpl) BroadcastNewBlock(cbm *model.ConsensusBlockMessage, group *GroupBrief) {
	body, e := types.MarshalBlock(&cbm.Block)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusBlockMessage because of marshal error:%s", e.Error())
		return
	}
	blockMsg := network.Message{Code: network.NewBlockMsg, Body: body}

	nextVerifyGroupID := group.Gid.GetHexString()
	groupMembers := id2String(group.MemIds)

	// Broadcast to a virtual group of heavy nodes
	heavyMinerMembers := core.MinerManagerImpl.GetHeavyMiners()

	validGroupMembers := make([]string, 0)
	for _, mid := range groupMembers {
		find := false
		for _, hid := range heavyMinerMembers {
			if hid == mid {
				find = true
				break
			}
		}
		if !find {
			validGroupMembers = append(validGroupMembers, mid)
		}
	}

	go ns.net.SpreadToGroup(network.FullNodeVirtualGroupID, heavyMinerMembers, blockMsg, []byte(blockMsg.Hash()))

	// Broadcast to the next group of light nodes
	//
	// Prevent duplicate broadcasts
	if len(validGroupMembers) > 0 {
		go ns.net.SpreadToGroup(nextVerifyGroupID, validGroupMembers, blockMsg, []byte(blockMsg.Hash()))
	}

	core.Logger.Debugf("Broad new block %d-%d,hash:%v, spread over group:%s", cbm.Block.Header.Height, cbm.Block.Header.TotalQN, cbm.Block.Header.Hash.Hex(), nextVerifyGroupID)
}

func (ns *NetworkServerImpl) AnswerSignPkMessage(msg *model.ConsensusSignPubKeyMessage, receiver groupsig.ID) {
	body, e := marshalConsensusSignPubKeyMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusSignPubKeyMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.AnswerSignPkMsg, Body: body}

	begin := time.Now()
	go ns.net.Send(receiver.GetHexString(), m)
	logger.Debugf("AnswerSignPkMessage %v, hash:%s, dummyId:%v, cost time:%v", receiver.GetHexString(), m.Hash(), msg.GHash.Hex(), time.Since(begin))
}

func (ns *NetworkServerImpl) AskSignPkMessage(msg *model.ConsensusSignPubkeyReqMessage, receiver groupsig.ID) {
	body, e := marshalConsensusSignPubKeyReqMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send ConsensusSignPubkeyReqMessage because of marshal error:%s", e.Error())
		return
	}

	m := network.Message{Code: network.AskSignPkMsg, Body: body}

	begin := time.Now()
	go ns.net.Send(receiver.GetHexString(), m)
	logger.Debugf("AskSignPkMessage %v, hash:%s, cost time:%v", receiver.GetHexString(), m.Hash(), time.Since(begin))
}

/*
Pre-establishment consensus
*/

// SendCreateGroupRawMessage start building a group
func (ns *NetworkServerImpl) SendCreateGroupRawMessage(msg *model.ConsensusCreateGroupRawMessage) {
	body, e := marshalConsensusCreateGroupRawMessage(msg)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusCreateGroupRawMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CreateGroupaRaw, Body: body}

	var groupID = msg.GInfo.GI.ParentID()
	go ns.net.SpreadAmongGroup(groupID.GetHexString(), m)
}

func (ns *NetworkServerImpl) SendCreateGroupSignMessage(msg *model.ConsensusCreateGroupSignMessage, parentGid groupsig.ID) {
	body, e := marshalConsensusCreateGroupSignMessage(msg)
	if e != nil {
		logger.Errorf("[peer]Discard send ConsensusCreateGroupSignMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CreateGroupSign, Body: body}

	go ns.net.SendWithGroupRelay(msg.Launcher.GetHexString(), parentGid.GetHexString(), m)
}

func (ns *NetworkServerImpl) SendCastRewardSignReq(msg *model.CastRewardTransSignReqMessage) {
	body, e := marshalCastRewardTransSignReqMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send CastRewardTransSignReqMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CastRewardSignReq, Body: body}

	gid := groupsig.DeserializeID(msg.Reward.GroupID)

	network.Logger.Debugf("send SendCastRewardSignReq to %v", gid.GetHexString())

	ns.net.SpreadAmongGroup(gid.GetHexString(), m)
}

func (ns *NetworkServerImpl) SendCastRewardSign(msg *model.CastRewardTransSignMessage) {
	body, e := marshalCastRewardTransSignMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send CastRewardTransSignMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.CastRewardSignGot, Body: body}

	ns.net.SendWithGroupRelay(msg.Launcher.GetHexString(), msg.GroupID.GetHexString(), m)
}

func (ns *NetworkServerImpl) SendGroupPingMessage(msg *model.CreateGroupPingMessage, receiver groupsig.ID) {
	body, e := marshalCreateGroupPingMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send SendGroupPingMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.GroupPing, Body: body}

	ns.net.Send(receiver.GetHexString(), m)
}

func (ns *NetworkServerImpl) SendGroupPongMessage(msg *model.CreateGroupPongMessage, group *GroupBrief) {
	body, e := marshalCreateGroupPongMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send SendGroupPongMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.GroupPong, Body: body}

	mems := id2String(group.MemIds)

	ns.net.SpreadToGroup(group.Gid.GetHexString(), mems, m, msg.SI.DataHash.Bytes())
}

func (ns *NetworkServerImpl) ReqSharePiece(msg *model.ReqSharePieceMessage, receiver groupsig.ID) {
	body, e := marshalSharePieceReqMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send marshalSharePieceReqMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.ReqSharePiece, Body: body}

	ns.net.Send(receiver.GetHexString(), m)
}

func (ns *NetworkServerImpl) ResponseSharePiece(msg *model.ResponseSharePieceMessage, receiver groupsig.ID) {
	body, e := marshalSharePieceResponseMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send marshalSharePieceResponseMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.ResponseSharePiece, Body: body}

	ns.net.Send(receiver.GetHexString(), m)
}

func (ns *NetworkServerImpl) ReqProposalBlock(msg *model.ReqProposalBlock, target string) {
	body, e := marshalReqProposalBlockMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send marshalReqProposalBlockMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.ReqProposalBlock, Body: body}

	ns.net.Send(target, m)
}

func (ns *NetworkServerImpl) ResponseProposalBlock(msg *model.ResponseProposalBlock, target string) {

	body, e := marshalResponseProposalBlockMessage(msg)
	if e != nil {
		network.Logger.Errorf("[peer]Discard send marshalResponseProposalBlockMessage because of marshal error:%s", e.Error())
		return
	}
	m := network.Message{Code: network.ResponseProposalBlock, Body: body}

	ns.net.Send(target, m)
}
