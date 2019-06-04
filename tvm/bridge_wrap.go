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
#cgo LDFLAGS: -L ./ -ltvm

#include "tvm.h"
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <string.h>

// The gateway function
int callOnMeGo_cgo(int in)
{
	//printf("C.callOnMeGo_cgo(): called with arg = %d\n", in);
	int callOnMeGo(int);
	return callOnMeGo(in);
}


void wrap_testAry(void* p)
{
    void go_testAry(void*);
    go_testAry(p);
}

void wrap_transfer(const char* p2, const char* value)
{
    void Transfer(const char*, const char* value);
    Transfer(p2, value);
}

void wrap_create_account(const char* address)
{
	void CreateAccount(const char*);
	CreateAccount(address);
}

void wrap_sub_balance(const char* address, const char* value)
{
	void SubBalance(const char*, const char*);
	SubBalance(address, value);
}

void wrap_add_balance(const char* address, const char* value)
{
	void AddBalance(const char*, const char*);
	AddBalance(address, value);
}

char* wrap_get_balance(const char* address)
{
	char* GetBalance(const char*);
	return GetBalance(address);
}

unsigned long long wrap_get_nonce(const char* address)
{
	unsigned long long GetNonce(const char*);
	return GetNonce(address);
}

void wrap_set_nonce(const char* address, unsigned long long nonce)
{
	void SetNonce(const char*, unsigned long long);
	SetNonce(address, nonce);
}

char* wrap_get_code_hash(char* address)
{
	char* GetCodeHash(char*);
	return GetCodeHash(address);
}

char* wrap_get_code(char* address)
{
	char* GetCode(char*);
	return GetCode(address);
}

void wrap_set_code(char* address, char* code)
{
	void SetCode(char*, char*);
	SetCode(address, code);
}

int wrap_get_code_size(char* address)
{
	int GetCodeSize(char*);
	return GetCodeSize(address);
}

void wrap_add_refund(unsigned long long refund)
{
	void AddRefund(unsigned long long);
	AddRefund(refund);
}

unsigned long long wrap_get_refund()
{
	unsigned long long GetRefund();
	return GetRefund();
}

void wrap_remove_data(char* key)
{
	void RemoveData(char* );
	RemoveData(key);
}

char* wrap_get_data(char* key)
{
	char* GetData(char*);
	return GetData(key);
}

void wrap_set_data(char* key, char* value)
{
	void SetData(char*, char*);
	SetData(key, value);
}

_Bool wrap_suicide(char* address)
{
	_Bool Suicide(char*);
	return Suicide(address);
}

_Bool wrap_has_suicide(char* address)
{
	_Bool HasSuicided(char*);
	return HasSuicided(address);
}

_Bool wrap_exists(char* address)
{
	_Bool Exist(char*);
	return Exist(address);
}

_Bool wrap_empty(char* address)
{
	_Bool Empty(char*);
	return Empty(address);
}

void wrap_revert_to_snapshot(int i)
{
	void RevertToSnapshot(int);
	RevertToSnapshot(i);
}

int wrap_snapshot()
{
	int Snapshot();
	return Snapshot();
}

void wrap_add_preimage(char* hash, char* preimage)
{
	void AddPreimage(char*, char*);
	AddPreimage(hash, preimage);
}

char* wrap_block_hash(unsigned long long height)
{
	char* BlockHash(unsigned long long);
	return BlockHash(height);
}

char* wrap_coin_base()
{
	char* CoinBase();
	return CoinBase();
}

unsigned long long wrap_difficulty()
{
	unsigned long long Difficulty();
	return Difficulty();
}

unsigned long long wrap_number()
{
	unsigned long long Number();
	return Number();
}

unsigned long long wrap_timestamp()
{
	unsigned long long Timestamp();
	return Timestamp();
}

char* wrap_tx_origin()
{
	char* TxOrigin();
	return TxOrigin();
}

unsigned long long wrap_tx_gas_limit()
{
	unsigned long long TxGasLimit();
	return TxGasLimit();
}

void wrap_contract_call(const char* address, const char* func_name, const char* json_parms, ExecuteResult *result)
{
    char* ContractCall();
    ContractCall(address, func_name, json_parms, result);
}

char* wrap_event_call(const char* address, const char* func_name, const char* json_parms)
{
    char* EventCall();
    return EventCall(address, func_name, json_parms);
}

