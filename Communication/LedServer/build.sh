#!/bin/bash

go build -o bin/ledController ledControls/controller.go

go build -o bin/analyzeAudio analyzeAudio.go communications.go pulseAudio.go utils.go
