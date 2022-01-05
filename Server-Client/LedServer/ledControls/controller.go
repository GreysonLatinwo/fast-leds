package main

import (
	"fmt"
	"log"
	"os"

	ws2811 "github.com/rpi-ws281x/rpi-ws281x-go"
)

const (
	brightness = 255 //max is 255
	width      = 21
	height     = 1
	ledCounts  = width * height
)

var ledController *ws2811.WS2811

func main() {
	opt := ws2811.DefaultOptions
	opt.Channels[0].Brightness = brightness
	opt.Channels[0].LedCount = ledCounts

	var err error
	ledController, err = ws2811.MakeWS2811(&opt)
	checkError(err)

	checkError(ledController.Init())
	defer ledController.Fini()

	visualizerLoop()
}

func visualizerLoop() {
	log.Println("Visualizing")
	rgbColor := make([]uint8, 3)
	for {
		os.Stdin.Read(rgbColor)
		intColor := uint32(rgbColor[0])*256*256 + uint32(rgbColor[1])*256 + uint32(rgbColor[2])
		for i := 0; i < ledCounts; i++ {
			ledController.Leds(0)[i] = intColor
		}
		checkError(ledController.Render())
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