void wrap_set_bytecode(const char* code, int len)
{
	void SetBytecode();
	SetBytecode(code, len);
}

unsigned long long wrap_get_data_iter(const char*prefix)
{
	unsigned long long DataIterator(const char*);
	return DataIterator(prefix);
}

char* wrap_get_data_iter_next(char* iter)
{
	char* DataNext(char*);
	return DataNext(iter);
}

_Bool wrap_miner_stake(const char* minerAddr, int _type, const char* value) {
	_Bool MinerStake(const char*, int, const char*);
	return MinerStake(minerAddr, _type, value);
}

_Bool wrap_miner_cancel_stake(const char* minerAddr, int _type, const char* value) {
	_Bool MinerCancelStake(const char*, int, const char*);
	return MinerCancelStake(minerAddr, _type, value);
}

_Bool wrap_miner_refund_stake(const char* minerAddr, int _type) {
	_Bool MinerRefundStake(const char*, int);
	return MinerRefundStake(minerAddr, _type);
}
*/
import "C"
import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unsafe"

	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/middleware/types"
)

type CallTask struct {
	Sender       *common.Address
	ContractAddr *common.Address
	FuncName     string
	Params       string
}

type ExecuteResult struct {
	ResultType int
	ErrorCode  int
	Content    string
	Abi        string
}

// RunBinaryCode Run python binary code
func RunBinaryCode(buf *C.char, len C.int) {
	C.runbytecode(buf, len)
}

// CallContract Execute the function of a contract which python code store in contractAddr
func CallContract(contractAddr string, funcName string, params string) *ExecuteResult {
	result := &ExecuteResult{}
	conAddr := common.HexStringToAddress(contractAddr)
	contract := LoadContract(conAddr)
	if contract.Code == "" {
		result.ResultType = C.RETURN_TYPE_EXCEPTION
		result.ErrorCode = types.NoCodeErr
		result.Content = fmt.Sprint(types.NoCodeErrorMsg, conAddr)
		return result
	}
	oneVM := &TVM{contract, controller.VM.ContractAddress, nil}

	// prepare vm environment
	controller.VM.createContext()
	finished := controller.StoreVMContext(oneVM)
	defer func() {
		// recover vm environment
		if finished {
			controller.VM.removeContext()
		}
	}()
	if !finished {
		result.ResultType = C.RETURN_TYPE_EXCEPTION
		result.ErrorCode = types.CallMaxDeepError
		result.Content = types.CallMaxDeepErrorMsg
		return result
	}

	msg := Msg{Data: []byte{}, Value: 0, Sender: conAddr.GetHexString()}
	errorCode, errorMsg, _ := controller.VM.CreateContractInstance(msg)
	if errorCode != 0 {
		result.ResultType = C.RETURN_TYPE_EXCEPTION
		result.ErrorCode = errorCode
		result.Content = errorMsg
		return result
	}

	if strings.EqualFold("[true]", params) {
		params = "[true]"
	} else if strings.EqualFold("[false]", params) {
		params = "[false]"
	}
	abi := ABI{}
	abiJSON := fmt.Sprintf(`{"FuncName": "%s", "Args": %s}`, funcName, params)
	abiJSONError := json.Unmarshal([]byte(abiJSON), &abi)
	if abiJSONError != nil {
		result.ResultType = C.RETURN_TYPE_EXCEPTION
		result.ErrorCode = types.ABIJSONError
		result.Content = types.ABIJSONErrorMsg
		return result
	}
	errorCode, errorMsg = controller.VM.checkABI(abi)
	if errorCode != 0 {
		result.ResultType = C.RETURN_TYPE_EXCEPTION
		result.ErrorCode = errorCode
		result.Content = errorMsg
		return result
	}
	return controller.VM.executeABIKindEval(abi)
}

// RunByteCode Run python byte code
func RunByteCode(code *C.char, len C.int) {
	C.runbytecode(code, len)
}

