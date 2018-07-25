package tvm


import (
	"testing"
	"math/big"
	"vm/common"
	"vm/core/types"
	"fmt"
)


func TestVmTest(t *testing.T) {
	a := &BlockChain{}
	vm := NewTvm(a)
	vm.Execute("import tas\ntas.test()")
}

func TestVmTestContract(t *testing.T) {
	VmTestContract()
}

func TestVmTestClass(t *testing.T) {
	VmTestClass()
}

type BlockChain struct {

}

func(b BlockChain) CreateAccount(address common.Address) {
	fmt.Println("CreateAccount")
}

func(b BlockChain) SubBalance(addr common.Address,value *big.Int) {
	fmt.Println("SubBalance")
}

func(b BlockChain) AddBalance(addr common.Address,value *big.Int) {
	fmt.Println("AddBalance")
}

func(b BlockChain) GetBalance(addr common.Address) *big.Int {
	fmt.Println("GetBalance")
	return big.NewInt(0)
}

func(b BlockChain) GetNonce(common.Address) uint64{
	fmt.Println("GetNonce")
	return 0
}

func(b BlockChain) SetNonce(common.Address, uint64){
	fmt.Println("SetNonce")
}

func(b BlockChain) GetCodeHash(addr common.Address) common.Hash {
	fmt.Println("GetCodeHash")
	return [32]byte{}
}

func(b BlockChain) GetCode(addr common.Address) []byte {
	fmt.Println("GetCode")
	return []byte{}
}

func(b BlockChain)SetCode(addr common.Address,code []byte) {
	fmt.Println("SetCode")
}

func(b BlockChain) GetCodeSize(addr common.Address) int {
	fmt.Println("GetCodeSize")
	return 0
}

func(b BlockChain) AddRefund(uint64) {
	fmt.Println("AddRefund")
}

func(b BlockChain) GetRefund() uint64 {
	fmt.Println("GetRefund")
	return 0
}

func(b BlockChain) GetState(addr common.Address,hash common.Hash) common.Hash {
	fmt.Println("GetState")
	return [32]byte{}
}

func(b BlockChain) SetState(addr common.Address,hash1 common.Hash,hash2 common.Hash) {
	fmt.Println("SetState")
}

func(b BlockChain) Suicide(common.Address) bool {
	fmt.Println("Suicide")
	return true
}

func(b BlockChain) HasSuicided(common.Address) bool{
	fmt.Println("HasSuicided")
	return true
}

func(b BlockChain) Exist(common.Address) bool {
	fmt.Println("Exist")
	return true
}

func(b BlockChain) Empty(common.Address) bool {
	fmt.Println("Empty")
	return true
}

func(b BlockChain) RevertToSnapshot(int) {
	fmt.Println("RevertToSnapshot")
}

func(b BlockChain) Snapshot() int {
	fmt.Println("Snapshot")
	return 1
}

func(b BlockChain) AddLog(*types.Log) {
	fmt.Println("AddLog")
}

func(b BlockChain) AddPreimage(common.Hash, []byte) {
	fmt.Println("AddPreimage")
}

func(b BlockChain) ForEachStorage(common.Address, func(common.Hash, common.Hash) bool) {
	fmt.Println("ForEachStorage")
}

func MockBlockChain() {

}

func TestVmTestABI(t *testing.T) {
	VmTestABI()
}