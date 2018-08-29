//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package logical

import (
	"consensus/groupsig"
	"crypto/sha1"
	"fmt"
	"testing"
	"time"

	"common"
	"log"
)

const TEST_MEMBERS = 5
const TEST_THRESHOLD = 3

type test_node_data struct {
	seed     util.Rand         //私密种子
	sk       groupsig.Seckey   //个人私钥(由私密种子和组公开信息衍生生成)
	shares   []groupsig.Seckey //秘密共享接收池
	sign_sk  groupsig.Seckey   //组成员签名私钥（由秘密共享接收池聚合而来）
	pubs     []groupsig.Pubkey //组成员签名公钥接收池
	group_pk groupsig.Pubkey   //组公钥（由组成员签名公钥接收池聚合而来）
}

func (tnd *test_node_data) AddPubPiece(pk groupsig.Pubkey) {
	//fmt.Printf("begin AddPubPiece, piece=%v...\n", pk.GetHexString())
	tnd.pubs = append(tnd.pubs, pk)
	//fmt.Printf("end AddPubPiece, sizeof pieces=%v.\n", len(tnd.pubs))
	return
}

//生成某个成员针对组的私钥，用于生成秘密。
func GenU4GSecKey(us util.Rand, gs util.Rand) *groupsig.Seckey {
	u4g_seed := us.DerivedRand(gs[:])
	return groupsig.NewSeckeyFromRand(u4g_seed.Deri(0))
}

//获得来自某个成员的秘密共享片段
//dest：消息接收者
//src：消息发送者
func SetSharePiece(dest groupsig.ID, node *test_node_data, src groupsig.ID, share groupsig.Seckey, pubs []groupsig.Pubkey) {
	//fmt.Printf("begin SetSharePiece...\n")
	pub_from_v := groupsig.SharePubkey(pubs, dest)    //从公钥验证向量恢复秘密公钥
	pub_from_s := groupsig.NewPubkeyFromSeckey(share) //从秘密私钥萃取秘密公钥
	if pub_from_v.GetHexString() != pub_from_s.GetHexString() {
		fmt.Printf("GetSharePiece failed, two pub key not equal.\n")
		fmt.Printf("Pub key from vvec=%v.\n", pub_from_v.GetHexString())
		fmt.Printf("Pub key from share=%v.\n", pub_from_s.GetHexString())
	} else {
		node.shares = append(node.shares, share)
	}
	//fmt.Printf("end SetSharePiece.\n")
	return
}

//某个成员生成针对全组所有成员的全部秘密共享片段
func GenSharePiece(uid groupsig.ID, info test_node_data, group_seed util.Rand, mems *map[groupsig.ID]test_node_data) (groupsig.SeckeyMapID, []groupsig.Pubkey) {
	//fmt.Printf("\nbegin GenSharePiece, uid=%v.\n", uid.GetHexString())
	shares := make(groupsig.SeckeyMapID)
	//var pubs []groupsig.Pubkey
	//生成当前节点针对组的种子
	u4g_seed := info.seed.DerivedRand(group_seed[:])
	//生成门限个密钥和公钥
	secs := make([]groupsig.Seckey, TEST_THRESHOLD)
	pubs := make([]groupsig.Pubkey, TEST_THRESHOLD)
	for i := 0; i < TEST_THRESHOLD; i++ {
		secs[i] = *groupsig.NewSeckeyFromRand(u4g_seed.Deri(i))
		pubs[i] = *groupsig.NewPubkeyFromSeckey(secs[i])
	}
	/*
		fmt.Printf("begin print THRESHOLD data...\n")
		for i, v := range secs {
			fmt.Printf("sec(%v)=%v.\n", i, v.GetHexString())
		}
		for i, v := range pubs {
			fmt.Printf("pub(%v)=%v.\n", i, v.GetHexString())
		}
		fmt.Printf("end print THRESHOLD data.\n")
	*/
	//生成成员数量个共享秘密
	for k, _ := range *mems { //组成员遍历
		shares[k.GetHexString()] = *groupsig.ShareSeckey(secs, k)
	}
	/*
		fmt.Printf("begin print share data...\n")
		for k, v := range shares {
			fmt.Printf("uid=%v, share=%v.", k.GetHexString(), v.GetHexString())
		}
		fmt.Printf("end print share data.\n")
	*/
	//fmt.Printf("end GenSharePiece.\n")
	return shares, pubs
}

