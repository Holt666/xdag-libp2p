package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	peerstore "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	mplex "github.com/libp2p/go-libp2p-mplex"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	secio "github.com/libp2p/go-libp2p-secio"
	yamux "github.com/libp2p/go-libp2p-yamux"
	"github.com/libp2p/go-libp2p/p2p/discovery"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/libp2p/go-tcp-transport"
	multiaddr "github.com/multiformats/go-multiaddr"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const xdagTopic = "/xdag/1.0.0"

type mdnsNotifee struct {
	h   host.Host
	ctx context.Context
}

func xdagHandler(ctx context.Context, sub *pubsub.Subscription) {
	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
		//TODO call xdag c function
		fmt.Fprintln(os.Stdout, msg)
	}
}

func (m *mdnsNotifee) HandlePeerFound(pi peer.AddrInfo) {
	m.h.Connect(m.ctx, pi)
}

func main() {
	var listenPort = 8001
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Generate a key pair for this host. We will use it at least
	// to obtain a valid host ID.
	var src io.Reader
	src = rand.Reader
	priv, _, err := crypto.GenerateECDSAKeyPair(src)
	if err != nil {
		panic(err)
	}

	nodeid := libp2p.Identity(priv)

	transports := libp2p.ChainOptions(
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Ping(false),
	)

	listenAddrs := libp2p.ListenAddrStrings(
		fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort),
	)

	var dht *kaddht.IpfsDHT
	newDHT := func(h host.Host) (routing.PeerRouting, error) {
		var err error
		dht, err = kaddht.New(ctx, h)
		return dht, err
	}
	routing := libp2p.Routing(newDHT)

	muxers := libp2p.ChainOptions(
		libp2p.Muxer("/yamux/1.0.0", yamux.DefaultTransport),
		libp2p.Muxer("/mplex/6.7.0", mplex.DefaultTransport),
	)

	security := libp2p.Security(secio.ID, secio.New)

	// start a libp2p node that listens on a random local TCP port
	node, err := libp2p.New(ctx,
		nodeid,
		transports,
		listenAddrs,
		muxers,
		security,
		routing,
	)
	if err != nil {
		panic(err)
	}

	ps, err := pubsub.NewGossipSub(ctx, node)
	if err != nil {
		panic(err)
	}
	sub, err := ps.Subscribe(xdagTopic)
	if err != nil {
		panic(err)
	}

	go xdagHandler(ctx, sub)

	mdns, err := discovery.NewMdnsService(ctx, node, time.Second*10, "")
	if err != nil {
		panic(err)
	}
	mdns.RegisterNotifee(&mdnsNotifee{h: node, ctx: ctx})

	err = dht.Bootstrap(ctx)
	if err != nil {
		panic(err)
	}

	// configure our own ping protocol
	pingService := &ping.PingService{Host: node}
	node.SetStreamHandler(ping.ID, pingService.PingHandler)
	//node.SetStreamHandler()

	// print the node's PeerInfo in multiaddr format
	peerInfo := peerstore.AddrInfo{
		ID:    node.ID(),
		Addrs: node.Addrs(),
	}
	addrs, err := peerstore.AddrInfoToP2pAddrs(&peerInfo)
	if err != nil {
		panic(err)
	}
	fmt.Println("xdag node address:", addrs[0])

	// if a remote peer has been passed on the command line, connect to it
	// and send it 5 ping messages, otherwise wait for a signal to stop
	if len(os.Args) > 1 {
		addr, err := multiaddr.NewMultiaddr(os.Args[1])
		if err != nil {
			panic(err)
		}
		peer, err := peerstore.AddrInfoFromP2pAddr(addr)
		if err != nil {
			panic(err)
		}
		if err := node.Connect(ctx, *peer); err != nil {
			panic(err)
		}
		fmt.Println("xdag-libp2p-node sending 5000 ping messages to", addr)
		ch := pingService.Ping(ctx, peer.ID)
		for i := 0; i < 5000; i++ {
			res := <-ch
			fmt.Println("xdag-libp2p-node pinged ", i, " times ", addr, "in", res.RTT)
		}
	} else {
		// wait for a SIGINT or SIGTERM signal
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
		<-ch
		fmt.Println("Received signal, shutting down...")
	}

	// shut the node down
	if err := node.Close(); err != nil {
		panic(err)
	}
}
