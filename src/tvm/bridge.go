package tvm

/*
#include <stdlib.h>
 */
import "C"
import (
	"fmt"
	"unsafe"
	"common"
	"math/big"
)

//export callOnMeGo
func callOnMeGo(in int) int {
	fmt.Printf("Go.callOnMeGo(): called with arg = %d\n", in)
	return in + 2
}

//export go_testAry
func go_testAry(ary unsafe.Pointer) {

	intary :=  (*[2]unsafe.Pointer)(ary)

	fmt.Println(intary)

	testInt := *(*int)(intary[0])
	fmt.Println(testInt)
	C.free(unsafe.Pointer(intary[0]))

	testStr := C.GoString((*C.char)(intary[1]))
	fmt.Println(testStr)
	C.free(unsafe.Pointer(intary[1]))
}

//export transfer
func transfer(fromAddresss *C.char, toAddress *C.char, amount int) {
	fmt.Printf("From %s Send to %s amout: %d\n", C.GoString(fromAddresss), C.GoString(toAddress), amount)
}

//export CreateAccount
func CreateAccount(addressC *C.char) {
	addr := common.StringToAddress(C.GoString(addressC))
	tvm.state.CreateAccount(addr)
}

//export SubBalance
func SubBalance(addressC *C.char,valueC *C.char) {
	address := common.StringToAddress(C.GoString(addressC))
	value, ok := big.NewInt(0).SetString(C.GoString(valueC), 10)
	if !ok {
		// TODO error
	}
	tvm.state.SubBalance(address, value)
}

//export AddBalance
func AddBalance(addressC *C.char,valueC *C.char) {
	address := common.StringToAddress(C.GoString(addressC))
	value, ok := big.NewInt(0).SetString(C.GoString(valueC), 10)
	if !ok {
		// TODO error
	}
	tvm.state.AddBalance(address, value)
}

//export GetBalance
func GetBalance(addressC *C.char) *C.char {
	address := common.StringToAddress(C.GoString(addressC))
	value := tvm.state.GetBalance(address)
	return C.CString(value.String())
}

//export GetNonce
func GetNonce(addressC *C.char) C.ulonglong {
	address := common.StringToAddress(C.GoString(addressC))
	value := tvm.state.GetNonce(address)
	return C.ulonglong(value)
}

//export SetNonce
func SetNonce(addressC *C.char, nonce C.ulonglong) {
	address := common.StringToAddress(C.GoString(addressC))
	tvm.state.SetNonce(address, uint64(nonce))
}

//export GetCodeHash
func GetCodeHash(addressC *C.char) *C.char {
	address := common.StringToAddress(C.GoString(addressC))
	hash := tvm.state.GetCodeHash(address)
	return C.CString(hash.String())
}

//export GetCode
func GetCode(addressC *C.char) *C.char {
	address := common.StringToAddress(C.GoString(addressC))
	code := tvm.state.GetCode(address)
	return C.CString(string(code))
}

//export SetCode
func SetCode(addressC *C.char,codeC *C.char) {
	address := common.StringToAddress(C.GoString(addressC))
	code := C.GoString(codeC)
	tvm.state.SetCode(address, []byte(code))
}

//export GetCodeSize
func GetCodeSize(addressC *C.char) C.int {
	address := common.StringToAddress(C.GoString(addressC))
	size := tvm.state.GetCodeSize(address)
	return C.int(size)
}

//export AddRefund
func AddRefund(re C.ulonglong) {
	tvm.state.AddRefund(uint64(re))
}

//export GetRefund
func GetRefund() C.ulonglong {
	return C.ulonglong(tvm.state.GetRefund())
}

//export GetState
func GetState(addressC *C.char, hashC *C.char) *C.char {
	address := common.StringToAddress(C.GoString(addressC))
	//hash := common.StringToHash(C.GoString(hashC))
	state := tvm.state.GetData(address, C.GoString(hashC))
	return C.CString(string(state))
}

//export SetState
func SetState(addressC *C.char, hashC *C.char, stateC *C.char) {
	address := common.StringToAddress(C.GoString(addressC))
	//hash := common.StringToHash(C.GoString(hashC))
	//state := common.StringToHash(C.GoString(stateC))
	tvm.state.SetData(address, C.GoString(hashC), []byte(C.GoString(stateC)))
}

//export Suicide
func Suicide(addressC *C.char) bool {
	address := common.StringToAddress(C.GoString(addressC))
	return tvm.state.Suicide(address)
}

//export HasSuicided
func HasSuicided(addressC *C.char) bool {
	address := common.StringToAddress(C.GoString(addressC))
	return tvm.state.HasSuicided(address)
}

//export Exist
func Exist(addressC *C.char) bool {
	address := common.StringToAddress(C.GoString(addressC))
	return tvm.state.Exist(address)
}

//export Empty
func Empty(addressC *C.char) bool {
	address := common.StringToAddress(C.GoString(addressC))
	return tvm.state.Empty(address)
}

//export RevertToSnapshot
func RevertToSnapshot(i int) {
	tvm.state.RevertToSnapshot(i)
}

//export Snapshot
func Snapshot() int {
	return tvm.state.Snapshot()
}

//export AddPreimage
func AddPreimage(hashC *C.char,preimageC *C.char) {
	//hash := common.StringToHash(C.GoString(hashC))
	//preimage := []byte(C.GoString(preimageC))
	//tvm.state.AddPreimage(hash, preimage)
}
