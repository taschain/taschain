package common

import "testing"

/*
**  Creator: pxf
**  Date: 2019/1/8 下午3:50
**  Description:
 */

func TestParseCoin(t *testing.T) {
	t.Log(ParseCoin("232a"))
	t.Log(ParseCoin("232ra"))
	t.Log(ParseCoin("232krara"))
	t.Log(ParseCoin("232mra"))
	t.Log(ParseCoin("2tas1"))
}
