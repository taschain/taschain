package logical

import (
	"testing"
	"log"
	"encoding/json"
	"common"
	"fmt"
	"consensus/model"
	"middleware"
	"consensus/base"
	"consensus/groupsig"
	"strconv"
)

const CONF_PATH_PREFIX = `/Users/daijia/code/tas/deploy/daily`

func TestBelongGroups(t *testing.T) {
	//groupsig.Init(1)
	common.InitConf(CONF_PATH_PREFIX + "/tas1.ini")
	InitConsensus()
	belongs := NewBelongGroups("/Users/pxf/workspace/tas_develop/tas/conf/aliyun/joined_group.config.1")
	belongs.load()
	gs := belongs.getAllGroups()
	for _, g := range gs {
		log.Println(g.GroupID.ShortS())
	}
	t.Log(belongs)
}

func initProcessor(conf string) *Processor {
	cm := common.NewConfINIManager(conf)
	proc := new(Processor)
	proc.Init(model.NewSelfMinerDO(cm.GetString("gtas", "secret", "")))
	log.Printf("%v", proc.mi.VrfPK)
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

func TestLoadGenesisGroup(t *testing.T) {
	file := CONF_PATH_PREFIX + "/genesis_sgi.config"
	gg := genGenesisStaticGroupInfo(file)

	json, _ := json.Marshal(gg)
	t.Log(string(json))
}

func TestNewID(t *testing.T) {
	var mi model.SelfMinerDO

	for i := 0; i < 100; i++ {

		mi.SecretSeed = base.RandFromString(strconv.Itoa(i))
		mi.SK = *groupsig.NewSeckeyFromRand(mi.SecretSeed)
		mi.PK = *groupsig.NewPubkeyFromSeckey(mi.SK)
		mi.ID = *groupsig.NewIDFromPubkey(mi.PK)

		fmt.Println(mi.ID.ShortS())
	}

}