func bridgeInit() {
	C.tvm_setup_func((C.callback_fcn)(unsafe.Pointer(C.callOnMeGo_cgo)))
	C.tvm_set_testAry_func((C.testAry_fcn)(unsafe.Pointer(C.wrap_testAry)))
	//C.setTransferFunc((C.TransferFunc)(unsafe.Pointer(C.wrap_transfer)))
	C.transferFunc = (C.TransferFunc)(unsafe.Pointer(C.wrap_transfer))
	C.create_account = (C.Function1)(unsafe.Pointer(C.wrap_create_account))
	C.sub_balance = (C.Function5)(unsafe.Pointer(C.wrap_sub_balance))
	C.add_balance = (C.Function5)(unsafe.Pointer(C.wrap_add_balance))
	C.get_balance = (C.Function2)(unsafe.Pointer(C.wrap_get_balance))
	C.get_nonce = (C.Function3)(unsafe.Pointer(C.wrap_get_nonce))
	C.set_nonce = (C.Function6)(unsafe.Pointer(C.wrap_set_nonce))
	C.get_code_hash = (C.Function2)(unsafe.Pointer(C.wrap_get_code_hash))
	C.get_code = (C.Function2)(unsafe.Pointer(C.wrap_get_code))
	C.set_code = (C.Function5)(unsafe.Pointer(C.wrap_set_code))
	C.get_code_size = (C.Function7)(unsafe.Pointer(C.wrap_get_code_size))
	C.add_refund = (C.Function8)(unsafe.Pointer(C.wrap_add_refund))
	C.get_refund = (C.Function9)(unsafe.Pointer(C.wrap_get_refund))
	C.get_data = (C.Function10)(unsafe.Pointer(C.wrap_get_data))
	C.set_data = (C.Function5)(unsafe.Pointer(C.wrap_set_data))
	C.remove_data = (C.Function1)(unsafe.Pointer(C.wrap_remove_data))
	C.func_suicide = (C.Function4)(unsafe.Pointer(C.wrap_suicide))
	C.has_suicide = (C.Function4)(unsafe.Pointer(C.wrap_has_suicide))
	C.func_exists = (C.Function4)(unsafe.Pointer(C.wrap_exists))
	C.func_empty = (C.Function4)(unsafe.Pointer(C.wrap_empty))
	C.func_revert_to_snapshot = (C.Function12)(unsafe.Pointer(C.wrap_revert_to_snapshot))
	C.func_snapshot = (C.Function13)(unsafe.Pointer(C.wrap_snapshot))
	C.add_preimage = (C.Function5)(unsafe.Pointer(C.wrap_add_preimage))
	// block
	C.func_blockhash = (C.Function14)(unsafe.Pointer(C.wrap_block_hash))
	C.func_coinbase = (C.Function15)(unsafe.Pointer(C.wrap_coin_base))
	C.func_difficulty = (C.Function9)(unsafe.Pointer(C.wrap_difficulty))
	C.func_number = (C.Function9)(unsafe.Pointer(C.wrap_number))
	C.func_timestamp = (C.Function9)(unsafe.Pointer(C.wrap_timestamp))
	C.func_origin = (C.Function15)(unsafe.Pointer(C.wrap_tx_origin))
	C.func_gaslimit = (C.Function9)(unsafe.Pointer(C.wrap_tx_gas_limit))
	C.contract_call = (C.Function17)(unsafe.Pointer(C.wrap_contract_call))
	C.set_bytecode = (C.Function16)(unsafe.Pointer(C.wrap_set_bytecode))
	//C.get_data_iter = (C.Function3)(unsafe.Pointer(C.wrap_get_data_iter))
	//C.get_data_iter_next = (C.Function10)(unsafe.Pointer(C.wrap_get_data_iter_next))
	C.event_call = (C.Function11)(unsafe.Pointer(C.wrap_event_call))
	C.miner_stake = (C.Function18)(unsafe.Pointer(C.wrap_miner_stake))
	C.miner_cancel_stake = (C.Function18)(unsafe.Pointer(C.wrap_miner_cancel_stake))
	C.miner_refund_stake = (C.Function19)(unsafe.Pointer(C.wrap_miner_refund_stake))
}

// Contract Contract contains the base message of a contract
type Contract struct {
	Code            string          `json:"code"`
	ContractName    string          `json:"contract_name"`
	ContractAddress *common.Address `json:"-"`
}

// LoadContract Load a contract-instance from a contract address
func LoadContract(address common.Address) *Contract {
	jsonString := controller.AccountDB.GetCode(address)
	con := &Contract{}
	json.Unmarshal([]byte(jsonString), con)
	con.ContractAddress = &address
	return con
}