//测试组签名
func testGroupInit(t *testing.T) {
	fmt.Printf("\begin testGroupInit...\n")

	//组初始化信息
	var gis ConsensusGroupInitSummary
	group_name := "64-2"
	gis.ParentID = *groupsig.NewIDFromString("TASchain")
	gis.DummyID = *groupsig.NewIDFromString(group_name)
	gis.Authority = 777
	copy(gis.Name[:], group_name[:])
	gis.BeginTime = time.Now()
	if !gis.ParentID.IsValid() || !gis.DummyID.IsValid() {
		t.Error("create group init summary failed")
	}

	gis_hash := gis.GenHash() //组初始化共识的哈希值（尝试以这个作为共享秘密的基。如不行再以成员ID合并作为基，但这样没法支持缩扩容。）
	gis_rand := util.RandFromBytes(gis_hash.Bytes())

	//组成员信息
	users := make(map[groupsig.ID]test_node_data)
	var node test_node_data
	node.shares = make([]groupsig.Seckey, TEST_MEMBERS)

	user := *groupsig.NewIDFromString("thiefox")
	node.seed = util.RandFromString("710208")
	node.sk = *GenU4GSecKey(node.seed, gis_rand)
	users[user] = node

	user = *groupsig.NewIDFromString("siren")
	node.seed = util.RandFromString("850701")
	node.sk = *GenU4GSecKey(node.seed, gis_rand)
	users[user] = node

	user = *groupsig.NewIDFromString("juanzi")
	node.seed = util.RandFromString("123456")
	node.sk = *GenU4GSecKey(node.seed, gis_rand)
	users[user] = node

	user = *groupsig.NewIDFromString("wild children")
	node.seed = util.RandFromString("111111")
	node.sk = *GenU4GSecKey(node.seed, gis_rand)
	users[user] = node

	user = *groupsig.NewIDFromString("gebaini")
	node.seed = util.RandFromString("999999")
	node.sk = *GenU4GSecKey(node.seed, gis_rand)
	users[user] = node

	if len(users) != TEST_MEMBERS {
		t.Error("create map failed, size error.")
	}

	fmt.Printf("direct gen group info....\n")
	//直接合并出组私钥和组公钥
	secs := make([]groupsig.Seckey, 0)
	for _, v := range users {
		secs = append(secs, v.sk)
	}
	gsk := groupsig.AggregateSeckeys(secs)
	if gsk == nil {
		t.Error("Aggr group sec key faild.")
	}
	gpk := groupsig.NewPubkeyFromSeckey(*gsk)
	if gpk == nil {
		t.Error("rip pub key from sec key failed.")
	}
	fmt.Printf("direct gen group sec key=%v.\n", gsk.GetHexString())
	fmt.Printf("rip group pub key from direct gen gsk, =%v.\n", gpk.GetHexString())

	pubs := make([]groupsig.Pubkey, 0)
	//直接生成组公钥
	for _, v := range users {
		temp_pub := groupsig.NewPubkeyFromSeckey(v.sk)
		if temp_pub == nil {
			t.Error("NewPubkeyFromSeckey failed.")
		}
		pubs = append(pubs, *temp_pub)
	}
	temp_gpk := groupsig.AggregatePubkeys(pubs)
	if temp_gpk == nil {
		t.Error("Aggr group pub key failed.")
	}
	fmt.Printf("aggr gen group pub key=%v.\n", temp_gpk.GetHexString())

	fmt.Printf("\nbegin exchange share pieces...\n")
	//生成和交换秘密分享
	for k, v := range users { //组成员遍历生成秘密分享
		shares, pubs := GenSharePiece(k, v, gis_rand, &users)
		if len(shares) != TEST_MEMBERS || len(pubs) != TEST_THRESHOLD {
			t.Error("GenSharePiece failed, len not matched.")
		}

		for uid, node := range users { //组成员遍历接收秘密分享
			share, ok := shares[uid.GetHexString()]
			if ok {
				SetSharePiece(uid, &node, k, share, pubs)
				users[uid] = node
			}

		}
	}
	fmt.Printf("end exchange share pieces.\n")

	//生成组成员签名私钥
	fmt.Printf("\nbegin gen group member sign seckey...\n")
	for k, v := range users { //组成员遍历
		temp_sk := groupsig.AggregateSeckeys(v.shares)
		if temp_sk != nil {
			v.sign_sk = *temp_sk
			users[k] = v
			fmt.Printf("group member sign sec key=%v.\n", v.sign_sk.GetHexString())
		} else {
			fmt.Printf("AggregateSeckeys ERROR.\n")
			panic("AggregateSeckeys ERROR.")
		}

	}
	fmt.Printf("end gen group member sign seckey.\n")

	//发送组成员签名公钥
	fmt.Printf("begin send group member sign pubkey...\n")
	for _, v := range users { //组成员遍历
		temp_pk := groupsig.NewPubkeyFromSeckey(v.sign_sk) //取得组成员签名公钥
		//fmt.Printf("   sec key=%v.\n", v.sign_sk.GetHexString())
		//fmt.Printf("   pub key=%v.\n", temp_pk.GetHexString())
		var sign_pk groupsig.Pubkey
		if temp_pk != nil {
			sign_pk = *temp_pk
		} else {
			fmt.Printf("NewPubkeyFromSeckey ERROR.\n")
			panic("NewPubkeyFromSeckey ERROR.")
		}
		for k, j := range users { //发送给每个成员
			j.AddPubPiece(sign_pk)
			users[k] = j
		}
	}
	fmt.Printf("end send group member sign pubkey.\n")

	/*
		//聚合组公钥（！错，这个聚合出来的不是组公钥）
		fmt.Printf("begin aggr group pubkey...\n")
		for k, v := range users {
			temp_pk := groupsig.AggregatePubkeys(v.pubs)
			if temp_pk != nil {
				v.group_pk = *temp_pk
				users[k] = v
				fmt.Printf("uid = %v.\n", k.GetHexString())
				fmt.Printf("aggr group pub key = %v.\n", v.group_pk.GetHexString()) //组公钥
			} else {
				fmt.Printf("AggregatePubkeys ERROR.\n")
				panic("AggregatePubkeys ERROR.")
			}

		}
		fmt.Printf("end aggr group pubkey.\n")
	*/

	//用阈值恢复法生成组私钥和组公钥
	fmt.Printf("\nbegin recover group sec key and group pub key...\n")
	sk_pieces := make([]groupsig.Seckey, 0)
	id_pieces := make([]groupsig.ID, 0)
	const RECOVER_BEGIN = 0 //range 0-2
	for k, v := range users {
		sk_pieces = append(sk_pieces, v.sign_sk)
		id_pieces = append(id_pieces, k)
	}
	sk_pieces = sk_pieces[RECOVER_BEGIN : TEST_THRESHOLD+RECOVER_BEGIN]
	id_pieces = id_pieces[RECOVER_BEGIN : TEST_THRESHOLD+RECOVER_BEGIN]
	fmt.Printf("sk_pieces len=%v, id_pieces len=%v.\n", len(sk_pieces), len(id_pieces))
	inner_gsk := groupsig.RecoverSeckey(sk_pieces, id_pieces)
	var inner_gpk *groupsig.Pubkey
	if inner_gsk != nil {
		fmt.Printf("recover group sec key=%v.\n", inner_gsk.GetHexString())
		inner_gpk = groupsig.NewPubkeyFromSeckey(*inner_gsk)
		if inner_gpk != nil {
			fmt.Printf("rip gpk from recover gsk=%v.\n", inner_gpk.GetHexString())
		}
	} else {
		fmt.Printf("RecoverSeckey group sec key ERROR.\n")
		panic("RecoverSeckey group sec key ERROR.")
	}
	fmt.Printf("end recover group sec key and group pub key.\n")

	//测试签名
	fmt.Printf("\nbegin test sign...\n")
	plain := []byte("this is a plain message.")
	//直接用组公钥和组私钥验证
	gs1 := groupsig.Sign(*gsk, plain)
	fmt.Printf("direct sign data=%v.\n", GetSignPrefix(gs1))
	result1 := groupsig.VerifySig(*gpk, plain, gs1)
	fmt.Printf("1 verify group sign direct, result = %v.\n", result1)
	if !result1 {
		t.Error("1 verify sign failed.")
	}
	//用阈值恢复法验证
	si_pieces := make([]groupsig.Signature, 0)
	id_pieces = make([]groupsig.ID, 0)

	for k, v := range users {
		sig_piece := groupsig.Sign(v.sign_sk, plain)
		si_pieces = append(si_pieces, sig_piece)
		id_pieces = append(id_pieces, k)
	}
	si_pieces = si_pieces[RECOVER_BEGIN : TEST_THRESHOLD+RECOVER_BEGIN]
	id_pieces = id_pieces[RECOVER_BEGIN : TEST_THRESHOLD+RECOVER_BEGIN]
	gs2 := groupsig.RecoverSignature(si_pieces, id_pieces)
	if gs2 == nil {
		t.Error("RecoverSignature failed.")
	}
	fmt.Printf("recover sign data=%v.\n", GetSignPrefix(*gs2))
	result2 := groupsig.VerifySig(*gpk, plain, *gs2)
	fmt.Printf("2 verify group sign from recover, result = %v.\n", result2)
	if !result2 {
		t.Error("2 verify sign failed.")
	}
	fmt.Printf("\nend test sign.\n")

	fmt.Printf("\nend testGroupInit.\n")
	return
}

