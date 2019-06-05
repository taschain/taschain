//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package model

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/base"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/middleware/types"
	"github.com/taschain/taschain/taslog"
)

var SlowLog taslog.Logger

type ISignedMessage interface {
	GenSign(ski SecKeyInfo, hasher Hasher) bool
	VerifySign(pk groupsig.Pubkey) bool
}

type Hasher interface {
	GenHash() common.Hash
}

type BaseSignedMessage struct {
	SI SignData
}

func (sign *BaseSignedMessage) GenSign(ski SecKeyInfo, hasher Hasher) bool {
	if !ski.IsValid() {
		return false
	}
	sign.SI = GenSignData(hasher.GenHash(), ski.GetID(), ski.SK)
	return true
}

func (sign *BaseSignedMessage) VerifySign(pk groupsig.Pubkey) (ok bool) {
	if !sign.SI.GetID().IsValid() {
		return false
	}
	ok = sign.SI.VerifySign(pk)
	if !ok {
		fmt.Printf("verifySign fail, pk=%v, id=%v, sign=%v, data=%v\n", pk.GetHexString(), sign.SI.SignMember.GetHexString(), sign.SI.DataSign.GetHexString(), sign.SI.DataHash.Hex())
	}
	return
}

// ConsensusGroupRawMessage received the startup group initialization message for the parent group
type ConsensusGroupRawMessage struct {
	GInfo ConsensusGroupInitInfo // Group initialization consensus
	BaseSignedMessage
}

func (msg *ConsensusGroupRawMessage) GenHash() common.Hash {
	return msg.GInfo.GI.GetHash()
}

func (msg *ConsensusGroupRawMessage) MemberExist(id groupsig.ID) bool {
	return msg.GInfo.MemberExists(id)
}

// ConsensusSharePieceMessage send secret fragment messages to all members
// of the group（different members have different messages）
type ConsensusSharePieceMessage struct {
	GHash  common.Hash // Group initialization consensus (ConsensusGroupInitSummary) hash
	Dest   groupsig.ID // Receiver (miner) ID
	Share  SharePiece  // Message plaintext (encrypted and decrypted by the transport layer with the recipient public key)
	MemCnt int32
	BaseSignedMessage
}

func (msg *ConsensusSharePieceMessage) GenHash() common.Hash {
	buf := msg.GHash.Bytes()
	buf = append(buf, msg.Dest.Serialize()...)
	buf = append(buf, msg.Share.Pub.Serialize()...)
	buf = append(buf, msg.Share.Share.Serialize()...)
	return base.Data2CommonHash(buf)
}

// ConsensusSignPubKeyMessage send a signed public key message to members
// of the group（All members are the same）
type ConsensusSignPubKeyMessage struct {
	GHash   common.Hash
	GroupID groupsig.ID     // Group id
	SignPK  groupsig.Pubkey // Group member signature public key
	MemCnt  int32
	BaseSignedMessage
}

func (msg *ConsensusSignPubKeyMessage) GenHash() common.Hash {
	buf := msg.GHash.Bytes()
	buf = append(buf, msg.GroupID.Serialize()...)
	buf = append(buf, msg.SignPK.Serialize()...)
	return base.Data2CommonHash(buf)
}

// ConsensusSignPubkeyReqMessage request signature public key
type ConsensusSignPubkeyReqMessage struct {
	BaseSignedMessage
	GroupID groupsig.ID
}

func (m *ConsensusSignPubkeyReqMessage) GenHash() common.Hash {
	return base.Data2CommonHash(m.GroupID.Serialize())
}

// ConsensusGroupInitedMessage Broadcasting to the group has been initialized (
// the node outside the group has to receive count of threshold message are same
// before it is add on chain)
type ConsensusGroupInitedMessage struct {
	GHash        common.Hash
	GroupID      groupsig.ID     // Group ID (can be generated by the group public key)
	GroupPK      groupsig.Pubkey // Group public key
	CreateHeight uint64          // The height at which the group started to be created
	ParentSign   groupsig.Signature
	MemMask      []byte // Group member mask, a value of 1 indicates that the candidate is in the group member list, and the group member list can be restored according to the mask table and the candidate set.
	MemCnt       int32
	BaseSignedMessage
}

func (msg *ConsensusGroupInitedMessage) GenHash() common.Hash {
	buf := bytes.Buffer{}
	buf.Write(msg.GHash.Bytes())
	buf.Write(msg.GroupID.Serialize())
	buf.Write(msg.GroupPK.Serialize())
	buf.Write(common.Uint64ToByte(msg.CreateHeight))
	buf.Write(msg.ParentSign.Serialize())
	buf.Write(msg.MemMask)
	return base.Data2CommonHash(buf.Bytes())
}

/*
cast block message family
The SI of the cast block message family is signed with the public key of the group member.
*/

// ConsensusCurrentMessage become the current processing group message, issued
// by the first member who finds the current group become the cast block group
type ConsensusCurrentMessage struct {
	GroupID     []byte      // Cast block group
	PreHash     common.Hash // Previous block hash
	PreTime     time.Time   // Last block completion time
	BlockHeight uint64      // Cast block height
	BaseSignedMessage
}

func (msg *ConsensusCurrentMessage) GenHash() common.Hash {
	buf := msg.PreHash.Hex()
	buf += string(msg.GroupID[:])
	buf += msg.PreTime.String()
	buf += strconv.FormatUint(msg.BlockHeight, 10)
	return base.Data2CommonHash([]byte(buf))
}

type ConsensusCastMessage struct {
	BH        types.BlockHeader
	ProveHash common.Hash
	BaseSignedMessage
}

