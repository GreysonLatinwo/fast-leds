package main

import (
	"flag"
	"fmt"
	"net/http"
)

var webServerPort *uint

// takes audio stream, analyses the audio and writes the output to color
func main() {
	startupSetting := *flag.Uint("startup", 0, "Startup settings")
	webServerPort = flag.Uint("port", 9001, "The port that the webserver will listem on.")
	flag.Parse()

	switch startupSetting {
	case 0: //off
		ledCommPipe <- [6]byte{2, 0, 0, 0, 0, 0}
	case 1: //listen to music
		go ProcessAudioStream()
	case 2: // set statc color
		ledCommPipe <- [6]byte{1, 255, 0, 0, 0, 0}
	case 3: // THE Preset
		ledCommPipe <- [6]byte{3, 4, 128, 198, 117, 0}
	}
	//start pulse audio callback listener
	go StartPulseAudio()
	//init webserver and start start listening led writes
	chkFatal(StartComms())
	//start web server
	http.ListenAndServe(fmt.Sprintf(":%d", *webServerPort), nil)
}
