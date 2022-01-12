#!/bin/bash
go build -o bin/analyzeAudio .
cd ledControls
go build -o ../bin/ledController .