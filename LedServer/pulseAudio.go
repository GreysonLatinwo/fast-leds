package main

import (
	"log"

	"github.com/godbus/dbus"
	"github.com/sqp/pulseaudio"

	utils "github.com/greysonlatinwo/fast-led/LedServer/utils"
)

// Create a pulse dbus service with 2 clients, listen to events,
// then use some properties.
var app *AppPulse
var pulse *pulseaudio.Client
var isModuleLoaded bool

//initalize pulseaudio
func init() {
	// Load pulseaudio DBus module if needed. This module is mandatory, but it
	// can also be configured in system files. See package doc.
	isLoaded, e := pulseaudio.ModuleIsLoaded()
	utils.ChkFatal(e)
	if !isLoaded {
		e = pulseaudio.LoadModule()
		utils.ChkFatal(e)
	}

	// Connect to the pulseaudio dbus service.
	pulse, e = pulseaudio.New()
	utils.ChkFatal(e)

	// Create and register a first client.
	app = &AppPulse{}
	pulse.Register(app)
}

func StartPulseAudio() {
	if isModuleLoaded {
		defer pulseaudio.UnloadModule() // has error to test
	}
	defer pulse.Close()         // has error to test
	defer pulse.Unregister(app) // has errors to test
	// Listen to registered events.
	defer pulse.StopListening()
	pulse.Listen()
}

//*************************Callback Event Func****************************

// AppPulse is a client that connects 5 callbacks.
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
func (ap *AppPulse) NewPlaybackStream(streamPath dbus.ObjectPath) {
	dev := pulse.Stream(streamPath)
	_sampleRate, _ := dev.Uint32("SampleRate") // uint32
	_numChannels, _ := dev.ListUint32("Channels")

	sampleRate = int(_sampleRate)
	numChannels = len(_numChannels)
	log.Println("new playback stream:", streamPath)
	props, e := dev.MapString("PropertyList") // map[string]string
	utils.ChkPrint(e)
	log.Println(props["media.name"])
	go ProcessAudioStream()
}

// PlaybackStreamRemoved is called when a playback stream is removed.
//
func (ap *AppPulse) DeviceVolumeUpdated(path dbus.ObjectPath, vol []uint32) {

}

// PlaybackStreamRemoved is called when a playback stream is removed.
//
func (ap *AppPulse) StreamVolumeUpdated(path dbus.ObjectPath, vol []uint32) {

}

// PlaybackStreamRemoved is called when a playback stream is removed.
//
func (ap *AppPulse) PlaybackStreamRemoved(path dbus.ObjectPath) {
	stopMusicListening <- struct{}{}
	log.Println("playback stream removed:", path)
}

// DeviceActiveCardUpdated is called when active card has changed on a device.
// i.e. headphones injected.
func (ap *AppPulse) DeviceActiveCardUpdated(path dbus.ObjectPath, port dbus.ObjectPath) {
	log.Println("device active card updated:", path, port)
}

func (ap *AppPulse) FallbackSinkUpdated(path dbus.ObjectPath) {
	log.Println("Fallback Sink Updated:", path)
}

func (ap *AppPulse) FallbackSinkUnset() {
	log.Println("Fallback Sink Unset")
}
