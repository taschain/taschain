package network

import (
	"taslog"
	//ma "github.com/multiformats/go-multiaddr"
	//pstore "github.com/libp2p/go-libp2p-peerstore"
	//"github.com/libp2p/go-libp2p-kad-dht"
	//"context"
	//"github.com/libp2p/go-libp2p-peer"
	"network/p2p"
	"common"

	"net"
)

const (
	SEED_ID_KEY = "seed_id"

	SEED_ADDRESS_KEY = "seed_address"
)

var Logger taslog.Logger

func InitNetwork(config *common.ConfManager, isSuper bool) error {

	Logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))

	node, e1 := makeSelfNode(config, isSuper)
	if e1 != nil {
		return e1
	}

	e2 := initServer(config, *node, isSuper)
	if e2 != nil {
		return e2
	}
	return nil
}

func initServer(config *common.ConfManager, node p2p.Node, isSuper bool) error {

	seedId, seedAddr, _ := getSeedInfo(config)
	seedIp, seedPort, err := p2p.GetIPPortFromAddr(seedAddr)
	seeds := make([]*p2p.Node, 0, 16)
	if err == nil {

		bnNode := p2p.NewNode(common.StringToAddress(seedId), net.ParseIP(seedIp), seedPort)
		if bnNode.ID != node.ID {
			seeds = append(seeds, bnNode)
		}
	}
	p2p.InitServer(seeds, &node)
	return nil
}

func makeSelfNode(config *common.ConfManager, isSuper bool) (*p2p.Node, error) {
	node, error := p2p.InitSelfNode(config, isSuper)
	if error != nil {
		Logger.Error("[Network]InitSelfNode error!\n" + error.Error())
		return nil, error
	}
	return node, nil
}

func getSeedInfo(config *common.ConfManager) (string, string, error) {
	seedIdStr := (*config).GetString(p2p.BASE_SECTION, SEED_ID_KEY, "Qmdeh5r5kT2je77JNYKTsQi6ncckpLa9aFnr6xYQaGAxaw")
	seedAddrStr := (*config).GetString(p2p.BASE_SECTION, SEED_ADDRESS_KEY, "/ip4/10.0.0.193/tcp/1122")
	return seedIdStr, seedAddrStr, nil
}
