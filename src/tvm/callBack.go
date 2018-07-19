package tvm

/*
#cgo CFLAGS:  -I ../../include

#include "tvm.h"

//int callOnMeGo_cgo(int in); // Forward declaration.
*/
import "C"
import (
	"fmt"
)




func testString(params []byte) string{
	return "testString"
}

func testNoString(params []byte) {
	fmt.Println("testNoString")
}