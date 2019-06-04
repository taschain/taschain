package groupsig

import (
	"fmt"
	"github.com/taschain/taschain/consensus/base"
	"github.com/taschain/taschain/consensus/groupsig/bncurve"
	"math/big"
)

const PREFIX = "0x"

func revertString(b string) string {
	len := len(b)
	buf := make([]byte, len)
	for i := 0; i < len; i++ {
		buf[i] = b[len-1-i]
	}
	return string(buf)
}

func HashToG1(m string) *bncurve.G1 {
	g := &bncurve.G1{}
	g.HashToPoint([]byte(m))
	return g
}

type BnInt struct {
	v big.Int
}

func (bi *BnInt) IsEqual(b *BnInt) bool {
	return 0 == bi.v.Cmp(&b.v)
}

func (bi *BnInt) SetDecString(s string) error {
	bi.v.SetString(s, 10)
	return nil
}

func (bi *BnInt) Add(b *BnInt) error {
	b1 := &BnInt{}
	b2 := &BnInt{}

	b1.v.Set(&bi.v)
	b2.v.Set(&b.v)

	bi.v.Add(&b1.v, &b2.v)
	return nil
}

func (bi *BnInt) Sub(b *BnInt) error {
	bi.v.Sub(&bi.v, &b.v)
	return nil
}

func (bi *BnInt) Mul(b *BnInt) error {
	nb := &big.Int{}
	nb.Set(&bi.v)
	bi.v.Mul(nb, &b.v)
	return nil
}

func (bi *BnInt) Mod() error {
	nb := &big.Int{}
	nb.Set(&bi.v)
	bi.v.Mod(nb, bncurve.Order)
	return nil
}

func (bi *BnInt) SetBigInt(b *big.Int) error {
	bi.v.Set(b)
	return nil
}

func (bi *BnInt) SetString(s string) error {
	bi.v.SetString(s, 10)
	return nil
}

func (bi *BnInt) SetHexString(s string) error {
	if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
		return fmt.Errorf("arg failed")
	}
	buf := s[len(PREFIX):]
	bi.v.SetString(buf[:], 16)
	return nil
}

// GetBigInt export BlsInt as big.Int
func (bi *BnInt) GetBigInt() *big.Int {
	x := new(big.Int)
	x.Set(&bi.v)
	return x
}

func (bi *BnInt) GetString() string {
	bigInt := bi.GetBigInt()
	b := bigInt.Bytes()
	return string(b)
}

func (bi *BnInt) GetHexString() string {
	buf := bi.v.Text(16)
	return PREFIX + buf
}

func (bi *BnInt) Serialize() []byte {
	return bi.v.Bytes()
}

func (bi *BnInt) Deserialize(b []byte) error {
	bi.v.SetBytes(b)
	return nil
}

type bnG2 struct {
	v bncurve.G2
}

func (bg *bnG2) Deserialize(b []byte) error {
	bg.v.Unmarshal(b)
	return nil
}

func (bg *bnG2) Serialize() []byte {
	return bg.v.Marshal()
}

func (bg *bnG2) Add(bh *bnG2) error {
	a := &bnG2{}
	b := &bnG2{}
	a.Deserialize(bh.Serialize())
	b.Deserialize(bh.Serialize())

	bg.v.Add(&a.v, &b.v)
	return nil
}

func (sec *Seckey) GetMasterSecretKey(k int) (msk []Seckey) {
	msk = make([]Seckey, k)
	msk[0] = *sec

	// Generating random number
	r := base.NewRand()
	for i := 1; i < k; i++ {
		msk[i] = *NewSeckeyFromRand(r.Deri(1))
	}
	return msk
}

func GetMasterPublicKey(msk []Seckey) (mpk []Pubkey) {
	n := len(msk)
	mpk = make([]Pubkey, n)
	for i := 0; i < n; i++ {
		mpk[i] = *NewPubkeyFromSeckey(msk[i])
	}
	return mpk
}
