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
	"github.com/vmihailenco/msgpack"
	"math/big"
	"middleware/types"
	"storage/account"
	"storage/serialize"
	"storage/trie"
	"time"
)

const ChainDataVersion = 2

var emptyHash = common.Hash{}

func calcTxTree(txs []*types.Transaction) common.Hash {
	if nil == txs || 0 == len(txs) {
		return emptyHash
	}

	buf := new(bytes.Buffer)
	//for i := 0; i < len(tx); i++ {
	//	encode, _ := msgpack.Marshal(tx[i])
	//	serialize.Encode(buf, encode)
	//	buf.Write(tx)
	//}
	for _, tx := range txs {
		buf.Write(tx.Hash.Bytes())
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

func setupGenesisStateDB(stateDB *account.AccountDB, genesisInfo *types.GenesisInfo) {
	tenThousandTasBi := big.NewInt(0).SetUint64(common.TAS2RA(10000))

	//管理员账户
	stateDB.SetBalance(common.HexStringToAddress("0xf77fa9ca98c46d534bd3d40c3488ed7a85c314db0fd1e79c6ccc75d79bd680bd"), big.NewInt(0).SetUint64(common.TAS2RA(5000000)))
	stateDB.SetBalance(common.HexStringToAddress("0xb055a3ffdc9eeb0c5cf0c1f14507a40bdcbff98c03286b47b673c02d2efe727e"), big.NewInt(0).SetUint64(common.TAS2RA(5000000)))

	//创世账户
	for _, mem := range genesisInfo.Group.Members {
		addr := common.BytesToAddress(mem)
		stateDB.SetBalance(addr, tenThousandTasBi)
	}

	// 交易脚本账户
	stateDB.SetBalance(common.HexStringToAddress("0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xcad6d60fa8f6330f293f4f57893db78cf660e80d6a41718c7ad75e76795000d4"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xca789a28069db6f1639b60a8bf1084333358672f65c6d6c2e6d58b69187fe402"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x94bdb92d329dac69d7f107995a7b666d1092c63eadeae2dd495ab2e554bb155d"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xb50eea221a1eb061dea7ca20f7b7508c2d9639e3558e69f758380e32624337b5"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xce59fd5e1c6c99d9990b08ccf685260a2b3a03889de56e91b25878a4bf2f89e9"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x5d9b2132ec1d2011f488648a8dc24f9b29ca40933ca89d8d19367280dff59a03"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x5afb7e2617f1dd729ea3557096021e2f4eaa1a9c8fe48d8132b1f6cf13338a8f"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x30c049d276610da3355f6c11de8623ec6b40fd2a73bb5d647df2ae83c30244bc"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xa2b7bc555ca535745a7a9c55f9face88fc286a8b316352afc457ffafb40a7478"), tenThousandTasBi)
	//小熊本地测试轻节点账户
	stateDB.SetBalance(common.HexStringToAddress("0x7c608b0e25bba75a6bde6e87b2b007354cfc42f9bda481f59082f68fe4a9446d"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xe75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xdc57f6994098a8ac04e1c73064ee93e5edc8dbe0e47e8865ed01df7304d2bf73"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x352dfb7802f3fccf3cb571dd8000ba5cd2386fa70db9c415d34588905f233017"), tenThousandTasBi)

	//阿里云账户
	stateDB.SetBalance(common.HexStringToAddress("0xe75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xd3d410ec7c917f084e0f4b604c7008f01a923676d0352940f68a97264d49fb76"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x9d2961d1b4eb4af2d78cb9e29614756ab658671e453ea1f6ec26b4e918c79d02"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xcb6389bf1afd2817477ea5c1e191fd65a08e9250b3a4f5ecf7ac3a82d5ece1b7"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x7b81717d0b812165b6a937a77c992e5192e00890caff654badb99a351cd326e7"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xf4970eb761a6ea682dd060eb0ef10bb0df4488b4cc7f856e7c87c4ca176b35db"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x16acfb210b71f09b4bcc698afc4335878d72676aaad8ae79266fc30369239ac8"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x98b67e8534d7d1ab736dcf30a495efa922c7a2d0ff5d268e3fbc4cfb24cf77a6"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x54d72f8a8107c84036af68921a2808435010ec8a6b0e34910337ffb336479d1b"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x67cdc0a9f615097aed3a040039e49b072af499cb35db5bfd889cb67173cb4e90"), tenThousandTasBi)

	stateDB.SetBalance(common.HexStringToAddress("0xe620172fa355ddfd4f7db2a3dd0fdf645b5b0fb1e14798e3dc8e54b71188d554"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x3948d5e745ce95857fd70d650a5088a12631aff96e360a721ff3aad9dc68ecc4"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xf163f9a9b956051b2166ea29cec4be4858ced322342dd14cfb14fbfea3fa791a"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xf46f475ab25be472338a61e6f4864de81b6561c0300dd9aaa427fa143e7152a8"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x92bb3e243e874f37a9c98805a24c4133c940a5827a4494482d19e41f9947083e"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x160ec57f11386ac80368644d43ef922b88badd563be0c2672408f89cd0e59e31"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xe0aa93b4c9ffa5fe78bfd3a4f7317b6e0b44e283fb76361187f5adf8eff16ebc"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xed4873146c2fb5b44c34cc24a448e0d6dcd515f3f2251d24de1229406cbe702a"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x35f1c07d83af4653c35c959a794757bef4df4fdf276a5cad2d1aba17d69e70d5"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x24f5ab29a3dade1d468b23f60d4d7bb7952b49c8bcd6e0e0d259bcd1ffd535c0"), tenThousandTasBi)

	stateDB.SetBalance(common.HexStringToAddress("0x71e66738332cdf7de6052dd9ec1f8ce7487a2c8064c34b26baaa37ff013b72c4"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xfe5a3c8fe23fd648bd6f58ed7bf10b30051cf5f0de1dbab809c31b96684ac7ef"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x6662f1aee90c6b982ed198aca01c58ee58124ea8d7f8df1fc5f7188b19f32402"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x46c427e1f96236342e42df65a61b5e78de822817b6403aaae319becde7fc8851"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xcd8b917af5dfb9e0842f8692587a37332bf06aaee2743e83a51cfa5c05b931dc"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xe100c6159ff0981b9a84525435f41101a9fd2d7874c726aa2d1d95b1ebdfcfc1"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xe44e8a382bddfa80f5308eb97dc21a8678e7988ac07151a1499ac24b9bc10bcc"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xac154005491b0a8d80ac24e4de8e7ebfccc49feb016cb67286a13f51b90ea537"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xf2248d3e889532a27f08a1db8192ba1c144e65fe6c58b59e08d070a93d5ec259"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x4b511e43d1c5006eafe88b8bcaaca3e2aec1c6b25ec2e23309a51d42bf2ae771"), tenThousandTasBi)

	stateDB.SetBalance(common.HexStringToAddress("0xc4ceca5aa89b90fb64854409fbba619dd16bfa1a2fb268b489f1f0278ff36cfb"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xe986f5acdcc427b25cc958743d5cf06dc348b71fdb2c586c2ea2b818d19ca541"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xfd5ad81aac34bd140701b3f6403a98c80f659b483bbed1b921d4d0f5b7f7dfa3"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x44e510ed3459380de08edf8a2b3d08931ef638ef1508e924e6a20ff386966c89"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x72ceaca6300453d35df24c9cb4f18769edcd3a974c580bb0458c1499d9953e25"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x87aea89cde1b54edafd32e501623a8c967c6bef504aa2dde274e38431f37923f"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xff70a465473fdda118574a47af326269adc3b123e7e519dcedde1d52a154afdf"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xb3a4f25bfc7a0506a94137f480c254ae84c58e4c5c9483af7c1d7a2d3864d0d6"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x3fd09a4434fdb19c606ac24f17bbced83b8146be90d35e7b982425aa2044de11"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xcaf4f10f1702d846cffdab4897646882be8a6aa4d771834e9d7fe1f85eff6b22"), tenThousandTasBi)

	stateDB.SetBalance(common.HexStringToAddress("0x892fe35dc0300073e3e395581ba595ec7870bab3cdaf6772614b9923df03592d"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x1e06b3138c71596b72f6ecc5e02ec46637c6a4608771dcfb7f03e6033c8176d0"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x74cd75f303000575159531b4dc49c6bd7a06f85a33c96af47d4e25fa72e9ac0e"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x16a2373685d18c99c39447f41ca278cd69a199b6d9fa267e046ce475d466e266"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x7f1a08e58ce142ae107570782f8c4138c5cbdb06a7515985b04c467a526aca7c"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x67ed53c210d08335f41bdd18b3faab9d2f969db0e75a9e080ec288fbd022eafe"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x450d1c40c69e1f0c3d0c1d0c5981d42d47b250f7279e01d4b812c0da6a5951d4"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xab3a6a7190e55f70e94b66ccdad1de16a8758204a15ec644c6617545eb43a217"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xb6404b846a71ba17762de9c4b5e0c55f3127cdcfa3559b2d7c2e857500e15f30"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x4347c0bd265b2a9033aae1f0847f260514a21a4b4b24e118cb8e2301dedc10dd"), tenThousandTasBi)

	stateDB.SetBalance(common.HexStringToAddress("0x4934bb3d8b402df0a4ee29f0c010c01958ea99d9e7969d3e0fa0cb71d1859d27"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xd45f821aa75d549fa22bd974acf00d5cf9fd3e430703cfe3d0fa56705a2d04be"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xdad08a11cc33fb5ffc7f25be611497337f764b797e172e5f4846bd3b4399a43f"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xfa754c3fbee8ec1ea67d7f52fc398215a7bb3a91cfde3a7e6ae9a648a7f2f59e"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x4090c936c3a8267330a94ce668cbf76dad699f80752252ea4681b48d45ce5359"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x5f487bd52b298fa419ba2a811a1a4b8c1ee11ef59000d25e6c438d32271cc46f"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xd89657480642f2c189396892c8509dda4134b8b21707f30c218a181b6048a4ec"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xf9a4b16220455da8075c5aa5e9b2ed6835c090374450d1611470e28a4bf59648"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xe086585848db46cf5d2b2f7c8cae48bcca87a9d070626dbbdabde58df808ed5a"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x8d2797fa1aa2b012e1c8268ffa8533359ad7ee5271f2c1c3ee85cc5c0a9f3949"), tenThousandTasBi)

	stateDB.SetBalance(common.HexStringToAddress("0x9ff0c002793b2d4e0abb2a29996f3e9bbf0e042fd3d8251b88b37a0fca57a15f"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xe646868830038c6eb7550c270fe915de0de396cdd285dd623508736066f1f515"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xc4e74a1c3438d8f6d561713b18d950825949d520c89d2377311a1c05ae8a3840"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x873c0ff7a3856aa9311fc622ea7515983a528d3f03543d2f5e4184b406a9231d"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x9736c7f65b4a838f8a9f49492b9fecffc5debf9185b4fd1a414d4c2584042926"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xcf33219cce7eb99d8864e53ee671df6aab8701e2eb85ac13052b9c73602e3c1a"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x2879d5478f35de6f05dba639c29ddd922848962ce5b02daeadbc13ec7b3edc91"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x90b7eb7cef4b1356de2baafe0a2fd85973519668ba03a5a53b96c85533c53604"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x965d8617a3a0549338ddbb1afc2f13174415eb615ea68b8b0da3d66c0bf85eb5"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xff1d7a8a179e3d87dbaa99fdb21965e6115cd31e7dc3725f6ca8b312a90f9f7f"), tenThousandTasBi)

	stateDB.SetBalance(common.HexStringToAddress("0x996121978a5385bcd5efa5fbdd2d2f411f2a0b9da7b8a2b608c7f033680b4ca3"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x6f65affe2a58503bbb404711be2e0887f224fba0009506dec44b8f4ac2c859ba"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xed4cf4e255714a0f2ac58914c357efc347b70b7479ac64d5ac4ea93239de144a"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xd593ce541a2cc6d654508fb102ff29428adac02f46926ee33d20d17a66e2bc3d"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xf989c10a288abf0580674bee752738a2b9ee4a5ee3de89436d6adadfc58c8d12"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x358603c02237295f5137b865539a6ac989e5488fba40d76cefb635d67a100f7f"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x40f3a027dad8e9e14b1837c761ace085073e3a492d372928c750aea4baad37ca"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x21e65c02d7620da3af945ffd4c05d39a662348cc96cdc059d56ed00bef04b530"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x39a17b07825ebadbb58c90c2c862511db062db1ef161abeb801b1add70fc29e5"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x18791b6f3f6f91a0d7c0ae5f81425e363554795df3f2373c453742d70c5e6f2f"), tenThousandTasBi)

	stateDB.SetBalance(common.HexStringToAddress("0xa2c91414afedc2bddee333d5f653bb865e6b460199609cf4811206655a972163"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xe03adda90d12d4ff4ac5b51adca8295e68103b3d27808115e562134b95de4c77"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xdc8be2cebe7608aa87e549d1fc962387f0137fdf3f338ceec8a0ac205ebf3f37"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xadccffe4968f49ab5651fb61f2f09e9e355ef7e7bb4673a5f8c7034157706d68"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x4ed235a94a078e89393e4171e92b2fdb481e29e59e93c559d50c4f3ec5d3ad57"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xcf4ea129d93a278983db127c6ed0694c557cb3bf02030167d96821cafc4c08f9"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x6c15eac04d105ed9938238dea8af205714b74a672f3e3dcc69ed8111178617f8"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x217ffb3394e429aad0aa1d08f45c7a57739554cee1e931392f8c8265f2e6cee2"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x2ad3f4c07025ea51aab81cdef1610410a85ef1d066156bbed8b0044749ee5eab"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x63adfa6753d3d4b369f88f61a9b75251a71b571030c15f91dd897d9583f3e872"), tenThousandTasBi)

	stateDB.SetBalance(common.HexStringToAddress("0x42280931f59c88a63f69620a5128094fbf8fc61bed936adf5d2206802c8277ae"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x6bf8f008839fcc69a7ce2f0bd2c5a79eeab1fdb391cd6363ce7b4fc07f5820d3"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xf49fdbb194697e1d793f552416b10345312e0b219c1aba722541c1c83d523610"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x2c8035d4c4c14309556409f11e3abf8555c2cfc64320056b80d2f5cf483edcef"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xe90cf652216c7d97e194988d24d5bdbd6d8111dc444c3e4829fc2646e552ed90"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xea4a97533b63f5c29d0bfb20a6ba606d3a2d9f6bb7cbf3ce2b91c809542fcbf2"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x5235642fcef1c671e31b85b7db72ae0369ff03ae9d3d2b8ff64eeca34ac3a9dd"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x9c9148a568b13a73dcfc7ba9647a1cd3c47c68ebab514e011cc8cd8d241b198d"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x3f1d433ede58e8708370ecb5ad265e69a5463c5631e3674b2b9fbcb683635259"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x9e669433261cdb34ebc9a65a9627d8db2f3170cf0f8cda1c2a36c87ac2e112d3"), tenThousandTasBi)

	stateDB.SetBalance(common.HexStringToAddress("0x8dcfea2ac789dcdfc24f000c8448777852774330c15cecc136622833e0ae2192"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xfbb794512d03e9f09ecd0209cc725c2696989d0a2ab95dfb5223f18be073ef61"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x5a136d12991d5aa8449b5c651569a3b44a1cf0ca6d3783bd71cf864f31aa9a1f"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xe7a08ad5e50480319f9336e0e2bd6b4670201ed83f1af13cafa1d9636c8f09e4"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x5d6656e53f2336400dea6be0e455d6d2c13e7b1d2ceb9a060fe1802eee9ae88a"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x84928e24d54c29a287746ea739565101933dd78f3f68ad354fdd219182dee488"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x52006e7c79b851bbc31071878817d2a642e9448dcffb2a91c6936ec61364fd51"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xf34107b25ddc02974491fe506ef4edd450ccc0b68c1ec6cdb7f31d325593642a"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xe2b772b155b1c546d663854a6a23c8b88dce1592cf6249e51e639eaff78f9ff6"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x2e44b7b6c1abdb2af7a9a8c270f9e34b75f66c4bc355ca6a21135e5ffdc913c1"), tenThousandTasBi)

	stateDB.SetBalance(common.HexStringToAddress("0xc9b4d51086b4f1158e66094e66d39a5af2ac3152f15b3a18bfce4c44ca12743e"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xe20fd601a8b8cf841855e226d1da3554783b545f1204504a067b85a40f3284b3"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x151e1fef174f691f4203e88f3436a5bd45f1b27fb0a7842a7445b08e4ee03a45"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x2549818f9b74c9b43c042dc7835026dd037443b05cf83f1821ad4ac8be7ffbe4"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x3f11697959b4c25f83d190c2f59aceee5533e16917796aa21863ab6e0155ff60"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x138d5e4fc80a910603668808a9edb8cb04e7c669b3d35388683032cf4740ec79"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xea10d91b7db5921b3b67812919eac2062747096b57ead0f0fcb74217d2cbb71f"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x1162f400c051fcd9a0c006225ec68cb42dce0731de181c84bd54369941d28a94"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xf77cd080da611946c9889214c0cb87053127de70d34bc77e82109afeab8e5b35"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x686bc07100ad3082c6d4b9b3bf37285407a562543ce8200dee4786f603e65304"), tenThousandTasBi)

	stateDB.SetBalance(common.HexStringToAddress("0xd4eef7f2bba3a7d0d82b379b49f6b6dd0e0936ea0e69c450a5397abd72edb97c"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xa1879558ed5e921074d4d201ccc42be8f8737ff61578f99cd3a663b6214f0d24"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x8099f7e9ff704baf411b6a09324a93d14aa01a2588d6a08d982683351835dd0d"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xa90235d4b53410e08369bd8f26482a8664d912c4b9aa02ff0ee3058e4f873092"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xec2c3b03e8eb2d0d527c92190d3a9f7441a503f1f3d78f1928f766bce2225860"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x310fb5ec6afd2d22f38734bfe11d67017d3a4ede156265b63399ac1cc49be345"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xe8a97a7f48d5b2c4296169ae0e21bb708572db246f6eec598b9e9666d73adc16"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x92c7a1448ce4d86e135260f9dcbb362424d1761fa870dcef418fe1adcce675a4"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x1bc691ba9d8485422c457113b004f8b4a9d9d92112099ea5eaae34281e2d2ebc"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xb8f9339d49dc45b23bc8ac8f113cc79dfc484dae9c889211688fb3ac46711bd8"), tenThousandTasBi)

	stateDB.SetBalance(common.HexStringToAddress("0x259be556ab4859dc59982c62cf8295cbae2b25583e0c7488dc757c42323b5394"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x2d7094789e6ff69b4bb31b5ef6b43164295adff1378bcc9a8787d0c8a7f2759b"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x304bf70f383459fe9013e81ec117347ed009b15e38159cf385daf3898c11217b"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0xec18b31abab7fb457684966310967780f0190305e72246c8e996a7fa98bc8bd2"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x5665956ed6d245323715b15b6f4b62c5c141e7422440fd596c00cf3696746dc7"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x93ee2d83911ae7abf0d89bb3a4c73b4169998b1a57d8589aedaa2bb51629395e"), tenThousandTasBi)
	stateDB.SetBalance(common.HexStringToAddress("0x2c6e9e2d789852af52427ee7e486e775107467740c73afd41456f402f0426013"), tenThousandTasBi)

	stateDB.SetBalance(common.HexStringToAddress("0x87c83e834e52fbea9ec7b47534d49fa9ae51c91e22e9b1dae0d3b31e8d8b9be3"), tenThousandTasBi)
}