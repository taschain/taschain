package net

import (
	"consensus/model"
	"common"
	"consensus/groupsig"
	"middleware/types"
)

/*
**  Creator: pxf
**  Date: 2018/7/27 下午3:53
**  Description: 
*/

type MessageProcessor interface {
	Ready() bool

	GetMinerID() groupsig.ID

	ExistInGroup(gHash common.Hash) bool

	OnMessageGroupInit(msg *model.ConsensusGroupRawMessage)

	OnMessageSharePiece(msg *model.ConsensusSharePieceMessage)

	OnMessageSignPK(msg *model.ConsensusSignPubKeyMessage)

	OnMessageSignPKReq(msg *model.ConsensusSignPubkeyReqMessage)

	OnMessageGroupInited(msg *model.ConsensusGroupInitedMessage)

	OnMessageCast(msg *model.ConsensusCastMessage)

	OnMessageVerify(msg *model.ConsensusVerifyMessage)

	OnMessageNewTransactions(txs []common.Hash)

	OnMessageBlock(msg *model.ConsensusBlockMessage)

	OnMessageCreateGroupRaw(msg *model.ConsensusCreateGroupRawMessage)

	OnMessageCreateGroupSign(msg *model.ConsensusCreateGroupSignMessage)

	OnMessageCastRewardSignReq(msg *model.CastRewardTransSignReqMessage)

	OnMessageCastRewardSign(msg *model.CastRewardTransSignMessage)
}

type GroupBrief struct {
	Gid    groupsig.ID
	MemIds []groupsig.ID
}

type NetworkServer interface {
	SendGroupInitMessage(grm *model.ConsensusGroupRawMessage)

	SendKeySharePiece(spm *model.ConsensusSharePieceMessage)

	SendSignPubKey(spkm *model.ConsensusSignPubKeyMessage)

	BroadcastGroupInfo(cgm *model.ConsensusGroupInitedMessage)

	SendCastVerify(ccm *model.ConsensusCastMessage, group *GroupBrief, body []*types.Transaction)

	SendVerifiedCast(cvm *model.ConsensusVerifyMessage)

	BroadcastNewBlock(cbm *model.ConsensusBlockMessage, group *GroupBrief)

	SendCreateGroupRawMessage(msg *model.ConsensusCreateGroupRawMessage)

	SendCreateGroupSignMessage(msg *model.ConsensusCreateGroupSignMessage, parentGid groupsig.ID)

	BuildGroupNet(groupIdentifier string, mems []groupsig.ID)

	ReleaseGroupNet(groupIdentifier string)

	SendCastRewardSignReq(msg *model.CastRewardTransSignReqMessage)

	SendCastRewardSign(msg *model.CastRewardTransSignMessage)

	AnswerSignPkMessage(msg *model.ConsensusSignPubKeyMessage, receiver groupsig.ID)

	AskSignPkMessage(msg *model.ConsensusSignPubkeyReqMessage, receiver groupsig.ID)
}
