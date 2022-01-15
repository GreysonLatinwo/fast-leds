package main

import (
	"os"
	"testing"
)

func TestRotateLeds(t *testing.T) {
	ledCount = 10
	leds = make([]uint32, ledCount)
	hue1 := 0.5
	hue2 := 0.75
	hue3 := 1.0

	populateTempLeds(hue1, hue2, hue3)
	rotateLeds(60)
}

func TestReverseArr(t *testing.T) {
	data, _ := os.ReadFile("preset.go")
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
	os.Stdout.Write(data)
}
