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
	"middleware/types"
	"storage/account"
	"storage/serialize"
	"storage/trie"
)

var testTxAccount = []string{"0xc2f067dba80c53cfdd956f86a61dd3aaf5abbba5609572636719f054247d8103","0xcad6d60fa8f6330f293f4f57893db78cf660e80d6a41718c7ad75e76795000d4",
	"0xca789a28069db6f1639b60a8bf1084333358672f65c6d6c2e6d58b69187fe402","0x94bdb92d329dac69d7f107995a7b666d1092c63eadeae2dd495ab2e554bb155d",
	"0xb50eea221a1eb061dea7ca20f7b7508c2d9639e3558e69f758380e32624337b5","0xce59fd5e1c6c99d9990b08ccf685260a2b3a03889de56e91b25878a4bf2f89e9",
	"0x5d9b2132ec1d2011f488648a8dc24f9b29ca40933ca89d8d19367280dff59a03","0x5afb7e2617f1dd729ea3557096021e2f4eaa1a9c8fe48d8132b1f6cf13338a8f",
	"0x30c049d276610da3355f6c11de8623ec6b40fd2a73bb5d647df2ae83c30244bc","0xa2b7bc555ca535745a7a9c55f9face88fc286a8b316352afc457ffafb40a7478"}

func IsTestTransaction(tx *types.Transaction) bool {
	if tx == nil || tx.Source == nil {
		return false
	}

	source := tx.Source.GetHexString()
	for _, testAccount := range testTxAccount {
		if source == testAccount {
			return true
		}
	}
	return false
}


