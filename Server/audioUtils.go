package main

import (
	"log"
	"time"

	aubio "github.com/coral/aubio-go"
)

var tempo *aubio.Tempo
var bpmChan chan float64

func InitBPM() (out chan float64, err error) {
	aubio.OpenSource("", 44100, 1024)
	tempo, err = aubio.NewTempo(aubio.HFC, 1024, 1024, 44100)
	if err != nil {
		return nil, err
	}
	defer tempo.Free()

	out = make(chan float64)
	bpmChan = out

	return out, nil
}

func ComputeBPM() {
	buffercomplex := make([]float64, audioStreamBufferSize)
	for i, v := range audioStreamBuffer {
		buffercomplex[i] = float64(v)
	}
	aubioBuffer := aubio.NewSimpleBufferData(uint(audioStreamBufferSize), buffercomplex)
	tempo.Do(aubioBuffer)

	ticker := time.NewTicker(time.Second)
	for {
		<-ticker.C
		bpm := tempo.GetBpm()
		bpmChan <- bpm
		log.Println(bpm)
	}
}
