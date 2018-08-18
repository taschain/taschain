package pow

import (
	"math/big"
	"time"
)

/*
**  Creator: pxf
**  Date: 2018/8/8 上午11:34
**  Description: 
*/

const BIT_POW_HASH = 256
var MAX_HASH_256 *big.Int
var DIFFCULTY_20_24 *Difficulty

const (
	DIFFCULTY_LEVEL_NONE uint32 = 0
	DIFFCULTY_LEVEL_1 = 1
	DIFFCULTY_LEVEL_2 = 2
)

func init() {
	MAX_HASH_256, _ = new(big.Int).SetString("ffffffffffffffffffffffffffffffff", 16)
	DIFFCULTY_20_24 = NewDifficultyWithPrefixZero(24, 20, 5, 7)
}

type Difficulty struct {
	TargetMax  *big.Int
	TargetMin  *big.Int
	PowSecsMax int
	ConfirmSecsMax int

}

func generateBigIntWithPrefixZero(znum int) *big.Int {
	s := ""
	zero := 0
	for zero < znum  {
		s += "0"
		zero++
	}
	for zero < BIT_POW_HASH {
		s += "1"
		zero++
	}
	if v, ok := new(big.Int).SetString(s, 2); !ok {
		return new(big.Int)
	} else {
		return v
	}
}

func NewDifficultyWithPrefixZero(zmaxnum int, zminnum int, powSec int, confirmSec int) *Difficulty {
	return &Difficulty{
		TargetMax:  generateBigIntWithPrefixZero(zminnum),
		TargetMin:  generateBigIntWithPrefixZero(zmaxnum),
		PowSecsMax: powSec,
		ConfirmSecsMax: confirmSec,
	}
}

func (d *Difficulty) Satisfy(v *big.Int) bool {
    return v.Cmp(d.TargetMax) < 0
}

func (d *Difficulty) Level(v *big.Int) uint32 {
	if v.Cmp(d.TargetMin) < 0 {
		return DIFFCULTY_LEVEL_2
	}
	if v.Cmp(d.TargetMax) < 0 {
		return DIFFCULTY_LEVEL_1
	}
	return DIFFCULTY_LEVEL_NONE
}

func (d *Difficulty) powDeadline(start time.Time) *time.Time {
    t := start.Add(time.Second * time.Duration(d.PowSecsMax))
    return &t
}

func (d *Difficulty) confirmDeadline(start time.Time) *time.Time {
	t := start.Add(time.Second * time.Duration(d.ConfirmSecsMax))
	return &t
}