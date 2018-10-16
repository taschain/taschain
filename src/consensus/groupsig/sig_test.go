package groupsig

import "testing"

/*
**  Creator: pxf
**  Date: 2018/10/16 下午2:39
**  Description: 
*/

func TestSignature_MarshalJSON(t *testing.T) {
	sig := Signature{}
	sig.SetHexString("0x123")
	bs := sig.GetHexString()
	t.Log(string(bs))
}