module github.com/greysonlatinwo/fast-leds/Server/ledcontroller

go 1.16

require (
	github.com/greysonlatinwo/fast-leds/ledcontrols v0.0.0-20220121153521-bb5cc5c51f7c
	github.com/greysonlatinwo/fast-leds/utils v0.0.0-20220121153521-bb5cc5c51f7c
	github.com/rpi-ws281x/rpi-ws281x-go v1.0.8
)

replace (
	github.com/greysonlatinwo/fast-leds/ledcontrols => ../../ledcontrols
	github.com/greysonlatinwo/fast-leds/utils => ../../utils
)
