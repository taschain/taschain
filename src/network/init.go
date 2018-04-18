package network

import (
	"taslog"
	"github.com/libp2p/go-libp2p-swarm"
	ma "github.com/multiformats/go-multiaddr"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p-kad-dht"
	"context"
	"github.com/libp2p/go-libp2p-peer"
	"network/p2p"
	"github.com/libp2p/go-libp2p-net"
	"github.com/libp2p/go-libp2p-host"
	"common"
	"fmt"
	"time"
	"github.com/libp2p/go-libp2p-blankhost"
)

const (
	SEED_ID_KEY = "seed_id"

	SEED_ADDRESS_KEY = "seed_address"
)

func InitNetwork(config *common.ConfManager) error {

	e1 := initSelfNode(config)
	if e1 != nil {
		return e1
	}

	ctx := context.Background()
	network, e2 := initSwarm(ctx)
	if e2 != nil {
		return e2
	}

	host := initHost(network)
	e4 := connectToSeed(ctx, host, config)
	if e4 != nil {
		return e4
	}

	dht, e5 := initDHT(ctx, host)
	if e5 != nil {
		return e5
	}

	p2p.InitServer(host, dht)
	return nil
}

func initSelfNode(config *common.ConfManager) error {
	e0 := p2p.InitSelfNode(config)
	if e0 != nil {
		taslog.P2pLogger.Error("InitSelfNode error!\n" + e0.Error())
		return e0
	}
	return nil
}

func initSwarm(ctx context.Context) (net.Network, error) {
	self := p2p.SelfNetInfo
	localId := self.Id
	multiaddr, e2 := ma.NewMultiaddr(self.GenMulAddrStr())
	if e2 != nil {
		taslog.P2pLogger.Error("new mlltiaddr error!\n" + e2.Error())
		return nil, e2
	}
	listenAddrs := []ma.Multiaddr{multiaddr}

	peerStore := pstore.NewPeerstore()
	p1 := &p2p.Pubkey{PublicKey: self.PublicKey}
	p2 := &p2p.Privkey{PrivateKey: self.PrivateKey}
	peerStore.AddPubKey(peer.ID(localId), p1)
	peerStore.AddPrivKey(peer.ID(localId), p2)
	//bwc  is a bandwidth metrics collector, This is used to track incoming and outgoing bandwidth on connections managed by this swarm.
	// It is optional, and passing nil will simply result in no metrics for connections being available.
	sw, e3 := swarm.NewNetwork(ctx, listenAddrs, peer.ID(localId), peerStore, nil)
	if e3 != nil {
		taslog.P2pLogger.Error("New swarm error!\n" + e3.Error())
		return nil, e3
	}
	peerStore.AddAddrs(peer.ID(localId), sw.ListenAddresses(), pstore.PermanentAddrTTL)
	return sw, nil
}

func initHost(n net.Network) (host.Host) {
	host := blankhost.NewBlankHost(n)
	return host
}

func connectToSeed(ctx context.Context, host host.Host, config *common.ConfManager) error {
	//get seed ID and sedd multi address from config file
	seedIdStr, seedAddrStr, e := getSeedInfo(config)
	if e != nil {
		return e
	}
	if p2p.SelfNetInfo.GenMulAddrStr() == seedAddrStr {
		return nil
	}
	seedMultiaddr, e6 := ma.NewMultiaddr(seedAddrStr)
	if e6 != nil {
		taslog.P2pLogger.Error("SeedIdStr to seedMultiaddr error!\n" + e6.Error())
		return e6
	}
	seedPeerInfo := pstore.PeerInfo{ID: peer.ID(seedIdStr), Addrs: []ma.Multiaddr{seedMultiaddr}}
	e7 := host.Connect(ctx, seedPeerInfo)
	if e7 != nil {
		taslog.P2pLogger.Error("Host connect to seed error!\n" + e7.Error())
		return e7
	}
	return nil
}

func initDHT(ctx context.Context, host host.Host) (*dht.IpfsDHT, error) {
	dss := dssync.MutexWrap(ds.NewMapDatastore())
	kadDht := dht.NewDHT(ctx, host, dss)

	cfg := dht.DefaultBootstrapConfig
	cfg.Queries = 3
	cfg.Period = time.Duration(20 * time.Second)
	process, e8 := kadDht.BootstrapWithConfig(cfg)
	if e8 != nil {
		process.Close()
		taslog.P2pLogger.Error("KadDht bootstrap error!\n" + e8.Error())
		return kadDht, e8
	}
	return kadDht, nil
}

func getSeedInfo(config *common.ConfManager) (peer.ID, string, error) {
	seedIdStrPretty:= (*config).GetString(p2p.BASE_SECTION, SEED_ID_KEY,"QmaGUeg9A1f2umu2ToPN8r7sJzMgQMuHYYAjaYwkkyrBz9")
	seedId, e := peer.IDB58Decode(seedIdStrPretty)
	if e != nil {
		fmt.Printf("Decode seed id error:%s\n", e.Error())
		return peer.ID(""), "", e
	}

	seedAddrStr := (*config).GetString(p2p.BASE_SECTION, SEED_ADDRESS_KEY,"/ip4/10.0.0.66/tcp/1122")
	return seedId, seedAddrStr, nil
}
