package utils

import (
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"time"
)

var FPSCount int = 0

//************************Helper Func***************************

// Equal tells whether a and b contain the same elements.
// A nil argument is equivalent to an empty slice.
func Equal(a, b []uint8) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

// normalizes the fft power data
func NormalizePower(pxx []float64) []float64 {
	var min = 0.0
	for i := range pxx {
		PDb := (((10 * math.Log10(float64(i)+10)) + 10) * math.Log10(pxx[i])) + 10
		pxx[i] = math.Max(PDb, float64(min))
	}
	return pxx
}

// forcefully make sure values are not outside of range
func ClampRGBColor(rgb []float64) []byte {
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

// hard caps between lower and upper values
func ClampVal(val, lower, upper float64) float64 {
	return math.Max(lower, math.Min(val, upper))
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

// int mod
func Mod(a, b float64) int {
	return int(math.Mod(float64(a), float64(b)))
}

// find the index of the requested frequency within the fft frequency array
func ComputeFreqIdx(freq, sampleRate, pad int) int {
	Fs := float64(sampleRate) * 2
	coef := Fs / float64(pad)
	pos := float64(freq) / coef
	return int(math.Floor(pos))
}

func VolumeText(mute bool, vals []uint32) string {
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

func PrintFPS() {
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

//**********************Error Handling************************

/*
	Meant to handle error inline

	exactly 2 values as input

	expects error value to be the last value

	returns other value
		HandleErrPrint(strconv.Atoi("69")).(int)
*/
func HandleErrPrint(out ...interface{}) interface{} {
	if out[1] != nil {
		CheckError(out[1].(error))
	}
	return out[0]
}

//Check error and write to stderr and log fatal
func ChkFatal(e error) {
	if e != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", e)
		log.Fatal(e)
	}
}

//Check error and log print the error
func CheckError(err error) {
	if err != nil {
		log.Println(err)
	}
}
