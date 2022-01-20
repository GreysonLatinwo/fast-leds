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

// normalizes the fft power data
func NormalizePower(pxx []float64) []float64 {
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

func Mod(a, b int) int {
	return int(math.Mod(float64(a), float64(b)))
}

func CheckError(err error) {
	if err != nil {
		log.Println(err)
	}
}

// find the index of the requested frequency.
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

func HandleErrPrint(out ...interface{}) interface{} {
	if out[1] != nil {
		ChkPrint(out[1].(error))
	}
	return out[0]
}

func ChkFatal(e error) {
	if e != nil {
		fmt.Fprintf(os.Stderr, "%+v\n", e)
		log.Fatal(e)
	}
}

func ChkPrint(err error) {
	if err != nil {
		log.Println(err)
	}
}
