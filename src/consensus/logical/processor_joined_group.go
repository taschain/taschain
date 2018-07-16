package logical

import (
	"sync"
	"consensus/groupsig"
	"log"
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
	GroupSec GroupSecret
}


func (jg *JoinedGroup) Init() {
	jg.Members = make(groupsig.PubkeyMapID, 0)
}

//取得组内某个成员的签名公钥
func (jg JoinedGroup) GetMemSignPK(mid groupsig.ID) groupsig.Pubkey {
	return jg.Members[mid.GetHexString()]
}

func (jg *JoinedGroup) setGroupSecretHeight(height uint64)  {
	jg.GroupSec.EffectHeight = height
}

type BelongGroups struct {
	groups sync.Map	//idHex -> *JoinedGroup
}

func NewBelongGroups() *BelongGroups {
	return &BelongGroups{
		groups: sync.Map{},
	}
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
	bg.groups.Store(jg.GroupID.GetHexString(), jg)
}

func (bg *BelongGroups) leaveGroups(gids []groupsig.ID)  {
	for _, gid := range gids {
		bg.groups.Delete(gid.GetHexString())
	}
}

//取得组内成员的签名公钥
func (p Processor) GetMemberSignPubKey(gmi GroupMinerID) (pk groupsig.Pubkey) {
	if jg := p.belongGroups.getJoinedGroup(gmi.gid); jg != nil {
		pk = jg.GetMemSignPK(gmi.uid)
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
	log.Printf("begin Processor::joinGroup, gid=%v...\n", GetIDPrefix(g.GroupID))
	if !p.IsMinerGroup(g.GroupID) {
		p.belongGroups.addJoinedGroup(g)
		if save {
			p.saveJoinedGroup(g)
		}
	} else {
		log.Printf("Error::Processor::joinGroup failed, already exist.\n")
	}
	log.Printf("SUCCESS:node=%v inited group=%v, sign key=%v.\n", p.getPrefix(), GetIDPrefix(g.GroupID), GetSecKeyPrefix(g.SignKey))
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

func (p *Processor) getGroupSecret(gid groupsig.ID) *GroupSecret {
	if jg := p.belongGroups.getJoinedGroup(gid); jg != nil {
		return &jg.GroupSec
	} else {
		return nil
	}
}

func (p *Processor) cleanDismissGroupRoutine() bool {
	topHeight := p.MainChain.QueryTopBlock().Height
	ids := p.globalGroups.DismissGroups(topHeight)
	log.Printf("cleanDismissGroupRoutine: clean group %v\n", len(ids))
	p.globalGroups.RemoveGroups(ids)
	p.blockContexts.removeContexts(ids)
	p.belongGroups.leaveGroups(ids)
	for _, gid := range ids {
		p.storage.Delete(gid.Serialize())
	}
	return true
}
