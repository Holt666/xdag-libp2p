package main // 这个文件一定要在main包下面

import (
	"C" // 这个 import 也是必须的，有了这个才能生成 .h 文件
	"context"
	"fmt"
	"github.com/libp2p/go-libp2p"
	peerstore "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	multiaddr "github.com/multiformats/go-multiaddr"
)

// 下面这一行不是注释，是导出为SO库的标准写法，注意 export前面不能有空格！！！
//export hello
func hello(value string) *C.char { // 如果函数有返回值，则要将返回值转换为C语言对应的类型
	return C.CString("hello " + value)
}

//export xdag_libp2p_send
func xdag_libp2p_send(send_addr string) *C.char {
	// create a background context (i.e. one that never cancels)
	ctx := context.Background()

	// start a libp2p node that listens on a random local TCP port,
	// but without running the built-in ping protocol
	// /ip4/127.0.0.1/tcp/0
	node, err := libp2p.New(ctx,
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
		libp2p.Ping(false),
	)
	if err != nil {
		panic(err)
	}

	// configure our own ping protocol
	pingService := &ping.PingService{Host: node}
	node.SetStreamHandler(ping.ID, pingService.PingHandler)

	// print the node's PeerInfo in multiaddr format
	peerInfo := peerstore.AddrInfo{
		ID:    node.ID(),
		Addrs: node.Addrs(),
	}

	addrs, err := peerstore.AddrInfoToP2pAddrs(&peerInfo)
	if err != nil {
		panic(err)
	}
	fmt.Println("libp2p node address:", addrs[0])

	// if a remote peer has been passed on the command line, connect to it
	// and send it 5 ping messages, otherwise wait for a signal to stop
	//if len(os.Args) > 1 {
	addr, err := multiaddr.NewMultiaddr(send_addr)
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
	fmt.Println("sending 1000 ping messages to", addr)
	ch := pingService.Ping(ctx, peer.ID)
	for i := 0; i < 1000; i++ {
		res := <-ch
		fmt.Println("pinged", addr, "in", res.RTT)
	}
	//}

	// shut the node down
	if err := node.Close(); err != nil {
		panic(err)
	}
	return C.CString("ok")
}

func main() {
	// 此处一定要有main函数，有main函数才能让cgo编译器去把包编译成C的库
}
