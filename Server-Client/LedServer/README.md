Assumptions:
    Running on a Raspberrypi
        Tested on Raspberry pi 4 8gb Raspian
    Leds strip
        ws2812
        63 pixels
        connected to pin 18
    pulseaudio installed

Installation:
    sudo apt install bluez-tools pulseaudio-module-bluetooth
    install go
        wget https://go.dev/dl/go1.17.5.linux-armv6l.tar.gz
        rm -rf /usr/local/go && tar -C /usr/local -xzf go1.17.5.linux-armv6l.tar.gz
        echo export\ PATH=$PATH:/usr/local/go/bin >> ~/.profile
        source ~/.profile
    install scons
    
    clone rpi_ws281x repo (https://github.com/jgarff/rpi_ws281x.git)
        build it //https://github.com/rpi-ws281x/rpi-ws281x-go#compiling-directly-on-the-raspberry-pi
            ```shell
                scons
                sudo cp *.a /usr/local/lib
                sudo cp *.h /usr/local/include
            ```
    clone fast-led repo (https://github.com/GreysonLatinwo/fast-leds.git)
    cd fast-leds/Server-Client/LedServer

Usage:
    run ./build.sh
    run ./run.sh
    Connect to RPi via Bluetooth and play music.
        Makesure you play the audio through the hdmi output
    
    Webserver running on port 9001