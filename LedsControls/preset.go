package main

import (
	"math"
	"math/rand"

	utils "github.com/greysonlatinwo/fast-leds/Utils"
)

// random colored speckles that blink in and fade smoothly (default 0.1)
func confetti(args []float64) {
	fadeToBlackBy(0.1)
	pos := rand.Intn(ledCount)
	randPresetHue := utils.RotateColor(presetHue, rand.Float64()*64)
	leds[pos] = utils.RGBToInt(randPresetHue[0], randPresetHue[1], randPresetHue[2])
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
	leds[pos] = utils.RGBToInt(presetHue[0], presetHue[1], presetHue[2])
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
		shiftedPresetHue := utils.RotateColor(presetHue, colorOffset)
		leds[pos] = utils.RGBToInt(shiftedPresetHue[0], shiftedPresetHue[1], shiftedPresetHue[2])
	}
}

func rotatingHues(args []float64) {
	var hue1, hue2, hue3 float64 = args[0], args[1], args[2]
	var brightness float64 = args[3]
	bps := 2.0

	//build rgb colors
	c1r, c1g, c1b := utils.HSLToRGB(hue1, 1, 0.5*brightness)
	color1 := []float64{c1r, c1g, c1b}
	c2r, c2g, c2b := utils.HSLToRGB(hue2, 1, 0.5*brightness)
	color2 := []float64{c2r, c2g, c2b}
	c3r, c3g, c3b := utils.HSLToRGB(hue3, 1, 0.5*brightness)
	color3 := []float64{c3r, c3g, c3b}

	offset := math.Mod(beat16(bps), float64(ledCount))
	palette := [][]float64{color1, color2, color3}
	for i := 0; i < ledCount; i++ {
		r, g, b := paletteLookup(palette, offset+(float64(i)/float64(ledCount)))
		//color = fancy.gamma_adjust(color, brightness=1.0)
		leds[i] = utils.RGBToInt(r, g, b)
	}
}
