package logical

import (
	"testing"
	"consensus/groupsig"
	"log"
	"encoding/json"
	"common"
	"fmt"
	"io/ioutil"
	"os"
	"consensus/model"
	"middleware"
	"consensus/base"
)

const CONF_PATH_PREFIX = `/Users/pxf/workspace/tas_develop/tas/local_test`
const ProcNum = 3

func TestBelongGroups(t *testing.T) {
	//groupsig.Init(1)
	middleware.InitMiddleware()
	common.InitConf(CONF_PATH_PREFIX + "/tas1.ini")
	InitConsensus()

	cm := common.NewConfINIManager(CONF_PATH_PREFIX + "/tas1.ini")
	proc := new(Processor)
	addr := common.HexToAddress(cm.GetString("gtas", "miner", ""))

	gstore := fmt.Sprintf("%v/groupstore%v", CONF_PATH_PREFIX, cm.GetString("instance", "index", ""))
	cm.SetString(ConsensusConfSection, "groupstore", gstore)
	jgFile := CONF_PATH_PREFIX + "/joined_group.config." + cm.GetString("instance", "index", "")
	cm.SetString(ConsensusConfSection, "joined_group_store", jgFile)

	proc.Init(model.NewSelfMinerDO(addr), cm)
	proc.belongGroups.initStore()
	gidstr := "0x15fc80b99d8904205b768e04ccea60b67756fd8176ce27e95f5db1da9e57735"
	var gid groupsig.ID
	gid.SetHexString(gidstr)
	jg := proc.belongGroups.getJoinedGroup(gid)

	t.Logf("%+v", jg.GroupID.GetHexString())
	t.Logf("%+v", jg.GroupPK.GetHexString())
	t.Logf("%+v", jg.SignKey.GetHexString())
	for id, mem := range jg.Members {
		t.Logf("%v: %v", id, mem.GetHexString())

	}
}

func initProcessor(conf string) *Processor {
	cm := common.NewConfINIManager(conf)
	proc := new(Processor)
	addr := common.HexToAddress(cm.GetString("gtas", "miner", ""))

	gstore := fmt.Sprintf("%v/groupstore%v", CONF_PATH_PREFIX, cm.GetString("instance", "index", ""))
	cm.SetString("consensus", "groupstore", gstore)

	proc.Init(model.NewSelfMinerDO(addr), cm)
	log.Printf("%v", proc.mi.VrfPK)
	return proc
}

