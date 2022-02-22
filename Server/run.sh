#!/bin/bash
librespot --name=fast-leds --device=/home/pi/Music/spotify --backend=pipe &
aplay -f cb /home/pi/Music/spotify &
./bin/analyzeAudio.out | sudo ./bin/ledcontroller.out $1 $2 $3 $4 $5 $6
