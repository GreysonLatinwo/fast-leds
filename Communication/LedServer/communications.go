package main

import (
	"log"
	"net"
	"net/http"
	"os"
)

const colorUpdateBufSize = 8

var webServerPort = ":9000"

var clients []*net.UDPAddr
var colorUpdate = make(chan []byte, colorUpdateBufSize)
var addr = net.UDPAddr{
	Port: 1234,
	IP:   nil,
}

func InitComms() (chan []byte, error) {

	server, err := net.ListenUDP("udp", &addr)
	if err != nil {
		return nil, err
	}

	go newClientListener(server)

	go sendColor(server, colorUpdate)

	go http.ListenAndServe(webServerPort, nil)

	return colorUpdate, nil
}

//takes the color output and tells the network
func sendColor(server *net.UDPConn, colorUpdate chan []byte) {
	// set FPS limit
	for {
		color := <-colorUpdate
		os.Stdout.Write(color)
		for _, client := range clients {
			_, err := server.WriteToUDP(color, client)
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
