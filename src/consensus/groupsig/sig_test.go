package groupsig

import (
	"testing"
	"common"
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
	sign.SetHexString("0x000e11889e2404af5c54e4f29be90335e24fc49d1acccb6b595a33d033e3d49600")
	var gpk Pubkey
	gpk.SetHexString("0x23a99e494324002b110c2c4dd97003f6acdb31a19788c55e748a5fbbfde8a08b0e8e11dc7c18d52230d9f2320504d70f97c0742b5cecef7b437b24b4da6064ec2a508994c275d96d6fbef1c8ab0680a452c959f461bab0740905844cb4c8990e22a211ae4e4e2c6ed96386ae0981ae9ad2f9097ea7a5bcda8dbbed5c1d0dd69e")
	var hash = common.HexToHash("0x5f8809e65011c325b028fa7ab02d7697fa5af733ee8574e6e4711677ff790543")

	t.Log(VerifySig(gpk, hash.Bytes(), sign))
}