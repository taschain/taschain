package model

import (
	"consensus/groupsig"
	"common"
	"consensus/base"
)

/*
**  Creator: pxf
**  Date: 2018/9/11 下午1:49
**  Description: 矿工节点
*/

type NodeType int8

const (
	LightNode  NodeType = 0
	WeightNode          = 1
)

type NodeInfo struct {
	NType  NodeType
}

func (bn *NodeInfo) IsLight() bool {
	return bn.NType == LightNode
}

func (bn *NodeInfo) IsWeight() bool {
	return bn.NType == WeightNode
}

type MinerDO struct {
	NodeInfo
	PK          groupsig.Pubkey
	ID          groupsig.ID
	Stake       uint64
	ApplyHeight uint64
	AbortHeight uint64
}

func (md *MinerDO) IsAbort(h uint64) bool {
	return md.AbortHeight > 0 && h >= md.AbortHeight
}

func (md *MinerDO) EffectAt(h uint64) bool {
	return h >= md.ApplyHeight+ Param.EffectGapAfterApply
}

//在该高度是否可以铸块
func (md *MinerDO) CanCastAt(h uint64) bool {
	return md.IsWeight() && !md.IsAbort(h) && md.EffectAt(h)
}

//在该高度是否可以加入组
func (md *MinerDO) CanJoinGroupAt(h uint64) bool {
	return md.IsLight() && !md.IsAbort(h) && md.EffectAt(h)
}


type MinerInfo struct {
	NodeInfo
	MinerID    groupsig.ID //矿工ID
	SecretSeed base.Rand   //私密随机数
}

func NewMinerInfo(id string, secert string) MinerInfo {
	var mi MinerInfo
	mi.MinerID = *groupsig.NewIDFromString(id)
	mi.SecretSeed = base.RandFromString(secert)
	return mi
}

func (mi *MinerInfo) Init(id groupsig.ID, secert base.Rand) {
	mi.MinerID = id
	mi.SecretSeed = secert
	return
}

func (mi MinerInfo) GetMinerID() groupsig.ID {
	return mi.MinerID
}

func (mi MinerInfo) GetSecret() base.Rand {
	return mi.SecretSeed
}

func (mi MinerInfo) GetDefaultSecKey() groupsig.Seckey {
	return *groupsig.NewSeckeyFromRand(mi.SecretSeed)
}

func (mi MinerInfo) GetDefaultPubKey() groupsig.Pubkey {
	return *groupsig.NewPubkeyFromSeckey(mi.GetDefaultSecKey())
}

func (mi MinerInfo) GenSecretForGroup(h common.Hash) base.Rand {
	r := base.RandFromBytes(h.Bytes())
	return mi.SecretSeed.DerivedRand(r[:])
}
