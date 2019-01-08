package common

import (
	"strings"
	"fmt"
	"regexp"
	"strconv"
)

/*
**  Creator: pxf
**  Date: 2019/1/8 下午3:33
**  Description: 
*/

const (
	RA uint64 = 1
	KRA = 1000
	MRA = 1000000
	TAS = 1000000000
)

var (
	ErrEmptyStr = fmt.Errorf("empty string")
	ErrIllegalStr = fmt.Errorf("illegal string")
)

var re, _ = regexp.Compile("^([0-9]+)(ra|kra|mra|tas)?$")


func ParseCoin(s string) (uint64, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, ErrEmptyStr
	}

	arr := re.FindAllStringSubmatch(s, -1)
	if arr == nil || len(arr) == 0 {
		return 0, ErrIllegalStr
	}
	ret := arr[0]
	if ret == nil || len(ret) < 2 {
		return 0, ErrIllegalStr
	}
	num, err := strconv.Atoi(ret[1])
	if err != nil {
		return 0, err
	}
	unit := RA
	if len(ret) == 3 {
		switch ret[2] {
		case "kra":
			unit = KRA
		case "mra":
			unit = MRA
		case "tas":
			unit = TAS
		}
	}
	//fmt.Println(re.FindAllString(s, -1))
	return uint64(num) * unit, nil
}