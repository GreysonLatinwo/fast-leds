package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		Port: 1234,
		IP:   net.ParseIP("192.168.0.36"),
	})
	if err != nil {
		fmt.Printf("Some error %v\n", err)
		return
	}

	conn.Write([]byte{0})

	buf := make([]byte, 3)
	for {
		conn.ReadFromUDP(buf)
		os.Stdout.Write(buf)
	}
}
