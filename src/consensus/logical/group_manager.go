package logical

import (
	"consensus/groupsig"
	"log"
	"time"
	"core"
	"consensus/rand"
	"middleware/types"
	"math/big"
	"fmt"
	"common"
)

/*
**  Creator: pxf
**  Date: 2018/6/23 下午4:07
**  Description: 组生命周期, 包括建组, 解散组
*/

const (
	EPOCH uint64 = 4
	CHECK_CREATE_GROUP_HEIGHT_AFTER uint64 = 10	//启动建组的块高度差
	MINER_MAX_JOINED_GROUP = 5	//一个矿工最多加入的组数
	CANDIDATES_MIN_RATIO = 1	//最小的候选人相对于组成员数量的倍数

	GROUP_GET_READY_GAP = EPOCH	//组准备就绪(建成组)的间隔为1个epoch
	GROUP_CAST_QUALIFY_GAP = EPOCH * 5	//组准备就绪后, 等待可以铸块的间隔为4个epoch
	GROUP_CAST_DURATION = EPOCH * 100	//组铸块的周期为100个epoch
)

type GroupManager struct {
	groupChain     *core.GroupChain
	mainChain      core.BlockChainI
	processor      *Processor
	creatingGroups *CreatingGroups
}

func NewGroupManager(processor *Processor) *GroupManager {
	return &GroupManager{
		processor:      processor,
		mainChain:      processor.MainChain,
		groupChain:     processor.GroupChain,
		creatingGroups: &CreatingGroups{},
	}
}

func (gm *GroupManager) addCreatingGroup(group *CreatingGroup)  {
    gm.creatingGroups.addCreatingGroup(group)
}

func (gm *GroupManager) removeCreatingGroup(id groupsig.ID)  {
    gm.creatingGroups.removeGroup(id)
}


//检查当前用户是否是属于建组的组, 返回组id
func (gm *GroupManager) checkCreateGroup(topHeight uint64) (create bool, sgi *StaticGroupInfo, theBH *types.BlockHeader) {
	defer func() {
		log.Printf("checkCreateNextGroup topHeight=%v, create %v\n", topHeight, create)
	}()
	blockHeight := topHeight - CHECK_CREATE_GROUP_HEIGHT_AFTER
	if blockHeight % EPOCH != 0 {
		return
	}
	theBH = gm.mainChain.QueryBlockByHeight(blockHeight)
	if theBH == nil || theBH.GroupId == nil || len(theBH.GroupId) == 0 {
		create = false
		return
	}

	castGID := groupsig.DeserializeId(theBH.GroupId)
	if gm.processor.IsMinerGroup(*castGID) {
		sgi = gm.processor.getGroup(*castGID)
		if sgi.CastQualified(topHeight) {
			log.Printf("checkCreateNextGroup, topHeight=%v, theBH height=%v, king=%v\n", topHeight, theBH.Height, gm.processor.getPrefix())
			create = true
			return
		}
	}

	create = false
	return
}


//检查当前用户是否是建组发起人
func (gm *GroupManager) checkKing(bh *types.BlockHeader, group *StaticGroupInfo) groupsig.ID {
	secret := gm.processor.getGroupSecret(group.GroupID)
	if secret == nil {
		return groupsig.ID{}
	}
	data := secret.SecretSign
	data = append(data, bh.Signature...)
	hash := rand.Data2CommonHash(data)
	biHash := hash.Big()

	var index int32 = -1
	mem := len(group.Members)
	if biHash.BitLen() > 0 {
		index = int32(biHash.Mod(biHash, big.NewInt(int64(mem))).Int64())
	}
	log.Printf("checkCreateNextGroup king index=%v, id=%v\n", index, GetIDPrefix(group.GetCastor(int(index))))
	if index < 0 {
		return groupsig.ID{}
	}
	return group.GetCastor(int(index))
}

//todo 从链上获取所有候选者
func (gm *GroupManager) getAllCandidates() []groupsig.ID {
    memBytes, _ := gm.groupChain.GetCandidates()
    ids := make([]groupsig.ID, 0)
	for _, mem := range memBytes {
		id := groupsig.DeserializeId(mem)
		ids = append(ids, *id)
	}
	str := ""
	for _, id := range ids {
		str += GetIDPrefix(id) + ","
	}
	log.Printf("=============getAllCandidates %v\n", str)
	//sgi := gm.processor.globalGroups.groups[0]
	//for _, mem := range sgi.Members {
	//	ids = append(ids, mem.ID)
	//}
	return ids
}

