package main

import (
	"encoding/binary"
	"math"
	"os/exec"
	"time"

	"github.com/mjibson/go-dsp/spectral"
	"github.com/mjibson/go-dsp/window"
	"github.com/sqp/pulseaudio"

	"log"
	"strconv"
)

var pxx, freq []float64
var streaming = true
var audioStreamBufferSize = 1 << 10
var numBins int = 1 << 13 // number of bins for fft (number of datapoints across the output fft array)

var fftColor = []byte{0, 0, 0}
var fftColorBuffer [][]float64
var fftColorBufferSize int = 16
var fftColorShift float64 = 0

var redLowerFreq int = 80
var redUpperFreq int = 200
var greenLowerFreq int = 160
var greenUpperFreq int = 1000
var blueLowerFreq int = 600
var blueupperFreq int = 2800
var colorBrightness float64 = 255

// read audio stream and computes fft and color
func ProcessAudioStream(client *pulseaudio.Client, udpClients chan []byte) {
	streams, _ := client.Core().ListPath("PlaybackStreams")
	if len(streams) == 0 {
		log.Println("Waiting for audio stream to process...")
	}
	for {
		streams, _ = client.Core().ListPath("PlaybackStreams")
		if len(streams) < 1 {
			time.Sleep(time.Millisecond * 100)
			continue
		}
		log.Println("Streams:", streams)
		stream := streams[0]
		log.Println("Stream found:", stream)
		// Get the device to query properties for the stream referenced by his path.
		dev := client.Stream(stream)

		// Get some informations about this stream.
		mute, _ := dev.Bool("Mute")               // bool
		vols, _ := dev.ListUint32("Volume")       // []uint32
		latency, _ := dev.Uint64("Latency")       // uint64
		sampleRate, _ := dev.Uint32("SampleRate") // uint32
		numChannels, _ := dev.ListUint32("Channels")
		log.Println("\tstream:", volumeText(mute, vols))
		log.Println("\tlatency:", latency)
		log.Println("\tsampleRate:", sampleRate)
		log.Println("\tChannels map:", numChannels)

		props, e := dev.MapString("PropertyList") // map[string]string
		chkPrint(e)
		log.Println(props["media.name"])

		cmd := exec.Command("parec")
		audioStreamReader, err := cmd.StdoutPipe()
		chkFatal(err)
		err = cmd.Start()
		chkFatal(err)

		audioStreamBuffer := make([]int16, audioStreamBufferSize)
		for streaming {
			//read in audiostream to buffer
			binary.Read(audioStreamReader, binary.LittleEndian, audioStreamBuffer)

			//convert buffer to float
			buffercomplex := make([]float64, audioStreamBufferSize)
			for i, v := range audioStreamBuffer {
				buffercomplex[i] = float64(v)
			}

			//FFT
			opt := &spectral.PwelchOptions{
				NFFT:      numBins, //nfft should be power of 2
				Pad:       numBins, //same as NFFT
				Window:    window.Blackman,
				Scale_off: false,
			}
			pxx, freq = spectral.Pwelch(buffercomplex, float64(int(sampleRate)*len(numChannels)), opt)
			rangeFreq := computeFreqIdx(3000, int(sampleRate), opt.Pad)
			pxx, freq = pxx[:rangeFreq], freq[:rangeFreq]
			pxx = normalizePower(pxx)

			colorOut := computeColor(pxx, sampleRate, opt.Pad)
			udpClients <- colorOut
		}
	}
}

// converts the pxx and freq to rgb values
// and saves them to the 'fftColor' variable
func computeColor(pxx []float64, sampleRate uint32, pad int) []byte {
	findMax := func(arr []float64) float64 {
		max := 0.0
		for _, v := range arr {
			if v > max {
				max = v
			}
		}
		return max
	}

	redLowerIdx := computeFreqIdx(redLowerFreq, int(sampleRate), pad)
	redUpperIdx := computeFreqIdx(redUpperFreq, int(sampleRate), pad)
	greenLowerIdx := computeFreqIdx(greenLowerFreq, int(sampleRate), pad)
	greenUpperIdx := computeFreqIdx(greenUpperFreq, int(sampleRate), pad)
	blueLowerIdx := computeFreqIdx(blueLowerFreq, int(sampleRate), pad)
	blueupperIdx := computeFreqIdx(blueupperFreq, int(sampleRate), pad)

	redFFTMax := findMax(pxx[redLowerIdx:redUpperIdx]) * 2
	greenFFTMax := findMax(pxx[greenLowerIdx:greenUpperIdx]) * 2
	blueFFTMax := findMax(pxx[blueLowerIdx:blueupperIdx]) * 2

	if blueFFTMax > greenFFTMax && blueFFTMax > redFFTMax {
		blueFFTMax *= 3
		greenFFTMax *= 2.25
	} else if greenFFTMax > blueFFTMax && greenFFTMax > redFFTMax {
		greenFFTMax *= 2.25
		redFFTMax *= 1.75
	} else if redFFTMax > greenFFTMax && redFFTMax > blueFFTMax {
		redFFTMax *= 2
		blueFFTMax *= 3
	}

	if redFFTMax < greenFFTMax && redFFTMax < blueFFTMax {
		redFFTMax *= 1.0
	} else if greenFFTMax < blueFFTMax && greenFFTMax < redFFTMax {
		greenFFTMax *= 0.9
	} else if blueFFTMax < greenFFTMax && blueFFTMax < redFFTMax {
		blueFFTMax *= 0.8
	}

	fftColorBuffer = append(fftColorBuffer, []float64{redFFTMax, greenFFTMax, blueFFTMax})
	if len(fftColorBuffer) > fftColorBufferSize {
		rmCount := len(fftColorBuffer) - fftColorBufferSize
		fftColorBuffer = fftColorBuffer[rmCount:]
	}

	var redAvg, greenAvg, blueAvg float64 = 0, 0, 0

	for _, fftColorX := range fftColorBuffer {
		redAvg += fftColorX[0]
		greenAvg += fftColorX[1]
		blueAvg += fftColorX[2]
	}

	redAvg /= float64(len(fftColorBuffer))
	greenAvg /= float64(len(fftColorBuffer))
	blueAvg /= float64(len(fftColorBuffer))

	redAvg = math.Min(redAvg*1.5, 255)
	greenAvg = math.Min(greenAvg*1.5, 255)
	blueAvg = math.Min(blueAvg*1.5, 255)

	rgbRotated := rotateColor([]float64{redAvg, greenAvg, blueAvg}, fftColorShift)
	rgbScaled := scaleColor2Brightness(rgbRotated)
	fftColor = clamp(rgbScaled)
	return fftColor
}

//************************Helper Func***************************

// rotates rgb float value by degrees. https://flylib.com/books/2/816/1/html/2/files/fig11_14.jpeg
func rotateColor(rgb []float64, rotDeg float64) []float64 {
	if rotDeg != 0 {
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
	out[2] = rgb[0]*matrix[2][0] + rgb[1]*matrix[2][1] + rgb[2]*matrix[2][2]
	out[0] = rgb[0]*matrix[0][0] + rgb[1]*matrix[0][1] + rgb[2]*matrix[0][2]
	out[1] = rgb[0]*matrix[1][0] + rgb[1]*matrix[1][1] + rgb[2]*matrix[1][2]
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

// scale the color to the brightness
func scaleColor2Brightness(color []float64) []float64 {
	scaledColor := color[:]
	for i, val := range color {
		if colorBrightness == 255 {
			break
		}
		scaler := colorBrightness / 255
		scaledColor[i] = val * scaler
	}
	return scaledColor
}

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
	return int(math.Round(pos))
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
