package network

import (
	"testing"
	"fmt"
	"network/p2p"
	"context"
	"common"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-crypto"
	"time"
	"taslog"
	"github.com/libp2p/go-libp2p-host"
	"consensus/groupsig"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"net"
)

func TestDHT(t *testing.T) {
	defer taslog.Close()

	groupsig.Init(1)
	crypto.KeyTypes = append(crypto.KeyTypes, 3)
	crypto.PubKeyUnmarshallers[3] = p2p.UnmarshalEcdsaPublicKey

	config := common.NewConfINIManager("dht_test.ini")
	ctx := context.Background()

	seedPrivateKey := "0x0423c75e7593a7e6b5ce489f7d3578f8f737b6dd0fc1d2b10dc12a3e88a0572c62b801e14a8864ebe2d7b8c32e31113ccb511a6ad597c008ea90d850439133819f0b682fe8ff4a9023712e74256fb628c8e97658d99f2a8880a3066f120c2e899b"
	seedDht, _, seedNode := mockDHT(seedPrivateKey, &config, ctx)
	fmt.Printf("Mock seed node success! seedId is:%s\n", seedNode.Id)

	node1Dht, node1Host, node1Node := mockDHT("", &config, ctx)
	fmt.Printf("Mock  node1 success! node1 is:%s\n", node1Node.Id)

	if node1Dht != nil && seedDht != nil {

		e2 := connectToSeed(ctx, &node1Host, &config, *node1Node)
		if e2 != nil {
			fmt.Errorf("connectToSeed error!%s\n", e2.Error())
			panic("connectToSeed error!")
		}
		dhts := []*dht.IpfsDHT{node1Dht, seedDht}
		bootDhts(dhts)
		time.Sleep(30 * time.Second)

		node11ID := p2p.ConvertToPeerID(node1Node.Id)
		r1 := seedDht.FindLocal(node11ID)
		fmt.Printf("Seed local find node1. node1 id is:%s\n", p2p.ConvertToID(r1.ID))

		seed11ID := p2p.ConvertToPeerID(seedNode.Id)
		r2 := node1Dht.FindLocal(seed11ID)
		fmt.Printf("Node1 local find seed. seed id is:%s\n", p2p.ConvertToID(r2.ID))

		r3, err := seedDht.FindPeer(ctx, node11ID)
		if err != nil {
			fmt.Printf("Seed find node1 error:%s\n", err.Error())
		}
		fmt.Printf("Seed find node1 result is:%s\n", p2p.ConvertToID(r3.ID))

		r4, err1 := node1Dht.FindPeer(ctx, seed11ID)
		if err1 != nil {
			fmt.Printf("Node1 find seed error:%s\n", err1.Error())
		}
		fmt.Printf("Node1 find seed result is:%s\n", p2p.ConvertToID(r4.ID))
	}
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

func mockNode(privateKey string, config *common.ConfManager) *p2p.Node {
	return p2p.NewSelfNetInfo(privateKey)
}

func mockDHT(privateKey string, config *common.ConfManager, ctx context.Context) (*dht.IpfsDHT, host.Host, *p2p.Node) {
	self := mockNode(privateKey, config)
	//fmt.Print(self.String())
	network, e1 := makeSwarm(ctx, *self)
	if e1 != nil {
		fmt.Errorf("make swarm error!%s\n", e1.Error())
		panic("make swarm error!")
	}
	host := makeHost(network)
	dss := dssync.MutexWrap(ds.NewMapDatastore())
	kadDht := dht.NewDHT(ctx, host, dss)
	return kadDht, host, self
}

func TestID(t *testing.T) {
	groupsig.Init(1)
	privateKey := common.GenerateKey("")
	publicKey := privateKey.GetPubKey()
	id := p2p.GetIdFromPublicKey(publicKey)
	fmt.Printf(id)
}

func TestUnmarshalEcdsaPublicKey(t *testing.T) {
	crypto.KeyTypes = append(crypto.KeyTypes, 3)
	crypto.PubKeyUnmarshallers[3] = p2p.UnmarshalEcdsaPublicKey
	privateKey := common.GenerateKey("")
	publicKey := privateKey.GetPubKey()
	pub := &p2p.Pubkey{PublicKey: publicKey}
	b1, i3 := pub.Bytes()
	if i3 != nil {
		fmt.Errorf("PublicKey to bytes error!\n")
	}

	bytes, e := crypto.MarshalPublicKey(pub)
	if e != nil {
		fmt.Errorf("MarshalPublicKey Error\n")
	}
	pubKey, i := crypto.UnmarshalPublicKey(bytes)
	if i != nil {
		fmt.Errorf("UnmarshalPublicKey Error\n")
	}
	b2, i4 := pubKey.Bytes()
	if i4 != nil {
		fmt.Errorf("PubKey to bytes Error\n")

	}
	fmt.Print("Origin public key length is :%d,marshal and unmaishal pub key length is:%d\n", len(b1), len(b2))
}


func TestContext(t *testing.T){
	ctx := context.Background()
	deadline, ok := ctx.Deadline()
	fmt.Print(deadline,ok)
}


func TestIp(t *testing.T){
	addrs, err := net.InterfaceAddrs()

	if err != nil {
		fmt.Printf("TestIp error:%s",err.Error())
	}

	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				fmt.Printf(ipnet.IP.String())
				break
			}
		}
	}
}