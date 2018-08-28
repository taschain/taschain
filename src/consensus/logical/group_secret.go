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

package logical

import (
	"consensus/groupsig"
	"common"
)

/*
**  Creator: pxf
**  Date: 2018/6/15 下午2:23
**  Description: 
*/

type GroupSecret struct {
	SecretSign   []byte      //秘密签名
	EffectHeight uint64      //生效高度
	DataHash     common.Hash //签名的数据hash
}

func NewGroupSecret(sign groupsig.Signature, height uint64, hash common.Hash) *GroupSecret {
	return &GroupSecret{
		SecretSign:   sign.Serialize(),
		EffectHeight: height,
		DataHash:     hash,
	}
}