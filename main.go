package main

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"net/http"
	"os/exec"
	"time"

	"github.com/godbus/dbus"
	"github.com/mjibson/go-dsp/spectral"
	"github.com/mjibson/go-dsp/window"
	"github.com/sqp/pulseaudio"

	"fmt"
	"log"
	"strconv"
)

// Create a pulse dbus service with 2 clients, listen to events,
// then use some properties.
var app *AppPulse
var pulse *pulseaudio.Client
var isModuleLoaded bool

var pxx, freq []float64
var fftColor []uint32 = []uint32{0, 0, 0}
var streaming = true
var audioStreamBufferSize = 1 << 12

const webServerAddr = ":9000"

//initalize webServer
func init() {
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) { http.ServeFile(rw, r, "public/index.html") })
	http.HandleFunc("/style.css", func(rw http.ResponseWriter, r *http.Request) { http.ServeFile(rw, r, "public/style.css") })
	http.HandleFunc("/app.js", func(rw http.ResponseWriter, r *http.Request) { http.ServeFile(rw, r, "public/app.js") })
	http.HandleFunc("/getData", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "utf-8")
		jsonOut, err := json.Marshal(struct {
			Power []float64
			Freq  []float64
			Color []uint32 // [r,g,b]
		}{
			Power: pxx,
			Freq:  freq,
			Color: fftColor,
		})
		chkPrint(err)
		rw.Write(jsonOut)

		//print strongest freq
		// var maxIdx int
		// for i, v := range pxx {
		// 	if v > pxx[maxIdx] {
		// 		maxIdx = i
		// 	}
		// }
		// log.Printf("Frequency: % 8d => Power: % 8f\n", int(freq[maxIdx]), pxx[maxIdx])
	})
}

//initalize pulseaudio
func init() {
	// Load pulseaudio DBus module if needed. This module is mandatory, but it
	// can also be configured in system files. See package doc.
	isLoaded, e := pulseaudio.ModuleIsLoaded()
	chkFatal(e, "test pulse dbus module is loaded")
	if !isLoaded {
		e = pulseaudio.LoadModule()
		chkFatal(e, "load pulse dbus module")
	}

	// Connect to the pulseaudio dbus service.
	pulse, e = pulseaudio.New()
	chkPrint(e)

	// Create and register a first client.
	app = &AppPulse{}
	pulse.Register(app)
}

func main() {
	// Start http server
	go http.ListenAndServe(webServerAddr, nil)

	if isModuleLoaded {
		defer pulseaudio.UnloadModule() // has error to test
	}
	defer pulse.Close()         // has error to test
	defer pulse.Unregister(app) // has errors to test
	// Listen to registered events.
	go pulse.Listen()
	defer pulse.StopListening()

	// Read audio stream
	ProcessAudioStream(pulse)
}

// read audio stream and compute fft
func ProcessAudioStream(client *pulseaudio.Client) {
	log.Println("Waiting for audio stream to process...")
	for {
		streams, _ := client.Core().ListPath("PlaybackStreams")
		if len(streams) < 1 {
			time.Sleep(time.Millisecond * 100)
			continue
		}
		stream := streams[0]
		log.Println("Stream found:", stream)
		// Get the device to query properties for the stream referenced by his path.
		dev := client.Stream(stream)

		// Get some informations about this stream.
		mute, _ := dev.Bool("Mute")               // bool
		vols, _ := dev.ListUint32("Volume")       // []uint32
		latency, _ := dev.Uint64("Latency")       // uint64
		sampleRate, _ := dev.Uint32("SampleRate") // uint32
		log.Println("stream", volumeText(mute, vols), "latency", latency, "sampleRate", sampleRate)

		props, e := dev.MapString("PropertyList") // map[string]string
		chkFatal(e, "get device PropertyList")
		log.Println(props)

		cmd := exec.Command("parec")
		audioStreamReader, err := cmd.StdoutPipe()
		chkFatal(err, "cmd.StdoutPipe()")
		err = cmd.Start()
		chkFatal(err, "cmd.Start()")

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
				NFFT:      1 << 13, //nfft should be power of 2
				Pad:       1 << 13, //same as NFFT
				Window:    window.Bartlett,
				Scale_off: false,
			}
			pxx, freq = spectral.Pwelch(buffercomplex, float64(sampleRate)*2, opt)
			pxx, freq = pxx[:len(pxx)/2], freq[:len(freq)/2]

			normalizePower(pxx)
			computeColor(sampleRate, opt.Pad)
		}
	}
}

