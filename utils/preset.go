package utils

import (
	"math"
	"math/rand"
)

// random colored speckles that blink in and fade smoothly (default 0.1)
func Confetti(leds []uint32, args []float64) {
	ledCount := len(leds)
	FadeToBlackBy(leds, 0.1)
	pos := rand.Intn(ledCount)
	randPresetHue := RotateColor(presetHue, rand.Float64()*64)
	leds[pos] = RGBToInt(randPresetHue[0], randPresetHue[1], randPresetHue[2])
}

// sine wave (default 45)
func Sinelon(leds []uint32, args []float64) {
	ledCount := len(leds)
	var bpm float64 = args[0]
	if bpm == 0 {
		bpm = 45
	}
	// a colored dot sweeping back and forth, with fading trails
	FadeToBlackBy(leds, 0.1)
	pos := beatsin16(bpm, 0, float64(ledCount)-1)
	leds[pos] = RGBToInt(presetHue[0], presetHue[1], presetHue[2])
}

// 8 sine waves (default 45)
func Juggle(leds []uint32, args []float64) {
	ledCount := len(leds)
	var bpm float64 = args[0]
	if bpm == 0 {
		bpm = 45
	}
	// a colored dot sweeping back and forth, with fading trails
	FadeToBlackBy(leds, 0.1)
	var i float64 = 0
	var numJuggles float64 = 8
	for ; i < numJuggles; i += 1 {
		colorOffset := (360 / numJuggles) * i
		pos := beatsin16(bpm+(i+7), 0, float64(ledCount)-1)
		shiftedPresetHue := RotateColor(presetHue, colorOffset)
		leds[pos] = RGBToInt(shiftedPresetHue[0], shiftedPresetHue[1], shiftedPresetHue[2])
	}
}

func RotatingHues(leds []uint32, args []float64) {
	ledCount := len(leds)
	var hue1, hue2, hue3 float64 = args[0], args[1], args[2]
	var brightness float64 = args[3]
	bps := 2.0

	//build rgb colors
	c1r, c1g, c1b := HueToRGB(hue1, brightness)
	color1 := []float64{c1r, c1g, c1b}
	c2r, c2g, c2b := HueToRGB(hue2, brightness)
	color2 := []float64{c2r, c2g, c2b}
	c3r, c3g, c3b := HueToRGB(hue3, brightness)
	color3 := []float64{c3r, c3g, c3b}

	offset := math.Mod(beat16(bps), float64(ledCount))
	palette := [][]float64{color1, color2, color3}
	for i := 0; i < ledCount; i++ {
		r, g, b := paletteLookup(palette, offset+(float64(i)/float64(ledCount)))
		//color = fancy.gamma_adjust(color, brightness=1.0)
		leds[i] = RGBToInt(r, g, b)
	}
}

// leds list
//
// args [0, 1]
//
// args = []float64{bpm, hue_1, hue_2,..., hue_n, brightness_1, brightness_2...brightness_n}
func RotatingColors(leds []uint32, args []float64) {
	bpm := args[0]
	hues := args[1:5]
	brightnesses := args[5:]
	ledCount := len(leds)
	colors := make([]uint32, 0)

	for i := range hues {
		r, g, b := HueToRGB(hues[i], brightnesses[i])
		colors = append(colors, RGBToInt(r, g, b))
	}

	offset := int(math.Mod(beat16(bpm*60), float64(ledCount)))

	colorCount := len(colors)
	coloridx := 0
	for i := 0; i < ledCount; {
		for j := i; i-j < 8; i++ {
			pos := (i + offset) % ledCount
			leds[pos] = colors[coloridx]
		}
		coloridx = (coloridx + 1) % colorCount
	}
}
