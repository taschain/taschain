package logical

import (
	"crypto/rand"
	"encoding/json"
	"github.com/hashicorp/golang-lru"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/groupsig"
	"github.com/taschain/taschain/consensus/model"
	"github.com/taschain/taschain/storage/tasdb"
	"io/ioutil"
	"os"
	"strings"
	"sync"
)

/*
**  Creator: pxf
**  Date: 2018/6/27 上午9:53
**  Description:
 */

const (
	suffixSignKey = "_signKey"
	suffixGInfo   = "_gInfo"
)

//当前节点参与的铸块组（已初始化完成）
type JoinedGroup struct {
	GroupID groupsig.ID //组ID
	//SeedKey groupsig.Seckey    //（组相关性的）私密私钥
	SignKey groupsig.Seckey    //矿工签名私钥
	GroupPK groupsig.Pubkey    //组公钥（backup,可以从全局组上拿取）
	Members groupsig.PubkeyMap //组成员签名公钥
	gHash   common.Hash
	lock    sync.RWMutex
}

type joinedGroupStore struct {
	GroupID groupsig.ID        //组ID
	GroupPK groupsig.Pubkey    //组公钥（backup,可以从全局组上拿取）
	Members groupsig.PubkeyMap //组成员签名公钥
}

func newJoindGroup(n *GroupNode, gHash common.Hash) *JoinedGroup {
	gpk := n.GetGroupPubKey()
	joinedGroup := &JoinedGroup{
		GroupPK: gpk,
		SignKey: n.getSignSecKey(),
		//Members: n.memberPubKeys,
		Members: make(groupsig.PubkeyMap, 0),
		GroupID: *groupsig.NewIDFromPubkey(gpk),
		//SeedKey: n.minerGroupSecret.GenSecKey(),
		gHash: gHash,
	}
	return joinedGroup
}

func signKeySuffix(gid groupsig.ID) []byte {
	return []byte(gid.GetHexString() + suffixSignKey)
}

func gInfoSuffix(gid groupsig.ID) []byte {
	return []byte(gid.GetHexString() + suffixGInfo)
}

func (jg *JoinedGroup) addMemSignPK(mem groupsig.ID, signPK groupsig.Pubkey) {
	jg.lock.Lock()
	defer jg.lock.Unlock()
	//stdLogger.Debugf("addMemSignPK %v %v", mem.GetHexString(), signPK.GetHexString())
	jg.Members[mem.GetHexString()] = signPK
}

func (jg *JoinedGroup) memSignPKSize() int {
	jg.lock.RLock()
	defer jg.lock.RUnlock()
	return len(jg.Members)
}

//取得组内某个成员的签名公钥
func (jg *JoinedGroup) getMemSignPK(mid groupsig.ID) (pk groupsig.Pubkey, ok bool) {
	jg.lock.RLock()
	defer jg.lock.RUnlock()
	pk, ok = jg.Members[mid.GetHexString()]
	return
}

func (jg *JoinedGroup) getMemberMap() groupsig.PubkeyMap {
	jg.lock.RLock()
	defer jg.lock.RUnlock()
	m := make(groupsig.PubkeyMap, 0)
	for key, pk := range jg.Members {
		m[key] = pk
	}
	return m
}

type BelongGroups struct {
	//groups    sync.Map //idHex -> *JoinedGroup
	cache *lru.Cache
	//storeFile string
	priKey   common.PrivateKey
	dirty    int32
	store    *tasdb.LDBDatabase
	storeDir string
	initMu   sync.Mutex
}

func NewBelongGroups(file string, priKey common.PrivateKey) *BelongGroups {
	return &BelongGroups{
		//cache: 		cache,
		//store: 		db,
		dirty:    0,
		priKey:   priKey,
		storeDir: file,
	}
}

func (bg *BelongGroups) initStore() {
	bg.initMu.Lock()
	defer bg.initMu.Unlock()

	if bg.ready() {
		return
	}
	db, err := tasdb.NewLDBDatabase(bg.storeDir, 1, 1)
	if err != nil {
		stdLogger.Errorf("newLDBDatabase fail, file=%v, err=%v\n", bg.storeDir, err.Error())
		return
	}

	bg.store = db
	bg.cache = common.MustNewLRUCache(30)
}

func (bg *BelongGroups) ready() bool {
	return bg.cache != nil && bg.store != nil
}

func (bg *BelongGroups) storeSignKey(jg *JoinedGroup) {
	if !bg.ready() {
		return
	}
	pubKey := bg.priKey.GetPubKey()
	ct, err := pubKey.Encrypt(rand.Reader, jg.SignKey.Serialize())
	if err != nil {
		stdLogger.Errorf("encrypt signkey fail, err=%v", err.Error())
		return
	}
	bg.store.Put(signKeySuffix(jg.GroupID), ct)
}

