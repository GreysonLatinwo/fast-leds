module github.com/greysonlatinwo/fastLed/analyzeAudioStream/Communication/CLILedClient

go 1.16

require (
	github.com/rpi-ws281x/rpi-ws281x-go v1.0.8
	github.com/stretchr/testify v1.7.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

// Ensure that examples always use the go-libp2p version in the same git checkout.
// replace github.com/libp2p/go-libp2p => ../
