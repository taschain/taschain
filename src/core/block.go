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
		EvictedTxs: make([]common.Hash, 0), //important!!
	}

	//blockByte, _ := json.Marshal(block)
	//block.Header.Hash = common.BytesToHash(common.Sha256(blockByte))
	block.Header.Signature = common.Sha256([]byte("tas"))
	block.Header.Random = common.Sha256([]byte("tas_initial_random"))

	// 创始块账户创建
	stateDB.SetBalance(common.BytesToAddress(common.Sha256([]byte("1"))), big.NewInt(1000000000))
	stateDB.SetBalance(common.BytesToAddress(common.Sha256([]byte("2"))), big.NewInt(2000000000))
	stateDB.SetBalance(common.BytesToAddress(common.Sha256([]byte("3"))), big.NewInt(3000000000))
	stateDB.SetBalance(common.BytesToAddress(common.Sha256([]byte("4"))), big.NewInt(1000000000))
	stateDB.SetBalance(common.BytesToAddress(common.Sha256([]byte("5"))), big.NewInt(2000000000))
	stateDB.SetBalance(common.BytesToAddress(common.Sha256([]byte("6"))), big.NewInt(3000000000))
	stateDB.SetBalance(common.BytesToAddress(common.Sha256([]byte("7"))), big.NewInt(1000000000))
	stateDB.SetBalance(common.BytesToAddress(common.Sha256([]byte("8"))), big.NewInt(2000000000))
	stateDB.SetBalance(common.BytesToAddress(common.Sha256([]byte("9"))), big.NewInt(3000000000))
	//小熊本地测试轻节点账户
	stateDB.SetBalance(common.HexStringToAddress("0xa59888012b8ee73d7ad673f38b4a7695310acb480454c1a484a4d2cde454dd2b"), big.NewInt(1000000))
	stateDB.SetBalance(common.HexStringToAddress("0x819241be6ab490dc17ac2172408ca0cc024880e3fddaaae80338ecdd7ec9d68c"), big.NewInt(1000000))
	stateDB.SetBalance(common.HexStringToAddress("0x5e5ba5be8d8b6c4d9f9bc8446c4295f6e40f0c6fd4e3d6a1e4db2e4931f674b0"), big.NewInt(1000000))
	//阿里云账户
	stateDB.SetBalance(common.HexStringToAddress("0xd196fb6b6ad3a788d61c61783e00f6568eb72f5ef170a874703a2b3b70eafc49"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x9ab304d9eabcbfe33fec3461261997cfcc981d9cd93e17372777547d1e642bb3"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x2ffd349c8db7f8587cd04846dc5ac4908bc5bdecd72d397aef7c5c6766b9311c"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x69df6234ab253eb65439377b886443bb6105240bfbf8d8ebdffcead4affcb830"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xf18951dbef0a34d54bb4d8da1bace6700bb1a68b7ecc5d8e937a388268ab33b2"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0x4f478623bcbe14acc9749d8d2bb2540b26768139c4c49dd2356e195e5df80357"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x39e9f20f2faea3e8854d1df01efcc32f4fed2f8fd65bcbb611bd1d7801ff1617"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xb4d24ee0cc9b736e8504839d82a9135a0e0c347f52ee2964faa7dda6023dff1f"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x9911b24c4551d0e7dfea9ed72a5ba8fddfe48529c6656ce7840ebf0ff3e71fa7"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xd4b27439e3c4fa4bd8a211b81238406c99290784f8cad522e72c9b3dfe50d012"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0x60371d2ca8a9cbb10e80af8673de81c6b35f6d171766dfea3e09e6db3bbe9b35"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x30930bce195e3e09c296de20aa6daf05b489a5dc308ee46b506ceba9838f7cbe"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x41960e6411aeb717a5447a7c1a80f002420b29827fe67c6bc2bbe27d705275ab"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x2cbcc61e7506fccba445fa9881b3e6067fcba238d75235d735724be89e099e56"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x69ad13a457741784185e0cd098789b6acb0c78af0ce9b37b22dc06abd7c92275"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0xef06131e0b3e5d2dcd0b2512ae0d7fe9d93ebeea255c4c1e8f074512610b2852"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x8a5ca0105d35e1ed1f2c1bafbea243256f89b88cf125605ff1c50e0926d46159"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x6e9fb7b05e9cbe3216e6a448aa5a6098b419881bf1cd5ecb6899ce1d011867ec"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x448b2c71f45b45ad13d6a22ed53dbbc3ee61892f705685bf419434ea6b13fd28"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xa77364106690e02bc6493fe361c6780327f48fbaa644e8961f47654d66decc34"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0x99d41cb6fc639987c75e0d588573e58c64e8bd2636221c1fb6157659e4e01401"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x5d86944b3ef6b118dac48fdb38729f0a07cf680cf6a6eed465fb5d1641ac9d2e"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x2a5a2e9b4212498c39d1896cebf49cae2309e9aed3b0950690ae2c83caa22289"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x8514500b043baa410c49f39e7cd733323cebef311e56e9c250a5f11e2d710d80"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xb9444ea71b5207372070ac6832705fc39044dc94c2b0ae056d5062036db4f950"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0x147849b9da3fae288517f3aac9b8a2d00ec39a0d295767fdb241adf8f5a49a99"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x7ed273b12f3582814d96fd4373b36956e31b02c5ba7d045730244ae375343829"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x2d7d83745e835414f07ec1573633fb874a659ac0a6efc3740c62a8e4acbdb900"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xc5bf9db32da35df78babd7c2ce60adf6acb3b6ed956dba0d3dc396a1a4995a7c"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xe061b9afdc81811c349b23076fdd106df4c04b19764cb81e0ca8c63526f74f6c"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0x4769f87166a2d9930353464298e1c3c332e5aea72abd2713d383e9eaff2579cd"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x7caba53eadda34d9549f48ade20b197c635181fbd643b69d77e51146561fde13"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xd43b9f6891f3862a6bd968ccd63393f8f09f21c5121e327bfe8465b69c520007"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x77b2228d176ec803a6fe524b89ca2fac98aec1ca486573f6f236b0b486bd7a8d"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x51429d57c80d52138c0bb6e2892be0df615627e9b237713247fd2c669fec6c81"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0xb14d1016233aac1075af137ca419da3c3101aa42d490fcfb138fc50c19d1e288"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xe354ebaa8c98c423b9d7b6d846a9189fb6af18770a6f8b4dce22557b8adf3bad"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xb082f21bee4ed0e94e770ff23ebc812a648ca9dd266d7c6c11a0612b85cbd78c"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x9615f3a58aac859b3fd077d139e48b63968e18e9efdc559a6fb91276ac798794"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x4a3e04a8a3c42019d96d82e9aae2932573094ca784fdbbc83c8b018b83a58713"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0xada7704d19d0f1061c7f4d912a606fc813fc2b363b9deda18e7d7bbd6b035389"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xe1603e4dbfe6ea9bb344014f529ce48cf29ab8faebd834bb663e06328ae59f34"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x67320dfddecebc6d878ee23448011df0dff48fc30518f70dd7dd428c899ea81f"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xbbc0fbd4f77c046fc4dae0ee1770dd2dc5c80bc8c037c31c7739ee6ef607152e"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x644981ced818a771e7e5b9277f56a6ddcd29a6c1401312afc019c28d9bdc4ead"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0x38ba29f3a2d36bb4a0b29f1b9294b892048c483b0db9177f6c4b5814979bb02f"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x57ad6acf483238546fa22d3491327b6972f7baba542716afecf663f3608feab9"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x9b5cb3c9ca48b4be90ca0dab8f1a4ab71e0510463036c46a0762b7f4d8055307"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x57a0497788887de9f1fddcebdfbe7544b19ace7f475cb093ea822615b4b29e6b"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xddc17ca30d647729d185f266b68c9be82ddd999604f4fbd901930e592bc2dd40"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0x75a272582652f438796473f0a2595539a6495a6992a954da20dc6ba49f28c9da"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xadc585fe24a188b84c0def0c2ffc555286e9b31f97115413067de0e1fac90565"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xd58b7d6ff5b21721baeefbddd77467f25d6222c75901010ca170ba155b81114a"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x52843863efcdaff44ed6fe73533f90963539a048ed6b6d747ad2973156bbdb0d"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xc7b440f45e37d40f65a8a568b69ef8b38939b32710f6a90c36124a60e2de5ebf"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0xe47a12a00ac1f6fde11cc3298666b608c26112da2b6f5486267ef2fec0e6a997"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x5462e7c565590bed5fb3b1aed9a8a6213f28c146f6bb748052154ed1bd235122"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xc8094e13acbe8772c7f11466bdfd788eccdc15dc66167f10c4c5368c9e570e81"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xef3d538162a64042c74243b3c169a53c5934155279c347d47d12f63e3a62dd68"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x372d564d6213be29016ccf27ae096983f13e240e79cc9347074248e9d19aceea"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0xf6f7c2ec69ae2c24efc7e38157c8019a0701c9f3a5a4f065f35389cea9293dfd"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xf48a96783668b0c5de0dbd2909c7670330e70ff1987c3a31f5cadd147099c47e"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xd436f9f59da569687d3e9248bb8580175aace497919cf6725bb69d50ae3f4560"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xcdbf9206e132388a1127e781cdf5db4eeb8d1d413b06bee32a9191c019829505"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xbd95c581fae4dee8bcce989336e585de0fab12480dfcbe1038e106a47c06a336"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0xd1843c2f1ff0a283c871b5fbf324f037afe2f6312f5e800af7789302d4ed7943"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x796ed294a38f6a8d4f6fd60b2c82c1613eae3ec6b3a3d304254f6968656fc85d"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x361b62ff78d2e8d68c7e96e10930685598fac11afbfbaa28ef8b2aba5fb0ab66"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xe27d2b4f1d70db08a9f55b7693d7d25ec895398a0cb61ea6706b915c542ebcf0"), big.NewInt(10000))

	stateDB.SetBalance(common.HexStringToAddress("0xdf63b5f27a6c087d5364b8307e9c866b3bc6d49ff6b42c15fee618e4c888055"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0xac868802da2fbc5fc98a242b7dddf09ddda453c7c19ecf0364c852be585ce75"), big.NewInt(10000))
	stateDB.SetBalance(common.HexStringToAddress("0x114a6325ceb1b6d0ab7e8bd8c13b63ed2c8c2cd1be08f12121edff0bb28b4db"), big.NewInt(10000))
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
