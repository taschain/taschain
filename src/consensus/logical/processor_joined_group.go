package logical

import (
	"sync"
	"consensus/groupsig"
	"log"
	"encoding/json"
	"io/ioutil"
	"os"
	"sync/atomic"
	"consensus/model"
)

/*
**  Creator: pxf
**  Date: 2018/6/27 上午9:53
**  Description: 
*/
//当前节点参与的铸块组（已初始化完成）
type JoinedGroup struct {
	GroupID groupsig.ID          //组ID
	SeedKey groupsig.Seckey      //（组相关性的）私密私钥
	SignKey groupsig.Seckey      //矿工签名私钥
	GroupPK groupsig.Pubkey      //组公钥（backup,可以从全局组上拿取）
	Members groupsig.PubkeyMapID //组成员签名公钥
}


func (jg *JoinedGroup) Init() {
	jg.Members = make(groupsig.PubkeyMapID, 0)
}

//取得组内某个成员的签名公钥
func (jg JoinedGroup) GetMemSignPK(mid groupsig.ID) groupsig.Pubkey {
	return jg.Members[mid.GetHexString()]
}

type BelongGroups struct {
	groups sync.Map	//idHex -> *JoinedGroup
	storeFile string
	dirty	int32
}

func NewBelongGroups(file string) *BelongGroups {
	return &BelongGroups{
		groups: sync.Map{},
		storeFile: file,
		dirty: 0,
	}
}

func (bg *BelongGroups) commit() bool {
	if atomic.LoadInt32(&bg.dirty) != 1 {
		return false
	}
	if bg.groupSize() == 0 {
		return false
	}
	log.Printf("belongGroups commit to file, %v, size %v\n", bg.storeFile, bg.groupSize())
    gs := make([]*JoinedGroup, 0)
    bg.groups.Range(func(key, value interface{}) bool {
		jg := value.(*JoinedGroup)
		gs = append(gs, jg)
		return true
	})
	buf, err := json.Marshal(gs)
	if err != nil {
		panic("BelongGroups::store Marshal failed ." + err.Error())
	}
	err = ioutil.WriteFile(bg.storeFile, buf, os.ModePerm)
	if err != nil {
		log.Println("store belongGroups fail", err)
		return false
	}
	atomic.CompareAndSwapInt32(&bg.dirty, 1, 0)
	return true
}

func (bg *BelongGroups) load() bool {
	log.Println("load belongGroups from", bg.storeFile)
    data, err := ioutil.ReadFile(bg.storeFile)
	if err != nil {
		log.Printf("load file %v fail, err %v", bg.storeFile, err.Error())
		return false
	}
	var gs []*JoinedGroup
	err = json.Unmarshal(data, &gs)
	if err != nil {
		log.Printf("unmarshal belongGroup store file %v fail, err %v", bg.storeFile, err.Error())
		return false
	}
	log.Println("load belongGroups size", bg.groupSize())
	for _, jg := range gs {
		bg.groups.Store(jg.GroupID.GetHexString(), jg)
	}
	return true
}

func (bg *BelongGroups) getJoinedGroup(id groupsig.ID) *JoinedGroup {
    if ret, ok := bg.groups.Load(id.GetHexString()); ok {
    	return ret.(*JoinedGroup)
	}
	return nil
}

func (bg *BelongGroups) groupSize() int32 {
	size := int32(0)
	bg.groups.Range(func(key, value interface{}) bool {
		size ++
		return true
	})
	return size
}

func (bg *BelongGroups) getAllGroups() map[string]JoinedGroup {
    m := make(map[string]JoinedGroup)
    bg.groups.Range(func(key, value interface{}) bool {
		m[key.(string)] = *(value.(*JoinedGroup))
		return true
	})
    return m
}

func (bg *BelongGroups) addJoinedGroup(jg *JoinedGroup) {
	newBizLog("addJoinedGroup").log("add gid=%v", jg.GroupID.ShortS())
	bg.groups.Store(jg.GroupID.GetHexString(), jg)
	atomic.CompareAndSwapInt32(&bg.dirty, 0, 1)
}

func (bg *BelongGroups) leaveGroups(gids []groupsig.ID)  {
	for _, gid := range gids {
		bg.groups.Delete(gid.GetHexString())
		atomic.CompareAndSwapInt32(&bg.dirty, 0, 1)
	}
	bg.commit()
}

//取得组内成员的签名公钥
func (p Processor) GetMemberSignPubKey(gmi *model.GroupMinerID) (pk groupsig.Pubkey) {
	if jg := p.belongGroups.getJoinedGroup(gmi.Gid); jg != nil {
		pk = jg.GetMemSignPK(gmi.Uid)
	}
	return
}

//取得组内自身的私密私钥（正式版本不提供）
// deprecated
func (p Processor) getGroupSeedSecKey(gid groupsig.ID) (sk groupsig.Seckey) {
	if jg := p.belongGroups.getJoinedGroup(gid); jg != nil {
		sk = jg.SeedKey
	}
	return
}


//加入一个组（一个矿工ID可以加入多个组）
//gid : 组ID(非dummy id)
//sk：用户的组成员签名私钥
func (p *Processor) joinGroup(g *JoinedGroup, save bool) {
	log.Printf("begin Processor(%v)::joinGroup, gid=%v...\n", p.getPrefix(), g.GroupID.ShortS())
	if !p.IsMinerGroup(g.GroupID) {
		p.belongGroups.addJoinedGroup(g)
		if save {
			p.belongGroups.commit()
		}
	}
	return
}

//取得矿工在某个组的签名私钥
func (p Processor) getSignKey(gid groupsig.ID) groupsig.Seckey {
	if jg := p.belongGroups.getJoinedGroup(gid); jg != nil {
		return jg.SignKey
	}
	return groupsig.Seckey{}
}

//检测某个组是否矿工的铸块组（一个矿工可以参与多个组）
func (p *Processor) IsMinerGroup(gid groupsig.ID) bool {
	return p.belongGroups.getJoinedGroup(gid) != nil
}


//取得矿工参与的所有铸块组私密私钥，正式版不提供
func (p Processor) getMinerGroups() map[string]JoinedGroup {
	return p.belongGroups.getAllGroups()
}
