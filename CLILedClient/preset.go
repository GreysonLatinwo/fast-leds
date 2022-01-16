package main

import (
	"math"
	"math/rand"
)

// random colored speckles that blink in and fade smoothly (default 0.1)
func confetti(args []float64) {
	fadeToBlackBy(0.2)
	pos := rand.Intn(ledCount)
	randPresetHue := rotateColor(presetHue, rand.Float64()*64)
	leds[pos] = RGBToInt(randPresetHue[0], randPresetHue[1], randPresetHue[2])
}

// sine wave (default 45)
func sinelon(args []float64) {
	var bpm float64 = args[0]
	if bpm == 0 {
		bpm = 45
	}
	// a colored dot sweeping back and forth, with fading trails
	fadeToBlackBy(0.1)
	pos := beatsin16(bpm, 0, float64(ledCount)-1)
	leds[pos] = RGBToInt(presetHue[0], presetHue[1], presetHue[2])
}

// 8 sine waves (default 45)
func juggle(args []float64) {
	var bpm float64 = args[0]
	if bpm == 0 {
		bpm = 45
	}
	// a colored dot sweeping back and forth, with fading trails
	fadeToBlackBy(0.1)
	var i float64 = 0
	var numJuggles float64 = 8
	for ; i < numJuggles; i += 1 {
		colorOffset := (360 / numJuggles) * i
		pos := beatsin16(bpm+(i+7), 0, float64(ledCount)-1)
		shiftedPresetHue := rotateColor(presetHue, colorOffset)
		leds[pos] = RGBToInt(shiftedPresetHue[0], shiftedPresetHue[1], shiftedPresetHue[2])
	}
}

func rotatingHues(args []float64) {
	var hue1, hue2, hue3 float64 = args[0], args[1], args[2]
	var bps float64 = args[3]
	if bps == 0 {
		bps = 6
	}
	bpm := bps / 60
	//build rgb colors
	c1r, c1g, c1b := HSLToRGB(hue1, 1, 0.5)
	color1 := []float64{c1r, c1g, c1b}
	c2r, c2g, c2b := HSLToRGB(hue2, 1, 0.5)
	color2 := []float64{c2r, c2g, c2b}
	c3r, c3g, c3b := HSLToRGB(hue3, 1, 0.5)
	color3 := []float64{c3r, c3g, c3b}

	offset := math.Mod(beat16(bpm*float64(ledCount)), float64(ledCount))
	palette := [][]float64{color1, color2, color3}
	for i := 0; i < ledCount; i++ {
		r, g, b := paletteLookup(palette, offset+(float64(i)/float64(ledCount)))
		//color = fancy.gamma_adjust(color, brightness=1.0)
		leds[i] = RGBToInt(r, g, b)
	}
}