func (bg *BelongGroups) storeGroupInfo(jg *JoinedGroup) {
	if !bg.ready() {
		return
	}
	st := joinedGroupStore{
		GroupID: jg.GroupID,
		GroupPK: jg.GroupPK,
		Members: jg.getMemberMap(),
	}
	bs, err := json.Marshal(st)
	if err != nil {
		stdLogger.Errorf("marshal joinedGroup fail, err=%v", err)
	} else {
		bg.store.Put(gInfoSuffix(jg.GroupID), bs)
	}
}

func (bg *BelongGroups) storeJoinedGroup(jg *JoinedGroup) {
	bg.storeSignKey(jg)
	bg.storeGroupInfo(jg)
}

func (bg *BelongGroups) loadJoinedGroup(gid groupsig.ID) *JoinedGroup {
	if !bg.ready() {
		return nil
	}
	jg := new(JoinedGroup)
	jg.Members = make(groupsig.PubkeyMap, 0)
	//加载签名私钥
	bs, err := bg.store.Get(signKeySuffix(gid))
	if err != nil {
		stdLogger.Errorf("get signKey fail, gid=%v, err=%v", gid.ShortS(), err.Error())
		return nil
	}
	m, err := bg.priKey.Decrypt(rand.Reader, bs)
	if err != nil {
		stdLogger.Errorf("decrypt signKey fail, err=%v", err.Error())
		return nil
	}
	jg.SignKey.Deserialize(m)

	//加载组信息
	infoBytes, err := bg.store.Get(gInfoSuffix(gid))
	if err != nil {
		stdLogger.Errorf("get gInfo fail, gid=%v, err=%v", gid.ShortS(), err.Error())
		return jg
	}
	if err := json.Unmarshal(infoBytes, jg); err != nil {
		stdLogger.Errorf("unmarsal gInfo fail, gid=%v, err=%v", gid.ShortS(), err.Error())
		return jg
	}
	return jg
}

//
//func (bg *BelongGroups) commit() bool {
//	if atomic.LoadInt32(&bg.dirty) != 1 {
//		return false
//	}
//	if bg.groupSize() == 0 {
//		return false
//	}
//	stdLogger.Debugf("belongGroups commit to file, %v, size %v\n", bg.storeFile, bg.groupSize())
//	gs := make([]*JoinedGroup, 0)
//	bg.groups.Range(func(key, value interface{}) bool {
//		jg := value.(*JoinedGroup)
//		gs = append(gs, jg)
//		return true
//	})
//	buf, err := json.Marshal(gs)
//	if err != nil {
//		panic("BelongGroups::store Marshal failed ." + err.Error())
//	}
//	err = ioutil.WriteFile(bg.storeFile, buf, os.ModePerm)
//	if err != nil {
//		stdLogger.Errorf("store belongGroups fail", err)
//		return false
//	}
//	atomic.CompareAndSwapInt32(&bg.dirty, 1, 0)
//	return true
//}
//

