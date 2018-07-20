package tvm

/*
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

	intary :=  (*[2]unsafe.Pointer)(ary)

	fmt.Println(intary)

	testInt := *(*int)(intary[0])
	fmt.Println(testInt)
	C.free(unsafe.Pointer(intary[0]))

	testStr := C.GoString((*C.char)(intary[1]))
	fmt.Println(testStr)
	C.free(unsafe.Pointer(intary[1]))
}

//export transfer
func transfer(fromAddresss *C.char, toAddress *C.char, amount int) {
	fmt.Printf("From %s Send to %s amout: %d\n", C.GoString(fromAddresss), C.GoString(toAddress), amount)
}

