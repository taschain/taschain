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

package core

import (
	"bytes"
	"common"
	"math/big"
	"time"
	"storage/account"
	"storage/trie"
	"middleware/types"
	"storage/serialize"
	"github.com/vmihailenco/msgpack"
)

var emptyHash = common.Hash{}

func calcTxTree(tx []*types.Transaction) common.Hash {
	if nil == tx || 0 == len(tx) {
		return emptyHash
	}

	buf := new(bytes.Buffer)
	for i := 0; i < len(tx); i++ {
		encode, _ := msgpack.Marshal(tx[i])
		serialize.Encode(buf, encode)
	}
	return common.BytesToHash(common.Sha256(buf.Bytes()))
}

func calcReceiptsTree(receipts types.Receipts) common.Hash {
	if nil == receipts || 0 == len(receipts) {
		return emptyHash
	}

	keybuf := new(bytes.Buffer)
	trie := new(trie.Trie)
	for i := 0; i < len(receipts); i++ {
		if receipts[i] != nil {
			keybuf.Reset()
			serialize.Encode(keybuf, uint(i))
			encode, _ := serialize.EncodeToBytes(receipts[i])
			trie.Update(keybuf.Bytes(), encode)
		}
	}
	hash := trie.Hash()

	return common.BytesToHash(hash.Bytes())
}

