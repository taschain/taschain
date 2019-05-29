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

// types

//POP即私钥对公钥的签名
type Pop Signature

//Proof-of-Possesion：拥有证明

//POP生成，用私钥对公钥的序列化值签名。
func GeneratePop(sec Seckey, pub Pubkey) Pop {
	return Pop(Sign(sec, pub.Serialize()))
}

//POP验证，用公钥对POP进行验证，确认POP是由该公钥对应的私钥生成的。
func VerifyPop(pub Pubkey, pop Pop) bool {
	return VerifySig(pub, pub.Serialize(), Signature(pop))
}
