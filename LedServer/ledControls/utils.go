package main

import (
	"fmt"
	"math"
)

// hard caps between lower and upper values
func clamp(val, lower, upper float64) float64 {
	return math.Max(lower, math.Min(val, upper))
}

// r g b color to integer color
// all values [0, 255]
func RGBToInt(r, g, b float64) uint32 {
	return uint32(r)<<16 + uint32(g)<<8 + uint32(b)
}

// integer color to r g b color
// [0, 1<<24)
func IntToRGB(x uint32) (float64, float64, float64) {
	red := float64(uint8(x >> 16))
	green := float64(uint8(x >> 8))
	blue := float64(uint8(x))
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
func HSLToRGB(h, s, l float64) (float64, float64, float64) {

	if s == 0 {
		// it's gray
		return l * 255, l * 255, l * 255
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

	return r * 255, g * 255, b * 255
}

// rotates rgb float value by degrees. https://flylib.com/books/2/816/1/html/2/files/fig11_14.jpeg
// rgb [0, 255], rotDeg [0, 360]
func rotateColor(rgb []float64, rotDeg float64) []float64 {
	if rotDeg == 0 {
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
	outf[0] = math.Min(math.Max(rgb[0]*matrix[0][0]+rgb[1]*matrix[0][1]+rgb[2]*matrix[0][2], 0), 255)
	outf[1] = math.Min(math.Max(rgb[0]*matrix[1][0]+rgb[1]*matrix[1][1]+rgb[2]*matrix[1][2], 0), 255)
	outf[2] = math.Min(math.Max(rgb[0]*matrix[2][0]+rgb[1]*matrix[2][1]+rgb[2]*matrix[2][2], 0), 255)
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

func mod(a, b int) int {
	return int(math.Mod(float64(a), float64(b)))
}

func checkError(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