// 创始块
func GenesisBlock(stateDB *account.AccountDB, triedb *trie.NodeDatabase, genesisInfo *types.GenesisInfo) *types.Block {
	block := new(types.Block)
	pv := big.NewInt(0)
	block.Header = &types.BlockHeader{
		Height:       0,
		ExtraData:    common.Sha256([]byte("tas")),
		CurTime:      time.Date(2018, 6, 14, 10, 0, 0, 0, time.Local),
		ProveValue:   pv,
		TotalQN:      0,
		Transactions: make([]common.Hash, 0), //important!!
		EvictedTxs:   make([]common.Hash, 0), //important!!
	}

	//blockByte, _ := json.Marshal(block)
	//block.Header.Hash = common.BytesToHash(common.Sha256(blockByte))
	block.Header.Signature = common.Sha256([]byte("tas"))
	block.Header.Random = common.Sha256([]byte("tas_initial_random"))

	//创世账户
	for _, mem := range genesisInfo.Group.Members {
		addr := common.BytesToAddress(mem)
		stateDB.SetBalance(addr, big.NewInt(10000))
	}

	// 交易脚本账户
	stateDB.SetBalance(common.HexStringToAddress("0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103"), big.NewInt(10000000000000))
	stateDB.SetBalance(common.HexStringToAddress("0xcad6d60fa8f6330f293f4f57893db78cf660e80d6a41718c7ad75e76795000d4"), big.NewInt(10000000000000))
	stateDB.SetBalance(common.HexStringToAddress("0xca789a28069db6f1639b60a8bf1084333358672f65c6d6c2e6d58b69187fe402"), big.NewInt(10000000000000))
	stateDB.SetBalance(common.HexStringToAddress("0x94bdb92d329dac69d7f107995a7b666d1092c63eadeae2dd495ab2e554bb155d"), big.NewInt(10000000000000))
	stateDB.SetBalance(common.HexStringToAddress("0xb50eea221a1eb061dea7ca20f7b7508c2d9639e3558e69f758380e32624337b5"), big.NewInt(10000000000000))
	stateDB.SetBalance(common.HexStringToAddress("0xce59fd5e1c6c99d9990b08ccf685260a2b3a03889de56e91b25878a4bf2f89e9"), big.NewInt(10000000000000))
	stateDB.SetBalance(common.HexStringToAddress("0x5d9b2132ec1d2011f488648a8dc24f9b29ca40933ca89d8d19367280dff59a03"), big.NewInt(10000000000000))
	stateDB.SetBalance(common.HexStringToAddress("0x5afb7e2617f1dd729ea3557096021e2f4eaa1a9c8fe48d8132b1f6cf13338a8f"), big.NewInt(10000000000000))
	stateDB.SetBalance(common.HexStringToAddress("0x30c049d276610da3355f6c11de8623ec6b40fd2a73bb5d647df2ae83c30244bc"), big.NewInt(10000000000000))
	stateDB.SetBalance(common.HexStringToAddress("0xa2b7bc555ca535745a7a9c55f9face88fc286a8b316352afc457ffafb40a7478"), big.NewInt(10000000000000))
	//小熊本地测试轻节点账户
	stateDB.SetBalance(common.HexStringToAddress("0x7c608b0e25bba75a6bde6e87b2b007354cfc42f9bda481f59082f68fe4a9446d"), big.NewInt(1000000))
	stateDB.SetBalance(common.HexStringToAddress("0xdc57f6994098a8ac04e1c73064ee93e5edc8dbe0e47e8865ed01df7304d2bf73"), big.NewInt(1000000))
	stateDB.SetBalance(common.HexStringToAddress("0x352dfb7802f3fccf3cb571dd8000ba5cd2386fa70db9c415d34588905f233017"), big.NewInt(1000000))
	//阿里云账户
	stateDB.SetBalance(common.HexStringToAddress("0x3ba8efc57a6b69a02635c35ca14ee54e789ccc4ca3d6b5812a2dd8abbc01bf4b"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x5d65a222b32675de37b4eae9a9e687069f02071c66134d61676a612513a2dd8a"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xd13f2d8ac0b33b3c42ece2eceb04a2eafe2f4dd2925d77ff6e07961ced24f291"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x97718a8347695ebb0daaf6e245c31ea77a5dbcd8bdf78295383e5a02a340968d"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x12bb66e48f405ee773d5aa1b8a829c3baf90982383effbaa4d4a11e15011c0d6"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xd80190bb77fb03bdbacdca48d5b64ad53ee25e761fd4165675b676645167c4e7"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x20583d59ad04067076df1a709ae82df7d1b67dfaff9b42546d85960abf07bc1f"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0xaedfa674cd46cfe94215807b3bd9b315d166d5bc83480287a0c4284365529695"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x1dd855a32c4ca04de65b34cfeea6263e90842b6c152ae3dd47a00cf366edbed6"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x104a07e6de077fe9321c771d22f6ee43e91a09d4b972d5dd1ab81a2b2d98e038"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x5f3c23cf1578cfe192ff6d2c38e15d239ce9aaae3ddde6c8481f0ddfce9da32c"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x6486865bcacedcaad04ed3519483dbe999f176ed492d28589c15f2b6035bb773"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x296af7568e617e3fbf9853ce96d743578dc72d44882268f6c3a07e7eb91e58c7"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xe000b5dd00ac03e2efb21392559238b9ec98f88ec69ce49e1758e99208711fdf"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xd6d7523026703c134f95d99b02b2c10a2126d8ea1a1bb3a583888131bd8d7f4f"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x96edfcb70afeb28ef4e89a10af8c53739bc0b66b7c90e339570175cd8b0369f9"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xe6ff51733492564716f4d8c403f8466a4c24fd2e4b616a62a98cfcace23e9652"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0xd4103e1c47e442351680b5329be14cdb5cc5f0e80c68678c9a9a5deb57ac9eeb"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xd6e25fdcb428d4cdf9df2e2a307f72120bad902ca09325ee66e8b734f1b2a048"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xc22768990ff2da63978791739694c7183cd7a1f070f3a969a3879bc21a5fcdcd"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xae4c3a94998e5c0ae0ff028af791d22b4307865c526ce13f0eeb5967aadadf48"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x7a9c9dae7f39ff2e6e67d33b469f7891f2c3fe89ca3d733a4ab6f5cf157a5ef7"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xfae5e5144f2cae87a07233772c48b118a4de90287fa2b25c71f8bf998e69c15b"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x3ff79ce7026c86877b8dac2855df40ae370bb1dd4410cb4eeed53d3688047aa2"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x1472c086cc690f219a0ec1e80572b89cf8049945995de60f29774df81295a3a8"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xe582318f0a29ef3983b04f2eea19ab65357438cc920f9b7e9142b59b7f8a4a88"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xaf501bbad93672d43c86591ca2f4ff2ea040f4333beea0da39ffbea98b15f789"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0xdf9c358537e2bae71c1cf5375f95f3c40313bb5967f0f014024901c04582329d"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x2d56a5654b7b5931459c1d240f6f00f48e6bedb7a118b2f2d23c1a1d9a1415bf"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xa27856f117fc845e9b9375b7f9e19a9fb5390a180a987adc8cf3773291bd2bf6"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x2d0ea661c77c65ff517e3df4635bd2c59bc5ad54e590a0186b6a42260496c62a"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x91e1f6cb08dbb5e6bbed7f2d8615962185e3e89cf1ff88bbfb415e3f93593aaa"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x365ca544445400c00ebd43347dc6b83bb8fa756a72f5878d5383ea63378a78a3"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xc0f6abac0f7f552649c43b660dd68a51690a71d4366727b44fea7ad155912506"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x1a8120bc0bbc4ed3f9a6e9c432290ceab75bbdb4007d94f7aeecaac3825f2b92"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xd3e0cb2d953a551264c7e987f93fb40852ec839591f080cb69dc7fb04b4468b6"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x502b233115ad3855bb2391a3864c0b817bdf839a2394dcf765fe78d4a8800837"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0xbfc64ab5c906e05e59a28960849eb7eb3797c5f4aac293abfc61b85fcf137990"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xe90fe1fdabc7e4811ee782dcf3dbd7ffc54ad683e4763aa2a0d707ab7f2072cc"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xb935722119dd63f8e38af1fbefab07524e298a6e0a3947d69c38c7d0c4d5ad00"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x5104da602c1fc5e799c9473ddb16f10778f40cf576e130b8fa2d0f8163985621"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x977988b6f632f61a68c75b1a17437ba62ed3ed3e1709c2280fd790bad01b3230"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xff610e6b0e503dc468e672a3e489899757734f3ecf04c5c692dc4fe36a2ab392"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x8d01a1c1e224c856fdc93c63aecfb626b9528d20c17973b8423eba6a075c21c5"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x33a88a28216eee1295b87bf4eb44d487c8106d1ef11441ba8d0d56dd97014344"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xffd693c69796bf1ac26530b0c62e5055e676da8357a9536aa8db2830f7a24959"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0x9f5c4256b4640e256458cc9c75aa925d7d071fd49ebcd685223ca932b7f84e42"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xbad287aa2b557a75d79dbfde38da9dc4191a2ec5a9cc66ce79551a3c0dbeb1af"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xc748a8817d18c34a1b1cb1fa7975a77c0fa2568aed443ebf800b5f9290bd706d"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x7f5871573d372c2a3f03791d6a073027fd65139d080f9fe008b12aaf9dad2cc7"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xa396d7331adeea6feab8ee3828575343ec6e28cf748a44cce2f8ae9013222eda"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x91d05de285c14b52f649f9a9113cf202dd3a579c885e818c4dca5a763c559412"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x4802cc273b7841ce21df693115663592f6a8acbc4a1cce941e9fb180f1720692"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xafdb9ba644cb3036d27689e5c91b502ea4ad9e7ec324ba39b326ad350531e2bc"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xc07120399da7ee6f4bba26c29801b1725d5ecf9b33054952606891043724aa52"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x5d60f112a4380ece4964a132f8335494e58bb47caafd958b051056bf0d6bfd19"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0x95762debb94e16549e162d1d208ba3a22ae048a3d3a6375f08a3a18bdfc9c0c1"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x9a6af10d7b2f0ca284f02ad6835a4f8bb611562a8085eb3a7ef75edcdbf81836"), big.NewInt(10000))

	stage := stateDB.IntermediateRoot(false)
	Logger.Debugf("GenesisBlock Stage1 Root:%s", stage.Hex())
	miners := make([]*types.Miner, 0)
	for i, member := range genesisInfo.Group.Members {
		miner := &types.Miner{Id: member, PublicKey: genesisInfo.Pks[i], VrfPublicKey: genesisInfo.VrfPKs[i], Stake: 10}
		miners = append(miners, miner)
	}
	MinerManagerImpl.AddGenesesMiner(miners, stateDB)
	stage = stateDB.IntermediateRoot(false)
	Logger.Debugf("GenesisBlock Stage2 Root:%s", stage.Hex())
	stateDB.SetNonce(common.BonusStorageAddress, 1)
	stateDB.SetNonce(common.HeavyDBAddress, 1)
	stateDB.SetNonce(common.LightDBAddress, 1)

	root, _ := stateDB.Commit(true)
	Logger.Debugf("GenesisBlock final Root:%s", root.Hex())
	triedb.Commit(root, false)
	block.Header.StateTree = common.BytesToHash(root.Bytes())
	block.Header.Hash = block.Header.GenHash()
	//block.Transactions = make([]*types.Transaction, 0)
	return block
}
