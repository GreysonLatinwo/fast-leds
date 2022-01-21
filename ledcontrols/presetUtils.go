package ledcontrols

import (
	"math"
	"time"

	utils "github.com/greysonlatinwo/fast-leds/utils"
)

var presetHue []float64 = []float64{255, 0, 0}

// rotate the presetHue variable
// rps: rotations per seconds [0, Inf)
func RotatePresetHue(rps float64) {
	rpm := rps / 60
	t := time.NewTicker(time.Duration(time.Second / 30))
	for {
		<-t.C
		presetHue[0], presetHue[1], presetHue[2] = utils.HSLToRGB(math.Mod(beat16(rpm), 1.0), 1, 0.5)
	}
}

// fadeby is a fraction (0, 1)
func FadeToBlackBy(leds []uint32, _fadeby float64) {

	fadeby := 1 - _fadeby
	for i := range leds {
		r, g, b := utils.IntToRGB(leds[i])
		r = r * fadeby
		g = g * fadeby
		b = b * fadeby
		leds[i] = utils.RGBToInt(r, g, b)
	}
}

// weight [0 To 1]
func mixColors(color1, color2 []float64, weight2 float64) (r, g, b float64) {
	weight2 = utils.ClampVal(weight2, 0, 1)
	weight1 := 1.0 - weight2
	r = color1[0]*weight1 + color2[0]*weight2
	g = color1[1]*weight1 + color2[1]*weight2
	b = color1[2]*weight1 + color2[2]*weight2
	return
}

func paletteLookup(palette [][]float64, position float64) (r, g, b float64) {
	position = math.Mod(position, 1.0)

	weight2 := position * float64(len(palette))
	idx := int(math.Floor(weight2))
	weight2 -= float64(idx)

	color1 := palette[idx]
	idx = (idx + 1) % len(palette)
	color2 := palette[idx]

	return mixColors(color1, color2, weight2)
}

//returns time since jan 1 1970
func millis() float64 {
	return float64(time.Now().UnixMilli())
}

//returns sawtooth wave at given bpm
func beat16(BPM float64) float64 {
	millis := millis()
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
