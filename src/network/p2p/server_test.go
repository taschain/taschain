package p2p

import (
	"taslog"
	"github.com/libp2p/go-libp2p-crypto"
	"common"
	"fmt"
	"testing"
	"context"
	gpeer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/libp2p/go-libp2p-kad-dht"
	"time"
	"github.com/libp2p/go-libp2p-swarm"
	"github.com/libp2p/go-libp2p-host"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	inet "github.com/libp2p/go-libp2p-net"

	"utility"
	"consensus/groupsig"
	"github.com/libp2p/go-libp2p/p2p/host/basic"
)

const (
	SEED_ID_KEY = "seed_id"

	SEED_ADDRESS_KEY = "seed_address"
)

func TestSendMessage(t *testing.T) {
	defer taslog.Close()

	groupsig.Init(1)
	crypto.KeyTypes = append(crypto.KeyTypes, 3)
	crypto.PubKeyUnmarshallers[3] = UnmarshalEcdsaPublicKey

	config := common.NewConfINIManager("server_test.ini")
	ctx := context.Background()

	seedPrivateKey := "0x0423c75e7593a7e6b5ce489f7d3578f8f737b6dd0fc1d2b10dc12a3e88a0572c62b801e14a8864ebe2d7b8c32e31113ccb511a6ad597c008ea90d850439133819f0b682fe8ff4a9023712e74256fb628c8e97658d99f2a8880a3066f120c2e899b"
	seedDht, seedHost, seedNet := mockDHT(seedPrivateKey, &config, ctx)
	fmt.Printf("Mock seed node success!\nseed :%s\n", seedNet.String())

	node1Dht, node1Host, node1Net := mockDHT("", &config, ctx)
	fmt.Printf("Mock  node1 success!\nnode1 :%s\n", node1Net.String())

	node2Dht, node2Host, node2Net := mockDHT("", &config, ctx)
	fmt.Printf("Mock  node2 success!\node2 :%s\n", node2Net.String())

	if node1Dht != nil && seedDht != nil {
		connectToSeed(ctx, node1Host, &config, *node1Net)
		connectToSeed(ctx, node2Host, &config, *node2Net)

		dhts := []*dht.IpfsDHT{node1Dht, seedDht,node2Dht}
		bootDhts(dhts)
		time.Sleep(30 * time.Second)

		//peerInfo1, err1 := node1.FindPeer(ctx1, ConvertToPeerID(seedId))
		//if err1 != nil {
		//	fmt.Printf("node find seed error:%s\n", err1.Error())
		//}
		//fmt.Printf("node find seed result is:%s\n", ConvertToID(peerInfo1.ID))
		//
		//
		//peerInfo, err := seedDht.FindPeer(ctx1, ConvertToPeerID(node1Id))
		//if err != nil {
		//	fmt.Printf("seed find node1 error:%s\n", err.Error())
		//}
		//fmt.Printf("seed find node1 result is:%s\n", ConvertToID(peerInfo.ID))
	}
	node2Server := server{node2Net, node2Host, node2Dht}

	seedServer := server{seedNet, seedHost, seedDht}

	node1Host.Network().SetStreamHandler(testSteamHandler)
	node1Server := server{node1Net, node1Host, node1Dht}

	messsage := mockMessage()
	seedServer.SendMessage(messsage, node1Net.Id)

	time.Sleep(1 * time.Second)
	conns := seedServer.GetConnInfo()
	for _, conn := range conns {
		fmt.Printf("seed server's conn:%s,%s,%s\n", conn.Id, conn.Ip, conn.TcpPort)
	}

	conn1 := node1Server.GetConnInfo()
	for _, conn := range conn1 {
		fmt.Printf("node1 server's conn:%s,%s,%s\n", conn.Id, conn.Ip, conn.TcpPort)
	}

	conn2 := node2Server.GetConnInfo()
	for _, conn := range conn2 {
		fmt.Printf("node2 server's conn:%s,%s,%s\n", conn.Id, conn.Ip, conn.TcpPort)
	}
}

func testSteamHandler(stream inet.Stream) {
	defer stream.Close()

	pkgLengthBytes := make([]byte, 4)
	n, err := stream.Read(pkgLengthBytes)
	if n != 4 || err != nil {
		fmt.Printf("Stream  read %d byte error:%s,received %d bytes\n", 4, err.Error(), n)
		return
	}
	//fmt.Printf("pkgLengthBytes length:%d\n",len(pkgLengthBytes))
	//for i:=0;i<len(pkgLengthBytes);i++{
	//	fmt.Printf("b:%d",int(pkgLengthBytes[i]))
	//}
	//fmt.Printf("\n")

	pkgLength := int(utility.ByteToUInt32(pkgLengthBytes))
	pkgBodyBytes := make([]byte, pkgLength)
	if pkgLength < PACKAGE_MAX_SIZE {
		n1, err1 := stream.Read(pkgBodyBytes)
		if n1 != pkgLength || err1 != nil {
			fmt.Printf("Stream  read %d byte error:%s,received %d bytes\n", pkgLength, err.Error(), n)
			return
		}
	} else {
		c := pkgLength / PACKAGE_MAX_SIZE
		left, right := 0, PACKAGE_MAX_SIZE
		for i := 0; i <= c; i++ {
			a := make([]byte, PACKAGE_MAX_SIZE)
			n1, err1 := stream.Read(a)
			if n1 != PACKAGE_MAX_SIZE || err1 != nil {
				logger.Errorf("Stream  read %d byte error:%s,received %d bytes\n", PACKAGE_MAX_SIZE, err.Error(), n1)
				return
			}
			copy(pkgBodyBytes[left:right], a)
			left += PACKAGE_MAX_SIZE
			right += PACKAGE_MAX_SIZE
			if right > pkgLength {
				right = pkgLength
			}
		}
	}
	//fmt.Printf("Before unmarshla message!,pkgLength：%d,length:%d\n",pkgLength,len(pkgBodyBytes))

	//for i:=0;i<len(pkgBodyBytes);i++{
	//	fmt.Printf("b:%d",int(pkgBodyBytes[i]))
	//}
	//fmt.Printf("\n")

	message, e := UnMarshalMessage(pkgBodyBytes)
	if e != nil {
		fmt.Printf("Unmarshal message error!" + e.Error())
		return
	}
	m := mockMessage()
	fmt.Printf("Reviced message compare result is:%t\n", messageEquals(*message, m))
}

