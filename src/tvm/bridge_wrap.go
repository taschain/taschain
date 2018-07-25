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

*/
import "C"
import (
	"unsafe"
	"vm/core/state"
	"encoding/json"
	"fmt"
	"bytes"
	"strconv"
)

func bridge_init() {
	C.tvm_setup_func((C.callback_fcn)(unsafe.Pointer(C.callOnMeGo_cgo)))
	C.tvm_set_testAry_func((C.testAry_fcn)(unsafe.Pointer(C.wrap_testAry)))
	C.setTransferFunc((C.TransferFunc)(unsafe.Pointer(C.wrap_transfer)))
}


type Tvm struct {
	state *state.StateDB
}

func NewTvm(state *state.StateDB)*Tvm {
	tvm := Tvm{}
	tvm.state = state

	C.tvm_start()
	bridge_init()

	return &tvm
}

func (tvm *Tvm)Execute(script string) {
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









