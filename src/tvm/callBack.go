package tvm

/*
#cgo CFLAGS:  -I ../../include

#include "tvm.h"

//int callOnMeGo_cgo(int in); // Forward declaration.
*/
import "C"
import (
	"fmt"
	"unsafe"
)
//export callBack
func callBack(function *C.char, cArray *C.char, length C.int) {
	params := C.GoStringN(cArray, length)
	switch function {
	case "test":
		testNoString(params)
	default:
		panic("")
	}
}
//export callBackString
func callBackString(function *C.char, cArray unsafe.Pointer, length C.int) string{
	params := C.GoBytes(cArray, length)
	switch function {
	case "test":
		return testString(params)
	default:
		panic("")
	}
}

func testString(params []byte) string{
	return "testString"
}

func testNoString(params []byte) {
	fmt.Println("testNoString")
}