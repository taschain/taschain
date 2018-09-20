package model

import (
	"consensus/groupsig"
	"common"
	"consensus/base"
	"consensus/vrf_ed25519"
	"middleware/types"
)

/*
**  Creator: pxf
**  Date: 2018/9/11 下午1:49
**  Description: 矿工节点
*/

type MinerDO struct {
	PK          groupsig.Pubkey
	VrfPK 		vrf_ed25519.PublicKey
	ID          groupsig.ID
	Stake       uint64
	NType  		byte
	ApplyHeight uint64
	AbortHeight uint64
}

func (md *MinerDO) IsAbort(h uint64) bool {
	return md.AbortHeight > 0 && h >= md.AbortHeight
}

func (md *MinerDO) EffectAt(h uint64) bool {
	return h >= md.ApplyHeight
}

//在该高度是否可以铸块
func (md *MinerDO) CanCastAt(h uint64) bool {
	return md.IsWeight() && !md.IsAbort(h) && md.EffectAt(h)
}

//在该高度是否可以加入组
func (md *MinerDO) CanJoinGroupAt(h uint64) bool {
	return md.IsLight() && !md.IsAbort(h) && md.EffectAt(h)
}

func (md *MinerDO) IsLight() bool {
	return md.NType == types.MinerTypeLight
}

func (md *MinerDO) IsWeight() bool {
	return md.NType == types.MinerTypeHeavy
}

type SelfMinerDO struct {
	MinerDO
	SecretSeed 	base.Rand   //私密随机数
	SK 			groupsig.Seckey
	VrfSK 		vrf_ed25519.PrivateKey
}

func (mi *SelfMinerDO) Read(p []byte) (n int, err error) {
	bs := mi.SecretSeed.Bytes()
	if p == nil || len(p) < len(bs) {
		p = make([]byte, len(bs))
	}
	copy(p, bs)
	return len(bs), nil
}

func NewSelfMinerDO(secert string) SelfMinerDO {
	var mi SelfMinerDO
	mi.SecretSeed = base.RandFromString(secert)
	mi.SK = *groupsig.NewSeckeyFromRand(mi.SecretSeed)
	mi.PK = *groupsig.NewPubkeyFromSeckey(mi.SK)
	mi.ID = *groupsig.NewIDFromPubkey(mi.PK)

	var err error
	mi.VrfPK, mi.VrfSK, err = vrf_ed25519.GenerateKey(&mi)
	if err != nil {
		panic("generate vrf key error, err=" + err.Error())
	}
	return mi
}

func (mi SelfMinerDO) GetMinerID() groupsig.ID {
	return mi.ID
}

func (mi SelfMinerDO) GetSecret() base.Rand {
	return mi.SecretSeed
}

func (mi SelfMinerDO) GetDefaultSecKey() groupsig.Seckey {
	return mi.SK
}

func (mi SelfMinerDO) GetDefaultPubKey() groupsig.Pubkey {
	return mi.PK
}

func (mi SelfMinerDO) GenSecretForGroup(h common.Hash) base.Rand {
	r := base.RandFromBytes(h.Bytes())
	return mi.SecretSeed.DerivedRand(r[:])
}
