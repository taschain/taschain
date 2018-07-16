package network

import (
	"taslog"
	"common"

	"net"
	"strings"
	"strconv"
	"errors"
)

const (
	SEED_ID_KEY = "seed_id"

	SEED_ADDRESS_KEY = "seed_address"
)

var Network network

var Logger taslog.Logger


func Init(config common.ConfManager, isSuper bool, chainHandler MsgHandler, consensusHandler MsgHandler) error {
	Logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))

	self,err:= InitSelfNode(config, isSuper)
	if err != nil {
		Logger.Errorf("[Network]InitSelfNode error:",err.Error())
		return err
	}

	seedId, seedIp,seedPort, err := getSeedInfo(config)
	seeds := make([]*Node, 0, 16)
	if err == nil {
		bnNode := NewNode(common.HexStringToAddress(seedId), net.ParseIP(seedIp), seedPort)
		if bnNode.ID != self.ID && !isSuper {
			seeds = append(seeds, bnNode)
		}
	}
	netConfig := Config{PrivateKey: &self.PrivateKey, ID: self.ID, ListenAddr: &net.UDPAddr{IP: self.IP, Port: self.Port}, Bootnodes: seeds}

	var netcore NetCore
	n,_ := netcore.InitNetCore(netConfig)

	Network = network{Self:self,netCore:n,consensusHandler:consensusHandler,chainHandler:chainHandler}
	return nil
}



func getSeedInfo(config common.ConfManager) (id string, ip string, port int, e error) {
	id = config.GetString(BASE_SECTION, SEED_ID_KEY, "0xa1cbfb3f2d4690016269a655df22f62a1b90a39b")
	seedAddr := config.GetString(BASE_SECTION, SEED_ADDRESS_KEY, "/ip4/10.0.0.193/tcp/1122")

	ip, port, e = getIPPortFromAddr(seedAddr)
	return
}

func getIPPortFromAddr(addr string) (string, int, error) {
	//addr /ip4/127.0.0.1/udp/1234"
	split := strings.Split(addr, "/")
	if len(split) != 5 {
		return "", 0, errors.New("wrong addr")
	}
	ip := split[2]
	port, _ := strconv.Atoi(split[4])
	return ip, port, nil
}
