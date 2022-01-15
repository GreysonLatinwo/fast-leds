package main

import (
	"math/rand"
)

func confetti() {
	// random colored speckles that blink in and fade smoothly
	fadeToBlackBy(0.1)
	pos := rand.Intn(ledCount)
	randPresetHue := rotateColor(presetHue, rand.Float64()*64)
	leds[pos] = RGBToInt(randPresetHue[0], randPresetHue[1], randPresetHue[2])
}

func sinelon() {
	// a colored dot sweeping back and forth, with fading trails
	fadeToBlackBy(0.1)
	pos := beatsin16(45, 0, float64(ledCount)-1)
	leds[pos] |= RGBToInt(presetHue[0], presetHue[1], presetHue[2])
}

func juggle() {
	// a colored dot sweeping back and forth, with fading trails
	fadeToBlackBy(0.1)
	bpm := float64(45)
	var i float64 = 0
	var numJuggles float64 = 8
	for ; i < numJuggles; i += 1 {
		colorOffset := (360 / numJuggles) * i
		pos := beatsin16(bpm+(i+7), 0, float64(ledCount)-1)
		shiftedPresetHue := rotateColor(presetHue, colorOffset)
		leds[pos] |= RGBToInt(shiftedPresetHue[0], shiftedPresetHue[1], shiftedPresetHue[2])
	}
}
