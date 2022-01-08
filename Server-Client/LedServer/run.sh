#!/bin/bash
if [ $# -gt 0 ]; then
echo './bin/analyzeAudio | sudo ./bin/ledController' $1 $2 $3 $4
./bin/analyzeAudio | sudo ./bin/ledController $1 $2 $3 $4
else
./bin/analyzeAudio | sudo ./bin/ledController --renderType running=spinning=center
fi