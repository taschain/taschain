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
	"errors"
	"github.com/taschain/taschain/common"
	"github.com/taschain/taschain/taslog"
	"log"
	"math/rand"
	nnet "net"
	"strconv"
	"time"
)

const (
	BASE_PORT = 22000

	SUPER_BASE_PORT = 1122

	BASE_SECTION = "network"

	PRIVATE_KEY = "private_key"

	NodeIdLength = 66
)

type NodeID [NodeIdLength]byte

func (nid NodeID) IsValid() bool {
	for i := 0; i < NodeIdLength; i++ {
		if nid[i] > 0 {
			return true
		}
	}
	return false
}

func (nid NodeID) GetHexString() string {
	return string(nid[:])
}
func NewNodeID(hex string) NodeID {
	var nid NodeID
	nid.SetBytes([]byte(hex))
	return nid
}

func (nid *NodeID) SetBytes(b []byte) {
	if len(nid) < len(b) {
		b = b[:len(nid)]
	}
	copy(nid[:], b)
}

func (nid NodeID) Bytes() []byte {
	return nid[:]
}

// Node Kad 节点
type Node struct {
	Id      NodeID
	Ip      nnet.IP
	Port    int
	NatType int

	// kad

	sha     []byte
	addedAt time.Time
	fails   int
	pingAt  time.Time
	pinged  bool
}

// NewNode 新建节点
func NewNode(id NodeID, ip nnet.IP, port int) *Node {
	if ipv4 := ip.To4(); ipv4 != nil {
		ip = ipv4
	}
	return &Node{
		Ip:   ip,
		Port: port,
		Id:   id,
		sha:  makeSha256Hash(id[:]),
	}
}

func (n *Node) addr() *nnet.UDPAddr {
	return &nnet.UDPAddr{IP: n.Ip, Port: int(n.Port)}
}

func (n *Node) Incomplete() bool {
	return n.Ip == nil
}

func (n *Node) validateComplete() error {
	if n.Incomplete() {
		return errors.New("incomplete node")
	}
	if n.Port == 0 {
		return errors.New("missing port")
	}

	if n.Ip.IsMulticast() || n.Ip.IsUnspecified() {
		return errors.New("invalid IP (multicast/unspecified)")
	}
	return nil
}

func distanceCompare(target, a, b []byte) int {
	for i := range target {
		da := a[i] ^ target[i]
		db := b[i] ^ target[i]
		if da > db {
			return 1
		} else if da < db {
			return -1
		}
	}
	return 0
}

var leadingZeroCount = [256]int{
	8, 7, 6, 6, 5, 5, 5, 5,
	4, 4, 4, 4, 4, 4, 4, 4,
	3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	2, 2, 2, 2, 2, 2, 2, 2,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0,
}

func logDistance(a, b []byte) int {
	lz := 0
	for i := range a {
		x := a[i] ^ b[i]
		if x == 0 {
			lz += 8
		} else {
			lz += leadingZeroCount[x]
			break
		}
	}
	return len(a)*8 - lz
}

func hashAtDistance(a []byte, n int) (b []byte) {
	if n == 0 {
		return a
	}

	b = a
	pos := len(a) - n/8 - 1
	bit := byte(0x01) << (byte(n%8) - 1)
	if bit == 0 {
		pos++
		bit = 0x80
	}
	b[pos] = a[pos]&^bit | ^a[pos]&bit
	for i := pos + 1; i < len(a); i++ {
		b[i] = byte(rand.Intn(255))
	}
	return b
}

func InitSelfNode(config common.ConfManager, isSuper bool, id NodeID) (*Node, error) {
	Logger = taslog.GetLoggerByIndex(taslog.P2PLogConfig, common.GlobalConf.GetString("instance", "index", ""))
	ip := getLocalIp()
	basePort := BASE_PORT
	port := SUPER_BASE_PORT
	if !isSuper {
		basePort += 16
		port = getAvailablePort(ip, BASE_PORT)
	}

	n := Node{Id: id, Ip: nnet.ParseIP(ip), Port: port}
	common.DefaultLogger.Info(n.String())
	return &n, nil
}

//内网IP
func getLocalIp() string {
	addrs, err := nnet.InterfaceAddrs()

	if err != nil {
	}

	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*nnet.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func getAvailablePort(ip string, port int) int {
	if port < 1024 {
		port = BASE_PORT
	}

	if port > 65535 {
		log.Printf("[Network]No available port!")
		return -1
	}

	rand.Seed(time.Now().UnixNano())
	port += rand.Intn(1000)
	//listener, e := net.ListenPacket("udp", ip+":"+strconv.Itoa(port))
	//if e != nil {
	//	//listener.Close()
	//	port++
	//	return getAvailablePort(ip, port)
	//}
	//listener.Close()

	return port
}

func (s *Node) String() string {
	str := "Self node net info:\n" + "ID is:" + s.Id.GetHexString() + "\nIP is:" + s.Ip.String() + "\nTcp port is:" + strconv.Itoa(s.Port) + "\n"
	return str
}

func getPrivateKeyFromConfigFile(config common.ConfManager) (privateKeyStr string) {
	privateKey := config.GetString(BASE_SECTION, PRIVATE_KEY, "")
	return privateKey
}

// insert into config file
func savePrivateKey(privateKeyStr string, config common.ConfManager) {
	config.SetString(BASE_SECTION, PRIVATE_KEY, privateKeyStr)
}
