package p2p

import (
	"net"
	"common"
)

type packet interface {
	handle(nc *NetCore, from *net.UDPAddr, fromID common.Address, mac []byte) error
	name() string
}
