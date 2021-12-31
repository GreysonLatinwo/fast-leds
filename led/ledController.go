package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"time"

	ws2811 "github.com/rpi-ws281x/rpi-ws281x-go"
)

const (
	brightness = 96 //max is 255
	width      = 8
	height     = 1
	ledCounts  = width * height
)

const FPS = 15

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

	ledController.Leds(0)[0] = 0xff0000

	go readAudioAnalyserLoop()
	//go visualizerLoop()
	for {
		time.Sleep(time.Second)
	}
}

func readAudioAnalyserLoop() {

	stdinReader := os.Stdin

	for {
		asdf := make([]byte, 9)
		stdinReader.Read(asdf)
		binary.Read(stdinReader, binary.LittleEndian, asdf)
		fmt.Println(asdf)
	}
}

func VisualizerLoop() {
	fmt.Println("Visualizing")
	for {
		ledController.Leds(0)[0] = fftAudioColor
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
