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
	"github.com/taschain/taschain/common"
	"math/big"
	"unsafe"

	"github.com/taschain/taschain/middleware/types"
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

//export Transfer
func Transfer(toAddressStr *C.char, value *C.char) {
	transValue, ok := big.NewInt(0).SetString(C.GoString(value), 10)
	if !ok {
		// TODO error
	}
	contractAddr := controller.VM.ContractAddress
	contractValue := controller.AccountDB.GetBalance(*contractAddr)
	if contractValue.Cmp(transValue) < 0 {
		return
	}
	toAddress := common.HexStringToAddress(C.GoString(toAddressStr))
	controller.AccountDB.AddBalance(toAddress, transValue)
	controller.AccountDB.SubBalance(*contractAddr, transValue)
}

//export CreateAccount
func CreateAccount(addressC *C.char) {
	addr := common.HexToAddress(C.GoString(addressC))
	controller.AccountDB.CreateAccount(addr)
}

//export SubBalance
func SubBalance(addressC *C.char, valueC *C.char) {
	address := common.HexToAddress(C.GoString(addressC))
	value, ok := big.NewInt(0).SetString(C.GoString(valueC), 10)
	if !ok {
		// TODO error
	}
	controller.AccountDB.SubBalance(address, value)
}

//export AddBalance
func AddBalance(addressC *C.char, valueC *C.char) {
	address := common.HexToAddress(C.GoString(addressC))
	value, ok := big.NewInt(0).SetString(C.GoString(valueC), 10)
	if !ok {
		// TODO error
	}
	controller.AccountDB.AddBalance(address, value)
}

//export GetBalance
func GetBalance(addressC *C.char) *C.char {
	address := common.HexToAddress(C.GoString(addressC))
	value := controller.AccountDB.GetBalance(address)
	return C.CString(value.String())
}

//export GetNonce
func GetNonce(addressC *C.char) C.ulonglong {
	address := common.HexToAddress(C.GoString(addressC))
	value := controller.AccountDB.GetNonce(address)
	return C.ulonglong(value)
}

//export SetNonce
func SetNonce(addressC *C.char, nonce C.ulonglong) {
	address := common.HexToAddress(C.GoString(addressC))
	controller.AccountDB.SetNonce(address, uint64(nonce))
}

//export GetCodeHash
func GetCodeHash(addressC *C.char) *C.char {
	address := common.HexToAddress(C.GoString(addressC))
	hash := controller.AccountDB.GetCodeHash(address)
	return C.CString(hash.String())
}

//export GetCode
func GetCode(addressC *C.char) *C.char {
	address := common.HexToAddress(C.GoString(addressC))
	code := controller.AccountDB.GetCode(address)
	return C.CString(string(code))
}

//export SetCode
func SetCode(addressC *C.char, codeC *C.char) {
	address := common.HexToAddress(C.GoString(addressC))
	code := C.GoString(codeC)
	controller.AccountDB.SetCode(address, []byte(code))
}

//export GetCodeSize
func GetCodeSize(addressC *C.char) C.int {
	address := common.HexToAddress(C.GoString(addressC))
	size := controller.AccountDB.GetCodeSize(address)
	return C.int(size)
}

//export AddRefund
func AddRefund(re C.ulonglong) {
	controller.AccountDB.AddRefund(uint64(re))
}

//export GetRefund
func GetRefund() C.ulonglong {
	return C.ulonglong(controller.AccountDB.GetRefund())
}

//export GetData
func GetData(hashC *C.char) *C.char {
	//hash := common.StringToHash(C.GoString(hashC))
	address := *controller.VM.ContractAddress
	state := controller.AccountDB.GetData(address, C.GoString(hashC))
	return C.CString(string(state))
}

//export SetData
func SetData(keyC *C.char, data *C.char) {
	address := *controller.VM.ContractAddress
	key := C.GoString(keyC)
	state := []byte(C.GoString(data))
	controller.AccountDB.SetData(address, key, state)
}

//export Suicide
func Suicide(addressC *C.char) bool {
	address := common.HexToAddress(C.GoString(addressC))
	return controller.AccountDB.Suicide(address)
}