func processors() (map[string]*Processor, map[string]int) {
	maxProcNum := ProcNum
	procs := make(map[string]*Processor, maxProcNum)
	indexs := make(map[string]int, maxProcNum)

	for i := 1; i <= maxProcNum; i++ {
		proc := initProcessor(fmt.Sprintf("%v/tas%v.ini", CONF_PATH_PREFIX, i))
		//proc.belongGroups.storeFile = fmt.Sprintf("%v/joined_group.config.%v", CONF_PATH_PREFIX, i)
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
	//err := core.InitCore(false, mediator.NewConsensusHelper(groupsig.ID{}))
	//if err != nil {
	//	panic(err)
	//}

	//network.Init(common.GlobalConf, true, new(handler.ChainHandler), chandler.MessageHandler, true, "127.0.0.1")

	InitConsensus()

	procs, _ := processors()

	mems := make([]groupsig.ID, 0)
	for _, proc := range procs {
		mems = append(mems, proc.GetMinerID())
	}
	gh := generateGenesisGroupHeader(mems)
	gis := &model.ConsensusGroupInitSummary{
		GHeader: gh,
	}
	grm := &model.ConsensusGroupRawMessage{
		GInfo: model.ConsensusGroupInitInfo{GI:*gis, Mems:mems},
	}

	procSpms := make(map[string][]*model.ConsensusSharePieceMessage)

	model.Param.GroupMemberMax = len(mems)
	model.Param.GroupMemberMin = len(mems)

	for _, p := range procs {
		//if p.globalGroups.AddInitingGroup(CreateInitingGroup(grm)) {
		//	//to do : 从链上检查消息发起人（父亲组成员）是否有权限发该消息（鸠兹）
		//	//dummy 组写入组链 add by 小熊
		//	//p.groupManager.AddGroupOnChain(staticGroupInfo, true)
		//}
		gc := p.joiningGroups.ConfirmGroupFromRaw(grm, nil, p.mi)
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
				GHash: grm.GInfo.GroupHash(),
				Dest: dest,
				Share: share,
			}
			spm.SI.SignMember = p.GetMinerID()
			spms = append(spms, spm)
			procSpms[id] = spms
		}
	}

	spks := make(map[string]*model.ConsensusSignPubKeyMessage)
	initedMsgs := make(map[string]*model.ConsensusGroupInitedMessage)

	for id, spms := range procSpms {
		p := procs[id]
		for _, spm := range spms {
			gc := p.joiningGroups.GetGroup(spm.GHash)
			ret := gc.PieceMessage(p.GetMinerID(), &spm.Share)
			if ret == 1 {
				jg := gc.GetGroupInfo()
				p.joinGroup(jg)
				msg := &model.ConsensusSignPubKeyMessage{
					GHash: spm.GHash,
					GroupID: jg.GroupID,
					SignPK:  *groupsig.NewPubkeyFromSeckey(jg.SignKey),
				}
				//msg.GenGSign(jg.SignKey)
				msg.SI.SignMember = p.GetMinerID()
				spks[id] = msg

				var initedMsg = &model.ConsensusGroupInitedMessage{
					GHash: spm.GHash,
					GroupID: jg.GroupID,
					GroupPK: jg.GroupPK,
					CreateHeight: 0,
				}
				ski := model.NewSecKeyInfo(p.mi.GetMinerID(), p.mi.GetDefaultSecKey())
				initedMsg.GenSign(ski, initedMsg)

				initedMsgs[id] = initedMsg
			}
		}
	}


	for _, p := range procs {
		for _, spkm := range spks {
			jg := p.belongGroups.getJoinedGroup(spkm.GroupID)
			p.belongGroups.addMemSignPk(spkm.SI.GetID(), spkm.GroupID, spkm.SignPK)
			log.Printf("processor %v join group gid %v\n", p.getPrefix(), jg.GroupID.ShortS())
			//if gc.SignPKMessage(spkm) == 1 {
			//	jg := gc.GetGroupInfo()
			//	p.joinGroup(jg, true)
			//	var msg = &model.ConsensusGroupInitedMessage{
			//		GHash: spkm.GHash,
			//		GroupID: jg.GroupID,
			//		GroupPK: jg.GroupPK,
			//	}
			//	ski := model.NewSecKeyInfo(p.mi.GetMinerID(), p.mi.GetDefaultSecKey())
			//	msg.GenSign(ski, msg)
			//
			//	initedMsgs[id] = msg
			//}
		}
	}

	for _, p := range procs {
		for _, msg := range initedMsgs {
			initingGroup := p.globalGroups.GetInitedGroup(msg.GHash)
			if initingGroup == nil {
				ginfo := &model.ConsensusGroupInitInfo{
					GI: model.ConsensusGroupInitSummary{
						Signature: groupsig.Signature{},
						GHeader: gh,
					},
					Mems: mems,
				}
				initingGroup = createInitedGroup(ginfo)
				p.globalGroups.generator.addInitedGroup(initingGroup)
			}
			if initingGroup.receive(msg.SI.GetID(), msg.GroupPK) == INIT_SUCCESS {
				staticGroup := NewSGIFromStaticGroupSummary(msg.GroupID, msg.GroupPK, initingGroup)
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
			log.Printf("jg is nil!!!!!! p=%v, gid=%v\n", p.getPrefix(),sgi.GroupID.ShortS())
			continue
		}
		jgByte, _ := json.Marshal(jg)

		if !write {
			write = true


			genesis := new(genesisGroup)
			genesis.Group = *sgi

			vrfpks := make([]base.VRFPublicKey, sgi.GetMemberCount())
			pks := make([]groupsig.Pubkey, sgi.GetMemberCount())
			for i, mem := range sgi.GetMembers() {
				_p := procs[mem.GetHexString()]
				vrfpks[i]= _p.mi.VrfPK
				pks[i] = _p.mi.GetDefaultPubKey()
			}
			genesis.VrfPK = vrfpks
			genesis.Pks = pks

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
