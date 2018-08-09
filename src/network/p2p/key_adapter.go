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

package p2p

import (
	"common"
	"github.com/libp2p/go-libp2p-crypto"
	pb "github.com/libp2p/go-libp2p-crypto/pb"
	"github.com/gogo/protobuf/proto"
	"bytes"
)

// adapt common.Privatekey and common.Publickey to use key for libp2p

var keyType pb.KeyType = 3

type Pubkey struct {
	PublicKey common.PublicKey
}

type Privkey struct {
	PrivateKey common.PrivateKey
}

func (pub *Pubkey) Verify(data []byte, sig []byte) (bool, error) {
	return pub.PublicKey.Verify(data, common.BytesToSign(sig)), nil
}

func (pub *Pubkey) Bytes() ([]byte, error) {
	pbmes := new(pb.PublicKey)
	pbmes.Type = &keyType
	pbmes.Data = pub.PublicKey.ToBytes()
	return proto.Marshal(pbmes)
}

func (pub *Pubkey) Equals(key crypto.Key) bool {
	p, ok := key.(*Pubkey)
	if !ok {
		return false
	}
	b1, e1 := pub.Bytes()
	b2, e2 := p.Bytes()
	if e1 != nil || e2 != nil {
		return false
	}
	return bytes.Equal(b1[:], b2[:])
}

func UnmarshalEcdsaPublicKey(b []byte) (crypto.PubKey, error) {
	pk := common.BytesToPublicKey(b)
	return &Pubkey{PublicKey: *pk}, nil
}

func (prv *Privkey) Sign(b []byte) ([]byte, error) {
	return prv.PrivateKey.Sign(b).Bytes(), nil
}

func (prv *Privkey) GetPublic() crypto.PubKey {
	pub := prv.PrivateKey.GetPubKey()
	return &Pubkey{pub}
}

func (prv *Privkey) Bytes() ([]byte, error) {
	pbmes := new(pb.PrivateKey)
	pbmes.Type = &keyType
	pbmes.Data = prv.PrivateKey.ToBytes()
	return proto.Marshal(pbmes)
}

func (prv *Privkey) Equals(key crypto.Key) bool {
	p, ok := key.(*Pubkey)
	if !ok {
		return false
	}
	b1, e1 := prv.Bytes()
	b2, e2 := p.Bytes()
	if e1 != nil || e2 != nil {
		return false
	}
	return bytes.Equal(b1[:], b2[:])
}

func UnmarshalEcdsaPrivateKey(b []byte) (crypto.PrivKey, error) {
	pk := common.BytesToSecKey(b)
	return &Privkey{PrivateKey: *pk}, nil
}

func init() {
	pb.KeyType_name[3] = "Ecdsa"
	pb.KeyType_value["Ecdsa"] = 3

	crypto.PubKeyUnmarshallers[keyType] = UnmarshalEcdsaPublicKey
	crypto.PrivKeyUnmarshallers[keyType] = UnmarshalEcdsaPrivateKey
}
