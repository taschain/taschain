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

	intary := (*[2]unsafe.Pointer)(ary)

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
	addr := common.HexToAddress(C.GoString(addressC))
	AccountDB.CreateAccount(addr)
}

//export SubBalance
func SubBalance(addressC *C.char, valueC *C.char) {
	address := common.HexToAddress(C.GoString(addressC))
	value, ok := big.NewInt(0).SetString(C.GoString(valueC), 10)
	if !ok {
		// TODO error
	}
	AccountDB.SubBalance(address, value)
}

//export AddBalance
func AddBalance(addressC *C.char, valueC *C.char) {
	address := common.HexToAddress(C.GoString(addressC))
	value, ok := big.NewInt(0).SetString(C.GoString(valueC), 10)
	if !ok {
		// TODO error
	}
	AccountDB.AddBalance(address, value)
}

//export GetBalance
func GetBalance(addressC *C.char) *C.char {
	address := common.HexToAddress(C.GoString(addressC))
	value := AccountDB.GetBalance(address)
	return C.CString(value.String())
}

//export GetNonce
func GetNonce(addressC *C.char) C.ulonglong {
	address := common.HexToAddress(C.GoString(addressC))
	value := AccountDB.GetNonce(address)
	return C.ulonglong(value)
}

//export SetNonce
func SetNonce(addressC *C.char, nonce C.ulonglong) {
	address := common.HexToAddress(C.GoString(addressC))
	AccountDB.SetNonce(address, uint64(nonce))
}

//export GetCodeHash
func GetCodeHash(addressC *C.char) *C.char {
	address := common.HexToAddress(C.GoString(addressC))
	hash := AccountDB.GetCodeHash(address)
	return C.CString(hash.String())
}

//export GetCode
func GetCode(addressC *C.char) *C.char {
	address := common.HexToAddress(C.GoString(addressC))
	code := AccountDB.GetCode(address)
	return C.CString(string(code))
}

//export SetCode
func SetCode(addressC *C.char, codeC *C.char) {
	address := common.HexToAddress(C.GoString(addressC))
	code := C.GoString(codeC)
	AccountDB.SetCode(address, []byte(code))
}

//export GetCodeSize
func GetCodeSize(addressC *C.char) C.int {
	address := common.HexToAddress(C.GoString(addressC))
	size := AccountDB.GetCodeSize(address)
	return C.int(size)
}

//export AddRefund
func AddRefund(re C.ulonglong) {
	AccountDB.AddRefund(uint64(re))
}

//export GetRefund
func GetRefund() C.ulonglong {
	return C.ulonglong(AccountDB.GetRefund())
}

//export GetData
func GetData(addressC *C.char, hashC *C.char) *C.char {
	address := common.HexToAddress(C.GoString(addressC))
	//hash := common.StringToHash(C.GoString(hashC))
	address = *tvmObj.ContractAddress
	state := AccountDB.GetData(address, C.GoString(hashC))

	return C.CString(string(state))
}

//export SetData
func SetData(addressC *C.char, keyC *C.char, data *C.char) {
	address := common.HexToAddress(C.GoString(addressC))
	address = *tvmObj.ContractAddress
	key := C.GoString(keyC)
	state := []byte(C.GoString(data))
	AccountDB.SetData(address, key, state)
}

//export Suicide
func Suicide(addressC *C.char) bool {
	address := common.HexToAddress(C.GoString(addressC))
	return AccountDB.Suicide(address)
}

//export HasSuicided
func HasSuicided(addressC *C.char) bool {
	address := common.HexToAddress(C.GoString(addressC))
	return AccountDB.HasSuicided(address)
}

//export Exist
func Exist(addressC *C.char) bool {
	address := common.HexToAddress(C.GoString(addressC))
	return AccountDB.Exist(address)
}

//export Empty
func Empty(addressC *C.char) bool {
	address := common.HexToAddress(C.GoString(addressC))
	return AccountDB.Empty(address)
}

//export RevertToSnapshot
func RevertToSnapshot(i int) {
	AccountDB.RevertToSnapshot(i)
}

//export Snapshot
func Snapshot() int {
	return AccountDB.Snapshot()
}

//export AddPreimage
func AddPreimage(hashC *C.char, preimageC *C.char) {

}

// block chain impl

//export BlockHash
func BlockHash(height C.ulonglong) *C.char {
	block := Reader.QueryBlockByHeight(uint64(height))
	return C.CString(block.Hash.String())
}

//export CoinBase
func CoinBase() *C.char {
	return C.CString(common.BytesToAddress(CurrentBlockHeader.Castor).GetHexString())
}

//export Difficulty
func Difficulty() C.ulonglong {
	return C.ulonglong(CurrentBlockHeader.TotalQN)
}

//export Number
func Number() C.ulonglong {
	return C.ulonglong(CurrentBlockHeader.Height)
}

//export Timestamp
func Timestamp() C.ulonglong {
	return C.ulonglong(uint64(CurrentBlockHeader.CurTime.Unix()))
}

//export TxOrigin
func TxOrigin() *C.char {
	return C.CString(CurrentTransaction.Source.GetHexString())
}

//export TxGasLimit
func TxGasLimit() C.ulonglong{
	return C.ulonglong(CurrentTransaction.GasLimit)
}

//export ContractCall
func ContractCall(addressC *C.char, funName *C.char, jsonParms *C.char) {

}

//TODO 合约call合约
func callContract() {
	// extern json

	// tvm.state.GetCode()

	// tvm.exec code

	// tvm.call abi
}