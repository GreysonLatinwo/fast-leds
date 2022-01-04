package main

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"net/http"
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

var fftColor [3]byte = [3]byte{0, 0, 0}
var fftColorBuffer [][]float64
var fftColorBufferSize int = 11
var fftColorShift float64 = 0

var colorBrightness byte = 255

//initalize webServer
func init() {
	http.HandleFunc("/music/", func(rw http.ResponseWriter, r *http.Request) {
		log.Println("Connection from", r.RemoteAddr, "Request:", r.URL.Path)
		http.ServeFile(rw, r, "public/index.html")
	})
	http.HandleFunc("/music/style.css", func(rw http.ResponseWriter, r *http.Request) { http.ServeFile(rw, r, "public/style.css") })
	http.HandleFunc("/music/app.js", func(rw http.ResponseWriter, r *http.Request) { http.ServeFile(rw, r, "public/app.js") })
	http.HandleFunc("/music/getData", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "utf-8")
		jsonOut, err := json.Marshal(struct {
			Power []float64
			Freq  []float64
			Color [3]byte // [r,g,b]
		}{
			Power: pxx,
			Freq:  freq,
			Color: fftColor,
		})
		ChkPrint(err)
		rw.Write(jsonOut)
	})
	http.HandleFunc("/music/setEnergyLevel", func(rw http.ResponseWriter, r *http.Request) {
		energyLevel := r.URL.RawQuery
		energyLevelInt, err := strconv.Atoi(energyLevel)
		ChkPrint(err)
		fftColorBufferSize = energyLevelInt
	})
	http.HandleFunc("/music/getVariables", func(rw http.ResponseWriter, r *http.Request) {
		vars, err := json.Marshal(struct {
			IsStreaming           bool
			AudioStreamBufferSize int
			FFTColorBufferSize    int
			ColorShift            float64
			ColorBrightness       byte
		}{
			IsStreaming:           streaming,
			AudioStreamBufferSize: audioStreamBufferSize,
			FFTColorBufferSize:    fftColorBufferSize,
			ColorShift:            fftColorShift,
			ColorBrightness:       colorBrightness,
		})
		ChkPrint(err)
		rw.Write(vars)
	})
	http.HandleFunc("/music/setColorShift", func(rw http.ResponseWriter, r *http.Request) {
		energyLevel := r.URL.RawQuery
		energyLevelInt, err := strconv.Atoi(energyLevel)
		ChkPrint(err)
		fftColorShift = float64(energyLevelInt)
	})
	http.HandleFunc("/music/setColorBrightness", func(rw http.ResponseWriter, r *http.Request) {
		energyLevel := r.URL.RawQuery
		energyLevelInt, err := strconv.Atoi(energyLevel)
		ChkPrint(err)
		colorBrightness = byte(energyLevelInt)
	})
}

// takes audio stream, analyses the audio and writes the output to color
func main() {

	go StartPulseAudio()
	colorUpdate := InitComms()

	// Read audio stream
	ProcessAudioStream(pulse, colorUpdate)
}

// read audio stream and computes fft and color
func ProcessAudioStream(client *pulseaudio.Client, colorUpdate chan [3]byte) {
	streams, _ := client.Core().ListPath("PlaybackStreams")
	if len(streams) == 0 {
		log.Println("Waiting for audio stream to process...")
	}
	for {
		streams, _ = client.Core().ListPath("PlaybackStreams")
		for len(streams) < 1 {
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
		ChkPrint(e)
		log.Println(props["media.name"])

		cmd := exec.Command("parec")
		audioStreamReader, err := cmd.StdoutPipe()
		ChkFatal(err, "cmd.StdoutPipe error")
		err = cmd.Start()
		ChkFatal(err, "cmd.Start()")

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
				Pad:       2800,    //same as NFFT
				Window:    window.Bartlett,
				Scale_off: false,
			}
			pxx, freq = spectral.Pwelch(buffercomplex, float64(int(sampleRate)*len(numChannels)), opt)
			rangeFreq := computeFreqIdx(3000, int(sampleRate), opt.Pad)
			pxx, freq = pxx[:rangeFreq], freq[:rangeFreq]
			pxx = normalizePower(pxx)

			colorOut := computeColor(pxx, sampleRate, opt.Pad)
			colorUpdate <- colorOut
		}
	}
}