//测试成为当前高度的铸块组
func testBlockCurrent(t *testing.T) {
	var proc Processor
	miner := NewMinerInfo("thiefox", "710208")
	if !proc.Init(miner) {
		return
	}

	return
}

func genAllNodes(gis ConsensusGroupInitSummary) map[groupsig.ID]GroupNode {
	nodes := make(map[groupsig.ID]GroupNode, 0)
	var node GroupNode
	node.InitForMinerStr("thiefox", "710208", gis)
	nodes[node.GetMinerID()] = node
	node.InitForMinerStr("siren", "850701", gis)
	nodes[node.GetMinerID()] = node
	node.InitForMinerStr("juanzi", "123456", gis)
	nodes[node.GetMinerID()] = node
	node.InitForMinerStr("wild children", "111111", gis)
	nodes[node.GetMinerID()] = node
	node.InitForMinerStr("gebaini", "999999", gis)
	nodes[node.GetMinerID()] = node
	return nodes
}

//测试一个组是否初始化完成
func testGroupInited(procs map[string]*Processor, gid_s string, t *testing.T) {
	fmt.Printf("\nbegin testGroupInited...\n")
	var gid groupsig.ID
	if gid.SetHexString(gid_s) != nil {
		panic("ID.SetHexString failed.")
	}
	//直接用内部函数计算组私钥和组公钥
	fmt.Printf("calc group sec key with private func...\n")
	secs := make([]groupsig.Seckey, 0)
	pubs := make([]groupsig.Pubkey, 0)
	for _, v := range procs {
		sec_piece := v.getGroupSeedSecKey(gid)
		secs = append(secs, sec_piece)
		pub_piece := groupsig.NewPubkeyFromSeckey(sec_piece)
		pubs = append(pubs, *pub_piece)
	}
	aggr_gsk := groupsig.AggregateSeckeys(secs)
	if aggr_gsk == nil {
		t.Error("Aggr group sec key from all member secret sec key faild.")
	}
	aggr_gpk := groupsig.NewPubkeyFromSeckey(*aggr_gsk)
	if aggr_gpk == nil {
		t.Error("rip pub key from sec key failed.")
	}
	fmt.Printf("aggr group sec key from all member secret sec key, aggr_gsk=%v.\n", GetSecKeyPrefix(*aggr_gsk))
	fmt.Printf("rip group pub key from aggr_gsk, aggr_gpk =%v.\n", GetPubKeyPrefix(*aggr_gpk))
	{
		temp_gpk := groupsig.AggregatePubkeys(pubs)
		if temp_gpk == nil {
			t.Error("Aggr group pub key failed.")
		}
		fmt.Printf("aggr gen group pub key=%v.\n", GetPubKeyPrefix(*temp_gpk))
		if !aggr_gpk.IsEqual(*temp_gpk) {
			t.Error("group pub key diff with direct private func.")
		}
	}

	//用阈值恢复法生成组私钥和组公钥
	sk_pieces := make([]groupsig.Seckey, 0) //组成员签名私钥列表
	id_pieces := make([]groupsig.ID, 0)     //组成员ID列表
	pk_pieces := make([]groupsig.Pubkey, 0) //组成员签名公钥列表
	for _, v := range procs {
		sign_sk := v.getSignKey(gid)
		sk_pieces = append(sk_pieces, sign_sk)
		sign_pk := *groupsig.NewPubkeyFromSeckey(sign_sk)
		pk_pieces = append(pk_pieces, sign_pk)
		id := v.GetMinerID()
		id_pieces = append(id_pieces, id)

	}
	const RECOVER_BEGIN = 0 //range 0-2
	sk_pieces = sk_pieces[RECOVER_BEGIN : TEST_THRESHOLD+RECOVER_BEGIN]
	id_pieces = id_pieces[RECOVER_BEGIN : TEST_THRESHOLD+RECOVER_BEGIN]
	fmt.Printf("sk_pieces len=%v, id_pieces len=%v.\n", len(sk_pieces), len(id_pieces))
	fmt.Printf("begin recover group sec key from member sign sec keys...\n")
	inner_gsk := groupsig.RecoverSeckey(sk_pieces, id_pieces)
	var inner_gpk *groupsig.Pubkey
	if inner_gsk != nil {
		fmt.Printf("recover group sec key=%v.\n", GetSecKeyPrefix(*inner_gsk))
		inner_gpk = groupsig.NewPubkeyFromSeckey(*inner_gsk)
		if inner_gpk != nil {
			fmt.Printf("rip gpk from recover gsk=%v.\n", GetPubKeyPrefix(*inner_gpk))
		} else {
			panic("rip gpk from recovered gsk ERROR.")
		}
	} else {
		fmt.Printf("RecoverSeckey group sec key ERROR.\n")
		panic("RecoverSeckey group sec key ERROR.")
	}

	fmt.Printf("end recover group sec key from member sign sec keys.\n")

	//测试签名
	fmt.Printf("\nbegin test sign...\n")
	plain := []byte("this is a plain message.")
	//直接用组公钥和组私钥验证
	gs1 := groupsig.Sign(*aggr_gsk, plain)
	fmt.Printf("direct sign data=%v.\n", gs1.GetHexString())
	result1 := groupsig.VerifySig(*aggr_gpk, plain, gs1)
	fmt.Printf("1 verify group sign direct, result = %v.\n", result1)
	if !result1 {
		t.Error("1 verify sign failed.")
	}
	//用阈值恢复法验证
	sig_pieces := make([]groupsig.Signature, 0)
	id_pieces = make([]groupsig.ID, 0)

	for _, v := range procs {
		sig_piece := groupsig.Sign(v.getSignKey(gid), plain)
		sig_pieces = append(sig_pieces, sig_piece)
		id_pieces = append(id_pieces, v.GetMinerID())
	}
	sig_pieces = sig_pieces[RECOVER_BEGIN : TEST_THRESHOLD+RECOVER_BEGIN]
	id_pieces = id_pieces[RECOVER_BEGIN : TEST_THRESHOLD+RECOVER_BEGIN]
	gs2 := groupsig.RecoverSignature(sig_pieces, id_pieces)
	if gs2 == nil {
		t.Error("RecoverSignature failed.")
	}
	fmt.Printf("recover sign data=%v.\n", gs2.GetHexString())
	result2 := groupsig.VerifySig(*aggr_gpk, plain, *gs2)
	fmt.Printf("2 verify group sign from recover, result = %v.\n", result2)
	if !result2 {
		t.Error("2 verify sign failed.")
	}
	fmt.Printf("end test sign.\n")
	fmt.Printf("end testGroupInited.\n\n")
	return
}

