package network

import (
	"testing"
	"fmt"
	"taslog"
	"network/p2p"
	"context"
	ma "github.com/multiformats/go-multiaddr"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	"github.com/libp2p/go-libp2p-swarm"
	//"github.com/libp2p/go-libp2p/p2p/host/basic"
	"common"
	"github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p-peer"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"


	"github.com/libp2p/go-libp2p-crypto"
	"time"
	"gx/ipfs/QmYvJhMM1SRTcGFxsHQ6gYZMUtHUphkAKUYp7VPrMtRAQ9/go-libp2p-blankhost"
	"github.com/multiformats/go-multihash"
)

func TestDHT(t *testing.T) {
	crypto.KeyTypes = append(crypto.KeyTypes, 3)
	crypto.PubKeyUnmarshallers[3] = p2p.UnmarshalEcdsaPublicKey

	config := common.NewConfINIManager("boot_test.ini")
	ctx := context.Background()

	seedPrivateKey := "0x0423c75e7593a7e6b5ce489f7d3578f8f737b6dd0fc1d2b10dc12a3e88a0572c62b801e14a8864ebe2d7b8c32e31113ccb511a6ad597c008ea90d850439133819f0b682fe8ff4a9023712e74256fb628c8e97658d99f2a8880a3066f120c2e899b"
	seedDht, seedId := mockDHT(seedPrivateKey, &config,ctx)
	fmt.Printf("Mock seed node success!\nseddId is:%s\n", peer.ID(seedId).Pretty())
	ctx1 := context.Background()
	node1, node1Id := mockDHT("", &config,ctx1)
	fmt.Printf("Mock  node1 success!\nnode1 is:%s\n", peer.ID(node1Id).Pretty())

	if node1 != nil && seedDht != nil {
		cfg := dht.DefaultBootstrapConfig
		cfg.Queries = 3
		cfg.Period = time.Duration(20 * time.Second)
		 go func(){
			 process, e8 := node1.BootstrapWithConfig(cfg)
			 if e8 != nil {
				 process.Close()
				 fmt.Print("KadDht bootstrap error! " + e8.Error())
				 return
			 }
		 }()

		p1, e9 := seedDht.BootstrapWithConfig(cfg)
		if e9 != nil {
			p1.Close()
			fmt.Print("KadDht bootstrap error! " + e9.Error())
			return
		}


		time.Sleep(60 * time.Second)

		r1 := seedDht.FindLocal(peer.ID(node1Id))
		fmt.Printf("Seed local find node1. node1 id is:%s\n", r1.ID.Pretty())

		r2 := node1.FindLocal(peer.ID(seedId))
		fmt.Printf("Node1 local find seed. seed id is:%s\n", r2.ID.Pretty())

		peerInfo, err := seedDht.FindPeer(ctx1, peer.ID(node1Id))
		if err !=nil{
			fmt.Printf("find node1 error:%s\n",err.Error())
		}
		fmt.Printf("find result is:%s\n",peerInfo.ID.Pretty())
	} else {
		return
	}
	taslog.Close()
}

