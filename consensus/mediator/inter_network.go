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

package mediator

import (
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/consensus/groupsig"
)

///////////////////////////////////////////////////////////////////////////////
//矿工消息签名函数
func SignMessage(msg common.Hash, sk groupsig.Seckey) (sign []byte) {
	s := groupsig.Sign(sk, msg.Bytes())
	sign = s.Serialize()
	return
}

//矿工消息验签函数
func VerifyMessage(msg common.Hash, sign []byte, pk groupsig.Pubkey) bool {
	var s groupsig.Signature
	if s.Deserialize(sign) != nil {
		return false
	}
	b := groupsig.VerifySig(pk, msg.Bytes(), s)
	return b
}
