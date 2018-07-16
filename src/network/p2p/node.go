package p2p

import (
	"common"
	"net"
	"strconv"
	"log"
	"taslog"
	"math/rand"
	"time"
	"fmt"
	"errors"
)

const (
	BASE_PORT = 22000
	SUPER_BASE_PORT = 1122

	BASE_SECTION = "network"

	PRIVATE_KEY = "private_key"
)

type NodeID =  common.Address

// Node Kad 节点
type Node struct {

	PrivateKey common.PrivateKey

	PublicKey common.PublicKey
	ID      NodeID
	IP     	net.IP
	Port    int
	NatType int


	// kad

	sha     []byte
	addedAt time.Time
	fails  int
	bondAt time.Time
	bonded bool

}


// NewNode 新建节点
func NewNode(id NodeID, ip net.IP, Port int) *Node {
	if ipv4 := ip.To4(); ipv4 != nil {
		ip = ipv4
	}
	return &Node{
		IP:   ip,
		Port: Port,
		ID:   id,
		sha:  SHA256Hash(id[:]),
	}
}

func (n *Node) addr() *net.UDPAddr {
	return &net.UDPAddr{IP: n.IP, Port: int(n.Port)}
}

// Incomplete returns true for nodes with no IP address.
func (n *Node) Incomplete() bool {
	return n.IP == nil
}

// checks whether n is a valid complete node.
func (n *Node) validateComplete() error {
	if n.Incomplete() {
		return errors.New("incomplete node")
	}
	if n.Port == 0 {
		return errors.New("missing port")
	}

	if n.IP.IsMulticast() || n.IP.IsUnspecified() {
		return errors.New("invalid IP (multicast/unspecified)")
	}
	return nil
}

// BytesID converts a byte slice to a NodeID
func BytesID(b []byte) (NodeID, error) {
	var id NodeID
	if len(b) != len(id) {
		return id, fmt.Errorf("wrong length, want %d bytes", len(id))
	}
	copy(id[:], b)
	return id, nil
}

// MustBytesID converts a byte slice to a NodeID.
// It panics if the byte slice is not a valid NodeID.
func MustBytesID(b []byte) NodeID {
	id, err := BytesID(b)
	if err != nil {
		panic(err)
	}
	return id
}

// distcmp compares the distances a->target and b->target.
// Returns -1 if a is closer to target, 1 if b is closer to target
// and 0 if they are equal.
func distcmp(target, a, b []byte) int {
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

// table of leading zero counts for bytes [0..255]
var lzcount = [256]int{
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

// logdist returns the logarithmic distance between a and b, log2(a ^ b).
func logdist(a, b []byte) int {
	lz := 0
	for i := range a {
		x := a[i] ^ b[i]
		if x == 0 {
			lz += 8
		} else {
			lz += lzcount[x]
			break
		}
	}
	return len(a)*8 - lz
}

// hashAtDistance 返回一个距离相同的随机哈希 logdist(a, b) == n
func hashAtDistance(a []byte, n int) (b []byte) {
	if n == 0 {
		return a
	}
	// flip bit at position n, fill the rest with random bits
	b = a
	pos := len(a) - n/8 - 1
	bit := byte(0x01) << (byte(n%8) - 1)
	if bit == 0 {
		pos++
		bit = 0x80
	}
	b[pos] = a[pos]&^bit | ^a[pos]&bit // TODO: randomize end bits
	for i := pos + 1; i < len(a); i++ {
		b[i] = byte(rand.Intn(255))
	}
	return b
}

func InitSelfNode(config *common.ConfManager, isSuper bool) (*Node, error) {
	logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))
	var privateKey common.PrivateKey

	privateKeyStr := getPrivateKeyFromConfigFile(config)
	if privateKeyStr == "" {
		privateKey = common.GenerateKey("")
		savePrivateKey(privateKey.GetHexString(), config)
	} else {
		privateKey = *common.HexStringToSecKey(privateKeyStr)
	}
	publicKey := privateKey.GetPubKey()
	id := publicKey.GetAddress()
	ip := getLocalIp()
	basePort := BASE_PORT
	port := SUPER_BASE_PORT;
	if !isSuper {
		basePort += 16
		port = getAvailablePort(ip, BASE_PORT)
	}


	n := Node{PrivateKey: privateKey, PublicKey: publicKey, ID: NodeID(id), IP: net.ParseIP(ip), Port: port}
	fmt.Print(n.String())
	return &n, nil
}


//内网IP
func getLocalIp() string {
	addrs, err := net.InterfaceAddrs()

	if err != nil {
	}

	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
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
	str := "Self node net info:\nPrivate key is:" + s.PrivateKey.GetHexString() +
		"\nPublic key is:" + s.PublicKey.GetHexString() + "\nID is:" + s.ID.GetHexString() + "\nIP is:" + s.IP.String() + "\nTcp port is:" + strconv.Itoa(s.Port)+"\n"
	return str
}

func getPrivateKeyFromConfigFile(config *common.ConfManager) (privateKeyStr string) {
	privateKey := (*config).GetString(BASE_SECTION, PRIVATE_KEY, "")
	return privateKey
}

// insert into config file
func savePrivateKey(privateKeyStr string, config *common.ConfManager) {
	(*config).SetString(BASE_SECTION, PRIVATE_KEY, privateKeyStr)
}

func (s Node) GenMulAddrStr() string {
	return ToMulAddrStr(s.IP.String(), "tcp", s.Port)
}

//"/ip4/127.0.0.1/udp/1234"
func ToMulAddrStr(ip string, protocol string, port int) string {
	addr := "/ip4/" + ip + "/" + protocol + "/" + strconv.Itoa(port)
	return addr
}

