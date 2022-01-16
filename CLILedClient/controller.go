package main

import (
	"flag"
	"log"
	"net"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	ws2811 "github.com/rpi-ws281x/rpi-ws281x-go"
)

var (
	brightness                    = 255 //0-255
	ledCount                      = 148
	renderFunc       func(uint32) = setStaticLeds
	rotate                        = false
	offset                        = 0.0
	runningchunkSize int

	isPresetRunning bool = false
)

var ledController *ws2811.WS2811
var leds []uint32

func parseRenderType(renderType string) error {
	renderParams := strings.Split(renderType, "-")
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
		log.Print("\tCenter")
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

func main() {
	flag.IntVar(&ledCount, "c", ledCount, "number of leds in the strip connected")
	flag.IntVar(&brightness, "b", brightness, "Max brightness of the leds")
	flag.Func("r", "Render Type (default static)\nstatic\nrunning[-spinning][-center][-#]\n(# is the chunk size of the pattern and if # omitted is equal to ledCount)\n", parseRenderType)
	flag.Parse()

	renderTypeName := runtime.FuncForPC(reflect.ValueOf(renderFunc).Pointer()).Name()
	log.Println("RenderType", renderTypeName)
	log.Println("\tledCount", ledCount)
	log.Println("\tBrightness", brightness)
	log.Println("\trunningchunkSize", runningchunkSize)
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
	go rotatePresetHue()
	renderLoop()
}

func setLeds(color uint32) {
	renderFunc(color)
	if rotate {
		offset += 0.1
	}
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

func runPreset(presetData []uint8, killPreset chan struct{}, presetDone chan struct{}) {
	var presetFunc func([]float64) = confetti
	arg := make([]float64, 5)
	fps := 150
	switch presetData[1] {
	case 0x1: // confetti
		fps = 20
		presetFunc = confetti
		arg[0] = float64(presetData[5] / 255)
	case 0x2: // sinelon
		presetFunc = sinelon
		arg[0] = float64(presetData[5] / 255)
	case 0x3: // juggle
		presetFunc = juggle
		arg[0] = float64(presetData[5] / 255)
	case 0x4: // spinning hue
		arg[0], arg[1], arg[2], arg[3] = float64(presetData[2])/255.0, float64(presetData[3])/255.0, float64(presetData[4])/255.0, float64(presetData[5])
		presetFunc = rotatingHues
	default: // invalid
		log.Println("not valid preset")
		return
	}
	//set fps
	isPresetRunning = true
	ticker := time.NewTicker(time.Second / time.Duration(fps))

	for {
		<-ticker.C
		select {
		case <-ticker.C:
			presetFunc(arg)
			ledController.Render()
		case <-killPreset:
			presetDone <- struct{}{}
			isPresetRunning = false
			return
		}
	}
}

func renderLoop() {
	pc, err := net.ListenPacket("udp4", ":1234")
	if err != nil {
		panic(err)
	}
	defer pc.Close()
	log.Println("Visualizing")
	renderData := make([]uint8, 6)
	killPreset := make(chan struct{})
	presetDone := make(chan struct{})
	for {
		pc.ReadFrom(renderData)
		// select led display type
		switch renderData[0] {
		case 0x1: // running
			if isPresetRunning {
				killPreset <- struct{}{}
				<-presetDone
			}
			intColor := RGBToInt(float64(renderData[1]), float64(renderData[2]), float64(renderData[3]))
			setLeds(intColor)
		case 0x2: // static
			if isPresetRunning {
				killPreset <- struct{}{}
				<-presetDone
			}
			intColor := RGBToInt(float64(renderData[1]), float64(renderData[2]), float64(renderData[3]))
			setStaticLeds(intColor)
		case 0x3: // preset
			if isPresetRunning {
				killPreset <- struct{}{}
				<-presetDone
			}
			go runPreset(renderData, killPreset, presetDone)
		}
		checkError(ledController.Render())
	}
}