// TVM TVM is the role who execute contract code
type TVM struct {
	*Contract
	Sender *common.Address

	// xtm for log
	Logs []*types.Log
}

// NewTVM new a TVM instance
func NewTVM(sender *common.Address, contract *Contract, libPath string) *TVM {
	tvm := &TVM{
		contract,
		sender,
		nil,
	}
	C.tvm_start()

	if !HasLoadPyLibPath {
		C.tvm_set_lib_path(C.CString(libPath))
		HasLoadPyLibPath = true
	}
	C.tvm_set_gas(1000000)
	bridgeInit()
	return tvm
}

// Gas Get the gas left of the TVM
func (tvm *TVM) Gas() int {
	return int(C.tvm_get_gas())
}

// SetGas Set the amount of gas that TVM can use
func (tvm *TVM) SetGas(gas int) {
	//fmt.Printf("SetGas: %d\n", gas);
	C.tvm_set_gas(C.int(gas))
}

// SetLibLine Correct the error line when python code is running
func (tvm *TVM) SetLibLine(line int) {
	C.tvm_set_lib_line(C.int(line))
}

// Pycode2Bytecode python code to byte code
func (tvm *TVM) Pycode2Bytecode(str string) {
	C.pycode2bytecode(C.CString(str))
}

// DelTVM Run tvm gc to collect mem
func (tvm *TVM) DelTVM() {
	//C.tvm_gas_report()
	C.tvm_gc()
}

func (tvm *TVM) checkABI(abi ABI) (int, string) {
	script := pycodeCheckAbi(abi)
	errorCode, errorMsg := tvm.ExecutedScriptVMSucceed(script)
	if errorCode != 0 {
		errorCode = types.SysCheckABIError
		errorMsg = fmt.Sprintf(`
			checkABI failed. abi:%s,msg=%s
		`, abi.FuncName, errorMsg)
	}
	return errorCode, errorMsg
}

// storeData flush data to db
func (tvm *TVM) storeData() (int, string) {
	script := pycodeStoreContractData()
	return tvm.ExecutedScriptVMSucceed(script)
}

// Msg Msg is msg instance which store running message when running a contract
type Msg struct {
	Data   []byte
	Value  uint64
	Sender string
}

// CreateContractInstance Create contract instance
func (tvm *TVM) CreateContractInstance(msg Msg) (int, string, int) {
	errorCode, errorMsg := tvm.loadMsg(msg)
	if errorCode != 0 {
		return errorCode, errorMsg, 0
	}
	script, codeLen := pycodeCreateContractInstance(tvm.Code, tvm.ContractName)
	errorCode, errorMsg = tvm.ExecutedScriptVMSucceed(script)
	return errorCode, errorMsg, codeLen
}

func (tvm *TVM) generateScript(res ABI) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("tas_%s.", tvm.ContractName))
	buf.WriteString(res.FuncName)
	buf.WriteString("(")
	for _, value := range res.Args {
		tvm.jsonValueToBuf(&buf, value)
		buf.WriteString(", ")
	}
	if len(res.Args) > 0 {
		buf.Truncate(buf.Len() - 2)
	}
	buf.WriteString(")")
	bufStr := buf.String()
	return bufStr
}

func (tvm *TVM) executABIVMSucceed(res ABI) (errorCode int, content string) {
	script := tvm.generateScript(res)
	result := tvm.executedPycode(script, C.PARSE_KIND_FILE)
	if result.ResultType == C.RETURN_TYPE_EXCEPTION {
		fmt.Printf("execute error,code=%d,msg=%s \n", result.ErrorCode, result.Content)
		return result.ErrorCode, result.Content
	}
	return types.Success, ""
}

func (tvm *TVM) executeABIKindFile(res ABI) *ExecuteResult {
	bufStr := tvm.generateScript(res)
	return tvm.executedPycode(bufStr, C.PARSE_KIND_FILE)
}

func (tvm *TVM) executeABIKindEval(res ABI) *ExecuteResult {
	bufStr := tvm.generateScript(res)
	return tvm.executedPycode(bufStr, C.PARSE_KIND_EVAL)
}

