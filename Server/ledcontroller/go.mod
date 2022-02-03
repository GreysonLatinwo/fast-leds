module github.com/greysonlatinwo/fast-leds/Server/ledcontroller

go 1.17

require (
	github.com/greysonlatinwo/fast-leds/utils v0.0.0-20220121162157-a011b68f7a69
	github.com/rpi-ws281x/rpi-ws281x-go v1.0.8
)

replace github.com/greysonlatinwo/fast-leds/utils => ../../utils

require (
	github.com/montanaflynn/stats v0.6.6 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/stretchr/testify v1.7.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