func calcTxTree(txs []*types.Transaction) common.Hash {
	if nil == txs || 0 == len(txs) {
		return common.EmptyHash
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
		return common.EmptyHash
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

	var testAccount = []string{
		"0xe75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019",
		"0xec6cddf1ee4d91832ad3fc658a1e4aae1741387b3c7bc176cb15399073454f23",
		"0xc9a2afdacceb7b2d0f50a19cd5bc2683f46e70b392cb66ddf759c3927fc591bf",
		"0x032f417a71b351cd4320a0a0859271ddbd257aa8ab22a2e35aeea2a02971bd8f",
		"0xa244d146ad492a4dacd2a1edd455292464e347687b331a1e5cd7484c8bec366c",
		"0x0553ad3d999a2439b12cf66d9822bcd0531ad9b29b6fc7e0b9b925a4e8720d33",
		"0xbb2a3d5b9dcec5bb4fbe251e29492eed2b0a813b9c69ab562c7a57bffe0da94b",
		"0xcc3532d62f9e88a5d0d4648a90c4248fb42eea81074d97f9abe94da83836ed33",
		"0xf4729dbefed95d8a3ecc77c5f860c1c239e8b0e018c02edc58f4a3654c959bd0",
		"0xdb282ff9c7ec38aa279240ed095770f2e4b9ddf9804d49d9736f025f33958bfc",
		"0xd92fc95c55c9852b38e4b2d97741f66204b4e3cf8ce70eab7d65de8e8f856551",
		"0xd3d410ec7c917f084e0f4b604c7008f01a923676d0352940f68a97264d49fb76",
		"0xa74b8fe517bed727970d1f33fbcb939178bb5aa0ea6b2143d419e6beca6d7b0c",
		"0x610a6280bfcebb78e071a55cf27f8a0673a432f732c28b181f031a0bae345d70",
		"0x0125fdef7e3412ffa4a48b4b58ba308e8b2ec9a32b90c9f013f5568ddd8c1e9f",
		"0xeb7f91f3a0bab3c5730e73920ebdcd6112f4768392ab3f55dfffd4e5e8719e2c",
		"0x5bb2eda528f490fef7d0152b36156c3667aebb46d1a5394d54e4bbc3de45d93b",
		"0x624921f7d4384eb0befc11ef5f529a42c50caaf715dd2ccff4216e60d0186c6e",
		"0xeac6dbaf89c9b71d1442ecff3fc165373fbf56b4846ed83cd8b501750168ccea",
		"0xa3654c83e3c21bf518f8fd4731c408c56ad4f65cf7f14b5288c47b87c47ce533",
		"0x17f1ebb309d7b0ee6185bc10c648508a3863e8fb260f961731d86abd556377ba",
		"0x9f65bd5ad9b5d996e1825b3e528d8b1e9c5acd6e576498775290316e7bf57df5",
		"0x9d2961d1b4eb4af2d78cb9e29614756ab658671e453ea1f6ec26b4e918c79d02",
		"0x213818b4e39f15d21be5e634dec892fa243f4df9d988b83d0879798bbba3750f",
		"0x6af174b568cd374a64a7d99b260425982a19595ac5a66f6bef8549bd3fa7fcae",
		"0x88044a4001ea4af165a17c42cfe5659eca814ece8a385b89e2e42990821e8022",
		"0xc0538db88e1a9d772a93436529de9cedb232bfdbd5a3efcd066ee493f2b8d5bb",
		"0x45e547325e74406a0e462c4c0b487946c60d2525b39696c849ee72087b50ead4",
		"0x25ba46660713091879416f1ca553fc05405c28e3c1fb1f424d96943966f047f1",
		"0xdaf452c828aa47d13f61230124ea8a2e03932680e472dff3eaea4fab195921db",
		"0xc0ee35293253548cdae1e4c7f825c30ed92cf9f3839c7763639e394d5d96b6c8",
		"0x730edb5e5b9a7a15a6987dce1e8a937d999be9d0104d5ae8d9e111e4799c872f",
		"0x1dc287607ad5ac0726f1d6ec494613ad002b06b237ff356e9677dcc49ffdba6d",
		"0x5fd1e048b6c8ce7658d162e2f6d344aaf3fa701c29e20a7ec579b579f18796f6",
		"0xb405b0d35775ebd803800ab091a08e42fe4635dd6399f8dd905b271f705bbac9",
		"0xb7b9ccec11b1c77f45f259848f8bd843d2929a0f01111d6177c1391f89cc68ac",
		"0x4cc808a28cdc995bd5d5769d0c3967102f72c3059980bdf275324c1fb5c4bf6f",
		"0x3c91ca654b251f92c4173e311c000d0b71f48b03d96424d8511a7710d4f9d339",
		"0xcc812b9345a343f9ef16bb8c5a541f5a0a6afb7727340a0f61ce673a438c987b",
		"0x8c292d6458b057306006d9eee671cb55d8fc29cb310173ef85473720e13f94d8",
		"0x4a56757d5fa748a24f48866c6e95b7cbf1e78e9b7fcd26f68f17a5aae1fd591a",
		"0x374170882cf5807aa2945cbf0f51b6f4c0a858322a34a4e6132032d7f87cd56d",
		"0x343333bb4bc294519e9faeddb7131898ea9db11762016d260bd93ff85ea69032",
		"0xc412daf6e4b21cd25434a35c4ddd06defdf18f27ad31bdc42e621c036cac0544",
		"0x17745995fe663966fa202f2e3caa6af3b1ecaaffa6dbc4c3ab67f495c0a90494",
		"0x175d181744807a984c0ec6b8a93f1b7f257375f90051183c7d75df6831f5e93a",
		"0xa3c154995610af39e043c745e56267237d1504ad3d71a0d4d015f263d8005fc8",
		"0x3b536d0879b2b3619c870a3163b4e912f85766a50fc35e2038dbbd3d8eeefbad",
		"0x16c7932d0bbe05e876c65dd0cb4bea08401d5d48f496ea2b7b6551aef11b8741",
		"0xb1cac579971aa21899ee134ebef0900603b5b4a875d88c47fb9c6fae8a00f254",
		"0xd9b4c8bfbe8aceb510b89d54188b007ba966fdb58a193bbad68c149c37ecc5d3",
		"0xeabee4810954784d56e9a202568f642c4a21ef4148e2a6519c5839de9cd8d6db",
		"0x3fc461710a3697da82f5833dc8cf0acc2d7dfd5e3a779f848f442b4d46acd168",
		"0x72881d39f34c5a3527948ede53c238010939b4f41ce544c2e3c22dfa3daacc99",
		"0x0a5b90b2b00a8ab8cc62d6636a7a9fbde2e8f819d7e6511780fca44ef170ab8b",
		"0xf46ff5aa545797326aa85f2787a330394c8bddba445712b3138debb3166b5bce",
		"0x4b6aa076ca99436c94e6ce00f4c9ac3a458f075907b134e63289a6fb35d1b84a",
		"0x24255b5e2ad7b757c6835060aa46ce8383da4b59ce62d18e60cbe5c828eea85d",
		"0xf02a123815cb0c8242d3a6d50cc89eba4130bc29162a0f6e46189fecf5738d86",
		"0x5e8f39328bda9fb8ec8863c1bdba8486b8be2d689f2a1fd5143665b05285dfb6",
		"0xee471215da18a0045fb05936e800194661997ecd9c4ca5ef2f6a4c2c0d065708",
		"0xdcf2ca61b54290a90e43771fbdb0754e94d7eb6ec076ddc589eda7e19e29dbb8",
		"0xa2eb845bad3c66b24a7098cde90a13a3395aa69d5322c2d8d1b7e040f75074c7",
		"0x139f35702cd747f6619fc3136f53781aa6abbb3e56236aab6a1802d0701439e1",
		"0xe76d4381e86f58cb9e135e21a0d0efd7fe66666b3d13e340987708a4c370f037",
		"0x11b5274bfd71774d13d9b00cc0a94997714940e35ad927ac0a1fcc1ae32756ff",
		"0x387377373f57ba60169479834e2e3f91c9840c48229cd8dd063ec06dadcf6817",
		"0x26c02aa788db60ae5789b310b4f4bed50a2d6fa92c094b973328e131d3600174",
		"0x3470d0d4c12a5d2a43b4814f5336711d83043a62b3330bed6568b3cc3145a615",
		"0xce56a93f359f92795eb7e5b5f5fe483d6c9ac78c84a40dfa27cde20e10b0f684",
		"0xc67141f5ec6bb2ecb859625e21ad04e51d5c28f174ec5066eaaf694483ed43b8",
		"0x9c42046eaff5bb720a6d1bacf59d03a1e02e9701a15963cc00fefd9995223e2f",
		"0x7980c4e0c8b9bb9d78cfd4465499da5613260b7a356f3dbca86d5f0494bcad36",
		"0x7be6e16f76b6e2546fed2bb45b6d96fd6243ae261bfbb0b3956eddbe91ebb4b2",
		"0x8f75245cb58356c60221fb25b63b204d858d343dba2400ab37f2b6fd18331d6a",
		"0xfa6793893203c2c08559d500e546b9d99654473d98c07b7484d1d0d81f1ff7a3",
		"0xaeb13ad14da36586b80d3685108934878fb9a9e2633f895f9de48892c6b73620",
		"0xeccf6cee119f066e10c03e58c61e6b09857d5e411cdc9fb50cf2608612d80439",
		"0x59641451082cd586340577f577aafecbd6f480a34022a482df0a97f3acdae810",
		"0x5f4dbe2c8a220283690e8fd2edfc86ddb7c2a969dff413869aba191269e0cfdb",
		"0xf686e559e91e4c1f86c33fba31a9909be8b30bcf4c3df7b1a2d74f09402bd7eb",
		"0x848d21af9feb8d77efd8c2fdfa220204d5dfacb5cc661dfc76343e019c4f2f89",
		"0x6127aa2fa855a267d4edd53d7b7c5395c95f0eb3fbb3bf478bcb74111b6a2c64",
		"0x3ecc759f20d24b2a253a33c868848dec1776d638a7c6ba2f93a92870b8fe8c4e",
		"0x1a68b95b1d47cf537015a3432177aff80bcbd5660645aa134b596da455ada1b7",
		"0x7c0f552a1e91f3a798d1d0e084f339c1c5d26c01b682226f5229d1a94486d7eb",
		"0xbe70fb754844405e926b56ce431b9d77b75608d6a02d2244dbca8af0d65c2c5d",
		"0x5bd5cd9758152ad59c3e19c761ec70632fb68260bae6e6f6d9cce53b7ebf2b91",
		"0xfdc1341e8accc087ed670026151d85854808be5d6e3b50784df62d9b3b51db75",
	}


	////创世账户
	for _, acct := range testAccount {
		addr := common.HexStringToAddress(acct)
		stateDB.SetBalance(addr, tenThousandTasBi)
	}


	// 交易脚本账户
	for _, acc := range testTxAccount {
		stateDB.SetBalance(common.HexStringToAddress(acc), tenThousandTasBi)
	}
}