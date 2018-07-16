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
	NodeOnline([]byte(node.Id), node.PublicKey.ToBytes())
}

func TestGetAllNodeIds(t *testing.T) {
	common.InitConf("/Users/Kaede/TasProject/tas1.ini")
	fmt.Println(GetAllNodeIds())
}

func TestGetNodeByIds(t *testing.T) {
	common.InitConf("/Users/Kaede/TasProject/tas1.ini")
	nodes,_ := GetAllNodeIds()
	pubKey,_ := GetPubKeyByIds(nodes)
	fmt.Println(pubKey)
}

func TestGetNodeById(t *testing.T) {
	common.InitConf("/Users/Kaede/TasProject/tas1.ini")
	nodes,_ := GetAllNodeIds()
	for _,id := range nodes{
		pubKey,_ := GetPubKeyById(id)
		fmt.Println(pubKey)
	}
}

func TestNodeOffline(t *testing.T) {
	common.InitConf("/Users/Kaede/TasProject/tas1.ini")
	nodes,_ := GetAllNodeIds()
	for _,id := range nodes{
		NodeOffline(id)
	}
}