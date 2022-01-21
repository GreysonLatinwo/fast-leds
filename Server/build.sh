#!/bin/bash
go build -o bin/analyzeAudio .
cd ledController
go build -o ../bin/ledController .