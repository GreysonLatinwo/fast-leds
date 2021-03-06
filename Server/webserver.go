package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	utils "github.com/greysonlatinwo/fast-leds/utils"
	"github.com/mjibson/go-dsp/window"
)

//initalize webServer handlers
func init() {
	http.HandleFunc("/static/", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "public/staticPicker.html")
	})
	http.HandleFunc("/preset/", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "public/presetPicker.html")
	})
	http.HandleFunc("/preset/setPreset", func(rw http.ResponseWriter, r *http.Request) {
		if isProcessAudioStream {
			stopMusicListening <- struct{}{}
		}
		presetData := strings.Split(r.URL.RawQuery, ",")
		presetStr := presetData[0]
		var presetInt uint8 = 0
		args := [10]byte{0, 0, 0, 0, 0, 0, 0, 0, 0}
		switch strings.ToLower(presetStr) {
		case "confetti":
			presetInt = 3
		case "sinelon":
			presetInt = 4
		case "juggle":
			presetInt = 5
		case "spinninghues":
			if len(presetData) < 5 {
				log.Println("Malformat:", r.URL.RawQuery)
				return
			}
			presetInt = 6
			//hues
			args[0] = uint8(utils.HandleErrPrint(strconv.Atoi(presetData[1])).(int))
			args[1] = uint8(utils.HandleErrPrint(strconv.Atoi(presetData[2])).(int))
			args[2] = uint8(utils.HandleErrPrint(strconv.Atoi(presetData[3])).(int))
			//brightness [0,255]
			args[3] = uint8(utils.HandleErrPrint(strconv.Atoi(presetData[4])).(int))
		case "spinningcolors":
			if len(presetData) < 10 {
				log.Println("Malformat:", r.URL.RawQuery)
				return
			}
			presetInt = 7
			// bpm
			args[0] = uint8(utils.HandleErrPrint(strconv.Atoi(presetData[1])).(int))
			// hues
			args[1] = uint8(utils.HandleErrPrint(strconv.Atoi(presetData[2])).(int))
			args[2] = uint8(utils.HandleErrPrint(strconv.Atoi(presetData[3])).(int))
			args[3] = uint8(utils.HandleErrPrint(strconv.Atoi(presetData[4])).(int))
			args[4] = uint8(utils.HandleErrPrint(strconv.Atoi(presetData[5])).(int))
			// brightness
			args[5] = uint8(utils.HandleErrPrint(strconv.Atoi(presetData[6])).(int))
			args[6] = uint8(utils.HandleErrPrint(strconv.Atoi(presetData[7])).(int))
			args[7] = uint8(utils.HandleErrPrint(strconv.Atoi(presetData[8])).(int))
			args[8] = uint8(utils.HandleErrPrint(strconv.Atoi(presetData[9])).(int))
		}

		// probably should convert isProcess(ing)AudioStream into a chan
		for isProcessAudioStream {
			time.Sleep(time.Duration(time.Millisecond * 100))
			continue
		}

		for i := 0; i < 4; i++ {
			ledCommPipe <- [10]byte{
				presetInt,
				args[0],
				args[1],
				args[2],
				args[3],
				args[4],
				args[5],
				args[6],
				args[7],
				args[8],
			}
		}
	})

	http.HandleFunc("/static/setColor", func(rw http.ResponseWriter, r *http.Request) {
		if isProcessAudioStream {
			stopMusicListening <- struct{}{}
		}
		rBody, err := ioutil.ReadAll(r.Body)
		utils.CheckError(err)
		body := strings.Split(string(rBody), ",")
		red, err := strconv.Atoi(body[0])
		if err != nil {
			return
		}
		green, err := strconv.Atoi(body[1])
		if err != nil {
			return
		}
		blue, err := strconv.Atoi(body[2])
		if err != nil {
			return
		}

		// probably should convert isProcess(ing)AudioStream into a chan
		for isProcessAudioStream {
			time.Sleep(time.Duration(time.Millisecond * 100))
			continue
		}

		ledCommPipe <- [10]byte{2, byte(red), byte(green), byte(blue), 0, 0, 0, 0}
		log.Println("Static color:", red, green, blue)
	})
	http.HandleFunc("/favicon.ico", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "public/favicon.ico")
	})
	http.HandleFunc("/music/", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "public/music.html")
	})
	http.HandleFunc("/music/start", func(rw http.ResponseWriter, r *http.Request) {
		go ProcessAudioStream()
	})
	http.HandleFunc("/music/style.css", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "public/style.css")
	})
	http.HandleFunc("/music/app.js", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "public/app.js")
	})
	http.HandleFunc("/music/getData", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "utf-8")
		jsonOut, err := json.Marshal(struct {
			Power []float64
			Freq  []float64
			Color []uint32 // [r,g,b]
		}{
			Power: pxx,
			Freq:  freq,
			Color: []uint32{uint32(fftColor[0]), uint32(fftColor[1]), uint32(fftColor[2])},
		})
		utils.CheckError(err)
		rw.Write(jsonOut)
	})
	http.HandleFunc("/music/setFFTWindow", func(rw http.ResponseWriter, r *http.Request) {
		fftTypeStr := r.URL.RawQuery
		fftTypeInt, err := strconv.Atoi(fftTypeStr)
		if err != nil {
			utils.CheckError(err)
			return
		}
		switch fftTypeInt {
		case 1:
			fftWindowType = window.Bartlett
		case 2:
			fftWindowType = window.Blackman
		case 3:
			fftWindowType = window.FlatTop
		case 4:
			fftWindowType = window.Hamming
		case 5:
			fftWindowType = window.Hann
		case 6:
			fftWindowType = window.Rectangular
		}
	})
	http.HandleFunc("/music/setEnergyLevel", func(rw http.ResponseWriter, r *http.Request) {
		energyLevelData := r.URL.RawQuery
		energyLevelParsed := strings.Split(energyLevelData, ":")
		energyLevelColor := energyLevelParsed[0]
		energyLevelInt, err := strconv.Atoi(energyLevelParsed[1])
		if err != nil {
			utils.CheckError(err)
			return
		}
		switch strings.ToLower(energyLevelColor) {
		case "red":
			fftRedBufferSize = energyLevelInt
		case "green":
			fftGreenBufferSize = energyLevelInt
		case "blue":
			fftBlueBufferSize = energyLevelInt
		}
	})
	http.HandleFunc("/music/setColorInScale", func(rw http.ResponseWriter, r *http.Request) {
		scaleData := r.URL.RawQuery
		scaleParsed := strings.Split(scaleData, "=")
		scaleColor := scaleParsed[0]
		scaleVal, err := strconv.ParseFloat(scaleParsed[1], 64)
		if err != nil {
			utils.CheckError(err)
			return
		}
		switch strings.ToLower(scaleColor) {
		case "red":
			redInScale = scaleVal
		case "green":
			greenInScale = scaleVal
		case "blue":
			blueInScale = scaleVal
		}
	})
	http.HandleFunc("/music/setColorOutScale", func(rw http.ResponseWriter, r *http.Request) {
		scaleData := r.URL.RawQuery
		scaleParsed := strings.Split(scaleData, "=")
		scaleColor := scaleParsed[0]
		scaleVal, err := strconv.ParseFloat(scaleParsed[1], 64)
		if err != nil {
			utils.CheckError(err)
			return
		}
		switch strings.ToLower(scaleColor) {
		case "red":
			redOutScale = scaleVal
		case "green":
			greenOutScale = scaleVal
		case "blue":
			blueOutScale = scaleVal
		}
	})
	http.HandleFunc("/music/setColorFreqRange", func(rw http.ResponseWriter, r *http.Request) {
		ColorFreqData := r.URL.RawQuery
		ColorFreqParsed := strings.Split(ColorFreqData, ":")

		ColorFreqColor := ColorFreqParsed[0]
		ColorFreqType := strings.ToLower(ColorFreqParsed[1])
		ColorFreqInt, err := strconv.Atoi(ColorFreqParsed[2])

		if err != nil {
			utils.CheckError(err)
			return
		}

		//cap at the max freq outputted
		ColorFreqInt = int(math.Min(float64(ColorFreqInt), float64(maxFreqOut)))

		switch strings.ToLower(ColorFreqColor) {
		case "red":
			if ColorFreqType == "lower" {
				if ColorFreqInt > redUpperFreq {
					ColorFreqInt = redUpperFreq
				}
				redLowerFreq = ColorFreqInt
			} else if ColorFreqType == "upper" {
				if ColorFreqInt < redLowerFreq {
					ColorFreqInt = redLowerFreq
				}
				redUpperFreq = ColorFreqInt
			}
		case "green":
			if ColorFreqType == "lower" {
				if ColorFreqInt > greenUpperFreq {
					ColorFreqInt = greenUpperFreq
				}
				greenLowerFreq = ColorFreqInt
			} else if ColorFreqType == "upper" {
				if ColorFreqInt < greenLowerFreq {
					ColorFreqInt = greenLowerFreq
				}
				greenUpperFreq = ColorFreqInt
			}
		case "blue":
			if ColorFreqType == "lower" {
				if ColorFreqInt > blueUpperFreq {
					ColorFreqInt = blueUpperFreq
				}
				blueLowerFreq = ColorFreqInt
			} else if ColorFreqType == "upper" {
				if ColorFreqInt < blueLowerFreq {
					ColorFreqInt = blueLowerFreq
				}
				blueUpperFreq = ColorFreqInt
			}
		}
	})
	http.HandleFunc("/music/getVariables", func(rw http.ResponseWriter, r *http.Request) {
		fftWindowTypeSplit := strings.Split(runtime.FuncForPC(reflect.ValueOf(fftWindowType).Pointer()).Name(), ".")
		windowType := fftWindowTypeSplit[len(fftWindowTypeSplit)-1]

		vars, err := json.Marshal(struct {
			AudioStreamBufferSize int
			FFTRedBufferSize      int
			FFTGreenBufferSize    int
			FFTBlueBufferSize     int
			FFTWindowType         string
			RedOutScale           float64
			GreenOutScale         float64
			BlueOutScale          float64
			RedInScale            float64
			GreenInScale          float64
			BlueInScale           float64
			RedLowerFreq          int
			RedUpperFreq          int
			GreenLowerFreq        int
			GreenUpperFreq        int
			BlueLowerFreq         int
			BlueUpperFreq         int
			MaxFreqOut            int
		}{
			AudioStreamBufferSize: audioStreamBufferSize,
			FFTRedBufferSize:      fftRedBufferSize,
			FFTGreenBufferSize:    fftGreenBufferSize,
			FFTBlueBufferSize:     fftBlueBufferSize,
			FFTWindowType:         windowType,
			RedOutScale:           redOutScale,
			GreenOutScale:         greenOutScale,
			BlueOutScale:          blueOutScale,
			RedInScale:            redInScale,
			GreenInScale:          greenInScale,
			BlueInScale:           blueInScale,
			RedLowerFreq:          redLowerFreq,
			RedUpperFreq:          redUpperFreq,
			GreenLowerFreq:        greenLowerFreq,
			GreenUpperFreq:        greenUpperFreq,
			BlueLowerFreq:         blueLowerFreq,
			BlueUpperFreq:         blueUpperFreq,
			MaxFreqOut:            maxFreqOut,
		})
		utils.CheckError(err)
		rw.Write(vars)
	})
}
