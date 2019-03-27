package logical

import (
	"testing"
	"math/big"
	"common"
	"consensus/base"
)

/*
**  Creator: pxf
**  Date: 2018/9/21 下午4:29
**  Description: 
*/

func TestBigInt(t *testing.T) {
	t.Log(max256, max256.String(), max256.FloatString(10))
}

func TestBigIntDiv(t *testing.T) {
	a, _ := new(big.Int).SetString("ffffffffffff", 16)
	b := new(big.Rat).SetInt(a)
	v := new(big.Rat).Quo(b, max256)
	t.Log(a, b, max256, v)
	t.Log(v.FloatString(5))

	a1 := new(big.Rat).SetInt64(10)
	a2 := new(big.Rat).SetInt64(30)
	v2 := a1.Quo(a1, a2)
	t.Log(v2.Float64())
	t.Log( v2.FloatString(5))
}

func TestCMP(t *testing.T) {
	rat := new(big.Rat).SetInt64(1)

	i := 1
	for i < 1000 {
		i++
		v := new(big.Rat).SetFloat64(1.66666666666666666666667)
		if v.Cmp(rat) > 0 {
			v = rat
		}
		t.Log(v.Quo(v, new(big.Rat).SetFloat64(0.5)), rat)
	}
}

func TestProve(t *testing.T) {
	random := "0x194b3d24ddb883a1fd7d3b1e0038ebf9bb739759719eb1093f40e489fdacf6c200"
	sk := "0x7e7707df15aa16256d0c18e9ddd59b36d48759ec5b6404cfb6beceea9a798879666a589f1bbc74ad4bc24c67c0845bd4e74d83f0e3efa3a4b465bf6e5600871c"
	pk := "0x666a589f1bbc74ad4bc24c67c0845bd4e74d83f0e3efa3a4b465bf6e5600871c"

	randomBytes := common.FromHex(random)
	msg := vrfM(randomBytes, 1)


	for i := 0; i < 10; i++ {
		pi, _ := base.VRF_prove(common.FromHex(pk), common.FromHex(sk), msg)

		pbytes := pi.Big().Bytes()

		t.Log(common.ToHex(base.VRF_proof2hash(pi)), common.ToHex(pbytes))

	}
}