func mockDHT(privateKey string, config *common.ConfManager,ctx context.Context) (*dht.IpfsDHT, string) {
	self := p2p.NewSelfNetInfo(privateKey)

	localId := self.Id
	multiaddr, e2 := ma.NewMultiaddr(self.GenMulAddrStr())
	if e2 != nil {
		fmt.Printf("new mlltiaddr error!" + e2.Error())
		return nil, self.Id
	}
	listenAddrs := []ma.Multiaddr{multiaddr}
	peerStore := pstore.NewPeerstore()
	p1 :=&p2p.Pubkey{PublicKey:self.PublicKey}
	p2 :=&p2p.Privkey{PrivateKey:self.PrivateKey}

	peerStore.AddPubKey(peer.ID(localId),p1)
	peerStore.AddPrivKey(peer.ID(localId),p2)

	//bwc  is a bandwidth metrics collector, This is used to track incoming and outgoing bandwidth on connections managed by this swarm.
	// It is optional, and passing nil will simply result in no metrics for connections being available.
	sw, e3 := swarm.NewNetwork(ctx, listenAddrs, peer.ID(localId), peerStore, nil)
	if e3 != nil {
		fmt.Printf("New swarm error!\n" + e3.Error())
		return nil, self.Id
	}
	peerStore.AddAddrs(peer.ID(localId),sw.ListenAddresses(),pstore.PermanentAddrTTL)

	//hostOpts := &basichost.HostOpts{}
	//host:= basichost.New(sw)
	host:= blankhost.NewBlankHost(sw)
	//if e4 != nil {
	//	fmt.Printf("New host error! " + e4.Error())
	//	return nil, self.Id
	//}

	seedIdStrPretty, b1 := (*config).GetString(p2p.BASE_SECTION, SEED_ID_KEY)
	if !b1 {
		fmt.Printf("Get seed id from config file error!\n")
		return nil, self.Id
	}
	seedId, e := peer.IDB58Decode(seedIdStrPretty)
	if e!=nil{
		fmt.Printf("Decode seed id error:%s\n",e.Error())
		return nil, self.Id
	}

	seedAddrStr, b2 := (*config).GetString(p2p.BASE_SECTION, SEED_ADDRESS_KEY)
	if !b2 {
		fmt.Printf("Get seed address from config file error!\n")
		return nil, self.Id
	}

	a := self.GenMulAddrStr()
	if a != seedAddrStr {
		seedMultiaddr, e6 := ma.NewMultiaddr(seedAddrStr)
		if e6 != nil {
			fmt.Printf("SeedIdStr to seedMultiaddr error! %s\n" , e6.Error())
		}
		fmt.Printf("Seed id pretty:%s\n",seedId.Pretty())
		seedPeerInfo := pstore.PeerInfo{ID: seedId, Addrs: []ma.Multiaddr{seedMultiaddr}}
		e7 := host.Connect(ctx, seedPeerInfo)
		if e7 != nil {
			fmt.Printf("Host connect to seed error! %s\n" + e7.Error())
		}
	}
	dss := dssync.MutexWrap(ds.NewMapDatastore())
	kadDht := dht.NewDHT(ctx, host, dss)
	return kadDht, self.Id
}

func TestIDB58(t *testing.T){
	privateKey := common.GenerateKey("")
	publicKey := privateKey.GetPubKey()
	b := publicKey.ToBytes()
	bytes, e := multihash.Sum(b, multihash.SHA2_256,-1)
	id:=peer.ID(bytes)
	if e!=nil{
		fmt.Errorf("multihash encode error:%s\n",e.Error())
	}
	fmt.Printf("ID pretty is:%s\n",id.Pretty())

	encodeStr := peer.IDB58Encode(id)
	r,error:= peer.IDB58Decode(encodeStr)
	if error!=nil{
		fmt.Printf("IDB58Decode error:%s",error.Error())
	}
	fmt.Printf("ID b58 encode and decode " +
		"result :%s",r.Pretty())
}


func TestUnmarshalEcdsaPublicKey(t *testing.T){
	crypto.KeyTypes = append(crypto.KeyTypes, 3)
	crypto.PubKeyUnmarshallers[3] = p2p.UnmarshalEcdsaPublicKey
	privateKey := common.GenerateKey("")
	publicKey := privateKey.GetPubKey()
	pub:= &p2p.Pubkey{PublicKey:publicKey}
	b1, i3 := pub.Bytes()
	if i3!=nil{
		fmt.Errorf("PublicKey to bytes error!\n")
	}


	bytes, e := crypto.MarshalPublicKey(pub)
	if e!=nil{
		fmt.Errorf("MarshalPublicKey Error\n")
	}
	pubKey, i := crypto.UnmarshalPublicKey(bytes)
	if i!=nil{
		fmt.Errorf("UnmarshalPublicKey Error\n")
	}
	b2, i4 := pubKey.Bytes()
	if i4!=nil{
		fmt.Errorf("PubKey to bytes Error\n")

	}
	fmt.Print("Origin public key length is :%d,marshal and unmaishal pub key length is:%d",len(b1),len(b2))
}
