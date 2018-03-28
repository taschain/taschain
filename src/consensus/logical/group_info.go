package logical

import (
	"consensus/groupsig"
	"fmt"
	"math/big"
	"net"

	"common"
)

type MemberInfo struct {
	pubkey groupsig.Pubkey //组内成员签名公钥
}

//静态组结构（组创建即确定）
type StaticGroupInfo struct {
	group_id  groupsig.ID            //组ID
	group_pk  groupsig.Pubkey        //组公钥
	members   []MemberInfo           //组内成员的静态信息(严格按照链上次序，全网一致，不然影响组铸块)
	map_mid_i map[groupsig.ID]uint32 //用ID查找成员信息
}

func (sgi StaticGroupInfo) MemExist(uid groupsig.ID) bool {
	_, ok := sgi.map_mid_i[uid]
	return ok
}

func (sgi StaticGroupInfo) GetMember(uid groupsig.ID) (m MemberInfo, result bool) {
	var i uint32
	i, result = sgi.map_mid_i[uid]
	if result {
		m = sgi.members[i]
	}
	return
}

//取得某个成员在组内的排位
func (sgi StaticGroupInfo) GetPosition(uid groupsig.ID) int32 {
	i, ok := sgi.map_mid_i[uid]
	if ok {
		return int32(i)
	} else {
		return int32(-1)
	}
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
	sgi       []StaticGroupInfo
	map_gid_i map[groupsig.ID]uint32 //用ID查找组信息
}

func (gg GlobalGroups) GetPosition(gid groupsig.ID) int32 {
	i, ok := gg.map_gid_i[gid]
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
	index, ok := gg.map_gid_i[id]
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
		ga = gg.sgi[index.Uint64()].group_id
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
	result = g.group_pk == pub_key
	return
}
