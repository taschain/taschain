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

package test

import (
	"fmt"
	"testing"
)

func TestSlice(t *testing.T) {
	fmt.Println("-----------a--------------------")
	a := make([]int, 10)
	fmt.Println(a)
	fmt.Printf("len(a):%d,cap(a):%d\n", len(a), cap(a))

	a = append(a, 1)
	fmt.Println(a)
	fmt.Printf("len(a):%d,cap(a):%d\n", len(a), cap(a))

	fmt.Println("\n-----------b--------------------")
	b := a[:]
	fmt.Println(b)
	fmt.Printf("len(ba):%d,cap(b):%d\n", len(b), cap(b))

	fmt.Println("\n-----------c--------------------")
	c := make([]int, 0, 4)
	fmt.Println(c)
	fmt.Printf("len(c):%d,cap(c):%d\n", len(c), cap(c))

	c = append(c, 1)
	fmt.Println(c)
	fmt.Printf("len(c):%d,cap(c):%d\n", len(c), cap(c))

	c = append(c, 2)
	c = append(c, 3)
	c = append(c, 4)
	fmt.Println(c)
	fmt.Printf("len(c):%d,cap(c):%d\n", len(c), cap(c))

	c = append(c, 5)
	fmt.Println(c)
	fmt.Printf("len(c):%d,cap(c):%d\n", len(c), cap(c))

	fmt.Println("\n-----------d--------------------")
	d := new([]int)
	fmt.Println(d)
	fmt.Printf("len(d):%d,cap(d):%d\n", len(*d), cap(*d))
	*d = append(*d, 1)
	fmt.Println(d)
	fmt.Printf("len(d):%d,cap(d):%d\n", len(*d), cap(*d))
	*d = append(*d, 2)
	fmt.Println(d)
	fmt.Printf("len(d):%d,cap(d):%d\n", len(*d), cap(*d))
	*d = append(*d, 3)
	fmt.Println(d)
	fmt.Printf("len(d):%d,cap(d):%d\n", len(*d), cap(*d))

}