func (msg *ConsensusCastMessage) GenHash() common.Hash {
	return msg.BH.GenHash()
}

func (msg *ConsensusCastMessage) VerifyRandomSign(pkey groupsig.Pubkey, preRandom []byte) bool {
	sig := groupsig.DeserializeSign(msg.BH.Random)
	if sig == nil || sig.IsNil() {
		return false
	}
	return groupsig.VerifySig(pkey, preRandom, *sig)
}

// ConsensusVerifyMessage is Verification message - issued by the certifier
// in the group (verify the casting block of KING)
type ConsensusVerifyMessage struct {
	BlockHash  common.Hash
	RandomSign groupsig.Signature
	BaseSignedMessage
}

func (msg *ConsensusVerifyMessage) GenHash() common.Hash {
	return msg.BlockHash
}

func (msg *ConsensusVerifyMessage) GenRandomSign(skey groupsig.Seckey, preRandom []byte) {
	sig := groupsig.Sign(skey, preRandom)
	msg.RandomSign = sig
}

// ConsensusBlockMessage is cast block Success Message - The group successfully
// completed an block and was issued by any member of the group who collected
// k signatures
type ConsensusBlockMessage struct {
	Block types.Block
}

func (msg *ConsensusBlockMessage) GenHash() common.Hash {
	buf := msg.Block.Header.GenHash().Bytes()
	buf = append(buf, msg.Block.Header.GroupID...)
	return base.Data2CommonHash(buf)
}

func (msg *ConsensusBlockMessage) VerifySig(gpk groupsig.Pubkey, preRandom []byte) bool {
	sig := groupsig.DeserializeSign(msg.Block.Header.Signature)
	if sig == nil {
		return false
	}
	b := groupsig.VerifySig(gpk, msg.Block.Header.Hash.Bytes(), *sig)
	if !b {
		return false
	}
	rsig := groupsig.DeserializeSign(msg.Block.Header.Random)
	if rsig == nil {
		return false
	}
	return groupsig.VerifySig(gpk, preRandom, *rsig)
}

/*
Parent group build consensus message
*/

// ConsensusCreateGroupRawMessage is parent group build consensus message
type ConsensusCreateGroupRawMessage struct {
	GInfo ConsensusGroupInitInfo // Group initialization consensus
	BaseSignedMessage
}

func (msg *ConsensusCreateGroupRawMessage) GenHash() common.Hash {
	return msg.GInfo.GI.GetHash()
}

type ConsensusCreateGroupSignMessage struct {
	GHash common.Hash
	BaseSignedMessage
	Launcher groupsig.ID
}

func (msg *ConsensusCreateGroupSignMessage) GenHash() common.Hash {
	return msg.GHash
}

/*
Reward transaction
*/

// CastRewardTransSignReqMessage is about reward transaction
type CastRewardTransSignReqMessage struct {
	BaseSignedMessage
	Reward       types.Bonus
	SignedPieces []groupsig.Signature
	ReceiveTime  time.Time // The timestamp of the request information is received, and the msg of the future is added. If the corresponding block is not uplinkd for a long time, it needs to be cleared according to this timestamp
}

func (msg *CastRewardTransSignReqMessage) GenHash() common.Hash {
	return msg.Reward.TxHash
}

type CastRewardTransSignMessage struct {
	BaseSignedMessage
	ReqHash   common.Hash
	BlockHash common.Hash

	// Not serialized
	GroupID  groupsig.ID
	Launcher groupsig.ID
}

func (msg *CastRewardTransSignMessage) GenHash() common.Hash {
	return msg.ReqHash
}

type CreateGroupPingMessage struct {
	BaseSignedMessage
	FromGroupID groupsig.ID
	PingID      string
	BaseHeight  uint64
}

func (msg *CreateGroupPingMessage) GenHash() common.Hash {
	buf := msg.FromGroupID.Serialize()
	buf = append(buf, []byte(msg.PingID)...)
	buf = append(buf, common.Uint64ToByte(msg.BaseHeight)...)
	return base.Data2CommonHash(buf)
}

type CreateGroupPongMessage struct {
	BaseSignedMessage
	PingID string
	Ts     time.Time
}

func (msg *CreateGroupPongMessage) GenHash() common.Hash {
	buf := []byte(msg.PingID)
	tb, _ := msg.Ts.MarshalBinary()
	buf = append(buf, tb...)
	return base.Data2CommonHash(tb)
}

type ReqSharePieceMessage struct {
	BaseSignedMessage
	GHash common.Hash
}

func (msg *ReqSharePieceMessage) GenHash() common.Hash {
	return msg.GHash
}

// ResponseSharePieceMessage send secret fragment messages to all
// members of the group (different members)
type ResponseSharePieceMessage struct {
	GHash common.Hash // Group initialization consensus (ConsensusGroupInitSummary) hash
	Share SharePiece  // Message plaintext (encrypted and decrypted by the transport layer with the recipient public key)
	BaseSignedMessage
}

func (msg *ResponseSharePieceMessage) GenHash() common.Hash {
	buf := msg.GHash.Bytes()
	buf = append(buf, msg.Share.Pub.Serialize()...)
	buf = append(buf, msg.Share.Share.Serialize()...)
	return base.Data2CommonHash(buf)
}

type BlockSignAggrMessage struct {
	Hash   common.Hash
	Sign   groupsig.Signature
	Random groupsig.Signature
}

type ReqProposalBlock struct {
	Hash common.Hash
}

type ResponseProposalBlock struct {
	Hash         common.Hash
	Transactions []*types.Transaction
}
