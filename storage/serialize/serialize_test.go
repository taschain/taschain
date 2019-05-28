package serialize

import (
	"fmt"
	"github.com/taschain/taschain/common"
	"math/big"
	"testing"
)

type Account struct {
	Nonce    uint64
	Balance  *big.Int
	Root     common.Hash
	CodeHash []byte
}

func TestSerialize(t *testing.T) {
	a := Account{Nonce: 100, Root: common.BytesToHash([]byte{1, 2, 3}), CodeHash: []byte{4, 5, 6}, Balance: new(big.Int)}
	accountDump(a)
	byte, err := EncodeToBytes(a)
	if err != nil {
		t.Errorf("encoding error")
	}


	var b = Account{}
	decodeErr := DecodeBytes(byte, &b)
	if decodeErr != nil {
		t.Errorf("decode error")
	}
	accountDump(b)
}

func accountDump(a Account) {
	fmt.Printf("Account nounce:%d,Root:%s,CodeHash:%v,Balance:%v\n", a.Nonce, a.Root.String(), a.CodeHash, a.Balance.Sign())
}
