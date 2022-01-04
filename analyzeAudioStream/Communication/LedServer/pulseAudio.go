package main

import (
	"log"

	"github.com/godbus/dbus"
	"github.com/sqp/pulseaudio"
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
	ChkFatal(e, "test pulse dbus module is loaded")
	if !isLoaded {
		e = pulseaudio.LoadModule()
		ChkFatal(e, "load pulse dbus module")
	}

	// Connect to the pulseaudio dbus service.
	pulse, e = pulseaudio.New()
	ChkPrint(e)

	// Create and register a first client.
	app = &AppPulse{}
	pulse.Register(app)
}

func startPulseAudio() {
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
