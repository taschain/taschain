package p2p

import (
	inet "github.com/libp2p/go-libp2p-net"

	"utility"
	"github.com/libp2p/go-libp2p-host"
	"context"
	"github.com/libp2p/go-libp2p-peer"
	"network/biz"
	"github.com/libp2p/go-libp2p-kad-dht"
)

const (
	PACKAGE_MAX_SIZE = 1024 * 1024

	PACKAGE_LENGTH_SIZE = 4

	CODE_SIZE = 4
)

var Server server

type server struct {
	host host.Host

	dht *dht.IpfsDHT

	bHandler biz.BlockChainMessageHandler

	cHandler biz.ConsensusMessageHandler
}

func InitServer(host host.Host, dht *dht.IpfsDHT) {
	bHandler := biz.NewBlockChainMessageHandler(nil, nil, nil, nil, nil, nil,
		nil, nil, nil, nil, nil)

	cHandler := biz.NewConsensusMessageHandler(nil, nil, nil, nil,
		nil, nil, nil)

	host.Network().SetStreamHandler(swarmStreamHandler)

	Server = server{host: host, dht: dht, bHandler: bHandler, cHandler: cHandler}
}

func (s *server) SendMessage(m Message, id string) {
	b1, e := MarshalMessage(m)
	if e != nil {
		return
	}
	length := len(b1)
	b2 := utility.UInt32ToByte(uint32(length))

	b := make([]byte, len(b1)+len(b2))
	copy(b, b1)
	copy(b[len(b1):], b2)

	s.send(b, id)

}

func (s *server) send(b []byte, id string) {
	stream, e := s.host.Network().NewStream(context.Background(), peer.ID(id))
	if e != nil {
		return
	}
	l := len(b)
	if l < PACKAGE_MAX_SIZE {
		r, err := stream.Write(b)
		if r != l || err != nil {

		}
	} else {
		n := l / PACKAGE_MAX_SIZE
		left, right := 0, PACKAGE_MAX_SIZE
		for i := 0; i <= n; i++ {
			a := make([]byte, PACKAGE_MAX_SIZE)
			copy(a, b[left:right])
			r, err := stream.Write(a)
			if r != PACKAGE_MAX_SIZE || err != nil {

			}
			left += PACKAGE_MAX_SIZE
			right += PACKAGE_MAX_SIZE
			if right > l {
				right = l
			}
		}
	}
}

//TODO 考虑读写超时
func swarmStreamHandler(stream inet.Stream) {
	pkgLengthBytes := make([]byte, PACKAGE_LENGTH_SIZE)
	n, err := stream.Read(pkgLengthBytes)
	if n != 4 || err != nil {

	}
	pkgLength := utility.ByteToInt(pkgLengthBytes)
	pkgBodyBytes := make([]byte, pkgLength)
	if pkgLength < PACKAGE_MAX_SIZE {
		n1, err1 := stream.Read(pkgBodyBytes)
		if n1 != pkgLength || err1 != nil {

		}
	} else {
		c := pkgLength / PACKAGE_MAX_SIZE
		left, right := 0, PACKAGE_MAX_SIZE
		for i := 0; i <= c; i++ {
			a := make([]byte, PACKAGE_MAX_SIZE)
			n1, err1 := stream.Read(a)
			if n1 != PACKAGE_MAX_SIZE || err1 != nil {

			}
			copy(pkgBodyBytes[left:right], a)
			left += PACKAGE_MAX_SIZE
			right += PACKAGE_MAX_SIZE
			if right > pkgLength {
				right = pkgLength
			}
		}
	}
	handleMessage(pkgBodyBytes, (string)(stream.Conn().RemotePeer()))
}

func handleMessage(pkgBodyBytes []byte, from string) {
	//code 都没有
	if len(pkgBodyBytes) < 4 {

		return
	}
	codeBytes := make([]byte, CODE_SIZE)
	copy(codeBytes, pkgBodyBytes[:3])
	code := utility.ByteToInt(codeBytes)
	switch code {
	case GROUP_INIT_MSG:
		//todo
	default:
		//not support message
	}
}

type ConnInfo struct {
	Id      string
	Ip      string
	TcpPort int
}

func GetConnInfo() []ConnInfo {
	return nil
}
