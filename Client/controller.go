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

	"github.com/greysonlatinwo/fast-leds/utils"
	ws2811 "github.com/rpi-ws281x/rpi-ws281x-go"
)

var (
	brightness                                   = 160 //0-255
	ledCount                                     = 128
	renderFunc       func([]uint32, int, uint32) = utils.SetStaticLeds
	runningChunkSize float64
	isPresetRunning  bool = false
)

var ledController *ws2811.WS2811
var leds []uint32

func parseRenderType(renderType string) error {
	renderParams := strings.Split(renderType, "-")
	runningChunkSize = float64(ledCount)
	if utils.Contains(renderParams, "static") {
		renderFunc = utils.SetStaticLeds
		return nil
	}
	if utils.Contains(renderParams, "running") {
		renderFunc = utils.SetRunningLeds
	}
	if centerIdx := utils.Index(renderParams, "center"); centerIdx >= 0 {
		renderFunc = utils.SetRunningCenterLeds
	}
	// if the last value is a number then thats that chunk size
	if num, err := strconv.Atoi(renderParams[len(renderParams)-1]); err == nil {
		runningChunkSize = float64(num)
	}
	return nil
}

func main() {
	//set and read flags
	flag.IntVar(&ledCount, "c", ledCount, "number of leds in the strip connected")
	flag.IntVar(&brightness, "b", brightness, "Max brightness of the leds")
	flag.Func("r", "Render Type (default static)\nstatic\nrunning[-center][-#]\n(# is the chunk size of the pattern and if # omitted is equal to ledCount)\n", parseRenderType)
	flag.Parse()

	//log startup vars
	renderTypeName := runtime.FuncForPC(reflect.ValueOf(renderFunc).Pointer()).Name()
	log.Println("RenderType", renderTypeName)
	log.Println("\tledCount", ledCount)
	log.Println("\tBrightness", brightness)
	log.Println("\trunningChunkSize", runningChunkSize)

	//init led strip
	opt := ws2811.DefaultOptions
	opt.Channels[0].Brightness = brightness
	opt.Channels[0].LedCount = ledCount
	var err error
	ledController, err = ws2811.MakeWS2811(&opt)
	utils.CheckError(err)
	utils.CheckError(ledController.Init())
	defer ledController.Fini()

	leds = ledController.Leds(0)

	initMDNS("fast-leds")

	go utils.RotatePresetHue(60)
	renderLoop()
}

func renderLoop() {
	udpip, _ := net.ResolveUDPAddr("udp4", ":1234")
	pc, err := net.ListenUDP("udp4", udpip)
	if err != nil {
		panic(err)
	}
	defer pc.Close()
	pc.WriteToUDP([]byte{7, 10, 136, 210, 113, 0, 128, 128, 128, 0}, udpip)
	log.Println("Visualizing")
	renderData := make([]uint8, 10)
	killPreset := make(chan struct{})
	presetDone := make(chan struct{})
	presetFPS := 150
	presetArgs := make([]float64, 9)
	var presetFunc func([]uint32, []float64)
	runPreset := func() {
		isPresetRunning = true
		defer func() {
			isPresetRunning = false
			presetDone <- struct{}{}
		}()
		ticker := time.NewTicker(time.Second / time.Duration(presetFPS))
		for {
			<-ticker.C
			select {
			case <-ticker.C:
				presetFunc(leds, presetArgs)
				ledController.Render()
			case <-killPreset:
				return
			}
		}
	}
	isNewData := make(chan struct{})
	go func() {
		for {
			pc.Read(renderData)
			select {
			case isNewData <- struct{}{}:
			default:
			}
		}
	}()
	ticker := time.NewTicker(time.Duration(time.Second / 60))
	for {
		// if the data is music then dont wait for new data
		if renderData[0] != 0x1 {
			<-isNewData
		}
		<-ticker.C
		// select led display type
		switch renderData[0] {
		case 0x1: // running
			if isPresetRunning {
				killPreset <- struct{}{}
				<-presetDone
			}
			intColor := utils.RGBToInt(float64(renderData[1]), float64(renderData[2]), float64(renderData[3]))
			renderFunc(leds, int(runningChunkSize), intColor)
		case 0x2: // static
			log.Println("static", renderData)
			if isPresetRunning {
				killPreset <- struct{}{}
				<-presetDone
			}
			intColor := utils.RGBToInt(float64(renderData[1]), float64(renderData[2]), float64(renderData[3]))
			renderFunc(leds, ledCount, intColor)
		case 0x3: // confetti
			log.Println("confetti", renderData)
			if isPresetRunning {
				killPreset <- struct{}{}
				<-presetDone
			}
			presetFPS = 50
			presetFunc = utils.Confetti
			presetArgs[0] = float64(renderData[5]) / 255
			go runPreset()
		case 0x4: // sinelon
			log.Println("sinelon", renderData)
			if isPresetRunning {
				killPreset <- struct{}{}
				<-presetDone
			}
			presetFunc = utils.Sinelon
			presetArgs[0] = float64(renderData[5]) / 255
			go runPreset()
		case 0x5: // juggle
			log.Println("juggle", renderData)
			if isPresetRunning {
				killPreset <- struct{}{}
				<-presetDone
			}
			presetFunc = utils.Juggle
			presetArgs[0] = float64(renderData[5]) / 255
			go runPreset()
		case 0x6: // spinning hue
			log.Println("spinning hue", renderData)
			if isPresetRunning {
				killPreset <- struct{}{}
				<-presetDone
			}
			//color
			presetArgs[0] = float64(renderData[1]) / 255
			presetArgs[1] = float64(renderData[2]) / 255
			presetArgs[2] = float64(renderData[3]) / 255
			//brightness
			presetArgs[3] = float64(renderData[4]) / 255
			presetFunc = utils.RotatingHues
			go runPreset()
		case 0x7: // spinning colors
			log.Println("spinning colors", renderData)
			if isPresetRunning {
				killPreset <- struct{}{}
				<-presetDone
			}
			//bpm
			presetArgs[0] = float64(renderData[1])
			//hues
			presetArgs[1] = float64(renderData[2]) / 255
			presetArgs[2] = float64(renderData[3]) / 255
			presetArgs[3] = float64(renderData[4]) / 255
			presetArgs[4] = float64(renderData[5]) / 255
			//brightness
			presetArgs[5] = float64(renderData[6]) / 255
			presetArgs[6] = float64(renderData[7]) / 255
			presetArgs[7] = float64(renderData[8]) / 255
			presetArgs[8] = float64(renderData[9]) / 255
			presetFunc = utils.RotatingColors
			go runPreset()
		}
		ledController.Render()
	}
}