func (gm *GroupManager) selectCandidates(randSeed common.Hash, height uint64) (bool, []groupsig.ID) {
	allCandidates := gm.getAllCandidates()
	groups := gm.processor.GetAvailableGroupsAt(height)
	log.Printf("selectCandidates available groupsize %v\n", len(groups))

	candidates := make([]groupsig.ID, 0)
	for _, id := range allCandidates {
		joinedNum := 0
		for _, g := range groups {
			if g.MemExist(id) {
				joinedNum++
			}
		}
		if joinedNum <= MINER_MAX_JOINED_GROUP {
			candidates = append(candidates, id)
		}
	}
	min := GetGroupMemberNum()*CANDIDATES_MIN_RATIO
	num := len(candidates)
	if len(candidates) < min {
		log.Printf("selectCandidates not enough candidates, expect %v, got %v\n", min, num)
		return false, []groupsig.ID{}
	}

	rand := rand.RandFromBytes(randSeed.Bytes())
	seqs := rand.RandomPerm(num, GetGroupMemberNum())

	result := make([]groupsig.ID, len(seqs))
	for i := 0; i < len(result); i++ {
		result[i] = candidates[seqs[i]]
	}
	str := ""
	for _, id := range result {
		str += GetIDPrefix(id) + ","
	}
	log.Printf("=============selectCandidates %v\n", str)
	return true, result
}

func (gm *GroupManager) getPubkeysByIds(ids []groupsig.ID) []groupsig.Pubkey {
	pubs := make([]groupsig.Pubkey, 0)

	idBytes := make([][]byte, 0)
	for _, id := range ids {
		idBytes = append(idBytes, id.Serialize())
	}

	pubBytes := gm.groupChain.GetMemberPubkeyByIDs(idBytes)
	log.Printf("=============getPubkeyByIds idsize %v, puksize %v\n", len(idBytes), len(pubBytes))
	for idx, pbyte := range pubBytes {
		if pbyte == nil || len(pbyte) == 0 {
			s := fmt.Sprintf("get pubkey bytes failed, idbytes=%v, id=%v, ret=%v", idBytes[idx], GetIDPrefix(ids[idx]), pbyte)
			panic(s)
		}
		pk := groupsig.DeserializePubkeyBytes(pbyte)
		if pk == nil {
			s := fmt.Sprintf("deserialize pubkey bytes failed, bytes=%v", pbyte)
			panic(s)
		}
		pubs = append(pubs, *pk)
	}
    return pubs
}

func (gm *GroupManager) CreateNextGroupRoutine() {
	topBH := gm.mainChain.QueryTopBlock()
	topHeight := topBH.Height

	create, group, bh := gm.checkCreateGroup(topHeight)
	//不是当前组铸
	if !create {
		return
	}
	castor := gm.checkKing(bh, group)
	//不是当期用户铸
	if !castor.IsEqual(gm.processor.GetMinerID()) {
		return
	}

	var gis ConsensusGroupInitSummary

	gis.ParentID = group.GroupID

	gn := rand.Data2CommonHash([]byte(fmt.Sprintf("%s-%v", group.GroupID.GetHexString(), bh.Height))).Str()
	gis.DummyID = *groupsig.NewIDFromString(gn)

	if gm.groupChain.GetGroupById(gis.DummyID.Serialize()) != nil {
		log.Printf("CreateNextGroupRoutine ingored, dummyId already onchain! dummyId=%v\n", GetIDPrefix(gis.DummyID))
		return
	}

	log.Printf("CreateNextGroupRoutine, group name=%v, group dummy id=%v.\n", gn, GetIDPrefix(gis.DummyID))
	gis.Authority = 777
	if len(gn) <= 64 {
		copy(gis.Name[:], gn[:])
	} else {
		copy(gis.Name[:], gn[:64])
	}
	gis.BeginTime = time.Now()
	gis.TopHeight = topHeight
	gis.GetReadyHeight = topHeight + GROUP_GET_READY_GAP
	gis.BeginCastHeight = gis.GetReadyHeight + GROUP_CAST_QUALIFY_GAP
	gis.DismissHeight = gis.BeginCastHeight + GROUP_CAST_DURATION

	if !gis.ParentID.IsValid() || !gis.DummyID.IsValid() {
		panic("create group init summary failed")
	}
	gis.Members = uint64(GetGroupMemberNum())
	gis.Extends = "Dummy"

	randSeed := rand.Data2CommonHash(bh.Signature)
	enough, memIds := gm.selectCandidates(randSeed, topHeight)
	if !enough {
		return
	}
	gis.MemberHash = genMemberHashByIds(memIds)

	msg := ConsensusCreateGroupRawMessage{
		GI: gis,
		IDs: memIds,
	}
	msg.GenSign(SecKeyInfo{ID: gm.processor.GetMinerID(), SK: gm.processor.getSignKey(group.GroupID)}, &msg)

	creatingGroup := newCreateGroup(&gis, memIds)
	gm.addCreatingGroup(creatingGroup)

	log.Printf("proc(%v) start Create Group consensus, send network msg to members...\n", gm.processor.getPrefix())
	log.Printf("call network service SendCreateGroupRawMessage...\n")

	SendCreateGroupRawMessage(&msg)
}

