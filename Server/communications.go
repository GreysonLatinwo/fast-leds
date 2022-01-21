package main

import (
	"log"
	"net"
	"os"
	"strings"

	utils "github.com/greysonlatinwo/fast-leds/utils"
	"github.com/libp2p/go-libp2p-core/peer"
)

const colorUpdateBufSize = 8
const UDPClientPort = ":1234"

var Uaddr *net.UDPAddr

var ledCommPipe = make(chan [6]byte, colorUpdateBufSize)

type remoteLeds struct {
	Server  *net.UDPConn
	Clients map[string]*net.UDPAddr
}

func StartComms() error {
	//start mdns listener
	newPeerChan := initMDNS("fast-leds")

	listenAddr := utils.HandleErrPrint(net.ResolveUDPAddr("udp4", UDPClientPort)).(*net.UDPAddr)
	server := utils.HandleErrPrint(net.ListenUDP("udp4", listenAddr)).(*net.UDPConn)

	piClients := make(map[string]*net.UDPAddr, 2)
	remote := &remoteLeds{Server: server, Clients: piClients}
	go listenForPeers(newPeerChan, remote)
	go colorServer(remote, ledCommPipe)
	return nil
}

func listenForPeers(peerListen chan peer.AddrInfo, remote *remoteLeds) {
	for peer := range peerListen {
		for _, peerMultiAddr := range peer.Addrs {
			peerIP := strings.Split(peerMultiAddr.String(), "/")[2]
			//dont add yourself or clients already added
			if _, ok := remote.Clients[peerIP]; ok {
				continue
			}
			if peerIP == getOutBoundAddress() {
				log.Println("self:", peerIP)
				continue
			}
			log.Println("Adding Client:", peerIP)
			piAddr := utils.HandleErrPrint(net.ResolveUDPAddr("udp4", peerIP+UDPClientPort)).(*net.UDPAddr)
			remote.Clients[peerIP] = piAddr
		}
	}
}

//takes the color output and tells the network
func colorServer(remote *remoteLeds, colorUpdate chan [6]byte) {
	for color := range colorUpdate {
		//log.Println("color:", color)
		go writeToLocalLeds(color)
		go remote.writeToLeds(color)
	}
}

func writeToLocalLeds(color [6]byte) {
	os.Stdout.Write(color[:])
}

func (r remoteLeds) writeToLeds(color [6]byte) {
	for _, client := range r.Clients {
		r.Server.WriteTo(color[:], client)
	}
}
