package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
)

const colorUpdateBufSize = 8
const Gport = ":9999"
const Gaddr = "192.168.0.255"

var Uaddr *net.UDPAddr

var colorUpdate = make(chan []byte, colorUpdateBufSize)

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
			Color []uint32 // [r,g,b]
		}{
			Power: pxx,
			Freq:  freq,
			Color: []uint32{uint32(fftColor[0]), uint32(fftColor[1]), uint32(fftColor[2])},
		})
		chkPrint(err)
		rw.Write(jsonOut)
	})
	http.HandleFunc("/music/setEnergyLevel", func(rw http.ResponseWriter, r *http.Request) {
		energyLevel := r.URL.RawQuery
		energyLevelInt, err := strconv.Atoi(energyLevel)
		chkPrint(err)
		fftColorBufferSize = energyLevelInt
	})
	http.HandleFunc("/music/getVariables", func(rw http.ResponseWriter, r *http.Request) {
		vars, err := json.Marshal(struct {
			IsStreaming           bool
			AudioStreamBufferSize int
			FFTColorBufferSize    int
			ColorShift            float64
			ColorBrightness       float64
		}{
			IsStreaming:           streaming,
			AudioStreamBufferSize: audioStreamBufferSize,
			FFTColorBufferSize:    fftColorBufferSize,
			ColorShift:            fftColorShift,
			ColorBrightness:       colorBrightness,
		})
		chkPrint(err)
		rw.Write(vars)
	})
	http.HandleFunc("/music/setColorShift", func(rw http.ResponseWriter, r *http.Request) {
		energyLevel := r.URL.RawQuery
		energyLevelInt, err := strconv.Atoi(energyLevel)
		chkPrint(err)
		fftColorShift = float64(energyLevelInt)
	})
	http.HandleFunc("/music/setColorBrightness", func(rw http.ResponseWriter, r *http.Request) {
		energyLevel := r.URL.RawQuery
		energyLevelInt, err := strconv.Atoi(energyLevel)
		chkPrint(err)
		colorBrightness = float64(energyLevelInt)
	})
}

func InitComms() (chan []byte, error) {

	listenAddr := handleErrPrint(net.ResolveUDPAddr("udp4", Gport)).(*net.UDPAddr)
	server := handleErrPrint(net.ListenUDP("udp4", listenAddr)).(*net.UDPConn)
	Uaddr = handleErrPrint(net.ResolveUDPAddr("udp4", Gaddr+Gport)).(*net.UDPAddr)

	// go newClientListener(list)
	go colorServer(server, Uaddr, colorUpdate)

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
