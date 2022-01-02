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
var numBins int = 1 << 13 // number of bins for fft (number of datapoints across the output fft array)

var fftColorBuffer [][]float64
var fftColorBufferSize int = 11
var fftColorShift float64 = 0

var colorBrightness float32 = 255

var colorOut *[3]uint8

//const webServerAddr = ":9002"

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
			Color: fftColor,
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
			ColorBrightness       float32
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
		colorBrightness = float32(energyLevelInt)
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

// takes audio stream, analyses the audio and writes the output to color
func AnalyzeAudio(color *[3]uint8) {
	colorOut = color
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

	// The distributed leds main function will start the server we dont have to
	// bc we are setting the main http server endpoints.
	//log.Println("Starting Server On", webServerAddr)
	//log.Fatal(http.ListenAndServe(webServerAddr, nil))
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
				NFFT:      numBins, //nfft should be power of 2
				Pad:       numBins, //same as NFFT
				Window:    window.Bartlett,
				Scale_off: false,
			}
			pxx, freq = spectral.Pwelch(buffercomplex, float64(int(sampleRate)*len(numChannels)), opt)
			rangeFreq := computeFreqIdx(3000, int(sampleRate), opt.Pad)
			pxx, freq = pxx[:rangeFreq], freq[:rangeFreq]
			normalizePower(pxx)
			computeColor(sampleRate, opt.Pad)
			(*colorOut)[0], (*colorOut)[1], (*colorOut)[2] = uint8(fftColor[0]), uint8(fftColor[1]), uint8(fftColor[2])
		}
	}
}

// converts the pxx and freq to rgb values
// and saves them to the 'fftColor' variable
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

	scale2Brightness := func(rgbvalue uint32) uint32 {
		if colorBrightness == 255 {
			return rgbvalue
		}
		rgbf := float32(rgbvalue)
		b := colorBrightness / 255
		return uint32(rgbf * b)
	}

	fftColor[0], fftColor[1], fftColor[2] = scale2Brightness(rgbRotated[0]), scale2Brightness(rgbRotated[1]), scale2Brightness(rgbRotated[2])

	//os.Stdout.Write([]byte{uint8(fftColor[0]), uint8(fftColor[1]), uint8(fftColor[2])})
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

func rotateColor(rgb []float64, rotDeg float64) []uint32 {
	pi := 3.14159265
	sqrtf := func(x float64) float64 {
		return math.Sqrt(x)
	}

	clamp := func(v float64) uint32 {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint32(v)
	}

	cosA := math.Cos(rotDeg * pi / 180) //convert degrees to radians
	sinA := math.Sin(rotDeg * pi / 180) //convert degrees to radians
	//calculate the rotation matrix, only depends on Hue
	matrix := [][]float64{{cosA + (1.0-cosA)/3.0, 1.0/3.0*(1.0-cosA) - sqrtf(1.0/3.0)*sinA, 1.0/3.0*(1.0-cosA) + sqrtf(1.0/3.0)*sinA},
		{1.0/3.0*(1.0-cosA) + sqrtf(1.0/3.0)*sinA, cosA + 1.0/3.0*(1.0-cosA), 1.0/3.0*(1.0-cosA) - sqrtf(1.0/3.0)*sinA},
		{1.0/3.0*(1.0-cosA) - sqrtf(1.0/3.0)*sinA, 1.0/3.0*(1.0-cosA) + sqrtf(1.0/3.0)*sinA, cosA + 1.0/3.0*(1.0-cosA)}}

	out := []uint32{0, 0, 0}

	//Use the rotation matrix to convert the RGB directly
	out[2] = clamp(rgb[0]*matrix[2][0] + rgb[1]*matrix[2][1] + rgb[2]*matrix[2][2])
	out[0] = clamp(rgb[0]*matrix[0][0] + rgb[1]*matrix[0][1] + rgb[2]*matrix[0][2])
	out[1] = clamp(rgb[0]*matrix[1][0] + rgb[1]*matrix[1][1] + rgb[2]*matrix[1][2])
	return out
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

func chkFatal(e error, msg string) {
	if e != nil {
		log.Fatalln(msg+":", e)
	}
}

func chkPrint(err error) {
	if err != nil {
		log.Println(err)
	}
}
