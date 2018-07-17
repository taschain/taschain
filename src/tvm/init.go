package tvm

/*
#cgo LDFLAGS: -L ../../lib/darwin_amd64 -lmicropython
#cgo CFLAGS:  -I ../../include
#include <tvm.h>
*/
import "C"

func VmTest() {
	C.tvm_start()
	C.tvm_test()
}


