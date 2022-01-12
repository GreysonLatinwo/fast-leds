package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	ws2811 "github.com/rpi-ws281x/rpi-ws281x-go"
)

var (
	brightness                    = 255 //0-255
	ledCount                      = 150
	renderFunc       func(uint32) = setStaticLeds
	rotate                        = false
	offset                        = 0.0
	runningchunkSize int
)

var ledController *ws2811.WS2811
var leds []uint32

func main() {
	flag.IntVar(&ledCount, "c", ledCount, "number of leds in the strip connected")
	flag.IntVar(&brightness, "b", brightness, "Max brightness of the leds")
	flag.Func("r", "Render Type (default static)\nstatic\nrunning[-spinning][-center][-#]\n(# is the chunk size of the pattern and if # omitted is equal to ledCount)\n", parseRenderType)
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
	go spinPresetHue()
	renderLoop()
}

var rotDeg float64 = 0.0
var presetHue []float64 = []float64{255, 0, 0}

func spinPresetHue() {
	for {
		rotDeg += 0.1
		_presetHue := rotateColor(presetHue, rotDeg)
		presetHue[0], presetHue[1], presetHue[2] = clamp(_presetHue[0]), clamp(_presetHue[1]), clamp(_presetHue[2])
		if rotDeg >= 360 {
			rotDeg = 0
		}
	}
}

func confetti() {
	// random colored speckles that blink in and fade smoothly
	fadeToBlackBy(20)
	pos := rand.Intn(ledCount)
	randPresetHue := rotateColor(presetHue, rand.Float64()*64)
	leds[pos] = RGBToInt(uint8(randPresetHue[0]), uint8(randPresetHue[1]), uint8(randPresetHue[2]))
}

func clamp(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 255 {
		return 255
	}
	return x
}

func fadeToBlackBy(fadeby float64) {

	for i := range leds {
		r, g, b := IntToRGB(leds[i])
		r, g, b = uint8(clamp(float64(r)-fadeby)), uint8(clamp(float64(g)-fadeby)), uint8(clamp(float64(b)-fadeby))

		leds[i] = RGBToInt(r, g, b)
	}
}

func RGBToInt(r, g, b uint8) uint32 {
	return uint32(r)<<16 + uint32(g)<<8 + uint32(b)
}

func IntToRGB(x uint32) (uint8, uint8, uint8) {
	red := uint8(x >> 16)
	green := uint8(x >> 8)
	blue := uint8(x)
	return red, green, blue
}

// hue [0,1]
func hueToRGB(v1, v2, h float64) float64 {
	if h < 0 {
		h += 1
	}
	if h > 1 {
		h -= 1
	}
	switch {
	case 6*h < 1:
		return (v1 + (v2-v1)*6*h)
	case 2*h < 1:
		return v2
	case 3*h < 2:
		return v1 + (v2-v1)*((2.0/3.0)-h)*6
	}
	return v1
}

//all values [0,1]
func HSLToRGB(h, s, l float64) (uint8, uint8, uint8) {

	if s == 0 {
		// it's gray
		return uint8(l) * 255, uint8(l) * 255, uint8(l) * 255
	}

	var v1, v2 float64
	if l < 0.5 {
		v2 = l * (1 + s)
	} else {
		v2 = (l + s) - (s * l)
	}

	v1 = 2*l - v2

	r := hueToRGB(v1, v2, h+(1.0/3.0))
	g := hueToRGB(v1, v2, h)
	b := hueToRGB(v1, v2, h-(1.0/3.0))

	return uint8(r) * 255, uint8(g) * 255, uint8(b) * 255
}

// rotates rgb float value by degrees. https://flylib.com/books/2/816/1/html/2/files/fig11_14.jpeg
// rgb [0, 255], rotDeg [0, 360]
func rotateColor(rgb []float64, rotDeg float64) []float64 {
	if 0 >= rotDeg || rotDeg >= 360 {
		return rgb
	}

	pi := 3.14159265
	sqrtf := func(x float64) float64 {
		return math.Sqrt(x)
	}

	cosA := math.Cos(rotDeg * pi / 180) //convert degrees to radians
	sinA := math.Sin(rotDeg * pi / 180) //convert degrees to radians
	//calculate the rotation matrix, only depends on Hue
	matrix := [][]float64{{cosA + (1.0-cosA)/3.0, 1.0/3.0*(1.0-cosA) - sqrtf(1.0/3.0)*sinA, 1.0/3.0*(1.0-cosA) + sqrtf(1.0/3.0)*sinA},
		{1.0/3.0*(1.0-cosA) + sqrtf(1.0/3.0)*sinA, cosA + 1.0/3.0*(1.0-cosA), 1.0/3.0*(1.0-cosA) - sqrtf(1.0/3.0)*sinA},
		{1.0/3.0*(1.0-cosA) - sqrtf(1.0/3.0)*sinA, 1.0/3.0*(1.0-cosA) + sqrtf(1.0/3.0)*sinA, cosA + 1.0/3.0*(1.0-cosA)}}

	outf := make([]float64, 3)

	//Use the rotation matrix to convert the RGB directly
	outf[0] = rgb[0]*matrix[0][0] + rgb[1]*matrix[0][1] + rgb[2]*matrix[0][2]
	outf[1] = rgb[0]*matrix[1][0] + rgb[1]*matrix[1][1] + rgb[2]*matrix[1][2]
	outf[2] = rgb[0]*matrix[2][0] + rgb[1]*matrix[2][1] + float64(rgb[2])*matrix[2][2]
	return outf
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
	rgbColor := make([]uint8, 4)
	var presetFunc func() = nil
	stopPreset := make(chan bool)
	for {
		os.Stdin.Read(rgbColor)
		intColor := RGBToInt(rgbColor[1], rgbColor[2], rgbColor[3])
		if presetFunc != nil {
			stopPreset <- true
		}
		// select led display type
		switch rgbColor[0] {
		case 0x1: //running
			setLeds(intColor)
		case 0x2: //static
			setStaticLeds(intColor)
		case 0x3: //preset
			setStaticLeds(0)
			// select preset
			switch rgbColor[1] {
			case 0x1:
				presetFunc = confetti
			default:
				continue
			}
			go func() {
				//render at 60 fps
				ticker := time.NewTicker(time.Second / time.Duration(10))
				for {
					select {
					case <-ticker.C:
						confetti()
						ledController.Render()
					case stopping := <-stopPreset:
						if stopping {
							presetFunc = nil
							return
						}
					}
				}
			}()
		}
		checkError(ledController.Render())
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
