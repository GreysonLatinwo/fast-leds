package main

import (
	"os"
	"testing"
)

func TestReverseArr(t *testing.T) {
	data, _ := os.ReadFile("preset.go")
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}
	os.Stdout.Write(data)
}