//normalizes the fft power data
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

var fftColorBuffer [][]float64
var fftColorBufferSize int = 6

func computeColor(sampleRate uint32, pad int) {

	findMax := func(arr []float64) float64 {
		max := 0.0
		for _, v := range arr {
			if v > max {
				max = v
			}
		}
		return max
	}

	computeFreqIdx := func(freq int) int {
		Fs := float64(sampleRate) * 2
		coef := Fs / float64(pad)
		pos := float64(freq) / coef
		return int(math.Round(pos))
	}

	var redLowerFreq int = 0
	var greenLowerFreq int = 200
	var blueLowerFreq int = 600
	var blueupperFreq int = 2800

	redLowerIdx := computeFreqIdx(redLowerFreq)
	greenLowerIdx := computeFreqIdx(greenLowerFreq)
	blueLowerIdx := computeFreqIdx(blueLowerFreq)
	blueupperIdx := computeFreqIdx(blueupperFreq)

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
	if len(fftColorBuffer) >= fftColorBufferSize {
		fftColorBuffer = fftColorBuffer[1:]
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

	fftColor[0], fftColor[1], fftColor[2] = uint32(redAvg), uint32(greenAvg), uint32(blueAvg)

	fmt.Println(fftColor[0]*256*256 + fftColor[1]*256 + fftColor[2])
	//log.Printf("% 4d,% 4d,% 4d\n", fftColor[0], fftColor[1], fftColor[2])
}

//*************************Callback Event Func****************************

// AppPulse is a client that connects 6 callbacks.
//
type AppPulse struct{}

// NewSink is called when a sink is added.
//
func (ap *AppPulse) NewSink(path dbus.ObjectPath) {
	log.Println("new sink:", path)
}

// SinkRemoved is called when a sink is removed.
//
func (ap *AppPulse) SinkRemoved(path dbus.ObjectPath) {
	log.Println("sink removed:", path)
}

// NewPlaybackStream is called when a playback stream is added.
//
func (ap *AppPulse) NewPlaybackStream(path dbus.ObjectPath) {
	streaming = true
	log.Println("new playback stream:", path)
}

// PlaybackStreamRemoved is called when a playback stream is removed.
//
func (ap *AppPulse) PlaybackStreamRemoved(path dbus.ObjectPath) {
	streaming = false
	log.Println("playback stream removed:", path)
}

// DeviceVolumeUpdated is called when the volume has changed on a device.
//
func (ap *AppPulse) DeviceVolumeUpdated(path dbus.ObjectPath, values []uint32) {
	log.Println("device volume updated:", path, values)
}

// DeviceActiveCardUpdated is called when active card has changed on a device.
// i.e. headphones injected.
func (ap *AppPulse) DeviceActiveCardUpdated(path dbus.ObjectPath, port dbus.ObjectPath) {
	log.Println("device active card updated:", path, port)
}

// StreamVolumeUpdated is called when the volume has changed on a stream.
//
func (ap *AppPulse) StreamVolumeUpdated(path dbus.ObjectPath, values []uint32) {
	log.Println("stream volume:", path, values)
}

//************************Helper Func***************************

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

func chkFatal(e error, msg string) {
	if e != nil {
		fmt.Println(msg)
		log.Fatalln(msg+":", e)
	}
}

func chkPrint(err error) {
	if err != nil {
		fmt.Println(err)
	}
}
