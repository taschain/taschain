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

void wrap_transfer(const char* p1, const char* p2, int p3)
{
    void transfer(const char*, const char*, int);
    transfer(p1, p2, p3);
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

char* wrap_get_state(char* address, char* hash)
{
	char* GetState(char*, char*);
	return GetState(address, hash);
}

void wrap_set_state(char* address, char* hash, char* state)
{
	void SetState(char*, char*, char*);
	SetState(address, hash, state);
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



*/
import "C"
import (
	"unsafe"
	"vm/core/vm"
	"encoding/json"
	"fmt"
	"bytes"
	"strconv"
)

var tvm *Tvm

func bridge_init() {
	C.tvm_setup_func((C.callback_fcn)(unsafe.Pointer(C.callOnMeGo_cgo)))
	C.tvm_set_testAry_func((C.testAry_fcn)(unsafe.Pointer(C.wrap_testAry)))
	C.setTransferFunc((C.TransferFunc)(unsafe.Pointer(C.wrap_transfer)))
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
	C.get_state = (C.Function10)(unsafe.Pointer(C.wrap_get_state))
	C.set_state = (C.Function11)(unsafe.Pointer(C.wrap_set_state))
	C.suicide = (C.Function4)(unsafe.Pointer(C.wrap_suicide))
	C.has_suicide = (C.Function4)(unsafe.Pointer(C.wrap_has_suicide))
	C.exists = (C.Function4)(unsafe.Pointer(C.wrap_exists))
	C.empty = (C.Function4)(unsafe.Pointer(C.wrap_empty))
	C.revert_to_snapshot = (C.Function12)(unsafe.Pointer(C.wrap_revert_to_snapshot))
	C.snapshot = (C.Function13)(unsafe.Pointer(C.wrap_snapshot))
	C.add_preimage = (C.Function5)(unsafe.Pointer(C.wrap_add_preimage))
}


type Tvm struct {
	state vm.StateDB
}

func NewTvm(state vm.StateDB)*Tvm {
	if tvm == nil {
		tvm = &Tvm{}
	}
	tvm.state = state

	C.tvm_start()
	bridge_init()

	return tvm
}

func (tvm *Tvm)Execute(script string) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	C.tvm_execute(C.CString(script))
}

type ABI struct {
	FuncName string
	Args []interface{}
}
func (tvm *Tvm) ExecuteABIJson(j string) {
	res := ABI{}
	json.Unmarshal([]byte(j), &res)
	fmt.Println(res)

	var buf bytes.Buffer
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
	fmt.Println(buf.String())
	tvm.Execute(buf.String())
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









