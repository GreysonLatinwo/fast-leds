package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/mjibson/go-dsp/window"
)

const colorUpdateBufSize = 8
const Gport = ":1234"
const Gaddr = "192.168.0.255"

var Uaddr *net.UDPAddr

var ledCommPipe = make(chan [4]byte, colorUpdateBufSize)

type remoteLeds struct {
	Server *net.UDPConn
	Client *net.UDPAddr
}

//initalize webServer
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
		presetStr := r.URL.RawQuery
		presetInt := 0
		switch strings.ToLower(presetStr) {
		case "confetti":
			presetInt = 1
		case "sinelon":
			presetInt = 2
		case "juggle":
			presetInt = 3
		}

		time.Sleep(time.Millisecond * 100)
		ledCommPipe <- [4]byte{3, byte(presetInt), 0, 0}
		log.Printf("Playing %s preset", presetStr)
	})

	http.HandleFunc("/static/setColor", func(rw http.ResponseWriter, r *http.Request) {
		if isProcessAudioStream {
			stopMusicListening <- struct{}{}
		}
		rBody, err := ioutil.ReadAll(r.Body)
		chkPrint(err)
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
		for i := 0; i < 4; i++ {
			time.Sleep(time.Millisecond * 50)
			ledCommPipe <- [4]byte{2, byte(red), byte(green), byte(blue)}
		}
		log.Println("Set Static color:", red, green, blue)
	})
	http.HandleFunc("/favicon.ico", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "public/favicon.ico")
	})
	http.HandleFunc("/music/", func(rw http.ResponseWriter, r *http.Request) {
		http.ServeFile(rw, r, "public/music.html")
	})
	http.HandleFunc("/music/start", func(rw http.ResponseWriter, r *http.Request) {
		go ProcessAudioStream()
		log.Println("Listening to music")
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
		chkPrint(err)
		rw.Write(jsonOut)
	})
	http.HandleFunc("/music/setFFTWindow", func(rw http.ResponseWriter, r *http.Request) {
		fftTypeStr := r.URL.RawQuery
		fftTypeInt, err := strconv.Atoi(fftTypeStr)
		if err != nil {
			chkPrint(err)
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
			chkPrint(err)
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
			chkPrint(err)
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
	http.HandleFunc("/music/setOutColorScale", func(rw http.ResponseWriter, r *http.Request) {
		scaleData := r.URL.RawQuery
		scaleVal, err := strconv.ParseFloat(scaleData, 64)
		if err != nil {
			chkPrint(err)
			return
		}
		colorOutScale = scaleVal
	})
	http.HandleFunc("/music/setColorFreqRange", func(rw http.ResponseWriter, r *http.Request) {
		ColorFreqData := r.URL.RawQuery
		ColorFreqParsed := strings.Split(ColorFreqData, ":")

		ColorFreqColor := ColorFreqParsed[0]
		ColorFreqType := strings.ToLower(ColorFreqParsed[1])
		ColorFreqInt, err := strconv.Atoi(ColorFreqParsed[2])

		if err != nil {
			chkPrint(err)
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
			ColorShift            float64
			ColorScaler           float64
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
			ColorShift:            fftColorShift,
			ColorScaler:           colorOutScale,
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
		chkPrint(err)
		rw.Write(vars)
	})
	http.HandleFunc("/music/setColorShift", func(rw http.ResponseWriter, r *http.Request) {
		colorShift, err := strconv.ParseFloat(r.URL.RawQuery, 64)
		if err != nil {
			chkPrint(err)
			return
		}
		fftColorShift = colorShift
	})
}

func StartComms() error {
	listenAddr := handleErrPrint(net.ResolveUDPAddr("udp4", Gport)).(*net.UDPAddr)
	server := handleErrPrint(net.ListenUDP("udp4", listenAddr)).(*net.UDPConn)
	Uaddr = handleErrPrint(net.ResolveUDPAddr("udp4", Gaddr+Gport)).(*net.UDPAddr)

	remote := remoteLeds{Server: server, Client: Uaddr}
	go colorServer(remote, ledCommPipe)
	ledCommPipe <- [4]byte{1, 0, 0, 0}

	return nil
}

//takes the color output and tells the network
func colorServer(remote remoteLeds, colorUpdate chan [4]byte) {
	for color := range colorUpdate {
		go writeToLocalLeds(color)
		go remote.writeToLeds(color)
	}
}

func writeToLocalLeds(color [4]byte) {
	os.Stdout.Write(color[:])
}

func (r remoteLeds) writeToLeds(color [4]byte) {
	r.Server.WriteTo(color[:], r.Client)

}
