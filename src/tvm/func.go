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

	intary :=  (*[5]unsafe.Pointer)(ary)

	fmt.Println(intary)

	for i := 0; i < 5; i++ {
		//var hdr reflect.SliceHeader
		//hdr.Data = uintptr(intary[i])
		//hdr.Len = (int)(unsafe.Sizeof(int(0)))

		testInt := *(*int)(intary[i])

		fmt.Println(testInt)


	}

	C.free(unsafe.Pointer(intary[0]))

}