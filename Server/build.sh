#!/bin/bash
go build -o bin/analyzeAudio.out .
cd ledcontroller
go build -o ../bin/ledcontroller.out .