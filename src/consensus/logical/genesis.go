package logical

import (
	"consensus/groupsig"
	"log"
	"time"
	"vm/common/math"
)

/*
**  Creator: pxf
**  Date: 2018/6/28 下午3:19
**  Description: 
*/


//生成创世组成员信息
func (p *Processor) BeginGenesisGroupMember() PubKeyInfo {
	gis := p.GenGenesisGroupSummary()
	temp_mi := p.getMinerInfo()
	temp_mgs := NewMinerGroupSecret(temp_mi.GenSecretForGroup(gis.GenHash()))
	gsk_piece := temp_mgs.GenSecKey()
	gpk_piece := *groupsig.NewPubkeyFromSeckey(gsk_piece)
	pki := PubKeyInfo{p.GetMinerID(), gpk_piece}
	log.Printf("\nBegin Genesis Group Member, ID=%v, gpk_piece=%v.\n", GetIDPrefix(pki.GetID()), GetPubKeyPrefix(pki.PK))
	return pki
}

func (p *Processor) GenGenesisGroupSummary() ConsensusGroupInitSummary {
	var gis ConsensusGroupInitSummary
	//gis.ParentID = P.GetMinerID()
	gis.DummyID = *groupsig.NewIDFromString("Trust Among Strangers")
	gis.Authority = 777
	gn := "TAS genesis group"
	if len(gn) <= 64 {
		copy(gis.Name[:], gn[:])
	} else {
		copy(gis.Name[:], gn[:64])
	}
	gis.BeginTime = time.Date(2018, time.June, 28, 18, 00, 00, 00, time.UTC)
	//gis.BeginTime = time.Unix(unix_time, 0)
	gis.Extends = "room 1003, BLWJXXJS6KYHX"
	gis.Members = uint64(GetGroupMemberNum())

	gis.BeginTime = time.Now()
	gis.GetReadyHeight = 0
	gis.BeginCastHeight = 0
	gis.DismissHeight = math.MaxUint64

	return gis
}