package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	utils "github.com/greysonlatinwo/fast-led/LedServer/utils"
)

var webServerPort *uint

// takes audio stream, analyses the audio and writes the output to color
func main() {
	if os.Getgid() == 0 {
		os.Stderr.WriteString("Run as non-root Pls :)")
		return
	}
	startupSetting := *flag.Uint("startup", 1, "Startup settings")
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
	utils.ChkFatal(StartComms())
	//start web server
	http.ListenAndServe(fmt.Sprintf(":%d", *webServerPort), nil)
}
