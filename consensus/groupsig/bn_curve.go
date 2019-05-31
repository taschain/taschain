package groupsig

//Added by FlyingSquirrel-Xu. 2018-08-24.

import (
	"fmt"
	"github.com/taschain/taschain/consensus/base"
	"github.com/taschain/taschain/consensus/groupsig/bn_curve"
	"math/big"
)

const PREFIX = "0x"

// GetMaxOpUnitSize --
//func GetMaxOpUnitSize() int {
//	return 4
//}

func revertString(b string) string {
	len := len(b)
	buf := make([]byte, len)
	for i := 0; i < len; i++ {
		buf[i] = b[len-1-i]
	}
	return string(buf)
}

func HashToG1(m string) *bn_curve.G1 {
	g := &bn_curve.G1{}
	g.HashToPoint([]byte(m))
	return g
}

type BnInt struct {
	v big.Int
}

func (bi *BnInt) IsEqual(b *BnInt) bool {
	return 0 == bi.v.Cmp(&b.v)
}

// SetDecString --
func (bi *BnInt) SetDecString(s string) error {
	bi.v.SetString(s, 10)
	return nil
}

//func (bi *BnInt) GetDecString () s string {
//
//}

func (bi *BnInt) Add(b *BnInt) error {
	bi.v.Add(&bi.v, &b.v)
	return nil
}

func (bi *BnInt) Sub(b *BnInt) error {
	bi.v.Sub(&bi.v, &b.v)
	return nil
}

func (bi *BnInt) Mul(b *BnInt) error {
	bi.v.Mul(&bi.v, &b.v)
	return nil
}

func (bi *BnInt) Mod() error {
	bi.v.Mod(&bi.v, bn_curve.Order)
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

//BlsInt导出为big.Int
func (bi *BnInt) GetBigInt() *big.Int {
	return new(big.Int).Set(&bi.v)
}

func (bi *BnInt) GetString() string {
	b := bi.GetBigInt().Bytes()
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
	v bn_curve.G2
}

func (bi *bnG2) Deserialize(b []byte) error {
	bi.v.Unmarshal(b)
	return nil
}

//序列化
func (bg *bnG2) Serialize() []byte {
	return bg.v.Marshal()
}

func (bg *bnG2) Add(bh *bnG2) error {
	bg.v.Add(&bg.v, &bh.v)
	return nil
}

// GetMasterSecretKey --
func (sec *Seckey) GetMasterSecretKey(k int) (msk []Seckey) {
	msk = make([]Seckey, k)
	msk[0] = *sec

	r := base.NewRand() //生成随机数
	for i := 1; i < k; i++ {
		msk[i] = *NewSeckeyFromRand(r.Deri(1))
	}
	return msk
}

// GetMasterPublicKey --
func GetMasterPublicKey(msk []Seckey) (mpk []Pubkey) {
	n := len(msk)
	mpk = make([]Pubkey, n)
	for i := 0; i < n; i++ {
		mpk[i] = *NewPubkeyFromSeckey(msk[i])
	}
	return mpk
}
