package p2p

import (
	"common"
	"net"
	"strconv"
	"taslog"
	"consensus/groupsig"
	"consensus/logical"
	"core"
	"github.com/multiformats/go-multihash"
	"github.com/libp2p/go-libp2p-peer"
)

const (
	BASE_PORT = 1122

	BASE_SECTION = "network"

	PRIVATE_KEY = "private_key"
)

type selfNode struct {
	PrivateKey common.PrivateKey

	PublicKey common.PublicKey

	Id string

	Ip string

	TcpPort int
}

var SelfNetInfo selfNode

func InitSelfNode(config *common.ConfManager) error {
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
	//该转换方式暂时不使用
	//id := publicKey.GetAddress().GetHexString()
	ip := getLocalIp()
	port := getAvailableTCPPort(ip, BASE_PORT)
	SelfNetInfo = selfNode{PrivateKey: privateKey, PublicKey: publicKey, Id: id, Ip: ip, TcpPort: port}
	taslog.P2pLogger.Debug(SelfNetInfo.String())
	return nil
}

//adpat to lib2p2. The whole p2p network use this id to be the only identity
func GetIdFromPublicKey(p common.PublicKey) string {
	b := p.ToBytes()
	idBytes, e := multihash.Sum(b, multihash.SHA2_256, -1)
	if e != nil {
		taslog.P2pLogger.Error("GetIdFromPublicKey error!:%s", e.Error())
		return ""
	}
	id := string(peer.ID(idBytes))
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

func (s *selfNode) String() string {
	str := "Self node net info:\n Private key is:" + s.PrivateKey.GetHexString() +
		"\nPublic key is:" + s.PublicKey.GetHexString() + "\nID is:" + s.Id + "\nIP is:" + s.Ip + "\n Tcp port is:" + strconv.Itoa(s.TcpPort)
	return str
}

func getPrivateKeyFromConfigFile(config *common.ConfManager) (privateKeyStr string) {
	privateKey, b := (*config).GetString(BASE_SECTION, PRIVATE_KEY)
	if !b {
		taslog.P2pLogger.Error("Get private key from config file error!\n")
		return ""
	}
	return privateKey
}

// insert into config file
func savePrivateKey(privateKeyStr string, config *common.ConfManager) {
	(*config).SetString(BASE_SECTION, PRIVATE_KEY, privateKeyStr)
}

func (s selfNode) GenMulAddrStr() string {
	return ToMulAddrStr(s.Ip, "tcp", s.TcpPort)
}

//"/ip4/127.0.0.1/udp/1234"
func ToMulAddrStr(ip string, protocol string, port int) string {
	addr := "/ip4/" + ip + "/" + protocol + "/" + strconv.Itoa(port)
	return addr
}

//only for test
//used to mock a new client
func NewSelfNetInfo(privateKeyStr string) *selfNode {
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
	return &selfNode{PrivateKey: privateKey, PublicKey: publicKey, Id: id, Ip: ip, TcpPort: port}
}

//----------------------------------------------------组初始化-----------------------------------------------------------
//广播 组初始化消息  组内广播
// param： id slice
//         signData
func sendGroupInitMessage(is []groupsig.ID, sd logical.SignData) {

}

//组内广播密钥   for each定向发送 组内广播
//param：密钥片段map
//       signData
func sendKeyPiece(km map[groupsig.ID]groupsig.Pubkey, sd logical.SignData) {

}

//广播组用户公钥  组内广播
//param:组用户公钥 memberPubkey
// signData
func sendMemberPubkey(mk groupsig.Pubkey, sd logical.SignData) {

}

//组初始化完成 广播组信息 全网广播
//参数: groupInfo senderId
func broadcastGroupInfo(gi logical.StaticGroupInfo, sd logical.SignData) {
	//上链 本地写数据库
}

//-----------------------------------------------------------------组铸币----------------------------------------------
//组内成员发现自己所在组成为铸币组 发消息通知全组 组内广播
//param: 组信息
//      SignData
func sendCurrentGroupCast(gi logical.StaticGroupInfo, sd logical.SignData) {

}

//铸币节点完成铸币，将blockheader  签名后发送至组内其他节点进行验证。组内广播
//param BlockHeader 组信息 signData
func sendCastVerify(bh core.BlockHeader, gi logical.StaticGroupInfo, sd logical.SignData) {}

//组内节点  验证通过后 自身签名 广播验证块 组内广播  验证不通过 保持静默
//param :BlockHeader
//       member signature
//       signData
func sendVerifiedCast(bh core.BlockHeader, gi logical.StaticGroupInfo, sd logical.SignData) {}

//验证节点 交易集缺失，索要、特定交易 全网广播
//param:hash slice of transaction slice
//      signData
func requestTransactionByHash(hs []common.Hash, sd logical.SignData) {}

//对外广播经过组签名的block 全网广播
//param: block
//       member signature
//       signData
func broadcastNewBlock(b core.Block, sd logical.SignData) {}

/////////////////////////////////////////////////////链同步/////////////////////////////////////////////////////////////
//广播索要链高度
//param: signData
func requestBlockChainHeight(sd logical.SignData) {}

//向某一节点请求Block
//param: target peer id
//       block height slice
//       sign data
func requestBlockByHeight(id groupsig.ID, hs []int, sd logical.SignData) {}

////////////////////////////////////////////////////////组同步//////////////////////////////////////////////////////////
//广播索要组链高度
//param: signData
func requestGroupChainHeight(sd logical.SignData) {}

//向某一节点请求GroupInfo
//param: target peer id
//       group height slice
//       sign data
func requestGroupInfoByHeight(id groupsig.ID, gs []int, sd logical.SignData) {}
