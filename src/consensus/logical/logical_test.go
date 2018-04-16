package logical

import (
	"consensus/groupsig"
	"consensus/rand"
	"fmt"
	"testing"
	"time"
)

const TEST_MEMBERS = 5
const TEST_THRESHOLD = 3

type test_node_data struct {
	seed     rand.Rand         //私密种子
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
func GenU4GSecKey(us rand.Rand, gs rand.Rand) *groupsig.Seckey {
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
func GenSharePiece(uid groupsig.ID, info test_node_data, group_seed rand.Rand, mems *map[groupsig.ID]test_node_data) (groupsig.SeckeyMapID, []groupsig.Pubkey) {
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
		shares[k] = *groupsig.ShareSeckey(secs, k)
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

	gis_hash := gis.GenHash().Sum(nil) //组初始化共识的哈希值（尝试以这个作为共享秘密的基。如不行再以成员ID合并作为基，但这样没法支持缩扩容。）
	gis_rand := rand.RandFromBytes(gis_hash)

	//组成员信息
	users := make(map[groupsig.ID]test_node_data)
	var node test_node_data
	node.shares = make([]groupsig.Seckey, TEST_MEMBERS)

	user := *groupsig.NewIDFromString("thiefox")
	node.seed = rand.RandFromString("710208")
	node.sk = *GenU4GSecKey(node.seed, gis_rand)
	users[user] = node

	user = *groupsig.NewIDFromString("siren")
	node.seed = rand.RandFromString("850701")
	node.sk = *GenU4GSecKey(node.seed, gis_rand)
	users[user] = node

	user = *groupsig.NewIDFromString("juanzi")
	node.seed = rand.RandFromString("123456")
	node.sk = *GenU4GSecKey(node.seed, gis_rand)
	users[user] = node

	user = *groupsig.NewIDFromString("wild children")
	node.seed = rand.RandFromString("111111")
	node.sk = *GenU4GSecKey(node.seed, gis_rand)
	users[user] = node

	user = *groupsig.NewIDFromString("gebaini")
	node.seed = rand.RandFromString("999999")
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
			share, ok := shares[uid]
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
	fmt.Printf("direct sign data=%v.\n", gs1.GetHexString())
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
	fmt.Printf("recover sign data=%v.\n", gs2.GetHexString())
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
	var proc Processer
	if !proc.Init() {
		return
	}

	return
}

func TestMain(t *testing.T) {
	groupsig.Init(1)
	testGroupInit(t)
}
