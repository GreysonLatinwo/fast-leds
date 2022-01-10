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

var (
	sampleRate            int                 = 44100
	numChannels           int                 = 2
	streaming             bool                = true
	pxx, freq             []float64           = []float64{}, []float64{}
	maxFreqOut            int                 = 3000
	audioStreamBufferSize int                 = 1 << 10
	numBins               int                 = 1 << 13 // number of bins for fft (number of datapoints across the output fft array)
	fftWindowType         func(int) []float64 = window.Bartlett
	fftRedBufferSize      int                 = 12
	fftGreenBufferSize    int                 = 32
	fftBlueBufferSize     int                 = 16
	fftColorShift         float64             = 0
	redLowerFreq          int                 = 80
	redUpperFreq          int                 = 200
	greenLowerFreq        int                 = 200
	greenUpperFreq        int                 = 800
	blueLowerFreq         int                 = 600
	blueUpperFreq         int                 = maxFreqOut
	redInScale            float64             = 1.25
	greenInScale          float64             = 1.33
	blueInScale           float64             = 1.5
	colorOutScale         float64             = 1.5
	fftColor              []byte              = []byte{0, 0, 0}
	fftRedBuffer          []float64           = []float64{}
	fftGreenBuffer        []float64           = []float64{}
	fftBlueBuffer         []float64           = []float64{}
	isStreaming           chan bool           = make(chan bool, 1)
)

// read audio stream and computes fft and color
func ProcessAudioStream(colorOut chan []byte) {
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
		if err != nil {
			panic(err)
		}
		err = cmd.Start()
		chkFatal(err)

		audioStreamBuffer := make([]int16, audioStreamBufferSize)
		for isStreaming {
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
			pxx, freq = spectral.Pwelch(buffercomplex, float64(sampleRate*numChannels), opt)
			rangeFreq := computeFreqIdx(maxFreqOut, int(sampleRate), opt.Pad)
			pxx, freq = pxx[:rangeFreq], freq[:rangeFreq]
			pxx = normalizePower(pxx)

			color := computeRGBColor(pxx, uint32(sampleRate), opt.Pad)
			colorOut <- color
			log.Println(color)
		}
	}
}

// converts the pxx and freq to rgb values
// and saves them to the 'fftColor' variable
func computeRGBColor(pxx []float64, sampleRate uint32, pad int) []byte {
	redLowerIdx := computeFreqIdx(redLowerFreq, int(sampleRate), pad)
	redUpperIdx := computeFreqIdx(redUpperFreq, int(sampleRate), pad)
	greenLowerIdx := computeFreqIdx(greenLowerFreq, int(sampleRate), pad)
	greenUpperIdx := computeFreqIdx(greenUpperFreq, int(sampleRate), pad)
	blueLowerIdx := computeFreqIdx(blueLowerFreq, int(sampleRate), pad)
	blueupperIdx := computeFreqIdx(blueUpperFreq, int(sampleRate), pad)

	findMax := func(arr []float64) float64 {
		max := 0.0
		for _, v := range arr {
			if v > max {
				max = v
			}
		}
		return max
	}

	//scale input colors bc fft values for higher freq sounds are not as strong
	redFFTMax := findMax(pxx[redLowerIdx:redUpperIdx]) * redInScale
	greenFFTMax := findMax(pxx[greenLowerIdx:greenUpperIdx]) * greenInScale
	blueFFTMax := findMax(pxx[blueLowerIdx:blueupperIdx]) * blueInScale

	//make strongest value more prominent to exaggerate them in the leds
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

	//we shouldnt have to recompute entire average every time
	//update based on previous buffer
	var redAvg, greenAvg, blueAvg float64 = 0, 0, 0

	// add value to respective buffers
	// and resize buffer appropriately to maintain expected size
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

	// compute averages for all buffers
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

	// scale output color by users request
	redAvg *= colorOutScale
	greenAvg *= colorOutScale
	blueAvg *= colorOutScale

	rgbRotated := rotateColor([]float64{redAvg, greenAvg, blueAvg}, fftColorShift)
	fftColor = clamp(rgbRotated)
	return fftColor
}

func startAudioListening() io.ReadCloser {
	cmd := exec.Command("parec")
	audioStreamReader, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	err = cmd.Start()
	chkFatal(err)
	return audioStreamReader
}
