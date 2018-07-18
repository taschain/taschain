package tvm

/*
#cgo CFLAGS:  -I ../../include

#include "tvm.h"

int callOnMeGo_cgo(int in); // Forward declaration.
*/
import "C"
import "fmt"

//export callOnMeGo
func callOnMeGo(in int) int {
	fmt.Printf("Go.callOnMeGo(): called with arg = %d\n", in)
	return in + 2
}
