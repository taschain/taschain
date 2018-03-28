package groupsig

import (
	"common"
	"consensus/bls"
	"log"
	"math/big"
)

// ID -- id for secret sharing, represented by big.Int
//秘密共享的ID，64位int，共256位
type ID struct {
	//	value big.Int
	value bls.ID
}

//判断2个ID是否相同
func (id ID) IsEqual(rhs ID) bool {
	// TODO : add IsEqual to bls.ID
	return id.value.GetHexString() == rhs.value.GetHexString() //hex string
}

//把big.Int转换到ID
func (id *ID) SetBigInt(b *big.Int) error {
	return id.value.SetHexString(b.Text(16))
}

//把十进制字符串转换到ID
func (id *ID) SetDecimalString(s string) error {
	return id.value.SetDecString(s)
}

//把十六进制字符串转换到ID
func (id *ID) SetHexString(s string) error {
	return id.value.SetHexString(s)
}

//把字节切片转换到ID
func (id *ID) Deserialize(b []byte) error {
	return id.value.SetLittleEndian(b)
}

//把ID转换到big.Int
func (id ID) GetBigInt() *big.Int {
	x := new(big.Int)
	x.SetString(id.value.GetHexString(), 16)
	return x
}

//把ID转换到十进制字符串
func (id ID) GetDecimalString() string {
	return id.value.GetDecString()
}

//把ID转换到十六进制字符串
func (id ID) GetHexString() string {
	return id.value.GetHexString()
}

//把ID转换到字节切片（小端模式）
func (id ID) Serialize() []byte {
	return id.value.GetLittleEndian()
}

//由big.Int创建ID
func NewIDFromBigInt(b *big.Int) *ID {
	id := new(ID)
	err := id.value.SetDecString(b.Text(10)) //bls C库函数
	if err != nil {
		log.Printf("NewIDFromBigInt %s\n", err)
		return nil
	}
	return id
}

//由int64创建ID
func NewIDFromInt64(i int64) *ID {
	return NewIDFromBigInt(big.NewInt(i))
}

//由int32创建ID-
func NewIDFromInt(i int) *ID {
	return NewIDFromBigInt(big.NewInt(int64(i)))
}

//从TAS 160位地址创建ID
func NewIDFromAddress(addr common.Address) *ID {
	return NewIDFromBigInt(addr.BigInteger())
}
