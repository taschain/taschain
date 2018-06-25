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
	"common"
	"time"
	"github.com/libp2p/go-libp2p/p2p/host/basic"
	"github.com/libp2p/go-libp2p-host"
)

const (
	SEED_ID_KEY = "seed_id"

	SEED_ADDRESS_KEY = "seed_address"
)

var Logger taslog.Logger

func InitNetwork(config *common.ConfManager, isSuper bool) error {

	Logger = taslog.GetLoggerByName("p2p" + common.GlobalConf.GetString("client", "index", ""))

	node, e1 := makeSelfNode(config)
	if e1 != nil {
		return e1
	}

	e2 := initServer(config, *node, isSuper)
	if e2 != nil {
		return e2
	}
	return nil
}

func initServer(config *common.ConfManager, node p2p.Node, isSuper bool) error {

	ctx := context.Background()
	context.WithTimeout(ctx, p2p.ContextTimeOut)
	network, e1 := makeSwarm(ctx, node)
	if e1 != nil {
		return e1
	}

	host := makeHost(network)

	dss := dssync.MutexWrap(ds.NewMapDatastore())
	kadDht := dht.NewDHT(ctx, host, dss)

	if !isSuper {
		e2 := connectToSeed(ctx, &host, config, node)
		if e2 != nil {
			return e2
		}
	}
	dht, e3 := initDHT(kadDht)
	if e3 != nil {
		return e3
	}

	p2p.InitServer(host, dht, &node)

	id, _, _ := getSeedInfo(config)
	if p2p.ConvertToID(id) != p2p.Server.SelfNetInfo.Id {
		for {
			info, e4 := p2p.Server.Dht.FindPeer(ctx, id)
			if e4 != nil {
				Logger.Infof("[Network]Find seed id %s error:%s", p2p.ConvertToID(id), e4.Error())
				time.Sleep(5 * time.Second)
			} else if p2p.ConvertToID(info.ID) == "" {
				Logger.Infof("[Network]Can not find seed node,finding....")
				time.Sleep(5 * time.Second)
			} else {
				Logger.Infof("[Network]Welcome to join TAS Network!")
				break
			}
		}
	}
	return nil
}
func makeSelfNode(config *common.ConfManager) (*p2p.Node, error) {
	node, error := p2p.InitSelfNode(config)
	if error != nil {
		Logger.Error("[Network]InitSelfNode error!\n" + error.Error())
		return nil, error
	}
	return node, nil
}

func makeSwarm(ctx context.Context, self p2p.Node) (net.Network, error) {
	localId := self.Id
	multiaddr, e1 := ma.NewMultiaddr(self.GenMulAddrStr())
	if e1 != nil {
		Logger.Error("[Network]new mlltiaddr error!\n" + e1.Error())
		return nil, e1
	}
	listenAddrs := []ma.Multiaddr{multiaddr}

	peerStore := pstore.NewPeerstore()
	p1 := &p2p.Pubkey{PublicKey: self.PublicKey}
	p2 := &p2p.Privkey{PrivateKey: self.PrivateKey}

	ID := p2p.ConvertToPeerID(localId)
	peerStore.AddPubKey(ID, p1)
	peerStore.AddPrivKey(ID, p2)

	peerStore.AddAddrs(ID, listenAddrs, pstore.PermanentAddrTTL)
	//bwc  is a bandwidth metrics collector, This is used to track incoming and outgoing bandwidth on connections managed by this swarm.
	// It is optional, and passing nil will simply result in no metrics for connections being available.
	sw, e2 := swarm.NewNetwork(ctx, listenAddrs, ID, peerStore, nil)
	if e2 != nil {
		Logger.Error("[Network]New swarm error!\n" + e2.Error())
		return nil, e2
	}
	return sw, nil
}

func makeHost(n net.Network) (host.Host) {
	opt := basichost.HostOpts{}
	opt.NegotiationTimeout = -1
	host := basichost.New(n)
	return host
}

func connectToSeed(ctx context.Context, host *host.Host, config *common.ConfManager, node p2p.Node) error {
	seedId, seedAddrStr, e1 := getSeedInfo(config)
	if e1 != nil {
		return e1
	}

	seedMultiaddr, e2 := ma.NewMultiaddr(seedAddrStr)
	if e2 != nil {
		Logger.Error("[Network]SeedIdStr to seedMultiaddr error!\n" + e2.Error())
		return e2
	}
	seedPeerInfo := pstore.PeerInfo{ID: seedId, Addrs: []ma.Multiaddr{seedMultiaddr}}
	(*host).Peerstore().AddAddrs(seedPeerInfo.ID, seedPeerInfo.Addrs, pstore.PermanentAddrTTL)
	e3 := (*host).Connect(ctx, seedPeerInfo)
	if e3 != nil {
		Logger.Error("[Network]Host connect to seed error!\n" + e3.Error())
		for i := 1; i <= 3; i++ {
			time.Sleep(time.Second * 5)
			Logger.Infof("[Network]Try to connect to seed:no %d\n", i)
			e := (*host).Connect(ctx, seedPeerInfo)
			if e == nil {
				break
			}
		}
		return e3
	}
	return nil
}

func initDHT(kadDht *dht.IpfsDHT) (*dht.IpfsDHT, error) {

	cfg := dht.DefaultBootstrapConfig
	cfg.Queries = 3
	cfg.Period = time.Duration(10 * time.Second)
	cfg.Timeout = time.Second * 30
	process, e := kadDht.BootstrapWithConfig(cfg)
	if e != nil {
		process.Close()
		Logger.Errorf("KadDht bootstrap error!" + e.Error())
		return kadDht, e
	}
	return kadDht, nil
}

func getSeedInfo(config *common.ConfManager) (peer.ID, string, error) {
	seedIdStr := (*config).GetString(p2p.BASE_SECTION, SEED_ID_KEY, "Qmdeh5r5kT2je77JNYKTsQi6ncckpLa9aFnr6xYQaGAxaw")
	seedAddrStr := (*config).GetString(p2p.BASE_SECTION, SEED_ADDRESS_KEY, "/ip4/10.0.0.66/tcp/1122")
	return p2p.ConvertToPeerID(seedIdStr), seedAddrStr, nil
}
