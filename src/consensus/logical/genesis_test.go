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
	"consensus/model"
	"middleware"
	"consensus/vrf_ed25519"
)

const CONF_PATH_PREFIX = `/Users/pxf/workspace/tas_develop/tas/deploy/daily`

func TestBelongGroups(t *testing.T) {
	//groupsig.Init(1)
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

func initProcessor(conf string) *Processor {
	cm := common.NewConfINIManager(conf)
	proc := new(Processor)
	proc.Init(model.NewSelfMinerDO(cm.GetString("gtas", "secret", "")))
	return proc
}

func processors() (map[string]*Processor, map[string]int) {
	maxProcNum := 3
	procs := make(map[string]*Processor, maxProcNum)
	indexs := make(map[string]int, maxProcNum)

	for i := 1; i <= maxProcNum; i++ {
		proc := initProcessor(fmt.Sprintf("%v/tas%v.ini", CONF_PATH_PREFIX, i))
		proc.belongGroups.storeFile = fmt.Sprintf("%v/joined_group.config.%v", CONF_PATH_PREFIX, i)
		procs[proc.GetMinerID().GetHexString()] = proc
		indexs[proc.getPrefix()] = i
	}

	return procs, indexs
}

func TestGenIdPubkey(t *testing.T) {
	//groupsig.Init(1)
	middleware.InitMiddleware()
	common.InitConf(CONF_PATH_PREFIX + "/tas1.ini")
	InitConsensus()
	procs, _ := processors()
	idPubs := make([]model.PubKeyInfo, 0)
	for _, p := range procs {
		idPubs = append(idPubs, p.GetPubkeyInfo())
	}


	bs, err := json.Marshal(idPubs)
	if err != nil {
		t.Fatal(err)
	}
	log.Println(string(bs))
}

func TestGenesisGroup(t *testing.T) {
	//groupsig.Init(1)
	middleware.InitMiddleware()
	common.InitConf(CONF_PATH_PREFIX + "/tas1.ini")
	// block初始化
	err := core.InitCore()
	if err != nil {
		panic(err)
	}

	//network.Init(common.GlobalConf, true, new(handler.ChainHandler), chandler.MessageHandler, true, "127.0.0.1")

	InitConsensus()
	model.InitParam()

	procs, _ := processors()

	mems := make([]model.PubKeyInfo, 0)
	for _, proc := range procs {
		mems = append(mems, proc.GetPubkeyInfo())
	}
	gis := GenGenesisGroupSummary()
	gis.WithMemberPubs(mems)
	grm := &model.ConsensusGroupRawMessage{
		GI: gis,
		MEMS: mems,
	}

	procSpms := make(map[string][]*model.ConsensusSharePieceMessage)

	model.Param.GroupMember = len(mems)

	for _, p := range procs {
		staticGroupInfo := new(StaticGroupInfo)
		staticGroupInfo.GIS = grm.GI
		staticGroupInfo.Members = grm.MEMS

		if p.globalGroups.AddInitingGroup(CreateInitingGroup(grm)) {
			//to do : 从链上检查消息发起人（父亲组成员）是否有权限发该消息（鸠兹）
			//dummy 组写入组链 add by 小熊
			//p.groupManager.AddGroupOnChain(staticGroupInfo, true)
		}

		gc := p.joiningGroups.ConfirmGroupFromRaw(grm, p.mi)
		shares := gc.GenSharePieces()
		for id, share := range shares {
			spms := procSpms[id]
			if spms == nil {
				spms = make([]*model.ConsensusSharePieceMessage, 0)
				procSpms[id] = spms
			}
			var dest groupsig.ID
			dest.SetHexString(id)
			spm := &model.ConsensusSharePieceMessage{
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

	spks := make(map[string]*model.ConsensusSignPubKeyMessage)

	for id, spms := range procSpms {
		p := procs[id]
		for _, spm := range spms {
			gc := p.joiningGroups.GetGroup(spm.DummyID)
			ret := gc.PieceMessage(spm)
			if ret == 1 {
				jg := gc.GetGroupInfo()
				msg := &model.ConsensusSignPubKeyMessage{
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

	initedMsgs := make(map[string]*model.ConsensusGroupInitedMessage)

	for id, p := range procs {
		for _, spkm := range spks {
			gc := p.joiningGroups.GetGroup(spkm.DummyID)
			if gc.SignPKMessage(spkm) == 1 {
				jg := gc.GetGroupInfo()
				log.Printf("processor %v join group gid %v\n", p.getPrefix(), GetIDPrefix(jg.GroupID))
				p.joinGroup(jg, true)
				var msg = new(model.ConsensusGroupInitedMessage)
				ski := model.NewSecKeyInfo(p.mi.GetMinerID(), p.mi.GetDefaultSecKey())
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
					//p.groupManager.AddGroupOnChain(staticGroup, false)

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
		if jg == nil {
			log.Printf("jg is nil!!!!!! p=%v, gid=%v\n", p.getPrefix(),GetIDPrefix(sgi.GroupID))
			continue
		}
		jgByte, _ := json.Marshal(jg)

		if !write {
			write = true


			genesis := new(genesisGroup)
			genesis.Group = *sgi

			vrfpks := make(map[string]vrf_ed25519.PublicKey, 0)
			for _, mem := range sgi.Members {
				vrfpks[mem.ID.GetHexString()]= p.mi.VrfPK
			}
			genesis.VrfPK = vrfpks

			log.Println("=======", id, "============")
			sgiByte, _ := json.Marshal(genesis)

			ioutil.WriteFile(fmt.Sprintf("%s/genesis_sgi.config", CONF_PATH_PREFIX), sgiByte, os.ModePerm)

			log.Println(string(sgiByte))
			log.Println("-----------------------")
			log.Println(string(jgByte))
		}


		log.Println()

		//ioutil.WriteFile(fmt.Sprintf("%s/genesis_jg.config.%v", CONF_PATH_PREFIX, index), jgByte, os.ModePerm)
	}





}

func TestLoadGenesisGroup(t *testing.T) {
	file := CONF_PATH_PREFIX + "/genesis_sgi.config"
	gg := genGenesisStaticGroupInfo(file)

	json, _ := json.Marshal(gg)
	t.Log(string(json))
}