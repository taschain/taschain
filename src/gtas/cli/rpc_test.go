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

package cli

import (
	"encoding/json"
	"log"
	"testing"
)

func TestRPC(t *testing.T) {
	gtas := NewGtas()
	gtas.simpleInit("tas.ini")
	err := gtas.fullInit()
	if err != nil {
		t.Error(err)
	}
	host := "127.0.0.1"
	var port uint = 8080
	StartRPC(host, port)
	tests := []struct {
		method string
		params []interface{}
	}{
		{"GTAS_newWallet", nil},
		{"GTAS_tx", []interface{}{"0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb", "0x5ca33e8ce7c3c97e0f7fa66db4371367e298621f", 1, ""}},
		{"GTAS_balance", []interface{}{"0x8ad32757d4dbcea703ba4b982f6fd08dad84bfcb"}},
		{"GTAS_blockHeight", nil},
		{"GTAS_getWallets", nil},
		//{},
	}
	for _, test := range tests {
		res, err := rpcPost(host, port, test.method, test.params...)
		if err != nil {
			t.Errorf("%s failed: %v", test.method, err)
			continue
		}
		if res.Error != nil {
			t.Errorf("%s failed: %v", test.method, res.Error.Message)
			continue
		}
		data, _ := json.Marshal(res.Result.Data)
		log.Printf("%s response data: %s", test.method, data)
	}
}
