package main

import (
	"net"
	"os"
)

func main() {
	pc, err := net.ListenPacket("udp4", ":9999")
	if err != nil {
		panic(err)
	}
	defer pc.Close()

	buf := make([]byte, 3)
	for {
		pc.ReadFrom(buf)
		os.Stdout.Write(buf)
	}
}
