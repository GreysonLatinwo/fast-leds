package main

import (
	"flag"
	"log"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	ws2811 "github.com/rpi-ws281x/rpi-ws281x-go"

	utils "github.com/greysonlatinwo/fast-led/LedServer/utils"
)

var (
	brightness                    = 200 //0-255
	ledCount                      = 21
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
	if utils.Contains(renderParams, "static") {
		renderFunc = setStaticLeds
		return nil
	}
	if utils.Contains(renderParams, "running") {
		renderFunc = setRunningLeds
	}
	if centerIdx := utils.Index(renderParams, "center"); centerIdx >= 0 {
		renderFunc = setRunningCenterLeds
	}
	// if the last value is a number then thats that chunk size
	if num, err := strconv.Atoi(renderParams[len(renderParams)-1]); err == nil {
		runningchunkSize = num
	}
	if utils.Contains(renderParams, "spinning") {
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
	utils.CheckError(err)
	utils.CheckError(ledController.Init())
	defer ledController.Fini()

	leds = ledController.Leds(0)
	go rotatePresetHue(60)
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
		leds[utils.Mod(i+int(offset), ledCount)] = leds[utils.Mod((i-1)+int(offset), ledCount)]
	}
	leds[int(offset)%ledCount] = color

	//duplicate for the reset of the leds
	for i := runningchunkSize; i < ledCount; i++ {
		chunkPos := utils.Mod(i, runningchunkSize)
		leds[utils.Mod(i+int(offset), ledCount)] = leds[utils.Mod(chunkPos+int(offset), ledCount)]
	}
}

func setRunningCenterLeds(color uint32) {
	//shift leds and set new color at center
	for i := 0; i < runningchunkSize/2; i++ {
		leds[utils.Mod(i+int(offset), ledCount)] = leds[utils.Mod(i+1+int(offset), ledCount)]
	}
	for i := runningchunkSize - 1; i > runningchunkSize/2; i-- {
		leds[utils.Mod(i+int(offset), ledCount)] = leds[utils.Mod(i-1+int(offset), ledCount)]
	}
	leds[utils.Mod((runningchunkSize/2)+int(offset), ledCount)] = color

	//duplicate for the reset of the leds
	for i := runningchunkSize; i < ledCount; i++ {
		chunkPos := utils.Mod(i, runningchunkSize)
		leds[utils.Mod(i+int(offset), ledCount)] = leds[utils.Mod(chunkPos+int(offset), ledCount)]
	}
}

func renderLoop() {
	log.Println("Visualizing")
	renderData := make([]uint8, 6)
	killPreset := make(chan struct{})
	presetDone := make(chan struct{})
	presetFPS := 150
	presetArgs := make([]float64, 6)
	presetFunc := confetti
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
				presetFunc(presetArgs)
				ledController.Render()
			case <-killPreset:
				return
			}
		}
	}
	//read stdin data
	go func() {
		for {
			os.Stdin.Read(renderData)
		}
	}()
	for {
		// select led display type
		switch renderData[0] {
		case 0x1: // running
			if isPresetRunning {
				killPreset <- struct{}{}
				<-presetDone
			}
			intColor := utils.RGBToInt(float64(renderData[1]), float64(renderData[2]), float64(renderData[3]))
			setLeds(intColor)
		case 0x2: // static
			if isPresetRunning {
				killPreset <- struct{}{}
				<-presetDone
			}
			intColor := utils.RGBToInt(float64(renderData[1]), float64(renderData[2]), float64(renderData[3]))
			setStaticLeds(intColor)
		case 0x3: // confetti
			if isPresetRunning {
				killPreset <- struct{}{}
				<-presetDone
			}
			presetFPS = 50
			presetFunc = confetti
			presetArgs[0] = float64(renderData[5]) / 255
			go runPreset()
		case 0x4: // sinelon
			if isPresetRunning {
				killPreset <- struct{}{}
				<-presetDone
			}
			presetFunc = sinelon
			presetArgs[0] = float64(renderData[5]) / 255
			go runPreset()
		case 0x5: // juggle
			if isPresetRunning {
				killPreset <- struct{}{}
				<-presetDone
			}
			presetFunc = juggle
			presetArgs[0] = float64(renderData[5]) / 255
			go runPreset()
		case 0x6: // spinning hue
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
			presetFunc = rotatingHues
			go runPreset()
		}
		ledController.Render()
	}
}