package main

import (
	"log"
	"os"
	"testing"
)

func TestSpinningHues(t *testing.T) {
	ledCount = 21
	leds = make([]uint32, ledCount)
	color1 := []float64{0, 255, 255}
	color2 := []float64{170, 0, 255}
	color3 := []float64{0, 255, 191}
	generateSpinningHuesLeds(color1, color2, color3)

	log.Println(leds)
}

func TestRotateLeds(t *testing.T) {
	ledCount = 10
	leds = make([]uint32, ledCount)
	color1 := []float64{0, 255, 255}
	color2 := []float64{170, 0, 255}
	color3 := []float64{0, 255, 191}

	leds = generateSpinningHuesLeds(color1, color2, color3)
	rotateLeds(60)
}

func TestReverseArr(t *testing.T) {
	data, _ := os.ReadFile("preset.go")
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
	os.Stdout.Write(data)
}
