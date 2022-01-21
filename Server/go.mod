module github.com/greysonlatinwo/fast-leds/Server

go 1.16

require (
	github.com/godbus/dbus v4.1.0+incompatible
	github.com/greysonlatinwo/fast-leds/utils v0.0.0-20220121143809-ab9e30a6cec6
	github.com/libp2p/go-libp2p v0.17.0
	github.com/libp2p/go-libp2p-core v0.14.0
	github.com/mjibson/go-dsp v0.0.0-20180508042940-11479a337f12
	github.com/sqp/pulseaudio v0.0.0-20180916175200-29ac6bfa231c
)

replace github.com/greysonlatinwo/fast-leds/utils => ../utils