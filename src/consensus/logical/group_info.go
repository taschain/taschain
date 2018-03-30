package logical

import (
	"common"
	"consensus/groupsig"
	"fmt"
	"math/big"
	"net"
)

type MemberInfo struct {
	id groupsig.ID //成员全局唯一ID
	//pubkey groupsig.Pubkey 	//成员全局唯一公钥（哪怕不加入组，该公钥也存在，用于接收交易）组内成员签名公钥
	pk groupsig.Pubkey //成员全局唯一公钥（哪怕不加入组，该公钥也存在，用于接收交易）
}

//组初始化结构
type InitGroupInfo struct {
	members []MemberInfo
}

//集齐所有组成员公钥后，可以聚合出组公钥。
func (igi *InitGroupInfo) UpdateMemberPubKey(id groupsig.ID, pk groupsig.Pubkey) bool {
	if !id.IsValid() || !pk.IsValid() {
		return false
	}
	for i := 0; i < len(igi.members); i++ {
		if igi.members[i].id == id {
			if !igi.members[i].pk.IsValid() {
				igi.members[i].pk = pk
			} else {
				if igi.members[i].pk != pk {
					panic("UpdateMemberPubKey failed, pub key diff.")
				}
			}
			return true
		}
	}
	return false
}

//静态组结构（组创建即确定）
type StaticGroupInfo struct {
	GroupID  groupsig.ID            //组ID
	GroupPK  groupsig.Pubkey        //组公钥
	members  []MemberInfo           //组内成员的静态信息(严格按照链上次序，全网一致，不然影响组铸块)
	mapCache map[groupsig.ID]uint32 //用ID查找成员信息
}

//创建一个未经过组初始化共识的静态组结构（尚未共识生成组私钥、组公钥和组ID）
//输入：组成员ID列表，该ID为组成员的私有ID（由该成员的交易私钥->公开地址处理而来），和组共识无关
func CreateWithRawMembers(mems []MemberInfo) *StaticGroupInfo {
	sgi := new(StaticGroupInfo)
	sgi.members = make([]MemberInfo, GROUP_MAX_MEMBERS)
	sgi.mapCache = make(map[groupsig.ID]uint32, GROUP_MAX_MEMBERS)
	for i := 0; i < len(mems); i++ {
		sgi.members = append(sgi.members, mems[i])
		sgi.mapCache[mems[i].id] = uint32(len(sgi.members)) - 1
	}
	return sgi
}

//按组内排位取得成员ID列表
func (sgi *StaticGroupInfo) GetIDSByOrder() []groupsig.ID {
	ids := make([]groupsig.ID, GROUP_MAX_MEMBERS)
	for i := 0; i < len(sgi.members); i++ {
		ids = append(ids, sgi.members[i].id)
	}
	return ids
}

func (sgi *StaticGroupInfo) Addmember(m MemberInfo) {
	if m.id.IsValid() {
		_, ok := sgi.mapCache[m.id]
		if !ok {
			sgi.members = append(sgi.members, m)
			sgi.mapCache[m.id] = uint32(len(sgi.members)) - 1
		}
	}
}

func (sgi *StaticGroupInfo) CanGroupSing() bool {
	return sgi.GroupPK.IsValid()
}

func (sgi StaticGroupInfo) MemExist(uid groupsig.ID) bool {
	_, ok := sgi.mapCache[uid]
	return ok
}

func (sgi StaticGroupInfo) GetMember(uid groupsig.ID) (m MemberInfo, result bool) {
	var i uint32
	i, result = sgi.mapCache[uid]
	if result {
		m = sgi.members[i]
	}
	return
}

//取得某个成员在组内的排位
func (sgi StaticGroupInfo) GetPosition(uid groupsig.ID) int32 {
	i, ok := sgi.mapCache[uid]
	if ok {
		return int32(i)
	} else {
		return int32(-1)
	}
}

//取得指定位置的铸块人
func (sgi StaticGroupInfo) GetCastor(i int) groupsig.ID {
	var m groupsig.ID
	if i >= 0 && i < len(sgi.members) {
		m = sgi.members[i].id
	}
	return m
}

//动态组结构（运行时变化）
type DynamicGroupInfo struct {
	members map[string]net.TCPAddr //组内成员的网络地址
}

//取得组成员网络地址
func (dgi DynamicGroupInfo) GetNetIP(ma string) string {
	addr, ok := dgi.members[ma]
	if ok {
		return addr.IP.String()
	}
	return ""
}

func (dgi DynamicGroupInfo) GetNetPort(ma string) int32 {
	addr, ok := dgi.members[ma]
	if ok {
		return int32(addr.Port)
	}
	return 0
}

///////////////////////////////////////////////////////////////////////////////
type GlobalGroups struct {
	//全网组的静态信息列表，用slice而不是map是为了求模定位
	sgi      []StaticGroupInfo
	mapCache map[groupsig.ID]uint32 //用ID查找组信息
}

func (gg GlobalGroups) GetPosition(gid groupsig.ID) int32 {
	i, ok := gg.mapCache[gid]
	if ok {
		return int32(i)
	} else {
		return int32(-1)
	}
}

//由index取得组信息
func (gg GlobalGroups) GetGroupByIndex(i int) (g StaticGroupInfo, err error) {
	if i < len(gg.sgi) {
		g = gg.sgi[i]
	} else {
		err = fmt.Errorf("out of range")
	}
	return
}

func (gg GlobalGroups) GetGroupByID(id groupsig.ID) (g StaticGroupInfo, err error) {
	index, ok := gg.mapCache[id]
	if ok {
		g, err = gg.GetGroupByIndex(int(index))
	}
	return
}

//根据上一块哈希值，确定下一块由哪个组铸块
func (gg GlobalGroups) SelectNextGroup(h common.Hash) (groupsig.ID, error) {
	var ga groupsig.ID
	value := h.Big()
	if value.BitLen() > 0 && len(gg.sgi) > 0 {
		index := value.Mod(value, big.NewInt(int64(len(gg.sgi))))
		ga = gg.sgi[index.Uint64()].GroupID
		return ga, nil
	} else {
		return ga, fmt.Errorf("SelectNextGroup failed, arg error.")
	}
}

//取得当前铸块组信息
//pre_hash : 上一个铸块哈希
func (gg GlobalGroups) GetCastGroup(pre_h common.Hash) (g StaticGroupInfo) {
	gid, e := gg.SelectNextGroup(pre_h)
	if e == nil {
		g, e = gg.GetGroupByID(gid)
	}
	return
}

//判断pub_key是否为合法铸块组的公钥
//h：上一个铸块的哈希
func (gg GlobalGroups) IsCastGroup(pre_h common.Hash, pub_key groupsig.Pubkey) (result bool) {
	g := gg.GetCastGroup(pre_h)
	result = g.GroupPK == pub_key
	return
}
