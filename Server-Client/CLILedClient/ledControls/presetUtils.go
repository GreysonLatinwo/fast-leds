package main

import (
	"math"
	"time"
)

var presetHue []float64 = []float64{255, 0, 0}

func spinPresetHue() {
	t := time.NewTicker(time.Duration(time.Second / 30))
	for {
		//<-t.C
		// rotDeg += 0.001
		// r, g, b := HSLToRGB(rotDeg, 1, 0.5)
		// presetHue[0], presetHue[1], presetHue[2] = r, g, b
		// if rotDeg >= 1 {
		// 	rotDeg = 0
		// }
		var r, g, b float64 = 255, 0, 0
		for g = 0; g <= 255; g++ {
			<-t.C
			presetHue[1] = g
		}
		for r = 255; r >= 0; r-- {
			<-t.C
			presetHue[0] = r
		}
		for b = 0; b <= 255; b++ {
			<-t.C
			presetHue[2] = b
		}
		for g = 255; g >= 0; g-- {
			<-t.C
			presetHue[1] = g
		}
		for r = 0; r <= 255; r++ {
			<-t.C
			presetHue[0] = r
		}
		for b = 255; b >= 0; b-- {
			<-t.C
			presetHue[2] = b
		}

	}
}

// fadeby is a fraction (0, 1)
func fadeToBlackBy(_fadeby float64) {
	fadeby := 1 - _fadeby
	for i := range leds {
		r, g, b := IntToRGB(leds[i])
		r = r * fadeby
		g = g * fadeby
		b = b * fadeby
		leds[i] = RGBToInt(r, g, b)
	}
}

//returns time since jan 1 1970
func GET_MILLIS() float64 {
	return float64(time.Now().UnixMilli())
}

func beat16(BPM float64) float64 {
	millis := GET_MILLIS()
	return (millis * BPM) / 60000
}

func beatsin16(BPM, lowest, highest float64) uint16 {
	var beat float64 = beat16(BPM)
	var beatsin float64 = (math.Sin(beat) * 32767.0) + 32768
	var rangewidth float64 = highest - lowest
	var scaledbeat float64 = (beatsin * rangewidth / 65535.0)
	var result uint16 = uint16(lowest + scaledbeat)
	return result
}