//export HasSuicided
func HasSuicided(addressC *C.char) bool {
	address := common.HexToAddress(C.GoString(addressC))
	return controller.AccountDB.HasSuicided(address)
}

//export Exist
func Exist(addressC *C.char) bool {
	address := common.HexToAddress(C.GoString(addressC))
	return controller.AccountDB.Exist(address)
}

//export Empty
func Empty(addressC *C.char) bool {
	address := common.HexToAddress(C.GoString(addressC))
	return controller.AccountDB.Empty(address)
}

//export RevertToSnapshot
func RevertToSnapshot(i int) {
	controller.AccountDB.RevertToSnapshot(i)
}

//export Snapshot
func Snapshot() int {
	return controller.AccountDB.Snapshot()
}

//export AddPreimage
func AddPreimage(hashC *C.char, preimageC *C.char) {

}

// block chain impl

//export BlockHash
func BlockHash(height C.ulonglong) *C.char {
	block := controller.Reader.QueryBlockHeaderByHeight(uint64(height))
	if block == nil {
		return C.CString("0x0000000000000000000000000000000000000000000000000000000000000000")
	}
	return C.CString(block.Hash.String())
}

//export CoinBase
func CoinBase() *C.char {
	return C.CString(common.BytesToAddress(controller.BlockHeader.Castor).GetHexString())
}

//export Difficulty
func Difficulty() C.ulonglong {
	return C.ulonglong(controller.BlockHeader.TotalQN)
}

//export Number
func Number() C.ulonglong {
	return C.ulonglong(controller.BlockHeader.Height)
}

//export Timestamp
func Timestamp() C.ulonglong {
	return C.ulonglong(uint64(controller.BlockHeader.CurTime.Unix()))
}

//export TxOrigin
func TxOrigin() *C.char {
	return C.CString(controller.Transaction.GetSource().GetHexString())
}

//export TxGasLimit
func TxGasLimit() C.ulonglong {
	return C.ulonglong(controller.Transaction.GetGasLimit())
}

//export ContractCall
func ContractCall(addressC *C.char, funName *C.char, jsonParms *C.char, cResult unsafe.Pointer) {
	goResult := CallContract(C.GoString(addressC), C.GoString(funName), C.GoString(jsonParms))
	ccResult := (*C.struct__ExecuteResult)(cResult)
	ccResult.resultType = C.int(goResult.ResultType)
	ccResult.errorCode = C.int(goResult.ErrorCode)
	if goResult.Content != "" {
		ccResult.content = C.CString(goResult.Content)
	}
	if goResult.Abi != "" {
		ccResult.abi = C.CString(goResult.Abi)
	}
}

//export EventCall
func EventCall(eventName *C.char, index *C.char, data *C.char) *C.char {
	//fmt.Println("111111111111111111111111111111")
	//fmt.Println(C.GoString(eventName))
	//fmt.Println(C.GoString(index))
	//fmt.Println(C.GoString(data))

	var log types.Log
	log.Topics = append(log.Topics, common.BytesToHash(common.Sha256([]byte(C.GoString(eventName)))))
	log.Topics = append(log.Topics, common.BytesToHash(common.Sha256([]byte(C.GoString(index)))))
	for i := 0; i < len(C.GoString(data)); i++ {
		log.Data = append(log.Data, C.GoString(data)[i])
	}
	log.TxHash = controller.Transaction.GetHash()
	log.Address = *controller.VM.ContractAddress //*(controller.Transaction.Target)
	log.BlockNumber = controller.BlockHeader.Height
	//block is running ,no blockhash this time
	// log.BlockHash = controller.BlockHeader.Hash

	controller.VM.Logs = append(controller.VM.Logs, &log)

	return nil //C.CString(contractResult);
}

//export SetBytecode
func SetBytecode(code *C.char, len C.int) {
	fmt.Println(C.GoString(code))
	RunByteCode(code, len)
	fmt.Println(C.GoString(code))
}

//export DataIterator
func DataIterator(prefix *C.char) C.ulonglong {
	address := *controller.VM.ContractAddress
	iter := controller.AccountDB.DataIterator(address, C.GoString(prefix))
	return C.ulonglong(uintptr(unsafe.Pointer(iter)))
}

