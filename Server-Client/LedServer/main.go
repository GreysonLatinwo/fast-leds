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

	//start pulse audio callback listener
	go StartPulseAudio()
	//init webserver and start start listening led writes
	chkFatal(StartComms())
	//start listen for audio
	go ProcessAudioStream()
	//start web server
	chkPrint(http.ListenAndServe(fmt.Sprintf(":%d", *webServerPort), nil))
}
