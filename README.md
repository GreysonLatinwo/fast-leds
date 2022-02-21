# fast-leds

## Assumptions:

Running on a Raspberrypi

Tested on Raspberry pi 4 8gb Raspian Buster - golang 17.5 and the most up-to-date software of the time.
    
Leds strip

ws2812
    
512 pixels
    
connected to pin 18
    
pulseaudio installed (do not install pulse audio if the device is only gonna be a client)

there is a problem with using pulseaudio and leds at the same time on rpi3

## Installation:
    
intall bluetooth tools

    // only for server
    sudo apt install bluez-tools pulseaudio-module-bluetooth

install go (https://go.dev/dl)
```
wget https://go.dev/dl/go1.17.5.linux-armv6l.tar.gz //replace with the most recent golang version
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.17.5.linux-armv6l.tar.gz //replace with the downloaded golang version
echo export\ PATH=$PATH:/usr/local/go/bin >> ~/.profile
source ~/.profile
```

install scons
    `sudo apt install scons`

clone rpi_ws281x repo

`git clone https://github.com/jgarff/rpi_ws281x.git`

build it (https://github.com/rpi-ws281x/rpi-ws281x-go#compiling-directly-on-the-raspberry-pi)

```shell
scons
sudo cp *.a /usr/local/lib && sudo cp *.h /usr/local/include
```
        
clone fast-led repo

`git clone https://github.com/GreysonLatinwo/fast-leds.git`

`cd fast-leds/Server`

## Usage:

shell run

`./build.sh`

`./run.sh -h`

`./run.sh`

Webserver running on port 9001