//测试逻辑功能
func TestLogicGroupInit(t *testing.T) {
	groupsig.Init(1)
	fmt.Printf("\nbegin testLogicGroupInit...\n")
	fmt.Printf("Group Size=%v, K THRESHOLD=%v.\n", GetGroupMemberNum(), GetGroupK(GetGroupMemberNum()))
	//初始化
	fmt.Printf("begin init data...\n")
	root := NewMinerInfo("root", "TASchain")
	gis := genDummyGIS(root, "64-2")

	nodes := genAllNodes(gis)

	//合并出组私钥和组公钥
	fmt.Printf("begin aggr group sec key and pub key direct...\n")
	secs := make([]groupsig.Seckey, 0)
	pubs := make([]groupsig.Pubkey, 0)
	for _, v := range nodes {
		secs = append(secs, v.getSeedSecKey())
		pubs = append(pubs, v.GetSeedPubKey())
	}
	gsk1 := *groupsig.AggregateSeckeys(secs)
	gpk1 := *groupsig.AggregatePubkeys(pubs)
	fmt.Printf("init aggr group sec key=%v.\n", gsk1.GetHexString())
	fmt.Printf("init aggr group pub key=%v.\n", gpk1.GetHexString())
	{
		temp_gpk := *groupsig.NewPubkeyFromSeckey(gsk1)
		fmt.Printf("rip gpk from aggr group sec key=%v.\n", temp_gpk.GetHexString())
	}
	//交换秘密
	fmt.Printf("begin exchange secure...\n")
	ids := make([]groupsig.ID, 0)
	for k, _ := range nodes {
		ids = append(ids, k)
	}
	for k, v := range nodes {
		shares := v.GenSharePiece(ids)
		var piece SharePiece
		piece.Pub = v.GetSeedPubKey()
		for x, y := range nodes {
			piece.Share = shares[x.GetHexString()]
			y.SetInitPiece(k, piece)
			nodes[x] = y
		}
	}
	for k, v := range nodes {
		v.beingValidMiner()
		fmt.Printf("node=%v, aggr group pub key=%v.\n", k.GetHexString(), v.GetGroupPubKey().GetHexString())
		nodes[k] = v
	}
	//恢复组私钥（内部函数）
	fmt.Printf("begin recover group sec key and group pub key...\n")
	const RECOVER_BEGIN = 0 //range 0-2
	secs = make([]groupsig.Seckey, 0)
	ids = make([]groupsig.ID, 0)
	for k, v := range nodes {

		ids = append(ids, k)
		secs = append(secs, v.getSignSecKey())
	}
	ids = ids[RECOVER_BEGIN : GetGroupK(GetGroupMemberNum())+RECOVER_BEGIN]
	secs = secs[RECOVER_BEGIN : GetGroupK(GetGroupMemberNum())+RECOVER_BEGIN]
	fmt.Printf("secs len=%v, ids len=%v.\n", len(secs), len(ids))
	gsk2 := *groupsig.RecoverSeckey(secs, ids)
	gpk2 := *groupsig.NewPubkeyFromSeckey(gsk2)
	fmt.Printf("recover group sec key=%v.\n", gsk2.GetHexString())
	fmt.Printf("rip gpk from recover gsk=%v.\n", gpk2.GetHexString())

	fmt.Printf("end testLogicGroupInit.\n")
	return
}

