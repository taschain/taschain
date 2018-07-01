package logical

import (
	"testing"
	"consensus/groupsig"
	"log"
	"encoding/json"
	"core"
	"common"
)

func processors() map[string]*Processor {
	procs := make(map[string]*Processor, GetGroupMemberNum())

	proc := new(Processor)
	proc.Init(NewMinerInfo("thiefox", "710208"))
	procs[proc.GetMinerID().GetHexString()] = proc

	proc = new(Processor)
	proc.Init(NewMinerInfo("siren", "850701"))
	procs[proc.GetMinerID().GetHexString()] = proc

	proc = new(Processor)
	proc.Init(NewMinerInfo("juanzi", "123456"))
	procs[proc.GetMinerID().GetHexString()] = proc

	proc = new(Processor)
	proc.Init(NewMinerInfo("wild children", "111111"))
	procs[proc.GetMinerID().GetHexString()] = proc

	proc = new(Processor)
	proc.Init(NewMinerInfo("gebaini", "999999"))
	procs[proc.GetMinerID().GetHexString()] = proc

	proc = new(Processor)
	proc.Init(NewMinerInfo("wenqin", "2342245"))
	procs[proc.GetMinerID().GetHexString()] = proc

	proc = new(Processor)
	proc.Init(NewMinerInfo("baozhu", "23420949"))
	procs[proc.GetMinerID().GetHexString()] = proc

	return procs
}

func TestGenesisGroup(t *testing.T) {
	groupsig.Init(1)
	common.InitConf("/Users/pxf/workspace/tas/conf/aliyun_3g21n/tas1.ini")
	// block初始化
	err := core.InitCore()
	if err != nil {
		panic(err)
	}
	InitConsensus()

	procs := processors()

	mems := make([]PubKeyInfo, 0)
	for _, proc := range procs {
		mems = append(mems, proc.GetMinerInfo())
	}
	gis := GenGenesisGroupSummary()
	gis.MemberHash = genMemberHash(mems)
	grm := &ConsensusGroupRawMessage{
		GI: gis,
		MEMS: mems,
	}

	procSpms := make(map[string][]*ConsensusSharePieceMessage)

	for _, p := range procs {
		staticGroupInfo := new(StaticGroupInfo)
		staticGroupInfo.GIS = grm.GI
		staticGroupInfo.Members = grm.MEMS

		if p.globalGroups.AddInitingGroup(CreateInitingGroup(grm)) {
			//to do : 从链上检查消息发起人（父亲组成员）是否有权限发该消息（鸠兹）
			//dummy 组写入组链 add by 小熊
			p.groupManager.AddGroupOnChain(staticGroupInfo, true)
		}

		gc := p.joiningGroups.ConfirmGroupFromRaw(grm, p.mi)
		shares := gc.GenSharePieces()
		for id, share := range shares {
			spms := procSpms[id]
			if spms == nil {
				spms = make([]*ConsensusSharePieceMessage, 0)
				procSpms[id] = spms
			}
			var dest groupsig.ID
			dest.SetHexString(id)
			spm := &ConsensusSharePieceMessage{
				GISHash: grm.GI.GenHash(),
				DummyID: grm.GI.DummyID,
				Dest: dest,
				Share: share,
			}
			spm.SI.SignMember = p.GetMinerID()
			spms = append(spms, spm)
			procSpms[id] = spms
		}
	}

	spks := make(map[string]*ConsensusSignPubKeyMessage)

	for id, spms := range procSpms {
		p := procs[id]
		for _, spm := range spms {
			gc := p.joiningGroups.GetGroup(spm.DummyID)
			ret := gc.PieceMessage(*spm)
			if ret == 1 {
				jg := gc.GetGroupInfo()
				msg := &ConsensusSignPubKeyMessage{
					GISHash: spm.GISHash,
					DummyID: spm.DummyID,
					SignPK:  *groupsig.NewPubkeyFromSeckey(jg.SignKey),
				}
				msg.GenGISSign(jg.SignKey)
				msg.SI.SignMember = p.GetMinerID()
				spks[id] = msg
			}
		}
	}

	initedMsgs := make(map[string]*ConsensusGroupInitedMessage)

	for id, p := range procs {
		for _, spkm := range spks {
			gc := p.joiningGroups.GetGroup(spkm.DummyID)
			if gc.SignPKMessage(spkm) == 1 {
				jg := gc.GetGroupInfo()
				p.joinGroup(jg, true)
				var msg ConsensusGroupInitedMessage
				ski := SecKeyInfo{p.mi.GetMinerID(), p.mi.GetDefaultSecKey()}
				msg.GI.GIS = gc.gis
				msg.GI.GroupID = jg.GroupID
				msg.GI.GroupPK = jg.GroupPK

				msg.GenSign(ski)
				initedMsgs[id] = &msg
			}
		}
	}

	for _, p := range procs {
		for _, msg := range initedMsgs {
			initingGroup := p.globalGroups.GetInitingGroup(msg.GI.GIS.DummyID)
			if p.globalGroups.GroupInitedMessage(&msg.GI, msg.SI.SignMember, 0) == INIT_SUCCESS {
				staticGroup := NewSGIFromStaticGroupSummary(&msg.GI, initingGroup)
				add := p.globalGroups.AddStaticGroup(staticGroup)
				if add {
					p.groupManager.AddGroupOnChain(staticGroup, false)

					//if p.IsMinerGroup(msg.GI.GroupID) && p.GetBlockContext(msg.GI.GroupID) == nil {
					//	p.prepareForCast(msg.GI.GroupID)
					//}
				}
			}
		}
	}

	for id, p := range procs {
		sgi := p.globalGroups.GetAvailableGroups(0)[0]
		jg := p.belongGroups.getJoinedGroup(sgi.GroupID)

		log.Println("=======", id, "============")
		sgiByte, _ := json.Marshal(sgi)
		log.Println(string(sgiByte))

		log.Println("-----------------------")

		jgByte, _ := json.Marshal(jg)
		log.Println(string(jgByte))
		log.Println()

		var sig groupsig.Signature
		sig.Deserialize(jg.GroupSec.SecretSign)
		log.Println(groupsig.VerifySig(sgi.GroupPK, jg.GroupSec.DataHash.Bytes(), sig))
	}


}