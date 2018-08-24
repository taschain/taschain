package groupsig

//Added by FlyingSquirrel-Xu. 2018-08-24.

import (
	"fmt"
	"consensus/groupsig/bn_curve"
	"math/big"
)

const PREFIX = "0x"

// GetMaxOpUnitSize --
func GetMaxOpUnitSize() int {
	return 4
}
const CurveFp254BNb = 1

func revertString(b string) string {
	len := len(b)
	buf := make([]byte, len)
	for i := 0; i < len; i++ {
		buf[i] = b[len-1-i]
	}
	return string(buf)
}

func HashToG1(m string) *bn_curve.G1 {
	b := &big.Int{}
	b.SetBytes([]byte(m))

	b.Mod(b, bn_curve.Order)

	bg := &bn_curve.G1{}
	bg.ScalarBaseMult(b)

	return bg
}

type BlsInt struct {
	v big.Int
}


//func blstest() error {
//	buf := make([]byte, 32)
//	rand.Read(buf)
//	fmt.Println("buf:", buf)
//	return nil
//}

func (bi *BlsInt) IsEqual(b *BlsInt) bool {
	return 0 == bi.v.Cmp(&b.v)
}

// SetDecString --
func (bi *BlsInt) SetDecString(s string) error {
	bi.v.SetString(s, 10)
	return nil
}

//func (bi *BlsInt) GetDecString () s string {
//
//}

func (bi *BlsInt) Add(b *BlsInt) error {
	b1 := &BlsInt{}
	b2 := &BlsInt{}

	b1.v.Set(&bi.v)
	b2.v.Set(&b.v)

	bi.v.Add(&b1.v, &b2.v)
	return nil
}

func (bi *BlsInt) Sub(b *BlsInt) error {
	bi.v.Sub(&bi.v, &b.v)
	return nil
}

func (bi *BlsInt) Mul(b *BlsInt) error {
	nb := &big.Int{}
	nb.Set(&bi.v)
	bi.v.Mul(nb, &b.v)
	return nil
}

func (bi *BlsInt) Mod() error {
	nb := &big.Int{}
	nb.Set(&bi.v)
	bi.v.Mod(nb, bn_curve.Order)
	return nil
}

func (bi *BlsInt) SetBigInt(b *big.Int) error {
	bi.v.Set(b)
	return nil
}

func (bi *BlsInt) SetString(s string) error {
	bi.v.SetString(s, 10)
	return nil
}

func (bi *BlsInt) SetHexString(s string) error {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return fmt.Errorf("arg failed")
	}
	buf := s[len(PREFIX):]
	bi.v.SetString(buf[:], 16)
	return nil
}

//BlsInt导出为big.Int
func (bi *BlsInt) GetBigInt() *big.Int {
	x := new(big.Int)
	x.Set(&bi.v)
	return x
}

func (bi *BlsInt) GetString() string {
	bigInt := bi.GetBigInt()
	b := bigInt.Bytes()
	return string(b)
}

func (bi *BlsInt) GetHexString() string {
	buf := bi.v.Text(16)
	return PREFIX + buf
}

func (bi *BlsInt) Serialize() []byte {
	return bi.v.Bytes()
}

func (bi *BlsInt) Deserialize(b []byte) error {
	bi.v.SetBytes(b)
	return nil
}

type blsG1 struct {
	v bn_curve.G1
}

// getPointer --
// func (s *blsG1) getPointer() (p *C.blsSignature) {
// 	return (*C.blsSignature)(unsafe.Pointer(sign))
// }

type blsG2 struct {
	v bn_curve.G2
}

func (bi *blsG2) Deserialize(b []byte) error {
	bi.v.Unmarshal(b)
	return nil
}

//序列化
func (bg *blsG2) Serialize() []byte {
	return bg.v.Marshal()
}

func (bg *blsG2) Add(bh *blsG2) error {
	a := &blsG2{}
	b := &blsG2{}
	a.Deserialize(bh.Serialize())
	b.Deserialize(bh.Serialize())

	bg.v.Add(&a.v, &b.v)
	return nil
}


