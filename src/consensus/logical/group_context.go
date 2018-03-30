package logical

import (
	"common"
	"consensus/groupsig"
	"consensus/rand"
)

//组共识上下文
type GroupContext struct {
}

//生成某个成员针对所有组内成员的秘密分享（私钥形式）
func (gc *GroupContext) GenSharePieces(h common.Hash, ids []groupsig.ID) []groupsig.Seckey {
	var shares []groupsig.Seckey
	if h.IsValid() && len(ids) > 0 {
		shares = make([]groupsig.Seckey, len(ids))
		master_seckeys := make([]groupsig.Seckey, len(ids))
		seed := rand.NewRand() //每个组成员自己生成的随机数
		for i := 0; i < len(ids); i++ {
			master_seckeys[i] = *groupsig.NewSeckeyFromRand(seed.Deri(i)) //生成master私钥数组（bls库函数）
		}
		for i := 0; i < len(ids); i++ {
			shares[i] = *groupsig.ShareSeckey(master_seckeys, ids[i]) //对每个组成员生成秘密分享
		}
	}
	return shares
}

//生成某个成员用于组内签名（分片）的组成员私钥
func (gc *GroupContext) GenMemberSignSeckey(shares []groupsig.Seckey) groupsig.Seckey {
	sk := *groupsig.AggregateSeckeys(shares) //通过bls底层库生成聚合私钥
	return sk
}

//所有节点生成组公钥片段
func (gc *GroupContext) GenGroupPubkeyPieces(sk groupsig.Seckey) groupsig.Pubkey {
	pk := *groupsig.NewPubkeyFromSeckey(sk) //从私钥萃取出公钥
	return pk
}

//某个凑齐了所有组公钥片段的节点生成组信息（组ID和组公钥），上链后用于组外验证签名。
func (gc *GroupContext) GenGroupInfo(pubs []groupsig.Pubkey) (groupsig.ID, groupsig.Pubkey) {
	gpk := *groupsig.AggregatePubkeys(pubs) //通过bls底层库生成聚合公钥
	gid := *groupsig.NewIDFromPubkey(gpk)
	return gid, gpk
}
