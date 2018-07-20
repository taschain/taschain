package tvm

/*
#cgo LDFLAGS: -L ../../lib/darwin_amd64 -lmicropython
#cgo CFLAGS:  -I ../../include
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


*/
import "C"
import "unsafe"

func VmTest() {
	C.tvm_start()
	C.tvm_setup_func((C.callback_fcn)(unsafe.Pointer(C.callOnMeGo_cgo)))
	C.tvm_set_testAry_func((C.testAry_fcn)(unsafe.Pointer(C.wrap_testAry)))
	C.tvm_execute(C.CString("import tas\ntas.test()"))
	//fmt.Printf("Go.main(): calling C function with callback to us\n")
	//C.some_c_func((C.callback_fcn)(unsafe.Pointer(C.callOnMeGo_cgo)))
}

func VmTestContract() {
	C.tvm_start()
	script := `
from TasAccount import *

def apply():
    myAccount = TasAccount()
    myAccount.address = "myAddress"
    otherAccount = "otherAddress"
    myAccount.transfer(otherAccount, 50)

apply()
`
	C.tvm_execute(C.CString(script))
}


