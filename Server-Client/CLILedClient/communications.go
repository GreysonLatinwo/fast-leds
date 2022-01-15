package main

import (
	"net"
	"os"
)

func main() {
	pc, err := net.ListenPacket("udp4", ":1234")
	if err != nil {
		panic(err)
	}
	defer pc.Close()

	buf := make([]byte, 4)
	for {
		pc.ReadFrom(buf)
		os.Stdout.Write(buf)
	}
}