func fileExists(f string) bool {
	_, err := os.Stat(f) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
func isDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func (bg *BelongGroups) joinedGroup2DBIfConfigExists(file string) bool {
	if !fileExists(file) || isDir(file) {
		return false
	}
	stdLogger.Debugf("load belongGroups from %v", file)
	data, err := ioutil.ReadFile(file)
	if err != nil {
		stdLogger.Debugf("load file %v fail, err %v", file, err.Error())
		return false
	}
	var gs []*JoinedGroup
	err = json.Unmarshal(data, &gs)
	if err != nil {
		stdLogger.Debugf("unmarshal belongGroup store file %v fail, err %v", file, err.Error())
		return false
	}
	n := 0
	bg.initStore()
	for _, jg := range gs {
		//for idStr, pk := range jg.Members {
		//	var id groupsig.ID
		//	id.SetHexString(idStr)
		//	jg.Members[id.GetHexString()] = pk
		//}
		if bg.getJoinedGroup(jg.GroupID) == nil {
			n++
			bg.addJoinedGroup(jg)
		}
	}
	stdLogger.Debugf("joinedGroup2DBIfConfigExists belongGroups size %v", n)
	return true
}

func (bg *BelongGroups) getJoinedGroup(id groupsig.ID) *JoinedGroup {
	if !bg.ready() {
		bg.initStore()
	}
	v, ok := bg.cache.Get(id.GetHexString())
	if ok {
		return v.(*JoinedGroup)
	}
	jg := bg.loadJoinedGroup(id)
	if jg != nil {
		bg.cache.Add(jg.GroupID.GetHexString(), jg)
	}
	return jg
}

//func (bg *BelongGroups) groupSize() int32 {
//	size := int32(0)
//	bg.groups.Range(func(key, value interface{}) bool {
//		size ++
//		return true
//	})
//	return size
//}
//
//func (bg *BelongGroups) getAllGroups() map[string]JoinedGroup {
//	m := make(map[string]JoinedGroup)
//	bg.groups.Range(func(key, value interface{}) bool {
//		m[key.(string)] = *(value.(*JoinedGroup))
//		return true
//	})
//	return m
//}

func (bg *BelongGroups) addMemSignPk(uid groupsig.ID, gid groupsig.ID, signPK groupsig.Pubkey) (*JoinedGroup, bool) {
	if !bg.ready() {
		bg.initStore()
	}
	jg := bg.getJoinedGroup(gid)
	//stdLogger.Debugf("getJoinedGroup gid=%v", gid.ShortS())
	//for mem, pk := range jg.getMemberMap() {
	//	stdLogger.Debugf("getJoinedGroup: %v, %v", mem, pk.GetHexString())
	//}
	if jg == nil {
		return nil, false
	}

	if _, ok := jg.getMemSignPK(uid); !ok {
		jg.addMemSignPK(uid, signPK)
		bg.storeGroupInfo(jg)
		return jg, true
	}
	return jg, false
}

func (bg *BelongGroups) addJoinedGroup(jg *JoinedGroup) {
	if !bg.ready() {
		bg.initStore()
	}
	newBizLog("addJoinedGroup").log("add gid=%v", jg.GroupID.ShortS())
	//bg.groups.Store(jg.GroupID.GetHexString(), jg)
	bg.cache.Add(jg.GroupID.GetHexString(), jg)
	bg.storeJoinedGroup(jg)
}

func (bg *BelongGroups) leaveGroups(gids []groupsig.ID) {
	if !bg.ready() {
		return
	}
	for _, gid := range gids {
		//bg.groups.Delete(gid.GetHexString())
		bg.cache.Remove(gid.GetHexString())
		//bg.store.Delete(signKeySuffix(gid))
		bg.store.Delete(gInfoSuffix(gid))
	}
}

func (bg *BelongGroups) close() {
	if !bg.ready() {
		return
	}
	bg.cache = nil
	bg.store.Close()
}

func (p *Processor) genBelongGroupStoreFile() string {
	storeFile := p.conf.GetString(ConsensusConfSection, "groupstore", "")
	if strings.TrimSpace(storeFile) == "" {
		storeFile = "groupstore" + p.conf.GetString("instance", "index", "")
	}
	return storeFile
}

//取得组内成员的签名公钥
func (p Processor) GetMemberSignPubKey(gmi *model.GroupMinerID) (pk groupsig.Pubkey, ok bool) {
	if jg := p.belongGroups.getJoinedGroup(gmi.Gid); jg != nil {
		pk, ok = jg.getMemSignPK(gmi.UID)
		if !ok && !p.GetMinerID().IsEqual(gmi.UID) {
			p.askSignPK(gmi)
		}
	}
	return
}

//取得组内自身的私密私钥（正式版本不提供）
// deprecated
//func (p Processor) getGroupSeedSecKey(gid groupsig.ID) (sk groupsig.Seckey) {
//	if jg := p.belongGroups.getJoinedGroup(gid); jg != nil {
//		sk = jg.SeedKey
//	}
//	return
//}

//加入一个组（一个矿工ID可以加入多个组）
//gid : 组ID(非dummy id)
//sk：用户的组成员签名私钥
func (p *Processor) joinGroup(g *JoinedGroup) {
	stdLogger.Debugf("begin Processor(%v)::joinGroup, gid=%v...\n", p.getPrefix(), g.GroupID.ShortS())
	if !p.IsMinerGroup(g.GroupID) {
		p.belongGroups.addJoinedGroup(g)
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

func (p *Processor) askSignPK(gmi *model.GroupMinerID) {
	if !addSignPkReq(gmi.UID) {
		return
	}
	msg := &model.ConsensusSignPubkeyReqMessage{
		GroupID: gmi.Gid,
	}
	ski := model.NewSecKeyInfo(p.GetMinerID(), p.mi.GetDefaultSecKey())
	if msg.GenSign(ski, msg) {
		newBizLog("AskSignPK").log("ask sign pk message, receiver %v, gid %v", gmi.UID.ShortS(), gmi.Gid.ShortS())
		p.NetServer.AskSignPkMessage(msg, gmi.UID)
	}
}
