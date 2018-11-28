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

package common

import (
	"testing"
	"encoding/json"
	"log"
)

/*
**  Creator: pxf
**  Date: 2018/9/30 下午3:11
**  Description: 
*/

func TestHash_Hex(t *testing.T) {
	var h Hash
	h = HexToHash("0x1234")
	t.Log(h.Hex())
	
	s := "0xf3be4592802e6bfa85bf449c41eea1fc7a695220590c677c46d84339a13eec1a"
	h = HexToHash(s)
	t.Log(h.Hex())
}


func TestAddress_MarshalJSON(t *testing.T) {
	addr := HexToAddress("0x123")

	bs, _ := json.Marshal(&addr)
	log.Printf(string(bs))
}