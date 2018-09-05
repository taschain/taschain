//   Copyright (C) 2018 TASChain
//
//   This program is free software: you can redistribute it and/or modify
//   it under the terms of the GNU General Public License as published by
//   the Free Software Foundation, either version 3 of the License, or
//   (at your option) any later version.
//
//   This program is distributed in the hope that it will be useful,
//   but WITHOUT ANY WARRANTY; without even the implied warranty of
//   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//   GNU General Public License for more details.
//
//   You should have received a copy of the GNU General Public License
//   along with this program.  If not, see <https://www.gnu.org/licenses/>.

package network

import (
	"taslog"
	"common"

	nnet "net"
	"middleware/statistics"
)

const (
	seedIdKey = "seed_id"

	seedIpKey = "seed_ip"

	seedPortKey = "seed_port"

	seedDefaultId = "0xa1cbfb3f2d4690016269a655df22f62a1b90a39b"

	seedDefaultIp = "47.105.70.31"

	seedDefaultPort = 1122
)

var net *server

var Logger taslog.Logger

func Init(config common.ConfManager, isSuper bool, chainHandler MsgHandler, consensusHandler MsgHandler, testMode bool,seedIp string)(id string,err error){
	Logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("instance", "index", ""))
	statistics.InitStatistics(config)
	self, err := InitSelfNode(config, isSuper)
	if err != nil {
		Logger.Errorf("[Network]InitSelfNode error:", err.Error())
		return "",err
	}
	id = self.Id.GetHexString()
	if seedIp == ""{
		seedIp = seedDefaultIp
	}
	seedId, _, seedPort := getSeedInfo(config)
	seeds := make([]*Node, 0, 16)

	bnNode := newNode(common.HexStringToAddress(seedId), nnet.ParseIP(seedIp), seedPort)

	if bnNode.Id != self.Id && !isSuper {
		seeds = append(seeds, bnNode)
	}
	listenAddr := nnet.UDPAddr{IP: self.Ip, Port: self.Port}

	var natEnable bool
	if testMode {
		natEnable = false
		listenAddr =  nnet.UDPAddr{IP:nnet.ParseIP(seedIp), Port: self.Port}
	} else {
		natEnable = true
	}
	netConfig := NetCoreConfig{ Id: self.Id, ListenAddr:&listenAddr , Seeds: seeds, NatTraversalEnable: natEnable}

	var netcore NetCore
	n, _ := netcore.InitNetCore(netConfig)

	net = &server{Self: self, netCore: n, consensusHandler: consensusHandler, chainHandler: chainHandler}
	return
}



func GetNetInstance()Network{
	return net
}

func getSeedInfo(config common.ConfManager) (id string, ip string, port int) {
	id = config.GetString(BASE_SECTION, seedIdKey, seedDefaultId)
	ip = config.GetString(BASE_SECTION, seedIpKey, seedDefaultIp)
	port = config.GetInt(BASE_SECTION, seedPortKey, seedDefaultPort)

	return
}
