#!/bin/bash

go build -o bin/ledController ledControls/main.go

go build -o bin/analyzeAudio analyzeAudioStream/main.go
