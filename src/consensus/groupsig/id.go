//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package groupsig

import (
	"common"
	"fmt"
	"log"
	"math/big"
	"golang.org/x/crypto/sha3"
)

// ID -- id for secret sharing, represented by big.Int
//秘密共享的ID，64位int，共256位
type ID struct {
	value BnInt
}

//判断2个ID是否相同
func (id ID) IsEqual(rhs ID) bool {
	// TODO : add IsEqual to bn_curve.ID
	return id.value.IsEqual(&rhs.value)
}

//把big.Int转换到ID  
func (id *ID) SetBigInt(b *big.Int) error {
	id.value.SetBigInt(b)
	return nil
}

//把十进制字符串转换到ID
func (id *ID) SetDecimalString(s string) error {
	return id.value.SetDecString(s)
}

//把十六进制字符串转换到ID
func (id *ID) SetHexString(s string) error {
	//if len(s) < len(PREFIX) || s[:len(PREFIX)] != PREFIX {
	//	return fmt.Errorf("arg failed")
	//}
	//buf := s[len(PREFIX):]
	return id.value.SetHexString(s)
}

// GetLittleEndian --
func (id *ID) GetLittleEndian() []byte {
	return id.Serialize()
}

// SetLittleEndian --
func (id *ID) SetLittleEndian(buf []byte) error {
	return id.Deserialize(buf)
}

//把字节切片转换到ID
func (id *ID) Deserialize(b []byte) error {
	return id.value.Deserialize(b)
}

//把ID转换到big.Int
func (id ID) GetBigInt() *big.Int {
	x := new(big.Int)
	x.Set(id.value.GetBigInt())
	return x
}

func (id ID) IsValid() bool {
	bi := id.GetBigInt()
	return bi.Cmp(big.NewInt(0)) != 0

}

//把ID转换到十六进制字符串
func (id ID) GetHexString() string {
	bs := id.Serialize()
	return common.ToHex(bs)
	//return id.value.GetHexString()
}

//把ID转换到字节切片（小端模式）
func (id ID) Serialize() []byte {
	idBytes := id.value.Serialize()
	if len(idBytes) == IDLENGTH {
		return idBytes
	}
	if len(idBytes) > IDLENGTH {
		panic("ID Serialize error: ID bytes is more than IDLENGTH")
	}
	buff := make([]byte, IDLENGTH)
	copy(buff[IDLENGTH-len(idBytes):IDLENGTH],idBytes)
	return buff
}

func (id ID) MarshalJSON() ([]byte, error) {
	str := "\"" + id.GetHexString() + "\""
	return []byte(str), nil
}

func (id *ID) UnmarshalJSON(data []byte) error {
	str := string(data[:])
	if len(str) < 2 {
		return fmt.Errorf("data size less than min.")
	}
	str = str[1:len(str)-1]
	return id.SetHexString(str)
}

func (id ID) ShortS() string  {
	str := id.GetHexString()
	return common.ShortHex12(str)
}

//由big.Int创建ID
func NewIDFromBigInt(b *big.Int) *ID {
	id := new(ID)
	err := id.value.SetDecString(b.Text(10)) //bn_curve C库函数
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

//由int32创建ID
func NewIDFromInt(i int) *ID {
	return NewIDFromBigInt(big.NewInt(int64(i)))
}

//从TAS 160位地址创建（FP254曲线256位或FP382曲线384位的）ID
//bn_curve.ID和common.Address不支持双向来回互转，因为两者的值域不一样（384位和160位），互转就会生成不同的值。
func NewIDFromAddress(addr common.Address) *ID {
	return NewIDFromBigInt(addr.BigInteger())
}

//由公钥构建ID，公钥->（缩小到160位）地址->（放大到256/384位）ID
func NewIDFromPubkey(pk Pubkey) *ID {
	h := sha3.Sum256(pk.Serialize())  //取得公钥的SHA3 256位哈希
	bi := new(big.Int).SetBytes(h[:])
	return NewIDFromBigInt(bi)
}

//从字符串生成ID 传入的STRING必须保证离散性
func NewIDFromString(s string) *ID {
	bi := new(big.Int).SetBytes([]byte(s))
	return NewIDFromBigInt(bi)
}
func DeserializeId(bs []byte) ID {
	var id ID
	if err := id.Deserialize(bs); err != nil {
		return ID{}
	}
	return id
}

func (id ID) String() string {
	//bigInt := id.GetBigInt()
	//b := bigInt.Bytes()
	//return string(b)
	return id.GetHexString()
}

func (id ID) ToAddress() common.Address {
    return common.BytesToAddress(id.Serialize())
}