//export RemoveData
func RemoveData(key *C.char) {
	address := *controller.VM.ContractAddress
	controller.AccountDB.RemoveData(address, C.GoString(key))
}

//export DataNext
func DataNext(cvalue *C.char) *C.char {
	//C.ulonglong
	value, ok := big.NewInt(0).SetString(C.GoString(cvalue), 10)
	if !ok {
		//TODO
	}
	data := controller.AccountDB.DataNext(uintptr(value.Uint64()))
	return C.CString(data)
}

//export MinerStake
func MinerStake(minerAddr *C.char, _type int, cvalue *C.char) bool {
	fmt.Println("VM MinerStake", _type)
	ss := controller.AccountDB.Snapshot()
	value, ok := big.NewInt(0).SetString(C.GoString(cvalue), 10)
	if !ok {
		//TODO
	}
	source := controller.VM.ContractAddress
	miner := common.HexStringToAddress(C.GoString(minerAddr))
	if CanTransfer(controller.AccountDB, *source, value) {
		mexist := controller.mm.GetMinerByID(miner.Bytes(), byte(_type), controller.AccountDB)
		if mexist != nil &&
			controller.mm.AddStake(mexist.ID, mexist, value.Uint64(), controller.AccountDB) &&
			controller.mm.AddStakeDetail(source.Bytes(), mexist, value.Uint64(), controller.AccountDB) {
			controller.AccountDB.SubBalance(*source, value)
			return true
		}
	}
	controller.AccountDB.RevertToSnapshot(ss)
	return false
}

//export MinerCancelStake
func MinerCancelStake(minerAddr *C.char, _type int, cvalue *C.char) bool {
	fmt.Println("VM MinerCancelStake", _type)
	ss := controller.AccountDB.Snapshot()
	value, ok := big.NewInt(0).SetString(C.GoString(cvalue), 10)
	if !ok {
		//TODO
	}
	source := controller.VM.ContractAddress
	miner := common.HexStringToAddress(C.GoString(minerAddr))
	mexist := controller.mm.GetMinerByID(miner.Bytes(), byte(_type), controller.AccountDB)
	if mexist != nil &&
		controller.mm.CancelStake(source.Bytes(), mexist, value.Uint64(), controller.AccountDB, controller.BlockHeader.Height) &&
		controller.mm.ReduceStake(mexist.ID, mexist, value.Uint64(), controller.AccountDB, controller.BlockHeader.Height) {
		return true
	}
	controller.AccountDB.RevertToSnapshot(ss)
	return false
}

//export MinerRefundStake
func MinerRefundStake(minerAddr *C.char, _type int) bool {
	fmt.Println("VM MinerRefundStake", _type)
	var success = false
	ss := controller.AccountDB.Snapshot()
	source := controller.VM.ContractAddress
	miner := common.HexStringToAddress(C.GoString(minerAddr))
	mexist := controller.mm.GetMinerByID(miner.Bytes(), byte(_type), controller.AccountDB)
	height := controller.BlockHeader.Height
	if mexist != nil {
		if mexist.Type == types.MinerTypeHeavy {
			latestCancelPledgeHeight := controller.mm.GetLatestCancelStakeHeight(source.Bytes(), mexist, controller.AccountDB)
			if height > latestCancelPledgeHeight+10 || (mexist.Status == types.MinerStatusAbort && height > mexist.AbortHeight+10) {
				value, ok := controller.mm.RefundStake(source.Bytes(), mexist, controller.AccountDB)
				if ok {
					refundValue := big.NewInt(0).SetUint64(value)
					controller.AccountDB.AddBalance(*source, refundValue)
					success = true
				}
			}
		} else {
			value, ok := controller.mm.RefundStake(source.Bytes(), mexist, controller.AccountDB)
			if ok {
				refundValue := big.NewInt(0).SetUint64(value)
				controller.AccountDB.AddBalance(*source, refundValue)
				success = true
			}
		}
	}
	if !success {
		controller.AccountDB.RevertToSnapshot(ss)
	}
	return success
}