// converts the pxx and freq to rgb values
// and saves them to the 'fftColor' variable
func computeColor(pxx []float64, sampleRate uint32, pad int) [3]byte {
	findMax := func(arr []float64) float64 {
		max := 0.0
		for _, v := range arr {
			if v > max {
				max = v
			}
		}
		return max
	}

	var redLowerFreq int = 0
	var greenLowerFreq int = 200
	var blueLowerFreq int = 600
	var blueupperFreq int = 2800

	redLowerIdx := computeFreqIdx(redLowerFreq, int(sampleRate), pad)
	greenLowerIdx := computeFreqIdx(greenLowerFreq, int(sampleRate), pad)
	blueLowerIdx := computeFreqIdx(blueLowerFreq, int(sampleRate), pad)
	blueupperIdx := computeFreqIdx(blueupperFreq, int(sampleRate), pad)

	redFFTMax := findMax(pxx[redLowerIdx:greenLowerIdx])
	greenFFTMax := findMax(pxx[greenLowerIdx:blueLowerIdx])
	blueFFTMax := findMax(pxx[blueLowerIdx:blueupperIdx])

	if redFFTMax > greenFFTMax && redFFTMax > blueFFTMax {
		redFFTMax *= 1.5
	} else if blueFFTMax > greenFFTMax && blueFFTMax > redFFTMax {
		blueFFTMax *= 1.5
	} else if greenFFTMax > blueFFTMax && greenFFTMax > redFFTMax {
		greenFFTMax *= 1.5
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

	fftColor = scaleColor2Brightness(rgbRotated)
	return fftColor
}

//************************Helper Func***************************

// rotates rgb float value by degrees. https://flylib.com/books/2/816/1/html/2/files/fig11_14.jpeg
func rotateColor(rgb []float64, rotDeg float64) [3]byte {
	if rotDeg != 0 {
		return [3]byte{byte(rgb[0]), byte(rgb[1]), byte(rgb[2])}
	}

	pi := 3.14159265
	sqrtf := func(x float64) float64 {
		return math.Sqrt(x)
	}

	clamp := func(v float64) byte {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return byte(v)
	}

	cosA := math.Cos(rotDeg * pi / 180) //convert degrees to radians
	sinA := math.Sin(rotDeg * pi / 180) //convert degrees to radians
	//calculate the rotation matrix, only depends on Hue
	matrix := [][]float64{{cosA + (1.0-cosA)/3.0, 1.0/3.0*(1.0-cosA) - sqrtf(1.0/3.0)*sinA, 1.0/3.0*(1.0-cosA) + sqrtf(1.0/3.0)*sinA},
		{1.0/3.0*(1.0-cosA) + sqrtf(1.0/3.0)*sinA, cosA + 1.0/3.0*(1.0-cosA), 1.0/3.0*(1.0-cosA) - sqrtf(1.0/3.0)*sinA},
		{1.0/3.0*(1.0-cosA) - sqrtf(1.0/3.0)*sinA, 1.0/3.0*(1.0-cosA) + sqrtf(1.0/3.0)*sinA, cosA + 1.0/3.0*(1.0-cosA)}}

	out := [3]byte{0, 0, 0}

	//Use the rotation matrix to convert the RGB directly
	out[2] = clamp(rgb[0]*matrix[2][0] + rgb[1]*matrix[2][1] + rgb[2]*matrix[2][2])
	out[0] = clamp(rgb[0]*matrix[0][0] + rgb[1]*matrix[0][1] + rgb[2]*matrix[0][2])
	out[1] = clamp(rgb[0]*matrix[1][0] + rgb[1]*matrix[1][1] + rgb[2]*matrix[1][2])
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
func scaleColor2Brightness(color [3]byte) [3]byte {
	scaledColor := [3]byte{color[0], color[1], color[2]}
	for i, val := range color {
		if colorBrightness == 255 {
			continue //we are able to continue bc we set the value equal at the beginning
		}
		valf := float32(val)
		scaler := float32(colorBrightness) / 255.0
		scaledColor[i] = byte(valf * scaler)
	}
	return scaledColor
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
