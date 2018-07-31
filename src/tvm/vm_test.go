package tvm


import (
	"testing"
	"vm/common"
	"vm/core/state"

	"vm/ethdb"
)

func TestVmTest(t *testing.T) {
	db, _ := ethdb.NewMemDatabase()
	statedb, _ := state.New(common.Hash{}, state.NewDatabase(db))
	vm := NewTvm(statedb)
	script := `import tas
import account
account.create_account("0x2234")
value = account.get_balance("0x1234")
value = account.add_balance("0x1234",10)
account.set_nonce("0x1234", -1)
print("")
print(account.get_nonce("0x1234"))
#tas.test()`
	vm.Execute(script)
}

func TestVmTestContract(t *testing.T) {
	VmTestContract()
}

func TestVmTestClass(t *testing.T) {
	VmTestClass()
}

func MockBlockChain() {

}

func TestVmTestABI(t *testing.T) {
	VmTestABI()
}

func TestVmTestException(t *testing.T) {
	VmTestException()
}

func TestVmTestToken(t *testing.T) {
	VmTestToken()
}

func TestVmTest2(t *testing.T) {
	VmTest()
}