func genAllProcessers() map[string]*Processor {
	procs := make(map[string]*Processor, GetGroupMemberNum())

	proc := new(Processor)
	proc.Init(NewMinerInfo("thiefox", "710208"))
	procs[proc.GetMinerID().GetHexString()] = proc

	proc = new(Processor)
	proc.Init(NewMinerInfo("siren", "850701"))
	procs[proc.GetMinerID().GetHexString()] = proc

	proc = new(Processor)
	proc.Init(NewMinerInfo("juanzi", "123456"))
	procs[proc.GetMinerID().GetHexString()] = proc

	proc = new(Processor)
	proc.Init(NewMinerInfo("wild children", "111111"))
	procs[proc.GetMinerID().GetHexString()] = proc

	proc = new(Processor)
	proc.Init(NewMinerInfo("gebaini", "999999"))
	procs[proc.GetMinerID().GetHexString()] = proc

	return procs
}

func testGenesisGroup(procs map[string]*Processor, t *testing.T) {
	fmt.Printf("begin being genesis group member...\n")
	pubs := make([]groupsig.Pubkey, 0)
	for _, v := range procs {
		pk_piece := v.BeginGenesisGroupMember()
		pubs = append(pubs, pk_piece.PK)
	}
	gpk := groupsig.AggregatePubkeys(pubs)
	if gpk == nil {
		t.Error("aggr gpk failed.")
	} else {
		fmt.Printf("end being genesis group member, gpk=%v.\n", GetPubKeyPrefix(*gpk))
	}
	return
}

