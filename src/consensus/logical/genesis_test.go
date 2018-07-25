package logical

import (
	"testing"
	"consensus/groupsig"
	"log"
	"encoding/json"
	"core"
	"common"
	"fmt"
	"io/ioutil"
	"os"
)

const CONF_PATH_PREFIX = `/Users/zhangchao/Documents/GitRepository/tas/conf/aliyun_3g21n_new_id`

func TestBelongGroups(t *testing.T) {
	groupsig.Init(1)
	common.InitConf(CONF_PATH_PREFIX + "/tas1.ini")
	InitConsensus()
	belongs := NewBelongGroups("/Users/pxf/workspace/tas_develop/tas/conf/aliyun/joined_group.config.1")
	belongs.load()
	gs := belongs.getAllGroups()
	for _, g := range gs {
		log.Println(GetIDPrefix(g.GroupID))
	}
	t.Log(belongs)
}
//
//func GetIdFromPublicKey(p common.PublicKey) string {
//	pubKey := &p2p.Pubkey{PublicKey: p}
//	pID, e := peer.IDFromPublicKey(pubKey)
//	if e != nil {
//		log.Printf("[Network]IDFromPublicKey error:%s", e.Error())
//		panic("GetIdFromPublicKey error!")
//	}
//	id := pID.Pretty()
//	return id
//}

func initProcessor(conf string) *Processor {
	cm := common.NewConfINIManager(conf)
	scm := cm.GetSectionManager("network")
	privateKey := common.HexStringToSecKey(scm.GetString("private_key", ""))
	pk := privateKey.GetPubKey()
	id := pk.GetAddress().GetHexString()
	proc := new(Processor)
	proc.Init(NewMinerInfo(id, cm.GetString("gtas", "secret", "")))
	return proc
}

func processors() (map[string]*Processor, map[string]int) {
	maxProcNum := 7
	procs := make(map[string]*Processor, maxProcNum)
	indexs := make(map[string]int, maxProcNum)

	for i := 1; i <= maxProcNum; i++ {
		proc := initProcessor(fmt.Sprintf("%v/tas%v.ini", CONF_PATH_PREFIX, i))
		proc.belongGroups.storeFile = fmt.Sprintf("%v/joined_group.config.%v", CONF_PATH_PREFIX, i)
		procs[proc.GetMinerID().GetHexString()] = proc
		indexs[proc.getPrefix()] = i
	}


	//proc = new(Processor)
	//proc.Init(NewMinerInfo("siren", "850701"))
	//procs[proc.GetMinerID().GetHexString()] = proc
	//
	//proc = new(Processor)
	//proc.Init(NewMinerInfo("juanzi", "123456"))
	//procs[proc.GetMinerID().GetHexString()] = proc
	//
	//proc = new(Processor)
	//proc.Init(NewMinerInfo("wild children", "111111"))
	//procs[proc.GetMinerID().GetHexString()] = proc
	//
	//proc = new(Processor)
	//proc.Init(NewMinerInfo("gebaini", "999999"))
	//procs[proc.GetMinerID().GetHexString()] = proc
	//
	//proc = new(Processor)
	//proc.Init(NewMinerInfo("wenqin", "2342245"))
	//procs[proc.GetMinerID().GetHexString()] = proc
	//
	//proc = new(Processor)
	//proc.Init(NewMinerInfo("baozhu", "23420949"))
	//procs[proc.GetMinerID().GetHexString()] = proc

	return procs, indexs
}

func TestGenIdPubkey(t *testing.T) {
	groupsig.Init(1)
	common.InitConf(CONF_PATH_PREFIX + "/tas1.ini")
	InitConsensus()
	procs, _ := processors()
	idPubs := make([]PubKeyInfo, 0)
	for _, p := range procs {
		idPubs = append(idPubs, p.getPubkeyInfo())
	}


	bs, err := json.Marshal(idPubs)
	if err != nil {
		t.Fatal(err)
	}
	log.Println(string(bs))
}

func TestGenesisGroup(t *testing.T) {
	groupsig.Init(1)
	common.InitConf(CONF_PATH_PREFIX + "/tas1.ini")
	// block初始化
	err := core.InitCore()
	if err != nil {
		panic(err)
	}
	InitConsensus()

	procs, _ := processors()

	mems := make([]PubKeyInfo, 0)
	for _, proc := range procs {
		mems = append(mems, proc.getPubkeyInfo())
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
				var msg = new(ConsensusGroupInitedMessage)
				ski := SecKeyInfo{p.mi.GetMinerID(), p.mi.GetDefaultSecKey()}
				msg.GI.GIS = gc.gis
				msg.GI.GroupID = jg.GroupID
				msg.GI.GroupPK = jg.GroupPK

				msg.GenSign(ski, msg)

				initedMsgs[id] = msg
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

	write := false
	for id, p := range procs {
		//index := indexs[p.getPrefix()]

		sgi := p.globalGroups.GetAvailableGroups(0)[0]
		jg := p.belongGroups.getJoinedGroup(sgi.GroupID)

		jgByte, _ := json.Marshal(jg)

		if !write {
			write = true
			log.Println("=======", id, "============")
			sgiByte, _ := json.Marshal(sgi)

			ioutil.WriteFile(fmt.Sprintf("%s/genesis_sgi.config", CONF_PATH_PREFIX), sgiByte, os.ModePerm)

			log.Println(string(sgiByte))
			log.Println("-----------------------")
			log.Println(string(jgByte))
		}


		log.Println()

		//ioutil.WriteFile(fmt.Sprintf("%s/genesis_jg.config.%v", CONF_PATH_PREFIX, index), jgByte, os.ModePerm)

		var sig groupsig.Signature
		sig.Deserialize(jg.GroupSec.SecretSign)
		log.Println(groupsig.VerifySig(sgi.GroupPK, jg.GroupSec.DataHash.Bytes(), sig))
	}





}

func writeGroup(p *Processor) {

}