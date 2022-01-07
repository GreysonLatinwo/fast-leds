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
var fftWindowType func(int) []float64 = window.Bartlett

var fftColor = []byte{0, 0, 0}

var fftRedBuffer []float64
var fftGreenBuffer []float64
var fftBlueBuffer []float64
var fftRedBufferSize int = 16
var fftGreenBufferSize int = 24
var fftBlueBufferSize int = 20
var fftColorShift float64 = 0

var redLowerFreq int = 80
var redUpperFreq int = 200
var greenLowerFreq int = 200
var greenUpperFreq int = 1000
var blueLowerFreq int = 1000
var blueupperFreq int = 2800
var colorBrightness float64 = 255
var colorOutScale float64 = 1.5

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
				Window:    fftWindowType,
				Scale_off: false,
			}
			pxx, freq = spectral.Pwelch(buffercomplex, float64(int(sampleRate)*len(numChannels)), opt)
			rangeFreq := computeFreqIdx(3000, int(sampleRate), opt.Pad)
			pxx, freq = pxx[:rangeFreq], freq[:rangeFreq]
			pxx = normalizePower(pxx)

			colorOut := computeRGBColor(pxx, sampleRate, opt.Pad)
			udpClients <- colorOut
		}
	}
}

func computeHueColor(pxx []float64, sampleRate uint32, pad int) []byte {

	findMax := func(arr []float64) float64 {
		max := 0.0
		maxIdx := 0.0
		for i, v := range arr {
			if v > max {
				max = v
				maxIdx = float64(i)
			}
		}
		return maxIdx
	}

	maxValueIdx := findMax(pxx)

	hue := (maxValueIdx / float64(len(pxx))) * 360 //scale maxValueIdx to hue range

	//add value to respective buffers and compute averages
	fftRedBuffer = append(fftRedBuffer, hue)

	if len(fftRedBuffer) > fftRedBufferSize {
		rmCount := len(fftRedBuffer) - fftRedBufferSize
		fftRedBuffer = fftRedBuffer[rmCount:]
	}

	var hueAvg float64 = 0

	for _, fftVal := range fftRedBuffer {
		hueAvg += fftVal
	}

	hueAvg /= float64(len(fftRedBuffer))

	// [0,360], [0,100], [0,100]
	hsl2rgb := func(h, s, l float64) (float64, float64, float64) {
		l /= 100
		var a = s * math.Min(l, 1-l) / 100
		f := func(n float64) float64 {
			k := math.Mod(n+h/30, 12)
			color := l - a*math.Max(math.Min(math.Min(k-3, 9-k), 1), -1)
			return math.Round(255 * color)
		}
		r := f(0)
		g := f(8)
		b := f(4)
		return r, g, b
	}

	r, g, b := hsl2rgb(hueAvg, 100, pxx[int(maxValueIdx)]/255*100)

	log.Print(r, g, b, hueAvg)

	r *= colorOutScale
	g *= colorOutScale
	b *= colorOutScale

	rgbRotated := rotateColor([]float64{r, g, b}, fftColorShift)
	rgbScaled := scaleColor2Brightness(rgbRotated)
	fftColor = clamp(rgbScaled)

	return fftColor
}

// converts the pxx and freq to rgb values
// and saves them to the 'fftColor' variable
func computeRGBColor(pxx []float64, sampleRate uint32, pad int) []byte {
	redLowerIdx := computeFreqIdx(redLowerFreq, int(sampleRate), pad)
	redUpperIdx := computeFreqIdx(redUpperFreq, int(sampleRate), pad)
	greenLowerIdx := computeFreqIdx(greenLowerFreq, int(sampleRate), pad)
	greenUpperIdx := computeFreqIdx(greenUpperFreq, int(sampleRate), pad)
	blueLowerIdx := computeFreqIdx(blueLowerFreq, int(sampleRate), pad)
	blueupperIdx := computeFreqIdx(blueupperFreq, int(sampleRate), pad)

	findMax := func(arr []float64) float64 {
		max := 0.0
		for _, v := range arr {
			if v > max {
				max = v
			}
		}
		return max
	}

	redFFTMax := findMax(pxx[redLowerIdx:redUpperIdx]) * 1.25
	greenFFTMax := findMax(pxx[greenLowerIdx:greenUpperIdx]) * 1.33
	blueFFTMax := findMax(pxx[blueLowerIdx:blueupperIdx]) * 1.33

	if blueFFTMax > greenFFTMax && blueFFTMax > redFFTMax {
		blueFFTMax *= 2
		greenFFTMax *= 1.5
	} else if greenFFTMax > blueFFTMax && greenFFTMax > redFFTMax {
		greenFFTMax *= 2
		redFFTMax *= 1.5
	} else if redFFTMax > greenFFTMax && redFFTMax > blueFFTMax {
		redFFTMax *= 2
		blueFFTMax *= 1.5
	}

	if redFFTMax < greenFFTMax && redFFTMax < blueFFTMax {
		redFFTMax *= 1.0
	} else if greenFFTMax < blueFFTMax && greenFFTMax < redFFTMax {
		greenFFTMax *= 0.5
	} else if blueFFTMax < greenFFTMax && blueFFTMax < redFFTMax {
		blueFFTMax *= 0.5
	}

	var redAvg, greenAvg, blueAvg float64 = 0, 0, 0

	//add value to respective buffers and compute averages
	fftRedBuffer = append(fftRedBuffer, redFFTMax)
	fftGreenBuffer = append(fftGreenBuffer, greenFFTMax)
	fftBlueBuffer = append(fftBlueBuffer, blueFFTMax)

	if len(fftRedBuffer) > fftRedBufferSize {
		rmCount := len(fftRedBuffer) - fftRedBufferSize
		fftRedBuffer = fftRedBuffer[rmCount:]
	}
	if len(fftGreenBuffer) > fftGreenBufferSize {
		rmCount := len(fftGreenBuffer) - fftGreenBufferSize
		fftGreenBuffer = fftGreenBuffer[rmCount:]
	}
	if len(fftBlueBuffer) > fftBlueBufferSize {
		rmCount := len(fftBlueBuffer) - fftBlueBufferSize
		fftBlueBuffer = fftBlueBuffer[rmCount:]
	}

	for _, fftRedVal := range fftRedBuffer {
		redAvg += fftRedVal
	}
	for _, fftGreenVal := range fftGreenBuffer {
		greenAvg += fftGreenVal
	}
	for _, fftBlueVal := range fftBlueBuffer {
		blueAvg += fftBlueVal
	}

	redAvg /= float64(len(fftRedBuffer))
	greenAvg /= float64(len(fftGreenBuffer))
	blueAvg /= float64(len(fftBlueBuffer))

	redAvg *= colorOutScale
	greenAvg *= colorOutScale
	blueAvg *= colorOutScale

	rgbRotated := rotateColor([]float64{redAvg, greenAvg, blueAvg}, fftColorShift)
	rgbScaled := scaleColor2Brightness(rgbRotated)
	fftColor = clamp(rgbScaled)
	return fftColor
}

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