// ExecutedScriptVMSucceed Execute script and returns result
func (tvm *TVM) ExecutedScriptVMSucceed(script string) (int, string) {
	result := tvm.executedPycode(script, C.PARSE_KIND_FILE)
	if result.ResultType == C.RETURN_TYPE_EXCEPTION {
		fmt.Printf("execute error,code=%d,msg=%s \n", result.ErrorCode, result.Content)
		return result.ErrorCode, result.Content
	}
	return types.Success, ""
}

func (tvm *TVM) executedScriptKindEval(script string) *ExecuteResult {
	return tvm.executedPycode(script, C.PARSE_KIND_EVAL)
}

// ExecutedScriptKindFile Execute file and returns result
func (tvm *TVM) ExecutedScriptKindFile(script string) *ExecuteResult {
	return tvm.executedPycode(script, C.PARSE_KIND_FILE)
}

func (tvm *TVM) executedPycode(code string, parseKind C.tvm_parse_kind_t) *ExecuteResult {
	cResult := &C.ExecuteResult{}
	C.initResult((*C.ExecuteResult)(unsafe.Pointer(cResult)))
	var param = C.CString(code)
	var contractName = C.CString(tvm.ContractName)

	//fmt.Println("-----------------code start-------------------")
	//fmt.Println(code)
	//fmt.Println("-----------------code end---------------------")
	C.tvm_execute(param, contractName, parseKind, (*C.ExecuteResult)(unsafe.Pointer(cResult)))
	C.free(unsafe.Pointer(param))
	C.free(unsafe.Pointer(contractName))

	result := &ExecuteResult{}
	result.ResultType = int(cResult.resultType)
	result.ErrorCode = int(cResult.errorCode)
	if cResult.content != nil {
		result.Content = C.GoString(cResult.content)
	}
	if cResult.abi != nil {
		result.Abi = C.GoString(cResult.abi)
	}
	//C.printResult((*C.ExecuteResult)(unsafe.Pointer(cResult)))
	C.deinitResult((*C.ExecuteResult)(unsafe.Pointer(cResult)))
	return result
}

func (tvm *TVM) loadMsg(msg Msg) (int, string) {
	script := pycodeLoadMsg(msg.Sender, msg.Value, tvm.ContractAddress.GetHexString())
	return tvm.ExecutedScriptVMSucceed(script)
}

// Deploy TVM Deploy the contract code and load msg
func (tvm *TVM) Deploy(msg Msg) (int, string) {
	errorCode, errorMsg := tvm.loadMsg(msg)
	if errorCode != 0 {
		return errorCode, errorMsg
	}
	script, libLen := pycodeContractDeploy(tvm.Code, tvm.ContractName)
	tvm.SetLibLine(libLen)
	errorCode, errorMsg = tvm.ExecutedScriptVMSucceed(script)
	return errorCode, errorMsg
}

func (tvm *TVM) createContext() {
	C.tvm_create_context()
}

func (tvm *TVM) removeContext() {
	C.tvm_remove_context()
}

// ABI ABI stores the calling msg when execute contract
type ABI struct {
	FuncName string
	Args     []interface{}
}

func (tvm *TVM) jsonValueToBuf(buf *bytes.Buffer, value interface{}) {
	switch value.(type) {
	case float64:
		buf.WriteString(strconv.FormatFloat(value.(float64), 'f', 0, 64))
	case bool:
		x := value.(bool)
		if x {
			buf.WriteString("True")
		} else {
			buf.WriteString("False")
		}
	case string:
		buf.WriteString(`"`)
		buf.WriteString(value.(string))
		buf.WriteString(`"`)
	case []interface{}:
		buf.WriteString("[")
		for _, item := range value.([]interface{}) {
			tvm.jsonValueToBuf(buf, item)
			buf.WriteString(", ")
		}
		if len(value.([]interface{})) > 0 {
			buf.Truncate(buf.Len() - 2)
		}
		buf.WriteString("]")
	case map[string]interface{}:
		buf.WriteString("{")
		for key, item := range value.(map[string]interface{}) {
			tvm.jsonValueToBuf(buf, key)
			buf.WriteString(": ")
			tvm.jsonValueToBuf(buf, item)
			buf.WriteString(", ")
		}
		if len(value.(map[string]interface{})) > 0 {
			buf.Truncate(buf.Len() - 2)
		}
		buf.WriteString("}")
	default:
		fmt.Println(value)
		//panic("")
	}
}
