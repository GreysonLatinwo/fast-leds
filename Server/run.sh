#!/bin/bash
#start the bluetooth server listening
sudo rfcomm -r watch 0 &
./bin/analyzeAudio.out | sudo ./bin/ledcontroller.out $1 $2 $3 $4 $5 $6
