package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	ws2811 "github.com/rpi-ws281x/rpi-ws281x-go"
)

var (
	brightness                    = 255 //0-255
	ledCount                      = 63
	renderFunc       func(uint32) = setStaticLeds
	rotate                        = false
	offset                        = 0.0
	runningchunkSize int
)

var ledController *ws2811.WS2811
var leds []uint32

func main() {
	flag.IntVar(&ledCount, "ledCount", ledCount, "number of leds in the strip connected")
	flag.IntVar(&brightness, "brightness", brightness, "Max brightness of the leds")
	flag.Func("renderType", "static\nrunning[=spinning][=center][=#]\n(# is the chunk size of the pattern and if # omitted is equal to ledCount)\n", parseRenderType)
	flag.Parse()

	log.Println("\trunningchunkSize", runningchunkSize)
	log.Println("\tledCount", ledCount)
	log.Println("\tspinning", rotate)

	opt := ws2811.DefaultOptions
	opt.Channels[0].Brightness = brightness
	opt.Channels[0].LedCount = ledCount

	var err error
	ledController, err = ws2811.MakeWS2811(&opt)
	checkError(err)
	checkError(ledController.Init())
	defer ledController.Fini()

	leds = ledController.Leds(0)

	renderLoop()
}

func Index(vs []string, t string) int {
	for i, v := range vs {
		if v == t {
			return i
		}
	}
	return -1
}

func Contains(vs []string, t string) bool {
	return Index(vs, t) >= 0
}

func parseRenderType(renderType string) error {
	renderParams := strings.Split(renderType, "=")
	runningchunkSize = ledCount
	if Contains(renderParams, "static") {
		renderFunc = setStaticLeds
		log.Println("Static Leds")
		return nil
	}
	if Contains(renderParams, "running") {
		renderFunc = setRunningLeds
		log.Print("Running Leds")
	}
	if centerIdx := Index(renderParams, "center"); centerIdx >= 0 {
		log.Print("\t(Center)")
		renderFunc = setRunningCenterLeds
	}
	// if the last value is a number then thats that chunk size
	if num, err := strconv.Atoi(renderParams[len(renderParams)-1]); err == nil {
		runningchunkSize = num
	}
	if Contains(renderParams, "spinning") {
		rotate = true
	}
	return nil
}

func rotateLeds() {
	if rotate {
		offset += 0.1
	}
}

func setLeds(color uint32) {
	renderFunc(color)
	rotateLeds()
}

func setStaticLeds(color uint32) {
	for i := 0; i < ledCount; i++ {
		leds[i] = color
	}
}

func setRunningLeds(color uint32) {
	//shift leds and set new color at beginning
	for i := runningchunkSize - 1; i > 0; i-- {
		leds[mod(i+int(offset), ledCount)] = leds[mod((i-1)+int(offset), ledCount)]
	}
	leds[int(offset)%ledCount] = color

	//duplicate for the reset of the leds
	for i := runningchunkSize; i < ledCount; i++ {
		chunkPos := mod(i, runningchunkSize)
		leds[mod(i+int(offset), ledCount)] = leds[mod(chunkPos+int(offset), ledCount)]
	}
}

func setRunningCenterLeds(color uint32) {
	//shift leds and set new color at center
	for i := 0; i < runningchunkSize/2; i++ {
		leds[mod(i+int(offset), ledCount)] = leds[mod(i+1+int(offset), ledCount)]
	}
	for i := runningchunkSize - 1; i > runningchunkSize/2; i-- {
		leds[mod(i+int(offset), ledCount)] = leds[mod(i-1+int(offset), ledCount)]
	}
	leds[mod((runningchunkSize/2)+int(offset), ledCount)] = color

	//duplicate for the reset of the leds
	for i := runningchunkSize; i < ledCount; i++ {
		chunkPos := mod(i, runningchunkSize)
		leds[mod(i+int(offset), ledCount)] = leds[mod(chunkPos+int(offset), ledCount)]
	}
}

func renderLoop() {
	log.Println("Visualizing")
	count := 0
	rgbColor := make([]uint8, 3)
	for {
		os.Stdin.Read(rgbColor)
		if count%100 == 0 {
			intColor := uint32(rgbColor[0])*256*256 + uint32(rgbColor[1])*256 + uint32(rgbColor[2])
			setLeds(intColor)
			checkError(ledController.Render())
		}
	}
}

func mod(a, b int) int {
	return int(math.Mod(float64(a), float64(b)))
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
