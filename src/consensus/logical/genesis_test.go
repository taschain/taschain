package logical

import (
	"common"
	"consensus/groupsig"
	"consensus/model"
	chandler "consensus/net"
	"core"
	"core/net/handler"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"middleware"
	"network"
	"os"
	"testing"
)

const CONF_PATH_PREFIX = `/Users/dongdexu/TASchain/taschain/deploy/daily`

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
	proc.Init(model.NewMinerInfo(id, cm.GetString("gtas", "secret", "")))
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

	network.Init(common.GlobalConf, true, new(handler.ChainHandler), chandler.MessageHandler, true, "127.0.0.1")

	InitConsensus()
	model.InitParam()

	procs, _ := processors()

	//生成miner信息ID, SK, PK.
	mems := make([]model.PubKeyInfo, 0)
	for _, proc := range procs {
		mems = append(mems, proc.GetPubkeyInfo())
	}
	gis := GenGenesisGroupSummary()
	gis.WithMemberPubs(mems)

	//组启初始化信息广播
	grm := &model.ConsensusGroupRawMessage{
		GI:   gis,
		MEMS: mems,
	}

	procSpms := make(map[string][]*model.ConsensusSharePieceMessage)

	model.Param.GroupMember = len(mems)

	for _, p := range procs {
		staticGroupInfo := new(StaticGroupInfo)
		staticGroupInfo.GIS = grm.GI
		staticGroupInfo.Members = grm.MEMS

		//log.Println("------dummyId:", gis.DummyID.GetHexString())
		//组上链.
		if p.globalGroups.AddInitingGroup(CreateInitingGroup(grm)) {
			//to do : 从链上检查消息发起人（父亲组成员）是否有权限发该消息（鸠兹）
			//dummy 组写入组链 add by 小熊

			//log.Println("------AddGroupOnChain()")
			p.groupManager.AddGroupOnChain(staticGroupInfo, true)
		}

		//创建组共识上下文
		gc := p.joiningGroups.ConfirmGroupFromRaw(grm, p.mi)

		//密钥分片广播
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
				Dest:    dest,
				Share:   share,
			}
			spm.SI.SignMember = p.GetMinerID()
			spms = append(spms, spm)
			procSpms[id] = spms
		}
	}

	//签名公钥信息广播
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
					SignPK:  *groupsig.NewPubkeyFromSeckey(jg.SignKey),  //PKi = si·Q.
				}
				msg.GenGISSign(jg.SignKey)
				msg.SI.SignMember = p.GetMinerID()
				spks[id] = msg

				//log.Println("SignKey:", jg.SignKey.Serialize())
			}
		}
	}

	//初始化完成消息
	initedMsgs := make(map[string]*model.ConsensusGroupInitedMessage)
	for id, p := range procs {
		for _, spkm := range spks {
			gc := p.joiningGroups.GetGroup(spkm.DummyID)
			if gc.SignPKMessage(spkm) == 1 {
				jg := gc.GetGroupInfo()
				if jg == nil {
					log.Println("jg is nil")
				}
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

	//组上链
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

	log.Println("ƒƒƒƒƒƒƒƒƒƒƒƒƒƒƒƒƒƒ")

	//把组信息写入到文件.
	write := false
	for id, p := range procs {
		//index := indexs[p.getPrefix()]
		sgi := p.globalGroups.GetAvailableGroups(0)[0]

		log.Println("=====getJoinedGroup by GroupID:",sgi.GroupID.GetHexString())
		jg := p.belongGroups.getJoinedGroup(sgi.GroupID)

		jgByte, _ := json.Marshal(jg)

		if !write {
			write = true
			log.Println("=======", id, "============")

			//log.Println("sgi.GroupPK:", sgi.GroupPK.GetHexString())

			sgiByte, _ := json.Marshal(sgi)

			ioutil.WriteFile(fmt.Sprintf("%s/genesis_sgi.config", CONF_PATH_PREFIX), sgiByte, os.ModePerm)

			log.Println("sgi:", string(sgiByte))
			log.Println("-----------------------")
			log.Println("jg:", string(jgByte))
		}

		log.Println()
		//ioutil.WriteFile(fmt.Sprintf("%s/genesis_jg.config.%v", CONF_PATH_PREFIX, index), jgByte, os.ModePerm)

		var sig groupsig.Signature

		if jg == nil {
			log.Println("jg is NULL")
		} else {
			sig.Deserialize(jg.GroupSec.SecretSign)
			log.Println(groupsig.VerifySig(sgi.GroupPK, jg.GroupSec.DataHash.Bytes(), sig))
		}
	}

}

func writeGroup(p *Processor) {

}
