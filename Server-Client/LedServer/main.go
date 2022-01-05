package main

import (
	"flag"
	"fmt"
	"net/http"
)

var webServerPort *uint

// takes audio stream, analyses the audio and writes the output to color
func main() {
	webServerPort = flag.Uint("port", 9001, "The port that the webserver will listem on.")
	flag.Parse()

	go StartPulseAudio()
	udpClients, err := InitComms()
	chkFatal(err)
	udpClients <- []byte{0, 0, 0}
	go ProcessAudioStream(pulse, udpClients)
	chkPrint(http.ListenAndServe(fmt.Sprintf(":%d", *webServerPort), nil))
}
