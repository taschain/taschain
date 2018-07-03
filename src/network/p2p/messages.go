package p2p

import (
	"net"
)

type packet interface {
	handle(nc *NetCore, from *net.UDPAddr, fromID NodeID, mac []byte) error
	name() string
}
