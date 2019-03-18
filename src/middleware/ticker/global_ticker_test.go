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

package ticker

import (
	"testing"
	"time"
	"log"
)

func handler(str string) RoutineFunc {
	return func() bool {
		log.Printf(str)
		return true
	}
}

func TestGlobalTicker_RegisterRoutine(t *testing.T) {

	ticker := NewGlobalTicker("test")

	time.Sleep(time.Second * 5)

	ticker.RegisterPeriodicRoutine("name1", handler("name1 exec1"), uint32(2))

	time.Sleep(time.Second * 5)
	ticker.RegisterPeriodicRoutine("name2", handler("name2 exec1"), uint32(3))
	time.Sleep(time.Second * 5)

	ticker.RegisterPeriodicRoutine("name3", handler("name3 exec1"), uint32(4))

	ticker.StopTickerRoutine("name1")


	time.Sleep(time.Second * 5)
	ticker.StopTickerRoutine("name3")
	time.Sleep(time.Second * 55)
}