func messageEquals(m1 Message, m2 Message) bool {

	if m1.Code != m2.Code || len(m1.Sign) != len(m2.Sign) || len(m1.Body) != len(m2.Body) {
		return false
	}
	for i := 0; i < len(m1.Sign); i++ {
		b1 := m1.Sign[i]
		b2 := m2.Sign[i]
		if b1 != b2 {
			return false
		}
	}

	for i := 0; i < len(m1.Body); i++ {
		b1 := m1.Body[i]
		b2 := m2.Body[i]
		if b1 != b2 {
			return false
		}
	}
	return true
}
func mockMessage() Message {
	code := GROUP_INIT_MSG
	sign := []byte{1, 2, 3, 4, 5, 6, 7}
	body := []byte{1, 1, 2, 3, 5, 8, 13}
	m := Message{Code: code, Sign: sign, Body: body}
	return m
}

func bootDhts(dhts []*dht.IpfsDHT) {
	for i := 0; i < len(dhts); i++ {
		d := dhts[i]
		cfg := dht.DefaultBootstrapConfig
		cfg.Queries = 3
		cfg.Period = time.Duration(20 * time.Second)

		process, e8 := d.BootstrapWithConfig(cfg)
		if e8 != nil {
			process.Close()
			fmt.Print("KadDht bootstrap error! " + e8.Error())
			return
		}
	}
}
func mockDHT(privateKey string, config *common.ConfManager, ctx context.Context) (*dht.IpfsDHT, host.Host, *Node) {
	self := NewSelfNetInfo(privateKey)

	localId := self.Id
	ID := ConvertToPeerID(localId)
	multiaddr, e2 := ma.NewMultiaddr(self.GenMulAddrStr())
	if e2 != nil {
		fmt.Printf("new mlltiaddr error!" + e2.Error())
		return nil, nil, self
	}
	listenAddrs := []ma.Multiaddr{multiaddr}
	peerStore := pstore.NewPeerstore()
	p1 := &Pubkey{PublicKey: self.PublicKey}
	p2 := &Privkey{PrivateKey: self.PrivateKey}

	peerStore.AddPubKey(ID, p1)
	peerStore.AddPrivKey(ID, p2)
	peerStore.AddAddrs(ID, listenAddrs, pstore.PermanentAddrTTL)
	//bwc  is a bandwidth metrics collector, This is used to track incoming and outgoing bandwidth on connections managed by this swarm.
	// It is optional, and passing nil will simply result in no metrics for connections being available.
	sw, e3 := swarm.NewNetwork(ctx, listenAddrs, ID, peerStore, nil)
	if e3 != nil {
		fmt.Printf("New swarm error!\n" + e3.Error())
		return nil, nil, self
	}

	//hostOpts := &basichost.HostOpts{}
	host := basichost.New(sw)
	//host := blankhost.NewBlankHost(sw)
	//if e4 != nil {
	//	fmt.Printf("New host error! " + e4.Error())
	//	return nil, self.Id
	//}

	dss := dssync.MutexWrap(ds.NewMapDatastore())
	kadDht := dht.NewDHT(ctx, host, dss)
	return kadDht, host, self
}

func connectToSeed(ctx context.Context, host host.Host, config *common.ConfManager, node Node) error {
	seedId, seedAddrStr, e1 := getSeedInfo(config)
	if e1 != nil {
		return e1
	}
	if node.GenMulAddrStr() == seedAddrStr {
		return nil
	}
	seedMultiaddr, e2 := ma.NewMultiaddr(seedAddrStr)
	if e2 != nil {
		logger.Error("SeedIdStr to seedMultiaddr error!\n" + e2.Error())
		return e2
	}
	seedPeerInfo := pstore.PeerInfo{ID: seedId, Addrs: []ma.Multiaddr{seedMultiaddr}}
	e3 := host.Connect(ctx, seedPeerInfo)
	if e3 != nil {
		logger.Error("Host connect to seed error!\n" + e3.Error())
		return e3
	}
	return nil
}

func getSeedInfo(config *common.ConfManager) (gpeer.ID, string, error) {
	seedIdStr := (*config).GetString(BASE_SECTION, SEED_ID_KEY, "QmPf7ArTTxDqd1znC9LF5r73YR85sbEU1t1SzTvt2fRry2")
	seedAddrStr := (*config).GetString(BASE_SECTION, SEED_ADDRESS_KEY, "/ip4/10.0.0.66/tcp/1122")
	return ConvertToPeerID(seedIdStr), seedAddrStr, nil
}
