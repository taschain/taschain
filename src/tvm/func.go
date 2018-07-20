package tvm

/*
#cgo CFLAGS:  -I ../../include
#include <stdlib.h>

*/
import "C"
import (
	"fmt"
	"unsafe"
)
//export callOnMeGo
func callOnMeGo(in int) int {
	fmt.Printf("Go.callOnMeGo(): called with arg = %d\n", in)
	return in + 2
}

//export go_testAry
func go_testAry(ary unsafe.Pointer) {
	//var identifier []unsafe.Pointer
	//var identifier unsafe.Pointer
	//identifier = C.CBytes(ary)

	intary :=  (*[2]unsafe.Pointer)(ary)

	fmt.Println(intary)

	testInt := *(*int)(intary[0])
	fmt.Println(testInt)
	C.free(unsafe.Pointer(intary[0]))

	testStr := C.GoString((*C.char)(intary[1]))
	fmt.Println(testStr)
	C.free(unsafe.Pointer(intary[1]))
}