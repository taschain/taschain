package core

import (
	"testing"
	"middleware/types"
	"time"
	"math/big"
	"consensus/groupsig"
	"common"
	"math/rand"
)

/*
**  Creator: pxf
**  Date: 2019/3/20 下午8:56
**  Description: 
*/
func genTx(source string, target string) *types.Transaction {
	var sourceAddr, targetAddr *common.Address

	sourcebyte := common.HexToAddress(source)
	sourceAddr = &sourcebyte
	if target == "" {
		targetAddr = nil
	} else {
		targetbyte := common.HexToAddress(target)
		targetAddr = &targetbyte
	}

	tx := &types.Transaction{
		Data:          []byte{13,23},
		GasPrice:      1,
		Source:        sourceAddr,
		Target:        targetAddr,
		Nonce:         rand.Uint64(),
		Value:         rand.Uint64(),
		ExtraData:     []byte{2,3,4},
		ExtraDataType: 1,
		GasLimit: 10000000,
		Type:		   1,
	}
	tx.Hash = tx.GenHash()
	return tx
}

func genBlockHeader() *types.BlockHeader {
	castor := groupsig.ID{}
	castor.SetBigInt(big.NewInt(1000))
	bh := &types.BlockHeader{
		CurTime:    time.Now(), //todo:时区问题
		Height:     rand.Uint64(),
		ProveValue: big.NewInt(rand.Int63()),
		Castor:     castor.Serialize(),
		GroupId:    castor.Serialize(),
		TotalQN:    rand.Uint64(),
		StateTree:  common.Hash{},
		ProveRoot:  common.HexToHash("0x123"),
	}
	return bh
}

func genBlock(txNum int) *types.Block {
	bh := genBlockHeader()
	txs := make([]*types.Transaction, 0)
	txHashs := make([]common.Hash, 0)
	for i := 0; i < txNum; i++ {
		tx := genTx("0x123", "0x234")
		txs = append(txs, tx)
		txHashs = append(txHashs, tx.Hash)
	}
	bh.Transactions = txHashs
	return &types.Block{
		Header: bh,
		Transactions: txs,
	}
}

func TestEncodeTransaction(t *testing.T) {
	b := genBlock(10)
	bs, err := encodeBlockTransactions(b)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(bs)
}
func TestDecodeBlockTransactionWithNoTransaction(t *testing.T) {
	b := genBlock(0)
	t.Logf("block is %+v", b.Header)
	bs, err := encodeBlockTransactions(b)
	if err != nil {
		t.Fatal(err)
	}

	txs, err := decodeBlockTransactions(bs)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("txs %v", txs)

}

func TestDecodeBlockTransactionWithTransactions(t *testing.T) {
	b := genBlock(10)
	t.Logf("block is %+v", b.Header)
	for i, tx := range b.Transactions {
		t.Logf("before %v: %+v", i, tx)
	}
	bs, err := encodeBlockTransactions(b)
	if err != nil {
		t.Fatal(err)
	}

	txs, err := decodeBlockTransactions(bs)
	if err != nil {
		t.Fatal(err)
	}
	for i, tx := range txs {
		t.Logf("after %v: %+v", i, tx)
	}
}

func TestDecodeTransactionByHash(t *testing.T) {
	b := genBlock(23)
	t.Logf("block is %+v", b.Header)
	var testHash common.Hash
	r := 24
	for i, tx := range b.Transactions {
		t.Logf("before %v: %+v", i, tx.Hash.String())
		if i == r {
			testHash = tx.Hash
			t.Log("test hash",i, testHash.String())
		}
	}
	bs, err := encodeBlockTransactions(b)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := decodeTransaction(b.Header, testHash, bs)
	if err != nil {
		t.Fatal(err)
	}
	if tx.Hash != tx.GenHash() {
		t.Fatal("gen hash diff")
	}
	if tx.Hash != testHash {
		t.Fatal("hash diff")
	}
	t.Log("success")
}