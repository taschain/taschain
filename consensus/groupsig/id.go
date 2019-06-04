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
	"fmt"
	"github.com/taschain/taschain/common"
	"golang.org/x/crypto/sha3"
	"log"
	"math/big"
)

// ID -- id for secret sharing, represented by big.Int
type ID struct {
	value BnInt
}

//check whether id is equal to rhs
func (id ID) IsEqual(rhs ID) bool {
	// TODO : add IsEqual to bncurve.ID
	return id.value.IsEqual(&rhs.value)
}

//Construct a ID with the specified big integer
func (id *ID) SetBigInt(b *big.Int) error {
	id.value.SetBigInt(b)
	return nil
}

//Construct a ID with the specified decimal string
func (id *ID) SetDecimalString(s string) error {
	return id.value.SetDecString(s)
}

//Construct a ID with the input hex string
func (id *ID) SetHexString(s string) error {
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

//Construct a ID with the input byte array
func (id *ID) Deserialize(b []byte) error {
	return id.value.Deserialize(b)
}

//Export ID into a big integer
func (id ID) GetBigInt() *big.Int {
	return new(big.Int).Set(id.value.GetBigInt())
}

//check id is valid
func (id ID) IsValid() bool {
	bi := id.GetBigInt()
	return bi.Cmp(big.NewInt(0)) != 0

}

//Export ID into a hex string
func (id ID) GetHexString() string {
	return common.ToHex(id.Serialize())
	//return id.value.GetHexString()
}

//Export ID into a byte array (little endian)
func (id ID) Serialize() []byte {
	idBytes := id.value.Serialize()
	if len(idBytes) == IDLENGTH {
		return idBytes
	}
	if len(idBytes) > IDLENGTH {
		panic("ID Serialize error: ID bytes is more than IDLENGTH")
	}
	buff := make([]byte, IDLENGTH)
	copy(buff[IDLENGTH-len(idBytes):IDLENGTH], idBytes)
	return buff
}

func (id ID) MarshalJSON() ([]byte, error) {
	str := "\"" + id.GetHexString() + "\""
	return []byte(str), nil
}

func (id *ID) UnmarshalJSON(data []byte) error {
	str := string(data[:])
	if len(str) < 2 {
		return fmt.Errorf("data size less than min")
	}
	str = str[1 : len(str)-1]
	return id.SetHexString(str)
}

func (id ID) ShortS() string {
	return common.ShortHex12(id.GetHexString())
}

//由big.Int创建ID
func NewIDFromBigInt(b *big.Int) *ID {
	id := new(ID)
	err := id.value.SetBigInt(b)
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

//Construct ID with the input address
func NewIDFromAddress(addr common.Address) *ID {
	return NewIDFromBigInt(addr.BigInteger())
}

//由公钥构建ID，公钥->（缩小到160位）地址->（放大到256/384位）ID
func NewIDFromPubkey(pk Pubkey) *ID {
	h := sha3.Sum256(pk.Serialize()) //取得公钥的SHA3 256位哈希
	bi := new(big.Int).SetBytes(h[:])
	return NewIDFromBigInt(bi)
}

//Construct ID with the input hex string
func NewIDFromString(s string) *ID {
	bi := new(big.Int).SetBytes(common.FromHex(s))
	return NewIDFromBigInt(bi)
}

//Construct ID with the input byte array
func DeserializeId(bs []byte) ID {
	var id ID
	if err := id.Deserialize(bs); err != nil {
		return ID{}
	}
	return id
}

//Convert ID to address
func (id ID) ToAddress() common.Address {
	return common.BytesToAddress(id.Serialize())
}
