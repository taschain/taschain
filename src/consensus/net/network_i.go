package net

import (
	"consensus/model"
	"common"
	"consensus/groupsig"
)

/*
**  Creator: pxf
**  Date: 2018/7/27 下午3:53
**  Description: 
*/

type MessageProcessor interface {

	Ready() bool

	GetMinerID() groupsig.ID

	ExistInDummyGroup(gid groupsig.ID) bool

	OnMessageGroupInit(msg *model.ConsensusGroupRawMessage)

	OnMessageSharePiece(msg *model.ConsensusSharePieceMessage)

	OnMessageSignPK(msg *model.ConsensusSignPubKeyMessage)

	OnMessageGroupInited(msg *model.ConsensusGroupInitedMessage)

	OnMessageCast(msg *model.ConsensusCastMessage)

	OnMessageVerify(msg *model.ConsensusVerifyMessage)

	OnMessageNewTransactions(txs []common.Hash)

	OnMessageBlock(msg *model.ConsensusBlockMessage)

	OnMessageCreateGroupRaw(msg *model.ConsensusCreateGroupRawMessage)

	OnMessageCreateGroupSign(msg *model.ConsensusCreateGroupSignMessage)

	OnMessagePowResult(msg *model.ConsensusPowResultMessage)

	OnMessagePowConfirm(msg *model.ConsensusPowConfirmMessage)
}

type NetworkServer interface {

	SendGroupInitMessage(grm *model.ConsensusGroupRawMessage)

	SendKeySharePiece(spm *model.ConsensusSharePieceMessage)

	SendSignPubKey(spkm *model.ConsensusSignPubKeyMessage)

	BroadcastGroupInfo(cgm *model.ConsensusGroupInitedMessage)

	SendCastVerify(ccm *model.ConsensusCastMessage)

	SendVerifiedCast(cvm *model.ConsensusVerifyMessage)

	BroadcastNewBlock(cbm *model.ConsensusBlockMessage)

	SendCreateGroupRawMessage(msg *model.ConsensusCreateGroupRawMessage)

	SendCreateGroupSignMessage(msg *model.ConsensusCreateGroupSignMessage)

	SendPowResultMessage(msg *model.ConsensusPowResultMessage)

	SendPowConfirmMessage(msg *model.ConsensusPowConfirmMessage)

	BuildGroupNet(gid groupsig.ID, mems []groupsig.ID)

	ReleaseGroupNet(gid groupsig.ID)
}