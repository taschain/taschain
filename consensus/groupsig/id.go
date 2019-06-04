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

// ID is id for secret sharing, represented by big.Int
// Secret shared ID, 64 bit int, a total of 256 bits
type ID struct {
	value BnInt
}

// IsEqual determine if the 2 IDs are the same
func (id ID) IsEqual(rhs ID) bool {
	return id.value.IsEqual(&rhs.value)
}

// SetBigInt convert big.Int to ID
func (id *ID) SetBigInt(b *big.Int) error {
	id.value.SetBigInt(b)
	return nil
}

// SetDecimalString convert a decimal string to an ID
func (id *ID) SetDecimalString(s string) error {
	return id.value.SetDecString(s)
}

// SetHexString converts the hexadecimal string to ID
func (id *ID) SetHexString(s string) error {
	return id.value.SetHexString(s)
}

func (id *ID) GetLittleEndian() []byte {
	return id.Serialize()
}

func (id *ID) SetLittleEndian(buf []byte) error {
	return id.Deserialize(buf)
}

// Deserialize convert byte slices to id
func (id *ID) Deserialize(b []byte) error {
	return id.value.Deserialize(b)
}

// GetBigInt convert the ID to big.int
func (id ID) GetBigInt() *big.Int {
	x := new(big.Int)
	x.Set(id.value.GetBigInt())
	return x
}

func (id ID) IsValid() bool {
	bi := id.GetBigInt()
	return bi.Cmp(big.NewInt(0)) != 0

}

// GetHexString converts the ID to a hexadecimal string
func (id ID) GetHexString() string {
	bs := id.Serialize()
	return common.ToHex(bs)
}

// Serialize convert ID to byte slice (LittleEndian)
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
	str := id.GetHexString()
	return common.ShortHex12(str)
}

// NewIDFromBigInt create ID by big.int
func NewIDFromBigInt(b *big.Int) *ID {
	id := new(ID)
	err := id.value.SetDecString(b.Text(10)) //bncurve C库函数
	if err != nil {
		log.Printf("NewIDFromBigInt %s\n", err)
		return nil
	}
	return id
}

// NewIDFromInt64 create ID by int64
func NewIDFromInt64(i int64) *ID {
	return NewIDFromBigInt(big.NewInt(i))
}

// NewIDFromInt Create ID by int32
func NewIDFromInt(i int) *ID {
	return NewIDFromBigInt(big.NewInt(int64(i)))
}

// NewIDFromAddress create ID from TAS 160-bit address (FP254 curve 256 bit or
// FP382 curve 384 bit)
//
// Bncurve.ID and common.Address do not support two-way back and forth conversions
// to each other, because their codomain is different (384 bits and 160 bits),
// and the interchange generates different values.
func NewIDFromAddress(addr common.Address) *ID {
	return NewIDFromBigInt(addr.BigInteger())
}

// NewIDFromPubkey construct ID by public key
//
// Public key -> (reduced to 160 bits) address -> (zoom in to 256/384 bit) ID
func NewIDFromPubkey(pk Pubkey) *ID {
	// Get the SHA3 256-bit hash of the public key
	h := sha3.Sum256(pk.Serialize())
	bi := new(big.Int).SetBytes(h[:])
	return NewIDFromBigInt(bi)
}

// NewIDFromString  generate ID by string, incoming string must guarantee discreteness
func NewIDFromString(s string) *ID {
	bi := new(big.Int).SetBytes([]byte(s))
	return NewIDFromBigInt(bi)
}
func DeserializeID(bs []byte) ID {
	var id ID
	if err := id.Deserialize(bs); err != nil {
		return ID{}
	}
	return id
}

func (id ID) String() string {
	return id.GetHexString()
}

func (id ID) ToAddress() common.Address {
	return common.BytesToAddress(id.Serialize())
}
