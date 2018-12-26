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
	"common"
	"taslog"

	"middleware/statistics"
	nnet "net"
)

const (
	seedIdKey = "seed_id"

	seedIpKey = "seed_ip"

	seedPortKey = "seed_port"

	seedDefaultId = "0xe75051bf0048decaffa55e3a9fa33e87ed802aaba5038b0fd7f49401f5d8b019"

	seedDefaultIp = "47.105.51.161"

	seedDefaultPort = 1122
)

var net *server

var Logger taslog.Logger

func Init(config common.ConfManager, isSuper bool, chainHandler MsgHandler, consensusHandler MsgHandler, testMode bool, seedIp string, nodeIDHex string) (err error) {
	Logger = taslog.GetLoggerByIndex(taslog.P2PLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	statistics.InitStatistics(config)

	self, err := InitSelfNode(config, isSuper, NewNodeID(nodeIDHex))
	if err != nil {
		Logger.Errorf("InitSelfNode error:", err.Error())
		return err
	}
	//id = self.Id
	if seedIp == "" {
		seedIp = seedDefaultIp
	}
	seedId, _, seedPort := getSeedInfo(config)
	seeds := make([]*Node, 0, 16)

	bnNode := NewNode(NewNodeID(seedId), nnet.ParseIP(seedIp), seedPort)

	if bnNode.Id != self.Id && !isSuper {
		seeds = append(seeds, bnNode)
	}
	listenAddr := nnet.UDPAddr{IP: self.Ip, Port: self.Port}

	var natEnable bool
	if testMode {
		natEnable = false
		listenAddr = nnet.UDPAddr{IP: nnet.ParseIP(seedIp), Port: self.Port}
	} else {
		natEnable = true
	}
	netConfig := NetCoreConfig{Id: self.Id, ListenAddr: &listenAddr, Seeds: seeds, NatTraversalEnable: natEnable}

	var netcore NetCore
	n, _ := netcore.InitNetCore(netConfig)

	net = &server{Self: self, netCore: n, consensusHandler: consensusHandler, chainHandler: chainHandler}
	return nil
}

func GetNetInstance() Network {
	return net
}

func getSeedInfo(config common.ConfManager) (id string, ip string, port int) {
	id = config.GetString(BASE_SECTION, seedIdKey, seedDefaultId)
	ip = config.GetString(BASE_SECTION, seedIpKey, seedDefaultIp)
	port = config.GetInt(BASE_SECTION, seedPortKey, seedDefaultPort)

	return
}
