package groupsig

import (
	"github.com/taschain/taschain/common"
	"testing"
)

/*
**  Creator: pxf
**  Date: 2018/10/16 下午2:39
**  Description:
 */
//
//func TestSignature_MarshalJSON(t *testing.T) {
//	sig := Signature{}
//	sig.SetHexString("0x123")
//	bs := sig.GetHexString()
//	t.Log(string(bs))
//}

func TestVerifySig(t *testing.T) {
	var sign Signature
	sign.SetHexString("0x041eda274745e471ea9b4ee4da8d1ba9667fee6f32399531c55cd288a40525b501")
	var gpk Pubkey
	gpk.SetHexString("0x1a471817d295dc93c8492a84116f1a7200f88504eec8754be234df104583f3b820de4dc3fb6aca4b4873fb714dc39879756dd8d2b6cbeaf1d722032b664b129421bf91f4ed295c0d2ed27fe05cb3f641f24f014641471eee492e3e961f8fd6c40523abd48aec1bf89d292ad2643e4daefc34f74c35dcb0df057c517dc9ea6887")
	var hash = common.HexToHash("0x05830417649117d587035cdd5f9f874c98ceba8423640277bf7ed8657ea2b211verifySign")

	t.Log(VerifySig(gpk, hash.Bytes(), sign))
}
