#!/bin/bash
sudo apt install scons > /dev/null &
cd ~/Documents && git clone https://github.com/jgarff/rpi_ws281x.git
cd rpi_ws281x && scons && sudo cp *.a /usr/local/lib && sudo cp *.h /usr/local/include
cd ~/Documents && git clone https://github.com/GreysonLatinwo/fast-leds.git
cd fast-leds/LedClient && ./build.sh

cat ~/.LedConsts.conf

./run.sh -c 