package common

import (
	"testing"
	"encoding/json"
	"log"
)

/*
**  Creator: pxf
**  Date: 2018/9/30 下午3:11
**  Description: 
*/

func TestHash_Hex(t *testing.T) {
	var h Hash
	h = HexToHash("0x1234")
	t.Log(h.Hex())
	
	s := "0xf3be4592802e6bfa85bf449c41eea1fc7a695220590c677c46d84339a13eec1a"
	h = HexToHash(s)
	t.Log(h.Hex())
}

func TestAddress_MarshalJSON(t *testing.T) {
	addr := HexToAddress("0x123")

	bs, _ := json.Marshal(&addr)
	log.Printf(string(bs))
}