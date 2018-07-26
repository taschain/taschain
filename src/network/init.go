package network

import (
	"taslog"
	"common"

	"net"
)

const (
	SEED_ID_KEY = "seed_id"

	SEED_IP_KEY = "seed_ip"

	SEED_PORT_KEY = "seed_port"

	SEED_DEFAULT_ID = "0xa1cbfb3f2d4690016269a655df22f62a1b90a39b"

	SEED_DEFAULT_IP = "47.106.39.118"

	SEED_DEFAULT_PORT = 1122
)

var netInstance *network

var Logger taslog.Logger

func Init(config common.ConfManager, isSuper bool, chainHandler MsgHandler, consensusHandler MsgHandler, testMode bool,seedIp string)(id string,err error){
	Logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("instance", "index", ""))

	self, err := InitSelfNode(config, isSuper)
	if err != nil {
		Logger.Errorf("[Network]InitSelfNode error:", err.Error())
		return "",err
	}
	id = self.Id.GetHexString()
	if seedIp == ""{
		seedIp = SEED_DEFAULT_IP
	}
	seedId, _, seedPort := getSeedInfo(config)
	seeds := make([]*Node, 0, 16)
	bnNode := NewNode(common.HexStringToAddress(seedId), net.ParseIP(seedIp), seedPort)
	if bnNode.Id != self.Id && !isSuper {
		seeds = append(seeds, bnNode)
	}

	var natEnable bool
	if testMode {
		natEnable = false
	} else {
		natEnable = true
	}
	netConfig := Config{PrivateKey: &self.PrivateKey, Id: self.Id, ListenAddr: &net.UDPAddr{IP: self.Ip, Port: self.Port}, Bootnodes: seeds, NatTraversalEnable: natEnable}

	var netcore NetCore
	n, _ := netcore.InitNetCore(netConfig)

	netInstance = &network{Self: self, netCore: n, consensusHandler: consensusHandler, chainHandler: chainHandler}
	return
}



func GetNetInstance()Server{
	return netInstance
}

func getSeedInfo(config common.ConfManager) (id string, ip string, port int) {
	id = config.GetString(BASE_SECTION, SEED_ID_KEY, SEED_DEFAULT_ID)
	ip = config.GetString(BASE_SECTION, SEED_IP_KEY, SEED_DEFAULT_IP)
	port = config.GetInt(BASE_SECTION, SEED_PORT_KEY, SEED_DEFAULT_PORT)

	return
}
