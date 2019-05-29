package core

import (
	"common"
	"fmt"
	"math/rand"
	"middleware/types"
	"os"
	"storage/account"
	"storage/tasdb"
	"taslog"
	"testing"
	"time"
)

var (
	executor *TVMExecutor
	adb *account.AccountDB
)
func init() {
	executor = &TVMExecutor{}

	ds, err := tasdb.NewDataSource("test_db")
	if err != nil {
		panic(err)
	}

	statedb, err := ds.NewPrefixDatabase("state")
	if err != nil {
		panic(fmt.Sprintf("Init block chain error! Error:%s", err.Error()))
	}
	db := account.NewDatabase(statedb)

	adb, err = account.NewAccountDB(common.Hash{}, db)
	if err != nil {
		panic(err)
	}
	Logger = taslog.GetLogger("")
}


func randomAddress() common.Address {
	r := rand.Uint64()
	return common.BytesToAddress(common.Uint64ToByte(r))
}

func genRandomTx() *types.Transaction {
	target := randomAddress()
	source := randomAddress()
	tx := &types.Transaction{
		Value: 1,
		Nonce:1,
		Target: &target,
		Source: &source,
		Type: types.TransactionTypeTransfer,
		GasLimit: 10000,
		GasPrice: 1000,
	}
	tx.Hash = tx.GenHash()
	return tx
}

func TestTVMExecutor_Execute(t *testing.T) {
	executor := &TVMExecutor{}

	ds, err := tasdb.NewDataSource("test_db")
	if err != nil {
		t.Fatalf("new datasource error:%v", err)
	}

	statedb, err := ds.NewPrefixDatabase("state")
	if err != nil {
		t.Fatalf("Init block chain error! Error:%s", err.Error())
	}
	db := account.NewDatabase(statedb)

	adb, err := account.NewAccountDB(common.Hash{}, db)
	if err != nil {
		t.Fatal(err)
	}

	txNum := 10
	txs := make([]*types.Transaction, txNum)
	for i := 0; i < txNum; i++ {
		txs[i] = genRandomTx()
	}
	stateHash, evts, executed, receptes, err := executor.Execute(adb, &types.BlockHeader{}, txs, false)
	if err != nil {
		t.Fatalf("execute error :%v", err)
	}
	t.Log(stateHash, evts, len(executed), len(receptes))
	if len(txs) != len(executed) {
		t.Error("executed tx num error")
	}
	for i, tx := range txs {
		if executed[i].Hash != tx.Hash {
			t.Error("execute tx error")
		}
	}
}


func BenchmarkTVMExecutor_Execute(b *testing.B) {
	txNum := 5400
	for i := 0; i < b.N; i++ {
		txs := make([]*types.Transaction, txNum)
		for i := 0; i < txNum; i++ {
			txs[i] = genRandomTx()
		}
		executor.Execute(adb, &types.BlockHeader{}, txs, false)
	}
}

func writeFile(f *os.File, bs *[]byte) {
	f.Write(*bs)
}
func TestReadWriteFile(t *testing.T) {
	file := "test_file"
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}
	begin := time.Now()
	cost := time.Duration(0)
	bs := make([]byte, 1024*1024*2)
	for i := 0; i < 100; i++ {
		b := time.Now()
		writeFile(f, &bs)
		cost += time.Since(b)
		//sha3.Sum256(randomAddress().Bytes())

	}
	t.Log(time.Since(begin).String(), cost.String())
}