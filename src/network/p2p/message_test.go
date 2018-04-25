package p2p

import (
	"testing"
	"fmt"
)

func TestTxMessage(t *testing.T) {
	//defer taslog.Close()
	//
	//groupsig.Init(1)
	//crypto.KeyTypes = append(crypto.KeyTypes, 3)
	//crypto.PubKeyUnmarshallers[3] = UnmarshalEcdsaPublicKey
	//config := common.NewConfINIManager("boot_test.ini")

	//mockClient(&config)

	fmt.Printf("mock client over!\n")


	//seedId := "0xe14f286058ed3096ab90ba48a1612564dffdc358"
	//peer1Id := "0x3f8ffdd38cbc6df7386868d098d0b95d637c881f"
	//
	//ctx := context.Background()
	//peerInfo, err := Server.dht.FindPeer(ctx, gpeer.ID(peer1Id))
	//if err != nil {
	//	fmt.Printf("find peer1 error:%s\n", err.Error())
	//}
	//fmt.Printf("find result is:%s\n", string(peerInfo.ID))
	//
	//pi, err := server1.dht.FindPeer(ctx, gpeer.ID(seedId))
	//if err != nil {
	//	fmt.Printf("find seed error:%s\n", err.Error())
	//}
	//fmt.Printf("find result is:%s\n", string(pi.ID))


	//txs := mockTxs()
	//peer1.SendTransactions(txs, seedId)

	//peer1.BroadcastTransactions(txs)
	//
	//trm := biz.TransactionRequestMessage{TransactionHashes: nil, SourceId: peer1Id, RequestTime: nil}
	//peer1.BroadcastTransactionRequest(trm)
}

//func mockClient(config *common.ConfManager) (*peer, *server) {
//	privKey := "0x0497367b933f4be262cbfeeb808c84d91180717c7f6224d057de96da3d0acd3eb342a83e01fbf7afc9b24ea7d66b3dd1ddce8dde0259003e4c7c75142a64e95d82faf001b42326cb7d8c7d1234e5b0257e0f4e1da771badc866c14a632aca14cc8"
//	ctx := context.Background()
//
//	dht1, host, id, selfNode := mockDHT(privKey, config, ctx)
//	fmt.Printf("Mock id is:%s\n", id)
//	dhts := []*dht.IpfsDHT{dht1}
//	bootDhts(dhts)
//	time.Sleep(30 * time.Second)
//
//	host.Network().SetStreamHandler(swarmStreamHandler)
//	bHandler := biz.NewBlockChainMessageHandler(nil, nil, nil, nil,
//		Peer.BroadcastTransactionRequest, Peer.SendTransactions,nil)
//
//	cHandler := biz.NewConsensusMessageHandler(nil, nil, nil,
//		nil, nil, nil)
//	server := server{host, dht1, bHandler, cHandler}
//	peer := peer{SelfNetInfo: selfNode}
//	return &peer, &server
//}
//
//func mockTxs() []*core.Transaction {
//	//source byte: 138,170,12,235,193,42,59,204,152,26,146,154,213,207,129,10,9,14,17,174
//	//target byte: 93,174,34,35,176,3,97,163,150,23,122,156,180,16,255,97,242,0,21,173
//	//hash : 112,155,85,189,61,160,245,168,56,18,91,208,238,32,197,191,221,124,171,161,115,145,45,66,129,202,232,22,183,154,32,27
//	t1 := genTestTx("tx1", 123, "111", "abc", 0, 1)
//	t2 := genTestTx("tx1", 456, "222", "ddd", 0, 1)
//	s := []*core.Transaction{t1, t2}
//	return s
//}
//
//func genTestTx(hash string, price uint64, source string, target string, nonce uint64, value uint64) *core.Transaction {
//
//	sourcebyte := common.BytesToAddress(core.Sha256([]byte(source)))
//	targetbyte := common.BytesToAddress(core.Sha256([]byte(target)))
//
//	//byte: 84,104,105,115,32,105,115,32,97,32,116,114,97,110,115,97,99,116,105,111,110
//	data := []byte("This is a transaction")
//	return &core.Transaction{
//		Data:     data,
//		Value:    value,
//		Nonce:    nonce,
//		Source:   &sourcebyte,
//		Target:   &targetbyte,
//		GasPrice: price,
//		GasLimit: 3,
//		Hash:     common.BytesToHash(core.Sha256([]byte(hash))),
//	}
//}

