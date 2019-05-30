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

package monitor

import (
	"fmt"
	"github.com/codeskyblue/go-sh"
	"os"
	"testing"
)

/*
**  Creator: pxf
**  Date: 2019/3/6 下午2:32
**  Description:
 */

func TestExecuteCmd(t *testing.T) {
	sess := sh.NewSession()
	sess.ShowCMD = true
	bs, err := sess.Command("top", "-b", "-n 1", fmt.Sprintf("-p %v", os.Getpid())).Command("grep", "gtas").Output()
	t.Log(string(bs), err)
}
