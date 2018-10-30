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
#cgo CFLAGS:  -I ../../include
#cgo LDFLAGS: -L ../../lib/darwin_amd64 -lmicropython

#include "tvm.h"
#include <stdio.h>


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

void wrap_contract_call(const char* address, const char* func_name, const char* json_parms)
{
    void ContractCall();
    ContractCall(address, func_name, json_parms);
}

*/
import "C"
import (
	"unsafe"
	"common"
	"encoding/json"
	"fmt"
	"bytes"
	"strconv"
	"storage/account/vm"
)

var controller *Controller = nil


type CallTask struct {
	Sender *common.Address
	ContractAddr *common.Address
	FuncName string
	Params string
}



func Call(_contractAddr string, funcName string, params string) bool {
	conAddr := common.HexStringToAddress(_contractAddr)
	task := CallTask{controller.Vm.ContractAddress, &conAddr, funcName, params}
	controller.Tasks = append(controller.Tasks, &task)
	return true
}

func bridge_init() {
	C.tvm_setup_func((C.callback_fcn)(unsafe.Pointer(C.callOnMeGo_cgo)))
	C.tvm_set_testAry_func((C.testAry_fcn)(unsafe.Pointer(C.wrap_testAry)))
	//C.setTransferFunc((C.TransferFunc)(unsafe.Pointer(C.wrap_transfer)))
	C.transferFunc = (C.TransferFunc)(unsafe.Pointer(C.wrap_transfer))
	C.create_account = (C.Function1)(unsafe.Pointer(C.wrap_create_account))
	C.sub_balance = (C.Function5)(unsafe.Pointer(C.wrap_sub_balance))
	C.add_balance = (C.Function5)(unsafe.Pointer(C.wrap_add_balance))
	C.get_balance = (C.Function2)(unsafe.Pointer(C.wrap_get_balance))
	C.get_nonce = (C.Function3)(unsafe.Pointer( C.wrap_get_nonce))
	C.set_nonce = (C.Function6)(unsafe.Pointer(C.wrap_set_nonce))
	C.get_code_hash = (C.Function2)(unsafe.Pointer(C.wrap_get_code_hash))
	C.get_code = (C.Function2)(unsafe.Pointer(C.wrap_get_code))
	C.set_code = (C.Function5)(unsafe.Pointer(C.wrap_set_code))
	C.get_code_size = (C.Function7)(unsafe.Pointer(C.wrap_get_code_size))
	C.add_refund = (C.Function8)(unsafe.Pointer(C.wrap_add_refund))
	C.get_refund = (C.Function9)(unsafe.Pointer(C.wrap_get_refund))
	C.get_data = (C.Function10)(unsafe.Pointer(C.wrap_get_data))
	C.set_data = (C.Function5)(unsafe.Pointer(C.wrap_set_data))
	C.suicide = (C.Function4)(unsafe.Pointer(C.wrap_suicide))
	C.has_suicide = (C.Function4)(unsafe.Pointer(C.wrap_has_suicide))
	C.exists = (C.Function4)(unsafe.Pointer(C.wrap_exists))
	C.empty = (C.Function4)(unsafe.Pointer(C.wrap_empty))
	C.revert_to_snapshot = (C.Function12)(unsafe.Pointer(C.wrap_revert_to_snapshot))
	C.snapshot = (C.Function13)(unsafe.Pointer(C.wrap_snapshot))
	C.add_preimage = (C.Function5)(unsafe.Pointer(C.wrap_add_preimage))
	// block
	C.blockhash = (C.Function14)(unsafe.Pointer(C.wrap_block_hash))
	C.coinbase = (C.Function15)(unsafe.Pointer(C.wrap_coin_base))
	C.difficulty = (C.Function9)(unsafe.Pointer(C.wrap_difficulty))
	C.number = (C.Function9)(unsafe.Pointer(C.wrap_number))
	C.timestamp = (C.Function9)(unsafe.Pointer(C.wrap_timestamp))
	C.origin = (C.Function15)(unsafe.Pointer(C.wrap_tx_origin))
	C.gaslimit = (C.Function9)(unsafe.Pointer(C.wrap_tx_gas_limit))
	C.contract_call = (C.Function11)(unsafe.Pointer(C.wrap_contract_call))
}

type Contract struct {
	Code string `json:"code"`
	ContractName string `json:"contract_name"`
	ContractAddress *common.Address `json:"-"`
}

func LoadContract(address common.Address) *Contract {
	jsonString := controller.AccountDB.GetCode(address)
	con := &Contract{}
	json.Unmarshal([]byte(jsonString), con)
	con.ContractAddress = &address
	return con
}

type Tvm struct {
	*Contract
	Sender *common.Address

	Block func() bool
}



func NewTvm(sender *common.Address, contract *Contract, libPath string)*Tvm {
	tvm := &Tvm{
		contract,
		sender,
		nil,
	}
	C.tvm_set_lib_path(C.CString(libPath))
	C.tvm_start()
	bridge_init()
	return tvm
}