func (gm *GroupManager) OnMessageCreateGroupRaw(msg *ConsensusCreateGroupRawMessage) bool {
	log.Printf("OnMessageCreateGroupRaw dummyId=%v, sender=%v\n", GetIDPrefix(msg.GI.DummyID), GetIDPrefix(msg.SI.SignMember))
	gis := &msg.GI
	if gis.GenHash() != msg.SI.DataHash {
		log.Printf("ConsensusCreateGroupRawMessage hash diff\n")
		return false
	}

	memHash := genMemberHashByIds(msg.IDs)
	if memHash != gis.MemberHash {
		log.Printf("ConsensusCreateGroupRawMessage memberHash diff\n")
		return false
	}
	create, group, bh := gm.checkCreateGroup(gis.TopHeight)
	if !create {
		log.Printf("ConsensusCreateGroupRawMessage current group is not the next CastGroup!")
		return false
	}
	castor := gm.checkKing(bh, group)
	if !castor.IsEqual(msg.SI.SignMember) {
		log.Printf("ConsensusCreateGroupRawMessage not the user for casting! expect user is %v, receive user is %v\n", GetIDPrefix(castor), GetIDPrefix(msg.SI.SignMember))
		return false
	}

	randSeed := rand.Data2CommonHash(bh.Signature)
	enough, memIds := gm.selectCandidates(randSeed, gis.TopHeight)
	if !enough {
		return false
	}
	if len(memIds) != len(msg.IDs) {
		log.Printf("ConsensusCreateGroupRawMessage member len not equal, expect len %v, receive len %v\n", len(memIds), len(msg.IDs))
		return  false
	}

	for idx, id := range memIds {
		if !id.IsEqual(msg.IDs[idx]) {
			log.Printf("ConsensusCreateGroupRawMessage member diff [%v, %v]", GetIDPrefix(id), GetIDPrefix(msg.IDs[idx]))
			return  false
		}
	}
	return true

}

func (gm *GroupManager) OnMessageCreateGroupSign(msg *ConsensusCreateGroupSignMessage) bool {
	log.Printf("ConsensusCreateGroupSignMessage dummyId=%v, sender=%v\n", GetIDPrefix(msg.GI.DummyID), GetIDPrefix(msg.SI.SignMember))
	gis := &msg.GI
	if gis.GenHash() != msg.SI.DataHash {
		log.Printf("OnMessageCreateGroupSign hash diff\n")
		return false
	}

	creating := gm.creatingGroups.getCreatingGroup(gis.DummyID)
	if creating == nil {
		log.Printf("OnMessageCreateGroupSign get creating group nil!\n")
		return false
	}

	memHash := genMemberHashByIds(creating.ids)
	if memHash != gis.MemberHash {
		log.Printf("OnMessageCreateGroupSign memberHash diff\n")
		return false
	}

	if gis.IsExpired() {
		log.Printf("OnMessageCreateGroupSign gis expired!\n")
		return false
	}
	accept := gm.creatingGroups.acceptPiece(gis.DummyID, msg.SI.SignMember, msg.SI.DataSign)
	if accept == PIECE_THRESHOLD {
		sign := groupsig.RecoverSignatureByMapI(creating.pieces, creating.threshold)
		msg.GI.Signature = *sign
		return true
	}
	return false
}

func (gm *GroupManager) AddGroupOnChain(sgi *StaticGroupInfo, isDummy bool)  {
	group := ConvertStaticGroup2CoreGroup(sgi, isDummy)
	err := gm.groupChain.AddGroup(group, nil, nil)
	if err != nil {
		log.Printf("ERROR:add group fail! isDummy=%v, dummyId=%v, err=%v\n", isDummy, GetIDPrefix(sgi.GIS.DummyID), err.Error())
		return
	}
	if isDummy {
		log.Printf("AddGroupOnChain success, dummyId=%v, height=%v\n", GetIDPrefix(sgi.GIS.DummyID), gm.groupChain.Count())
	} else {
		log.Printf("AddGroupOnChain success, ID=%v, height=%v\n", GetIDPrefix(sgi.GroupID), gm.groupChain.Count())
	}
}