func testLogicGroupInitEx(t *testing.T) {
	fmt.Printf("\nbegin testLogicGroupInitEx..\n")
	root := NewMinerInfo("root", "TASchain")
	fmt.Printf("root mi: id=%v, seed=%v.\n", GetIDPrefix(root.MinerID), root.SecretSeed.GetHexString())
	{
		save_buf := root.Serialize()
		var root2 MinerInfo
		root2.Deserialize(save_buf)
		fmt.Printf("root2 Deserialized, mi: id=%v, seed=%v.\n", GetIDPrefix(root.MinerID), root.SecretSeed.GetHexString())
	}

	procs := genAllProcessers() //生成矿工进程
	var first_proc *Processor
	var mems []PubKeyInfo
	var proc_index int
	for _, v := range procs {
		if first_proc == nil && v != nil {
			first_proc = v
		}
		pki := v.GetPubkeyInfo()
		fmt.Printf("proc(%v) miner_id=%v, pub_key=%v.\n", proc_index, GetIDPrefix(pki.GetID()), GetPubKeyPrefix(pki.PK))
		proc_index++
		mems = append(mems, pki)
		v.setProcs(procs)
	}

	testGenesisGroup(procs, t)

	var grm ConsensusGroupRawMessage
	grm.MEMS = make([]PubKeyInfo, len(mems))
	fmt.Printf("mems size=%v.\n", len(mems))
	copy(grm.MEMS[:], mems[:])
	fmt.Printf("grm.MEMS size=%v, mems size=%v.\n", len(grm.MEMS), len(mems))
	//grm.GI = genDummyGIS(root, "64-2")

	grm.GI = GenGenesisGroupSummary()

	//grm.SI = GenSignData(grm.GI.GenHash(), root.GetMinerID(), root.GetDefaultSecKey())
	fmt.Printf("grm msg member size=%v.\n", len(grm.MEMS))

	//通知所有节点这个待初始化的组合法
	//ngc := CreateInitingGroup(sgiinfo)
	for _, v := range procs {
		v.globalGroups.AddInitingGroup(CreateInitingGroup(&grm))
	}

	//启动所有节点进行初始化
	for _, v := range procs {
		v.OnMessageGroupInit(grm) //启动
	}

	//聚合组公钥
	var pub_pieces []groupsig.Pubkey
	for _, v := range procs {
		pub_piece := v.GetMinerPubKeyPieceForGroup(grm.GI.DummyID)
		pub_pieces = append(pub_pieces, pub_piece)
	}
	group_pub := groupsig.AggregatePubkeys(pub_pieces)
	if group_pub != nil {
		fmt.Printf("direct aggr group pub key=%v.\n", GetPubKeyPrefix(*group_pub))
	} else {
		panic("direct aggr group pub key failed.")
	}

	var first_gid groupsig.ID
	//初始化结果测试
	fmt.Printf("after inited, print processers info: %v---\n", GetIDPrefix(first_gid))
	index := 0
	for k, v := range procs {
		groups := v.getMinerGroups()
		fmt.Printf("---i(%v) proc(%v), joined groups=%v.\n", index, k, len(groups))
		for _, g_info := range groups {
			fmt.Printf("------g_id=%v, sign key=%v.\n", GetIDPrefix(g_info.GroupID), GetSecKeyPrefix(g_info.SignKey))
			if !first_gid.IsValid() {
				fmt.Printf("first_gid set value=%v.\n", GetIDPrefix(g_info.GroupID))
				first_gid = g_info.GroupID
			} else {
				fmt.Printf("first_gid valided, value=%v.", GetIDPrefix(first_gid))
			}
		}
		index++
	}
	fmt.Printf("print first_gid=%v.\n", GetIDPrefix(first_gid))
	if !first_gid.IsValid() {
		panic("first_gid not valid.")
	}
	fmt.Printf("after inited, first group id=%v.\n", GetIDPrefix(first_gid))
	testGroupInited(procs, first_gid.GetHexString(), t)
	return

	//铸块测试
	fmt.Printf("\n\nbegin group cast test, time=%v, init_piece_status=%v...\n", time.Now().Format(time.Stamp), CBMR_IGNORE_KING_ERROR)
	var ccm ConsensusCurrentMessage
	pre_hash := sha1.Sum([]byte("tas root block"))
	ccm.PreHash = common.BytesToHash(pre_hash[:])
	ccm.BlockHeight = 1
	ccm.PreTime = time.Now()
	fmt.Printf("pre block cast time=%v.\n", ccm.PreTime.Format(time.Stamp))

	for _, v := range procs { //进程遍历
		groups := v.getMinerGroups()
		var sign_sk groupsig.Seckey
		for _, g_info := range groups { //加入的组遍历
			ccm.GroupID = g_info.GroupID.Serialize()
			sign_sk = g_info.SignKey
			break //只处理第一个组
		}
		mi := v.getMinerInfo()
		//func (p Processor) getGroupSeedSecKey(gid groupsig.ID) (sk groupsig.Seckey) {
		//ccm.GenSign(SecKeyInfo{mi.GetMinerID(), mi.GetDefaultSecKey()})
		sign_pk := groupsig.NewPubkeyFromSeckey(sign_sk)
		fmt.Printf("ccm sender's id=%v, sign_pk=%v.\n\n", GetIDPrefix(mi.GetMinerID()), GetPubKeyPrefix(*sign_pk))
		ccm.GenSign(SecKeyInfo{mi.GetMinerID(), sign_sk}, &ccm)
		break
	}

	//for _, v := range procs { //向组内每个成员发送“成为当前铸块组”消息
	//	//v.OnMessageCurrent(ccm)
	//}
	//主进程堵塞
	sleep_d, err := time.ParseDuration("8s")
	if err == nil {
		fmt.Printf("main func begin sleep, time=%v...", time.Now().Format(time.Stamp))
		time.Sleep(sleep_d)
		fmt.Printf("main func end sleep, time=%v.", time.Now().Format(time.Stamp))
	}

	return
}

