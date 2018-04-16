package utility

import (
	"fmt"
	"testing"
)

func TestByteToInt(t *testing.T){

	var a uint32
	a=16
	bytes:= UInt32ToByte(a)
	i:= ByteToInt(bytes)

	if i ==16{
		fmt.Printf("OK")
	}else {
		fmt.Errorf("Failed")
	}
}
