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
