package p2p

import (
	"common"
	"net"
	"strconv"
	"taslog"
	"consensus/groupsig"
)

const (
	BASE_PORT = 1122

	BASE_SECTION = "network"

	PRIVATE_KEY = "private_key"
)

type Node struct {
	PrivateKey common.PrivateKey

	PublicKey common.PublicKey

	Id string

	Ip string

	TcpPort int
}

func InitSelfNode(config *common.ConfManager) (*Node, error) {
	var privateKey common.PrivateKey

	privateKeyStr := getPrivateKeyFromConfigFile(config)
	if privateKeyStr == "" {
		privateKey = common.GenerateKey("")
		savePrivateKey(privateKey.GetHexString(), config)
	} else {
		privateKey = *common.HexStringToSecKey(privateKeyStr)
	}
	publicKey := privateKey.GetPubKey()
	id := GetIdFromPublicKey(publicKey)
	ip := getLocalIp()
	port := getAvailableTCPPort(ip, BASE_PORT)

	n := Node{PrivateKey: privateKey, PublicKey: publicKey, Id: id, Ip: ip, TcpPort: port}
	taslog.P2pLogger.Debug(n.String())
	return &n, nil
}

//adpat to lib2p2. The whole p2p network use this id to be the only identity
func GetIdFromPublicKey(p common.PublicKey) string {
	//b := p.ToBytes()
	//idBytes, e := multihash.Sum(b, multihash.SHA2_256, -1)
	//if e != nil {
	//	taslog.P2pLogger.Error("GetIdFromPublicKey error!:%s", e.Error())
	//	return ""
	//}
	addr := p.GetAddress()
	i := groupsig.NewIDFromAddress(addr)
	id := i.GetHexString()
	return id
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

func getAvailableTCPPort(ip string, port int) int {
	if port < 1024 {
		port = BASE_PORT
	}

	if port > 65535 {
		taslog.P2pLogger.Error("No available port!\n")
		return -1
	}

	listener, e := net.Listen("tcp", ip+":"+strconv.Itoa(port))
	if e != nil {
		//listener.Close()
		port++
		return getAvailableTCPPort(ip, port)
	}
	listener.Close()
	return port
}

func (s *Node) String() string {
	str := "Self node net info:\nPrivate key is:" + s.PrivateKey.GetHexString() +
		"\nPublic key is:" + s.PublicKey.GetHexString() + "\nID is:" + s.Id + "\nIP is:" + s.Ip + "\nTcp port is:" + strconv.Itoa(s.TcpPort)+"\n"
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
	return ToMulAddrStr(s.Ip, "tcp", s.TcpPort)
}

//"/ip4/127.0.0.1/udp/1234"
func ToMulAddrStr(ip string, protocol string, port int) string {
	addr := "/ip4/" + ip + "/" + protocol + "/" + strconv.Itoa(port)
	return addr
}

//only for test
//used to mock a new client
func NewSelfNetInfo(privateKeyStr string) *Node {
	var privateKey common.PrivateKey

	if privateKeyStr == "" {
		privateKey = common.GenerateKey("")
	} else {
		privateKey = *common.HexStringToSecKey(privateKeyStr)
	}
	publicKey := privateKey.GetPubKey()
	id := GetIdFromPublicKey(publicKey)
	ip := getLocalIp()
	port := getAvailableTCPPort(ip, BASE_PORT)
	return &Node{PrivateKey: privateKey, PublicKey: publicKey, Id: id, Ip: ip, TcpPort: port}
}
