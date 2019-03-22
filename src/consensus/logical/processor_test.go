package logical

import (
	"testing"
	"encoding/json"
	"consensus/groupsig"
	"middleware"
	"common"
	"consensus/model"
	"core"
	"consensus/mediator"
	"middleware/types"
)

/*
**  Creator: pxf
**  Date: 2019/1/17 下午12:37
**  Description: 
*/

func TestLoadJoinGroup(t *testing.T) {
	s := `[{"GroupID":"0x4e2748e5bbdd8dc53d0cf3f5690ea5341ddbb7bc3b31c7dd576bed30f4a057de","SeedKey":"0x20ab30c9abf284c7248335f18111f0f615ac958468e7750422fa9158af75c61b","SignKey":"0x28ea8ae23228b408ef026b5c358c7a383b6449076c92ef1615fa76cea7c236cc","GroupPK":"0x2e711ddbd310e16cd31af352f5039b9f135d0c5af01983c2616e2f80c495b3990830f8e7f2dd94558ed39978e798e8258cf1eb73f07a6f238f3019623e96e85d21c53ab62de4608077bdc5da742cb1593f1441f5ebdd2b2ef8c92b17596cadd823fa890ff2a10255345f9f149a23d8f591d074ea8485485e89fd2cc003a2cf00","Members":{"0x10b94f335f1842befc329f996b9bee0d3f4fe034306842bb301023ca38711779":"0x2d0c41e9e09f69f622d58b79c9a00663bd5b2f48e6e2832bb561b1fa6cf7d41320294cebddb60a576250c2d6ce6c76c9c254fc5f2e984309cfd7dbbb22a4cd880d5f074e4cd68165c16c5041e2867d50183e4be5a1650b8797a990970da35dcb19c919aace3090e0c6181e69e0923cac24763cb753e7097474c2c4d7e5f5c55c","0x146bf2f6c42e7509174a23e66e3aaa78c8c7138334173be515140014ba657eaf":"0x16a0a1d4b39fb210f2731a9a69b55bb6cd7c32bb506307c84623586478c760622f8c5d532cd2a3bab430ea4b00c02728b23ac80a449ed5de4c04beaa5b5a251a28cc3358010f8715908919f3dbb52e32add29c7bc265afdc1ffd7a65bca2124f2c64bbb712b7cd9353891fc72e7dc89020e5c984535c50e7a2ed05b74f7c2492","0x16d1808f010a84da654e8ad2729c484eee053665bac830a8b7fd491a50783a6f":"0x2619f1b7f78d1dfd745da8ea1b82b698eb6828335eed9de1cfcad49ee0bda48b13db4b98503662cc430ca1a18698a3cb0941cf916cc3210b8f18727999b261ab0e1810615d91364d6f516056e28b272f93b389056a686194824d4bfdf004a20602298658cbd85ca99e7e6a9f52ece1c3bfcb4ebce3b2d22a7fa3572128bcd14d","0x290790f33fb9ffc3396b7b10abb689e98fef0156f0f5a68637b961aefb485652":"0x0689024f679807f5cac55818b5d3790a0104de74f20829c8afbe14ae9eeb60e5281ce0d0f332d2951555f02a15317bd79d9a72a40f970a98eca2c82382b71c66178ec692313ab00cc1596b8e9d1640a39bbb17e9d7b34dd213bd15201bcfc05027c9237fece5940bde5f0e64ce7e44666245cbbc9b6bb6fa25cd7ff6c96a86f9","0x5c9e92188e3adf51b133921bf0c65af561f4be50243cb184af8314d07880d030":"0x1e483a749ed4e67e0192bc11a0204c1c66fb3e6803e6a51d2d72ff6354fe2bfa05d7667b05a4161acc9728d82e654e9fda3b75707ce5845240e2fc3308c3f3fb1141f950186a38e220332de4b87cec3a036a699e8f1d4de71ae8d1633c74b493252b0f784092a453ffdfdbeb557233be08d9b794b8a6653d8f5b2a5acc483619","0xa85447e3e82cff8dae54748793a914f7740a34eafcdd05fe153189cb87fe08de":"0x2fc7495a862ed9a25611ecf7c047bc89973e37c6cadad68808fa0b96027139db22d9d0b843891d35f31dc68f0f64d8cb16f67a735825a775f72bbbaa19e71abc21cc1140cfc7dd6235ae3a0a564698183034ac5df6100cc83ae0b67db221fd5b17317e4ef8bd6b9649fa8fe73984294f9c419cc3eafdbf0ef1186bf86aaf7309","0xbf03e69b31aa1caa45e79dd8d7f8031bfe81722d435149ffa2d0b66b9e9b6b7":"0x17a33c7339b9dc90797719aedbff22230d5e0dd20e6dcb83ce34c6e24066f15d1153953b400be24703dcd3d41149f15d4fc271b30dce0ff927d68ef6a74bb0be2c2ee3953fe880cc8aa626fe1475d85c91f625d3a46d32ee8d6700084692c5c819b9c4707f565b8352f42dff1f168060c1d457f41da2c4da07c481f80f65a47c","0xcf73fdef8e9ef48b6b04dc1dbfb943af1f5a2ec4052c3e5ae34dd3dee5a3ffb9":"0x0ce105b102b7b8040344f52e8d9a008facd4d8b5b2071817209b49aebaed7de4073298e5754b378ea0aaae88b3edbb6d70189d15fba481fedba7195ae2410a371b699b67d4ba9aeca206450d5c8776dd67c6f7653f4d3cf39088396437b143da2a4974f69a1e189673f4fbc646f1883a9b0287d6c24579cd4c06598bf5e1043c","0xed890e78fc5d07e85e66b7926d8370c095570abb5259e346438abd3ea7a56a8a":"0x2c91f218f7d08d70bd74718f337f9c2fba79c6f6a3bce9d9327dc8c5d3199cf02fc5312de3ea8c443bed105af4ed153b4673235db63c6f804bf1794a138c95c61b2c3d7bd28dd6e762ec316bdbf2d23d547c228b995b988b2834d0fe3a3e7eb61c14b444a1e3833de06aac9373f90edc20026c736ee4274fdf842a9380c3f9a6"}}]`

	var gs []*JoinedGroup
	err := json.Unmarshal([]byte(s), &gs)
	if err != nil {
		t.Fatal(err)
	}
	for _, jg := range gs {
		for idStr, pk := range jg.Members {
			var id groupsig.ID
			id.SetHexString(idStr)
			jg.Members[id.GetHexString()] = pk
		}
		if jg.GroupID.GetHexString() == "0x4e2748e5bbdd8dc53d0cf3f5690ea5341ddbb7bc3b31c7dd576bed30f4a057de" {
			for id, pk := range jg.Members {
				t.Log(id, pk.GetHexString())
			}
		}
	}
}

func TestCalcVerifyGroup(t *testing.T) {
	common.InitConf("tas1.ini")
	middleware.InitMiddleware()

	addr := common.HexToAddress(common.GlobalConf.GetString("gtas", "miner", ""))
	mdo := model.NewSelfMinerDO(addr)
	core.InitCore(false, mediator.NewConsensusHelper(mdo.ID))
	p := new(Processor)

	p.Init(mdo, common.GlobalConf)

	top := p.MainChain.Height()
	pre := p.MainChain.QueryBlockByHeight(0)
	for h := uint64(1); h <= top; h++ {
		bh := p.MainChain.QueryBlockByHeight(h)
		if bh == nil {
			continue
		}
		gid := groupsig.DeserializeId(bh.GroupId)
		expectGid := p.CalcVerifyGroupFromChain(pre, h)
		pre = bh
		if !gid.IsEqual(*expectGid) {
			t.Fatalf("gid not equal, height %v, real gid %v, expect gid %v", h, gid.GetHexString(), expectGid.GetHexString())
		}
	}
	t.Log("ok")
}
