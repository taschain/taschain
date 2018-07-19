package tvm

/*
#cgo CFLAGS:  -I ../../include

#include "tvm.h"


*/
import "C"
import "fmt"
//export callOnMeGo
func callOnMeGo(in int) int {
	fmt.Printf("Go.callOnMeGo(): called with arg = %d\n", in)
	return in + 2
}

//export go_testAry
func go_testAry(i int) {
	fmt.Println(i)
}