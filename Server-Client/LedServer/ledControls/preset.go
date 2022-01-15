package main

import (
	"math"
	"math/rand"
)

var tempLeds []uint32 = make([]uint32, ledCount)

// random colored speckles that blink in and fade smoothly (default 0.1)
func confetti(prob float64) {
	if prob == 0 {
		prob = 0.1
	}
	if rand.Float64() >= prob {
		return
	}
	fadeToBlackBy(0.1)
	pos := rand.Intn(ledCount)
	randPresetHue := rotateColor(presetHue, rand.Float64()*64)
	leds[pos] = RGBToInt(randPresetHue[0], randPresetHue[1], randPresetHue[2])
}

// sine wave (default 45)
func sinelon(bpm float64) {
	if bpm == 0 {
		bpm = 45
	}
	// a colored dot sweeping back and forth, with fading trails
	fadeToBlackBy(0.1)
	pos := beatsin16(bpm, 0, float64(ledCount)-1)
	leds[pos] |= RGBToInt(presetHue[0], presetHue[1], presetHue[2])
}

// 8 sine waves (default 45)
func juggle(bpm float64) {
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
		leds[pos] |= RGBToInt(shiftedPresetHue[0], shiftedPresetHue[1], shiftedPresetHue[2])
	}
}

// takes exactly 3 values and generates the pixel values
func populateTempLeds(hue1, hue2, hue3 float64) {
	c1r, c1g, c1b := HSLToRGB(hue1, 1, 0.5)
	color1 := []float64{c1r, c1g, c1b}
	c2r, c2g, c2b := HSLToRGB(hue2, 1, 0.5)
	color2 := []float64{c2r, c2g, c2b}
	c3r, c3g, c3b := HSLToRGB(hue3, 1, 0.5)
	color3 := []float64{c3r, c3g, c3b}

	//unixmillis := millis()
	chunkSize := float64(ledCount / 3)
	var i float64
	for i = 0; i <= chunkSize; i++ {
		weight1 := 1.0 - (i / chunkSize)
		weight2 := i / chunkSize
		newR := uint32(color1[0]*weight1) | uint32(color2[0]*weight2)
		newG := uint32(color1[1]*weight1) | uint32(color2[1]*weight2)
		newB := uint32(color1[2]*weight1) | uint32(color2[2]*weight2)
		tempLeds[int(i)] = RGBToInt(float64(newR), float64(newG), float64(newB))
	}

	for i = chunkSize; i <= chunkSize*2; i++ {
		weight1 := 1.0 - (i / chunkSize)
		weight2 := i / chunkSize
		newR := uint32(color3[0]*weight1) | uint32(color2[0]*weight2)
		newG := uint32(color3[1]*weight1) | uint32(color2[1]*weight2)
		newB := uint32(color3[2]*weight1) | uint32(color2[2]*weight2)
		tempLeds[int(i)] = RGBToInt(float64(newR), float64(newG), float64(newB))
	}

	for i = chunkSize * 2; i < float64(ledCount); i++ {
		weight1 := 1.0 - (i / chunkSize)
		weight2 := i / chunkSize
		newR := uint32(color3[0]*weight1) | uint32(color1[0]*weight2)
		newG := uint32(color3[1]*weight1) | uint32(color1[1]*weight2)
		newB := uint32(color3[2]*weight1) | uint32(color1[2]*weight2)
		tempLeds[int(i)] = RGBToInt(float64(newR), float64(newG), float64(newB))
	}
}

// spin the led strip at the given rpm
func rotateLeds(rpm float64) {
	if rpm == 0 {
		rpm = 45
	}
	offset := math.Mod(beat16(rpm*float64(ledCount)), float64(ledCount))
	var arr1, arr2 []uint32 = make([]uint32, len(tempLeds[int(offset):])), make([]uint32, len(tempLeds[:int(offset)]))
	copy(arr1, tempLeds[int(offset):])
	copy(arr2, tempLeds[:int(offset)])
	for i, j := 0, len(arr1)-1; i < j; i, j = i+1, j-1 {
		arr1[i], arr1[j] = arr1[j], arr1[i]
	}
	for i, j := 0, len(arr2)-1; i < j; i, j = i+1, j-1 {
		arr2[i], arr2[j] = arr2[j], arr2[i]
	}
	leds = append(arr2, arr1...)
}
