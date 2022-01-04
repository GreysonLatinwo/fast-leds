package main

import (
	"fmt"
	"log"
	"os"
	"time"

	ws2811 "github.com/rpi-ws281x/rpi-ws281x-go"
)

const (
	brightness = 255 //max is 255
	width      = 63
	height     = 1
	ledCounts  = width * height
)

const FPS = 30

var ledController *ws2811.WS2811

var fftAudioColor uint32 = 256 * 256 * 255

func main() {
	opt := ws2811.DefaultOptions
	opt.Channels[0].Brightness = brightness
	opt.Channels[0].LedCount = ledCounts

	var err error
	ledController, err = ws2811.MakeWS2811(&opt)
	checkError(err)

	checkError(ledController.Init())
	defer ledController.Fini()

	go readAudioAnalyserLoop()
	go visualizerLoop()
	for {
		time.Sleep(time.Second)
	}
}

func readAudioAnalyserLoop() {
	for {
		rgbColor := make([]uint8, 3)
		os.Stdin.Read(rgbColor)
		fftAudioColor = uint32(rgbColor[0])*256*256 + uint32(rgbColor[1])*256 + uint32(rgbColor[2])
	}
}

func setLeds(color uint32) {
	for i := 0; i < ledCounts; i++ {
		ledController.Leds(0)[i] = color
	}
}

func visualizerLoop() {
	log.Println("Visualizing")
	for {
		setLeds(fftAudioColor)
		checkError(ledController.Render())
		time.Sleep(time.Second / FPS)
	}
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
		panic(err)
	}
}
