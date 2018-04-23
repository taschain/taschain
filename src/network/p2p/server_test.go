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
	"github.com/libp2p/go-libp2p-blankhost"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	inet "github.com/libp2p/go-libp2p-net"

	"utility"
	"network/biz"
	"consensus/groupsig"
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

	config := common.NewConfINIManager("boot_test.ini")
	ctx := context.Background()

	seedPrivateKey := "0x0423c75e7593a7e6b5ce489f7d3578f8f737b6dd0fc1d2b10dc12a3e88a0572c62b801e14a8864ebe2d7b8c32e31113ccb511a6ad597c008ea90d850439133819f0b682fe8ff4a9023712e74256fb628c8e97658d99f2a8880a3066f120c2e899b"
	seedDht, seedHost, seedId := mockDHT(seedPrivateKey, &config, ctx)
	fmt.Printf("Mock seed node success!\nseddId is:%s\n", seedId)

	ctx1 := context.Background()
	node1, node1Host, node1Id := mockDHT("", &config, ctx1)
	fmt.Printf("Mock  node1 success!\nnode1 is:%s\n", node1Id)

	if node1 != nil && seedDht != nil {

		if node1 != nil && seedDht != nil {
			dhts := []*dht.IpfsDHT{seedDht, node1}
			bootDhts(dhts)
			time.Sleep(30 * time.Second)

			r1 := seedDht.FindLocal(gpeer.ID(node1Id))
			fmt.Printf("Seed local find node1. node1 id is:%s\n", string(r1.ID))

			r2 := node1.FindLocal(gpeer.ID(seedId))
			fmt.Printf("Node1 local find seed. seed id is:%s\n", string(r2.ID))

			peerInfo, err := seedDht.FindPeer(ctx1, gpeer.ID(node1Id))
			if err != nil {
				fmt.Printf("find node1 error:%s\n", err.Error())
			}
			fmt.Printf("find result is:%s\n", string(peerInfo.ID))
		}

		bHandler := biz.NewBlockChainMessageHandler(nil, nil, nil, nil,
			nil)

		cHandler := biz.NewConsensusMessageHandler(nil, nil, nil,nil,nil,nil)

		seedServer := server{seedHost, seedDht, bHandler, cHandler}
		//seedHost.Network().SetStreamHandler(testSteamHandler)

		node1Host.Network().SetStreamHandler(testSteamHandler)
		//node1Server := server{node1Host, node1, bHandler, cHandler}

		messsage := mockMessage()
		//node1Server.SendMessage(messsage,seedId)
		seedServer.SendMessage(messsage,node1Id)

		time.Sleep(1*time.Second)
		conns := seedServer.GetConnInfo()
		for _,conn:= range conns{
			fmt.Printf("conn:%s,%s,%s\n",conn.Id,conn.Ip,conn.TcpPort)
		}
	}
}

func testSteamHandler(stream inet.Stream) {
	defer stream.Close()
	fmt.Printf("in testSteamHandler\n")

	pkgLengthBytes := make([]byte, 4)
	n, err := stream.Read(pkgLengthBytes)
	if n != 4 || err != nil {
		fmt.Printf("Stream  read %d byte error:%s,received %d bytes\n", 4, err.Error(), n)
		return
	}
	fmt.Printf("pkgLengthBytes length:%d\n",len(pkgLengthBytes))
	for i:=0;i<len(pkgLengthBytes);i++{
		fmt.Printf("b:%d",int(pkgLengthBytes[i]))
	}
	fmt.Printf("\n")



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
	fmt.Printf("Before unmarshla message!,pkgLength：%d,length:%d\n",pkgLength,len(pkgBodyBytes))

	for i:=0;i<len(pkgBodyBytes);i++{
		fmt.Printf("b:%d",int(pkgBodyBytes[i]))
	}
	fmt.Printf("\n")

	message, e := UnMarshalMessage(pkgBodyBytes)
	if e != nil {
		fmt.Printf("Unmarshal message error!" + e.Error())
		return
	}
	m := mockMessage()
	fmt.Printf("Reviced message compare result is:%t\n",messageEquals(*message,m))
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
func mockDHT(privateKey string, config *common.ConfManager, ctx context.Context) (*dht.IpfsDHT, host.Host, string) {
	self := NewSelfNetInfo(privateKey)
	fmt.Print(self.String())

	localId := self.Id
	multiaddr, e2 := ma.NewMultiaddr(self.GenMulAddrStr())
	if e2 != nil {
		fmt.Printf("new mlltiaddr error!" + e2.Error())
		return nil, nil, self.Id
	}
	listenAddrs := []ma.Multiaddr{multiaddr}
	peerStore := pstore.NewPeerstore()
	p1 := &Pubkey{PublicKey: self.PublicKey}
	p2 := &Privkey{PrivateKey: self.PrivateKey}

	peerStore.AddPubKey(gpeer.ID(localId), p1)
	peerStore.AddPrivKey(gpeer.ID(localId), p2)

	peerStore.AddAddrs(gpeer.ID(localId), listenAddrs, pstore.PermanentAddrTTL)
	//bwc  is a bandwidth metrics collector, This is used to track incoming and outgoing bandwidth on connections managed by this swarm.
	// It is optional, and passing nil will simply result in no metrics for connections being available.
	sw, e3 := swarm.NewNetwork(ctx, listenAddrs, gpeer.ID(localId), peerStore, nil)
	if e3 != nil {
		fmt.Printf("New swarm error!\n" + e3.Error())
		return nil, nil, self.Id
	}
	//peerStore.AddAddrs(peer.ID(localId), sw.ListenAddresses(), pstore.PermanentAddrTTL)

	//hostOpts := &basichost.HostOpts{}
	//host:= basichost.New(sw)
	host := blankhost.NewBlankHost(sw)
	//if e4 != nil {
	//	fmt.Printf("New host error! " + e4.Error())
	//	return nil, self.Id
	//}

	seedIdStr := (*config).GetString(BASE_SECTION, SEED_ID_KEY, "0xe14f286058ed3096ab90ba48a1612564dffdc358")
	//seedId, e := peer.IDB58Decode(seedIdStrPretty)
	//if e != nil {
	//	fmt.Printf("Decode seed id error:%s\n", e.Error())
	//	return nil, host,self.Id
	//}

	seedAddrStr := (*config).GetString(BASE_SECTION, SEED_ADDRESS_KEY, "/ip4/10.0.0.66/tcp/1122")

	a := self.GenMulAddrStr()
	if a != seedAddrStr {
		seedMultiaddr, e6 := ma.NewMultiaddr(seedAddrStr)
		if e6 != nil {
			fmt.Printf("SeedIdStr to seedMultiaddr error! %s\n", e6.Error())
		}
		seedPeerInfo := pstore.PeerInfo{ID: gpeer.ID(seedIdStr), Addrs: []ma.Multiaddr{seedMultiaddr}}
		e7 := host.Connect(ctx, seedPeerInfo)
		if e7 != nil {
			fmt.Printf("Host connect to seed error! %s\n" + e7.Error())
		}
	}
	dss := dssync.MutexWrap(ds.NewMapDatastore())
	kadDht := dht.NewDHT(ctx, host, dss)
	return kadDht, host, self.Id
}


func TestPubToId(t *testing.T){
	groupsig.Init(1)
	privateKey := common.GenerateKey("")
	publicKey := privateKey.GetPubKey()
	fmt.Printf("pub:"+publicKey.GetHexString()+"\n")
	addr := publicKey.GetAddress()
	fmt.Printf("addr:%s \n",addr.GetHexString())
	id := groupsig.NewIDFromAddress(addr)
	id.Serialize()
}