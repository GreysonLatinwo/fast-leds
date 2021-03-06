package main

import (
	"encoding/binary"
	"io"
	"log"
	"os/exec"
	"time"

	"github.com/mjibson/go-dsp/spectral"
	"github.com/mjibson/go-dsp/window"

	utils "github.com/greysonlatinwo/fast-leds/utils"
)

var (
	isProcessAudioStream  bool                = false
	sampleRate            int                 = 44100
	numChannels           int                 = 2
	audioStreamBuffer     []int16             = make([]int16, audioStreamBufferSize)
	pxx, freq             []float64           = []float64{}, []float64{}
	maxFreqOut            int                 = 3000
	audioStreamBufferSize int                 = 1 << 10
	numBins               int                 = 1 << 13 // number of bins for fft (number of datapoints across the output fft array)
	fftWindowType         func(int) []float64 = window.Bartlett
	fftRedBufferSize      int                 = 12
	fftGreenBufferSize    int                 = 32
	fftBlueBufferSize     int                 = 16
	redLowerFreq          int                 = 80
	redUpperFreq          int                 = 200
	greenLowerFreq        int                 = 200
	greenUpperFreq        int                 = 800
	blueLowerFreq         int                 = 600
	blueUpperFreq         int                 = maxFreqOut
	redInScale            float64             = 1.25
	greenInScale          float64             = 1.35
	blueInScale           float64             = 1.5
	redOutScale           float64             = 1.75
	greenOutScale         float64             = 1.5
	blueOutScale          float64             = 1.5
	fftColor              []byte              = []byte{0, 0, 0}
	fftRedBuffer          []float64           = []float64{}
	fftGreenBuffer        []float64           = []float64{}
	fftBlueBuffer         []float64           = []float64{}
	stopMusicListening    chan struct{}       = make(chan struct{})
)

// read audio stream and computes fft and color
func ProcessAudioStream() {
	isProcessAudioStream = true
	//audioStreamBuffer := make([]int16, audioStreamBufferSize)
	//function reads all data coming from parec
	go func() {
		parecCmd := exec.Command("parec")
		audioStreamReader := utils.HandleErrPrint(parecCmd.StdoutPipe()).(io.ReadCloser)
		utils.CheckError(parecCmd.Start())
		for {
			//read in audiostream to buffer
			binary.Read(audioStreamReader, binary.LittleEndian, audioStreamBuffer)
			//return when we are told
			select {
			case <-stopMusicListening:
				isProcessAudioStream = false
				if err := parecCmd.Process.Kill(); err != nil {
					log.Fatalln("failed to kill process\n", err)
				}
				return
			default:
			}
		}
	}()
	log.Println("Listening to audio stream")
	for isProcessAudioStream {
		if sampleRate == 0 {
			time.Sleep(100 * time.Millisecond)
			log.Println("Error: incorrect sample rate")
			continue
		}
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
		rangeFreq := 278 //utils.ComputeFreqIdx(maxFreqOut, int(sampleRate), opt.Pad)
		pxx, freq = pxx[:rangeFreq], freq[:rangeFreq]
		pxx = utils.NormalizePower(pxx)

		color := computeRGBColor(pxx, uint32(sampleRate), opt.Pad)
		ledCommPipe <- [10]byte{1, color[0], color[1], color[2], 0, 0, 0, 0}
	}
}

// converts the pxx and freq to rgb values
// and saves them to the 'fftColor' variable
func computeRGBColor(pxx []float64, sampleRate uint32, pad int) []byte {
	redLowerIdx := utils.ComputeFreqIdx(redLowerFreq, int(sampleRate), pad)
	redUpperIdx := utils.ComputeFreqIdx(redUpperFreq, int(sampleRate), pad)
	greenLowerIdx := utils.ComputeFreqIdx(greenLowerFreq, int(sampleRate), pad)
	greenUpperIdx := utils.ComputeFreqIdx(greenUpperFreq, int(sampleRate), pad)
	blueLowerIdx := utils.ComputeFreqIdx(blueLowerFreq, int(sampleRate), pad)
	blueupperIdx := utils.ComputeFreqIdx(blueUpperFreq, int(sampleRate), pad)

	if redLowerIdx > numBins {
		log.Println(sampleRate, pad)
	}

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

	//make stronger values more prominent to exaggerate them in the leds
	if blueFFTMax > greenFFTMax && blueFFTMax > redFFTMax {
		blueFFTMax *= 1.0
		greenFFTMax *= 0.66
		redFFTMax *= 0.66
	} else if greenFFTMax > blueFFTMax && greenFFTMax > redFFTMax {
		greenFFTMax *= 1.25
		redFFTMax *= 0.5
		blueFFTMax *= 0.5
	} else if redFFTMax > greenFFTMax && redFFTMax > blueFFTMax {
		redFFTMax *= 1.5
		blueFFTMax *= 0.33
		greenFFTMax *= 0.33
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
	redAvg *= redOutScale
	greenAvg *= greenOutScale
	blueAvg *= blueOutScale

	fftColor = utils.ClampRGBColor([]float64{redAvg, greenAvg, blueAvg})
	return fftColor
}
