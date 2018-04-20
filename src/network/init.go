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
	"network/biz"
)

const (
	SEED_ID_KEY = "seed_id"

	SEED_ADDRESS_KEY = "seed_address"
)

func InitNetwork(config *common.ConfManager) error {

	e1 := initPeer(config)
	if e1 != nil {
		return e1
	}

	e2 := initServer(config)
	if e2 != nil {
		return e2
	}

	initBlockSyncer()

	initGroupSyncer()
	return nil
}

func initPeer(config *common.ConfManager) error {
	node, error := makeSelfNode(config)
	if error != nil {
		return error
	}
	p2p.InitPeer(node)
	return nil
}

func initServer(config *common.ConfManager) error {

	ctx := context.Background()
	network, e1 := makeSwarm(ctx)
	if e1 != nil {
		return e1
	}

	host := makeHost(network)
	e2 := connectToSeed(ctx, host, config)
	if e2 != nil {
		return e2
	}

	dht, e3 := initDHT(ctx, host)
	if e3 != nil {
		return e3
	}

	bHandler := biz.NewBlockChainMessageHandler(nil, nil, nil, nil,
		nil, nil, )

	cHandler := biz.NewConsensusMessageHandler(nil, nil, nil, nil, nil, nil)

	p2p.InitServer(host, dht, bHandler, cHandler)
	return nil
}
func makeSelfNode(config *common.ConfManager) (*p2p.Node, error) {
	node, error := p2p.InitSelfNode(config)
	if error != nil {
		taslog.P2pLogger.Error("InitSelfNode error!\n" + error.Error())
		return nil, error
	}
	return node, nil
}

func makeSwarm(ctx context.Context) (net.Network, error) {
	self := p2p.Peer.SelfNetInfo
	localId := self.Id
	multiaddr, e1 := ma.NewMultiaddr(self.GenMulAddrStr())
	if e1 != nil {
		taslog.P2pLogger.Error("new mlltiaddr error!\n" + e1.Error())
		return nil, e1
	}
	listenAddrs := []ma.Multiaddr{multiaddr}

	peerStore := pstore.NewPeerstore()
	p1 := &p2p.Pubkey{PublicKey: self.PublicKey}
	p2 := &p2p.Privkey{PrivateKey: self.PrivateKey}
	peerStore.AddPubKey(peer.ID(localId), p1)
	peerStore.AddPrivKey(peer.ID(localId), p2)
	//bwc  is a bandwidth metrics collector, This is used to track incoming and outgoing bandwidth on connections managed by this swarm.
	// It is optional, and passing nil will simply result in no metrics for connections being available.
	sw, e2 := swarm.NewNetwork(ctx, listenAddrs, peer.ID(localId), peerStore, nil)
	if e2 != nil {
		taslog.P2pLogger.Error("New swarm error!\n" + e2.Error())
		return nil, e2
	}
	peerStore.AddAddrs(peer.ID(localId), sw.ListenAddresses(), pstore.PermanentAddrTTL)
	return sw, nil
}

func makeHost(n net.Network) (host.Host) {
	host := blankhost.NewBlankHost(n)
	return host
}

func connectToSeed(ctx context.Context, host host.Host, config *common.ConfManager) error {
	seedIdStr, seedAddrStr, e1 := getSeedInfo(config)
	if e1 != nil {
		return e1
	}
	if p2p.Peer.SelfNetInfo.GenMulAddrStr() == seedAddrStr {
		return nil
	}
	seedMultiaddr, e2 := ma.NewMultiaddr(seedAddrStr)
	if e2 != nil {
		taslog.P2pLogger.Error("SeedIdStr to seedMultiaddr error!\n" + e2.Error())
		return e2
	}
	seedPeerInfo := pstore.PeerInfo{ID: peer.ID(seedIdStr), Addrs: []ma.Multiaddr{seedMultiaddr}}
	e3 := host.Connect(ctx, seedPeerInfo)
	if e3 != nil {
		taslog.P2pLogger.Error("Host connect to seed error!\n" + e3.Error())
		return e3
	}
	return nil
}

func initDHT(ctx context.Context, host host.Host) (*dht.IpfsDHT, error) {
	dss := dssync.MutexWrap(ds.NewMapDatastore())
	kadDht := dht.NewDHT(ctx, host, dss)

	cfg := dht.DefaultBootstrapConfig
	cfg.Queries = 3
	cfg.Period = time.Duration(20 * time.Second)
	process, e := kadDht.BootstrapWithConfig(cfg)
	if e != nil {
		process.Close()
		taslog.P2pLogger.Error("KadDht bootstrap error!\n" + e.Error())
		return kadDht, e
	}
	return kadDht, nil
}

func getSeedInfo(config *common.ConfManager) (peer.ID, string, error) {
	seedIdStrPretty := (*config).GetString(p2p.BASE_SECTION, SEED_ID_KEY, "QmaGUeg9A1f2umu2ToPN8r7sJzMgQMuHYYAjaYwkkyrBz9")
	seedId, e := peer.IDB58Decode(seedIdStrPretty)
	if e != nil {
		fmt.Printf("Decode seed id error:%s\n", e.Error())
		return peer.ID(""), "", e
	}

	seedAddrStr := (*config).GetString(p2p.BASE_SECTION, SEED_ADDRESS_KEY, "/ip4/10.0.0.66/tcp/1122")
	return seedId, seedAddrStr, nil
}

func initBlockSyncer() {
	p2p.InitBlockSyncer(nil, nil, nil, nil)
}

func initGroupSyncer() {
	p2p.InitGroupSyncer(nil, nil, nil, nil)
}
