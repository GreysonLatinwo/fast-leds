package main

import (
	"context"
	"flag"
	"log"
	"net"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

// DiscoveryInterval is how often we re-publish our mDNS records.
const DiscoveryInterval = time.Hour

const PubSubListeningAddr = "/ip4/0.0.0.0/tcp/0"

// DiscoveryServiceTag is used in our mDNS advertisements to discover other chat peers.
var DiscoveryServiceTagFlag *string

var clients []*net.UDPAddr

func initComms() chan [3]byte {
	// parse some flags to set our nickname and the room to join
	DiscoveryServiceTagFlag = flag.String("DiscoveryServiceTag", "GreysonsLEDs", "Led group to join")
	flag.Parse()

	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	ChkPrint(err)

	setupDiscovery(h)

	log.Println("Discovery Service Tag:", *DiscoveryServiceTagFlag)

	addr := net.UDPAddr{
		Port: 1234,
		IP:   nil,
	}
	server, err := net.ListenUDP("udp", &addr)
	if err != nil {
		ChkPrint(err)
	}

	go newClientListener(server)

	colorUpdate := make(chan [3]byte, fftColorBufferSize)
	go sendColor(server, colorUpdate)

	return colorUpdate
}

//takes the color output and tells the network
func sendColor(server *net.UDPConn, colorUpdate chan [3]byte) {
	for {
		color := <-colorUpdate
		//log.Println(color)
		for _, client := range clients {
			_, err := server.WriteToUDP(color[:], client)
			ChkPrint(err)
		}
	}
}

// function that listens for new clients to add
func newClientListener(server *net.UDPConn) {
	// check ip
	// dont allow loopback ips
	for {
		_, raddr, _ := server.ReadFromUDP([]byte{})
		log.Println("Adding Client:", raddr)
		clients = append(clients, raddr)
	}
}

// discoveryNotifee gets notified when we find a new peer via mDNS discovery
type discoveryNotifee struct {
	h host.Host
}

// HandlePeerFound connects to peers discovered via mDNS. Once they're connected,
// the PubSub system will automatically start interacting with them if they also~`	`
// support PubSub.
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	log.Printf("discovered new peer %s\n", pi.ID.Pretty())
	n.h.Connect(context.Background(), pi)
}

// setupDiscovery creates an mDNS discovery service and attaches it to the libp2p Host.
// This lets us automatically discover peers on the same LAN and connect to them.
func setupDiscovery(h host.Host) error {
	// setup mDNS discovery
	s := mdns.NewMdnsService(h, *DiscoveryServiceTagFlag, &discoveryNotifee{h: h})
	return s.Start()
}
