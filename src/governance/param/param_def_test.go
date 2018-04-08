package param

import (
	"testing"
	"fmt"
)

/*
**  Creator: pxf
**  Date: 2018/3/27 下午2:06
**  Description: 
*/

func newInstance() *ParamDef {
	def := newParamDef(10, func(input interface{}) error {
		return nil
	})
	return def
}

func TestParamDef_CurrentValue(t *testing.T) {
	def := newInstance()
	fmt.Print(def.CurrentValue())
}



func TestParamDef_AddFuture(t *testing.T) {
	def := newInstance()

	future1 := newMeta(11)
	future1.ValidBlock = 5

	future2 := newMeta(12)
	future2.ValidBlock = 2

	future3 := newMeta(13)
	future3.ValidBlock = 3

	future4 := newMeta(16)
	future4.ValidBlock = 1

	def.AddFuture(future1)
	def.AddFuture(future4)
	def.AddFuture(future3)
	def.AddFuture(future2)
	def.AddFuture(future3)

	def.printFuture()
	fmt.Println("=====")

	fmt.Println(def.CurrentValue())

	def.printFuture()
	fmt.Println("---")
	def.printHistory()
}