func TestMain1(t *testing.T) {
	PROC_TEST_MODE = true
	common.InitConf("/home/thiefox/src/tas/src/consensus/logical/logical_test.ini")
	groupsig.Init(1)

	//testGroupInit(t)
	//testLogicGroupInit(t)
	testLogicGroupInitEx(t)
	return
}

func TestName(t *testing.T) {
	now := time.Now()
	time.Sleep(2)
	d := time.Since(now)
	var secs uint64 = uint64(d.Seconds())
		qn := int64(secs / uint64(MAX_USER_CAST_TIME))
		fmt.Println(secs, uint64(MAX_GROUP_BLOCK_TIME), qn)
}

//to do 给屮逸：
//2个签名验证函数
//一个组初始化完成后的上链结构

func TestTime(t *testing.T) {
	log.Printf(time.Now().String())
	log.Printf(time.Now().Format("2006-01-02 15:04:05.000"))
}

func TestTimeAdd(t *testing.T) {
	now := time.Now()
	add := now.Add(time.Minute * 100)
	
	t.Log(now, add)
}

func TestSwitch(t *testing.T) {
	a := TRANS_INVALID_SLOT
	switch a {
	case TRANS_INVALID_SLOT, TRANS_DENY:
		log.Println("TRANS_INVALID_SLOT full")
		break
	case TRANS_ACCEPT_NOT_FULL:
		log.Println("TRANS_ACCEPT_NOT_FULL full")
	case TRANS_ACCEPT_FULL_PIECE:
		log.Println("TRANS_ACCEPT_FULL_PIECE full")
	case TRANS_ACCEPT_FULL_THRESHOLD:
		log.Println("TRANS_ACCEPT_FULL_THRESHOLD full")
	}
}