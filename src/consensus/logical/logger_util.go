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
	"fmt"
	"time"
)

/*
**  Creator: pxf
**  Date: 2018/6/8 上午9:52
**  Description: 
*/
const TIMESTAMP_LAYOUT = "2006-01-02/15:04:05.000"

func logStart(mtype string, height uint64, qn uint64, sender string, format string, params ...interface{}) {
	var s string
	if params == nil || len(params) == 0 {
		s = format
	} else {
		s = fmt.Sprintf(format, params...)
	}
	consensusLogger.Infof("%v,%v-begin,#%v-%v#,%v,%v", time.Now().Format(TIMESTAMP_LAYOUT), mtype, height, qn, sender, s)
}

func logEnd(mtype string, height uint64, qn uint64, sender string) {
	consensusLogger.Infof("%v,%v-end,#%v-%v#,%v,%v", time.Now().Format(TIMESTAMP_LAYOUT), mtype, height, qn, sender, "")
}


func logHalfway(mtype string, height uint64, qn uint64, sender string, format string, params ...interface{}) {
	var s string
	if params == nil || len(params) == 0 {
		s = format
	} else {
		s = fmt.Sprintf(format, params...)
	}
	consensusLogger.Infof("%v,%v-half,#%v-%v#,%v,%v", time.Now().Format(TIMESTAMP_LAYOUT), mtype, height, qn, sender, s)
}