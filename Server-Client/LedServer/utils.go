package main

import (
	"log"
	"math"
	"strconv"
	"time"
)

var FPSCount int = 0

//************************Helper Func***************************

// rotates rgb float value by degrees. https://flylib.com/books/2/816/1/html/2/files/fig11_14.jpeg
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

	out := []float64{0, 0, 0}

	//Use the rotation matrix to convert the RGB directly
	out[0] = rgb[0]*matrix[0][0] + rgb[1]*matrix[0][1] + rgb[2]*matrix[0][2]
	out[1] = rgb[0]*matrix[1][0] + rgb[1]*matrix[1][1] + rgb[2]*matrix[1][2]
	out[2] = rgb[0]*matrix[2][0] + rgb[1]*matrix[2][1] + rgb[2]*matrix[2][2]
	return out
}

// normalizes the fft power data
func normalizePower(pxx []float64) []float64 {
	var min = 0
	for i := range pxx {
		PDb := 20 * math.Log10(pxx[i])
		if PDb < 0 {
			pxx[i] = math.Max(PDb, float64(min))
		} else {
			pxx[i] = math.Max(PDb, float64(min))
		}
	}
	return pxx
}

// forcefully make sure values are not outside of range
func clamp(rgb []float64) []byte {
	clampedRgb := []byte{0, 0, 0}
	for i, v := range rgb {
		if v < 0 {
			clampedRgb[i] = 0
		} else if v > 255 {
			clampedRgb[i] = 255
		} else {
			clampedRgb[i] = uint8(v)
		}
	}
	return clampedRgb
}

// find the index of the requested frequency.
func computeFreqIdx(freq, sampleRate, pad int) int {
	Fs := float64(sampleRate) * 2
	coef := Fs / float64(pad)
	pos := float64(freq) / coef
	return int(math.Floor(pos))
}

func volumeText(mute bool, vals []uint32) string {
	if mute {
		return "muted"
	}
	vol := int(volumeAverage(vals)) * 100 / 65535
	return " " + strconv.Itoa(vol) + "% "
}

func volumeAverage(vals []uint32) uint32 {
	var vol uint32
	if len(vals) > 0 {
		for _, cur := range vals {
			vol += cur
		}
		vol /= uint32(len(vals))
	}
	return vol
}

//**********************Error Handling************************

func printFPS() {
	ticker := time.NewTicker(time.Minute)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				log.Println("FPS:", FPSCount/60)
				FPSCount = 0
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
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

func handleErrPrint(out ...interface{}) interface{} {
	if out[1] != nil {
		chkPrint(out[1].(error))
	}
	return out[0]
}

func chkFatal(e error) {
	if e != nil {
		log.Fatalf("%+v\n", e)
	}
}

func chkPrint(err error) {
	if err != nil {
		log.Println(err)
	}
}
