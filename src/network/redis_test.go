package network

import (
	"common"
	"network/p2p"
	"testing"
	"fmt"
)


func TestNodeOnline(t *testing.T)  {
	common.InitConf("/Users/Kaede/TasProject/tas1.ini")
	node, _ := p2p.InitSelfNode(&common.GlobalConf)
	NodeOnline(node)
}

func TestGetAllNodeIds(t *testing.T) {
	common.InitConf("/Users/Kaede/TasProject/tas1.ini")
	fmt.Println(GetAllNodeIds())
}

func TestGetNodeById(t *testing.T) {
	common.InitConf("/Users/Kaede/TasProject/tas1.ini")
	nodes,_ := GetAllNodeIds()
	for _,id := range nodes{
		pubKey,_ := GetPubKeyById(string(id))
		fmt.Println(pubKey)
	}
}

func TestNodeOffline(t *testing.T) {
	common.InitConf("/Users/Kaede/TasProject/tas1.ini")
	nodes,_ := GetAllNodeIds()
	for _,id := range nodes{
		NodeOffline(string(id))
	}
}