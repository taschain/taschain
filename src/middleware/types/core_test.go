package types

import (
	"testing"
	"common"
	"storage/serialize"
	"fmt"
)

func TestTransaction(t *testing.T) {
	transaction := &Transaction{Value:5000,Nonce:2,GasLimit:1000000000,GasPrice:0,ExtraDataType:0}
	addr := common.HexStringToAddress("0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
	transaction.Source = &addr
	fmt.Println(&addr)
	addr = common.HexStringToAddress("0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4c")
	transaction.Target = &addr
	fmt.Println(&addr)
	b,_ := serialize.EncodeToBytes(transaction)
	fmt.Println(b)
	addr2 := common.HexStringToAddress("0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4b")
	transaction.Source = &addr2
	fmt.Println(&addr2)
	addr2 = common.HexStringToAddress("0xff5a3f5747ada4eaa22f1d49c01e52ddb7875b4c")
	transaction.Target = &addr2
	fmt.Println(&addr2)
	c,_ := serialize.EncodeToBytes(transaction)
	fmt.Println(c)
}