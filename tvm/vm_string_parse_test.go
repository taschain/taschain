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

package tvm

import (
	"testing"
)

func TestVmStringParse(t *testing.T) {

	// 测试用例1
	parse := vmStringParse("1|2|dd")
	if len(parse) != 3 {
		t.Fatal("len is not 3")
	}
	if parse[0] != "1" {
		t.Fatal("wrong value")
	}
	if parse[1] != "2" {
		t.Fatal("wrong value")
	}
	if parse[2] != "dd" {
		t.Fatal("wrong value")
	}

	// 测试用例2
	parse = vmStringParse("1|33|3|3|")
	if len(parse) != 3 {
		t.Fatal("len is not 3")
	}
	if parse[0] != "1" {
		t.Fatal("wrong value")
	}
	if parse[1] != "33" {
		t.Fatal("wrong value")
	}
	if parse[2] != "3|3|" {
		t.Fatal("wrong value")
	}

}

func TestExecutedVmSucceed(t *testing.T) {
	isSuccess, _ := ExecutedVmSucceed("1|2|dd")
	if !isSuccess {
		t.Fatal("should Succeed")
	}

	isSuccess, _ = ExecutedVmSucceed("4|2|dd")
	if isSuccess {
		t.Fatal("should Succeed")
	}

}
