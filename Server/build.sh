#!/bin/bash
go build -o bin/analyzeAudio .
cd ledcontroller
go build -o ../bin/ledcontroller .