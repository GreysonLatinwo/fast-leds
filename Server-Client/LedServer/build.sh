#!/bin/bash

go build -o bin/pulseaudio PulseAudio/pulseAudio.go

go build -o bin/analyzeAudio .
