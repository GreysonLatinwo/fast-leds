package main

import (
	"encoding/json"
	"net"
	"net/http"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/mjibson/go-dsp/window"
)

const colorUpdateBufSize = 8
const Gport = ":6969"
const Gaddr = "192.168.0.255"

var Uaddr *net.UDPAddr

var colorUpdate = make(chan []byte, colorUpdateBufSize)

//initalize webServer
func init() {
	http.HandleFunc("/music/", func(rw http.ResponseWriter, r *http.Request) { http.ServeFile(rw, r, "public/index.html") })
	http.HandleFunc("/music/style.css", func(rw http.ResponseWriter, r *http.Request) { http.ServeFile(rw, r, "public/style.css") })
	http.HandleFunc("/music/app.js", func(rw http.ResponseWriter, r *http.Request) { http.ServeFile(rw, r, "public/app.js") })
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
	http.HandleFunc("/music/getVariables", func(rw http.ResponseWriter, r *http.Request) {
		fftWindowTypeSplit := strings.Split(runtime.FuncForPC(reflect.ValueOf(fftWindowType).Pointer()).Name(), ".")
		windowType := fftWindowTypeSplit[len(fftWindowTypeSplit)-1]

		vars, err := json.Marshal(struct {
			IsStreaming           bool
			AudioStreamBufferSize int
			FFTRedBufferSize      int
			FFTGreenBufferSize    int
			FFTBlueBufferSize     int
			FFTWindowType         string
			ColorShift            float64
			ColorBrightness       float64
			ColorScaler           float64
		}{
			IsStreaming:           streaming,
			AudioStreamBufferSize: audioStreamBufferSize,
			FFTRedBufferSize:      fftRedBufferSize,
			FFTGreenBufferSize:    fftGreenBufferSize,
			FFTBlueBufferSize:     fftBlueBufferSize,
			FFTWindowType:         windowType,
			ColorShift:            fftColorShift,
			ColorBrightness:       colorBrightness,
			ColorScaler:           colorOutScale,
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
	http.HandleFunc("/music/setColorBrightness", func(rw http.ResponseWriter, r *http.Request) {
		brightness, err := strconv.ParseFloat(r.URL.RawQuery, 64)
		if err != nil {
			chkPrint(err)
			return
		}
		colorBrightness = brightness
	})
	http.HandleFunc("/music/setColorScale", func(rw http.ResponseWriter, r *http.Request) {
		scale, err := strconv.ParseFloat(r.URL.RawQuery, 64)
		if err != nil {
			chkPrint(err)
			return
		}
		colorOutScale = scale
	})
}

func InitComms() (chan []byte, error) {
	listenAddr := handleErrPrint(net.ResolveUDPAddr("udp4", Gport)).(*net.UDPAddr)
	server := handleErrPrint(net.ListenUDP("udp4", listenAddr)).(*net.UDPConn)
	Uaddr = handleErrPrint(net.ResolveUDPAddr("udp4", Gaddr+Gport)).(*net.UDPAddr)

	go colorServer(server, Uaddr, colorUpdate)

	colorUpdate <- []byte{0, 0, 0}
	return colorUpdate, nil
}

//takes the color output and tells the network
func colorServer(server *net.UDPConn, Uaddr *net.UDPAddr, colorUpdate chan []byte) {
	for color := range colorUpdate {
		os.Stdout.Write(color)
		_, err := server.WriteTo(color, Uaddr)
		if err != nil {
			panic(err)
		}
	}
}
