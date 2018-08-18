package groupsig

import "testing"

/*
**  Creator: pxf
**  Date: 2018/8/17 下午3:37
**  Description: 
*/

func TestRandK(t *testing.T) {
	Init(1)
	m := make(SignatureIMap)
	var sign Signature
	sign.SetHexString("0x1")
	m["1"] = sign
	sign.SetHexString("0x2")
	m["2"] = sign
	sign.SetHexString("0x3")
	m["3"] = sign
	sign.SetHexString("0x4")
	m["4"] = sign
	m["6"] = sign
	m["8"] = sign
	m["9"] = sign
	m1 := randK(m, 5)

	for k, s := range m1 {
		t.Log(k, s)
	}
}