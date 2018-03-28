package param

import (
	"testing"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2018/3/27 下午4:59
**  Description: 
*/

func TestParamDefs_GetParamByIndex(t *testing.T) {
	ptr := Params.GetParamByIndex(0)
	fmt.Println(ptr.CurrentValue())
}