// 获取剩余gas
func (tvm *Tvm)Gas() int {
	return int(C.getGas())
}

// 设置可使用gas, init成功后设置
func (tvm *Tvm)SetGas(gas int) {
	C.setGas(C.int(gas))
}

func (tvm *Tvm)DelTvm(){
	//TODO 释放tvm环境 tvmObj
}

func(tvm *Tvm) StoreData() bool {
	var c_bool C._Bool
	script := fmt.Sprintf(`
import account
import ujson
for k in tas_%s.__dict__:
    #	print(k)
    #	print(type(k))
    #	print(tas_%s.__dict__[k])
    #	print(type(tas_%s.__dict__[k]))
    value = ujson.dumps(tas_%s.__dict__[k])
    if TAS_PARAMS_DICT.get(k) != value:
        account.set_data(k, value)`, tvm.ContractName, tvm.ContractName, tvm.ContractName, tvm.ContractName)
	c_bool = C.tvm_execute(C.CString(script))
	return bool(c_bool)
}

func NewTvmTest(accountDB vm.AccountDB, chainReader vm.ChainReader)*Tvm {
	//if tvmObj == nil {
	//	tvmObj = NewTvm(nil, nil, "")
	//}
	//Reader = chainReader
	//AccountDB = accountDB
	//
	//C.tvm_start()
	//C.tvm_set_lib_path(C.CString("/Users/guangyujing/workspace/tas/src/tvm/py"))
	//bridge_init()
	//
	//return tvmObj
	return nil
}

func (tvm *Tvm) AddLibPath(path string) {
	C.tvm_set_lib_path(C.CString(path))
}

type Msg struct {
	Data []byte
	Value uint64
	Sender string
}

func(tvm *Tvm) LoadContractCode() bool {
	var c_bool C._Bool
	script := fmt.Sprintf("%s\ntas_%s = %s()",tvm.Code, tvm.ContractName, tvm.ContractName)
	c_bool = C.tvm_execute(C.CString(script))
	if !bool(c_bool) {
		return false
	}
	script = fmt.Sprintf(`
import account
import ujson
TAS_PARAMS_DICT = {}
for k in tas_%s.__dict__:
    #	print(k)
    #	print(type(k))
    #	value = ujson.loads(account.get_state("", k))
    #	print(value)
    value = account.get_data(k)
    TAS_PARAMS_DICT[k] = value
    setattr(tas_%s, k, ujson.loads(value))`, tvm.ContractName, tvm.ContractName)
	c_bool = C.tvm_execute(C.CString(script))
	return bool(c_bool)
}

func (tvm *Tvm)Execute(script string) bool {
	var c_bool C._Bool
	c_bool = C.tvm_execute(C.CString(script))
	return bool(c_bool)
}

func (tvm *Tvm)loadMsg(msg Msg) bool{
	script := fmt.Sprintf(`
from clib.tas_runtime import glovar
from clib.tas_runtime.msgxx import Msg
from clib.tas_runtime.address_tas import Address

glovar.msg = Msg(data=bytes(), sender="%s", value=%d)
glovar.this = "%s"
`, msg.Sender, msg.Value, tvm.ContractAddress.GetHexString())
	return tvm.Execute(script)
}

func (tvm *Tvm)Deploy(msg Msg) bool {
	tvm.Execute(tvm.Code)
	if tvm.loadMsg(msg) != true {
		return false
	}

	script := fmt.Sprintf(`
TAS_PARAMS_DICT = {}
tas_%s = %s()
tas_%s.deploy()
`, tvm.ContractName, tvm.ContractName, tvm.ContractName)
	return tvm.Execute(script)
}

type ABI struct {
	FuncName string
	Args []interface{}
}

// `{"FuncName": "Test", "Args": [10.123, "ten", [1, 2], {"key":"value", "key2":"value2"}]}`
func (tvm *Tvm) ExecuteABIJson(msg Msg, j string) bool{
	if tvm.loadMsg(msg) != true {
		return false
	}

	res := ABI{}
	json.Unmarshal([]byte(j), &res)
	fmt.Println(res)

	var buf bytes.Buffer
	//类名
	buf.WriteString(fmt.Sprintf("tas_%s.", tvm.ContractName))
	//函数名
	buf.WriteString(res.FuncName)
	//参数
	buf.WriteString("(")
	for _, value := range res.Args {
		tvm.jsonValueToBuf(&buf, value)
		buf.WriteString(", ")
	}
	if len(res.Args) > 0 {
		buf.Truncate(buf.Len() - 2)
	}
	buf.WriteString(")")
	fmt.Println(buf.String())
	return tvm.Execute(buf.String())
}

func (tvm *Tvm) jsonValueToBuf(buf *bytes.Buffer, value interface{}) {
	switch value.(type) {
	case float64:
		buf.WriteString(strconv.FormatFloat(value.(float64), 'f', 0, 64))
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
		panic("")
	}
}









