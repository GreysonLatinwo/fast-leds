package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	utils "github.com/greysonlatinwo/fast-leds/utils"
)

var webServerPort *uint

// takes audio stream, analyses the audio and writes the output to color
func main() {
	if os.Getuid() == 0 {
		fmt.Println("Run as non-root Pls :)")
		return
	}

	startupSetting := *flag.Uint("startup", 0, "Startup settings")
	webServerPort = flag.Uint("port", 9001, "The port that the webserver will listen on.")
	flag.Parse()

	switch startupSetting {
	case 0: //off
		ledCommPipe <- [6]byte{2, 0, 0, 0, 0, 0}
	case 1: //listen to music
		go ProcessAudioStream()
	}
	//start pulse audio callback listener
	go StartPulseAudio()
	//init webserver and start start listening led writes
	utils.ChkFatal(StartComms())
	//start web server
	http.ListenAndServe(fmt.Sprintf(":%d", *webServerPort), nil)
}
