package utils

func SetStaticLeds(leds []uint32, ledCount int, color uint32) {
	for i := 0; i < ledCount; i++ {
		leds[i] = color
	}
}

func SetRunningLeds(leds []uint32, runningchunkSize int, color uint32) {
	ledCount := len(leds)
	//shift leds and set new color at beginning
	for i := runningchunkSize - 1; i > 0; i-- {
		leds[i%ledCount] = leds[(i-1)%ledCount]
	}
	leds[ledCount] = color

	//duplicate for the reset of the leds
	for i := runningchunkSize; i < (ledCount); i++ {
		chunkPos := i % runningchunkSize
		leds[i%ledCount] = leds[chunkPos%ledCount]
	}
}

func SetRunningCenterLeds(leds []uint32, runningchunkSize int, color uint32) {
	ledCount := len(leds)
	//shift leds and set new color at center
	for i := 0; i < runningchunkSize/2; i++ {
		leds[i%ledCount] = leds[i+1%ledCount]
	}
	for i := runningchunkSize - 1; i > runningchunkSize/2; i-- {
		leds[i%ledCount] = leds[i-1%ledCount]
	}
	leds[int(runningchunkSize/2)%ledCount] = color

	//duplicate for the reset of the leds
	for i := runningchunkSize; i < ledCount; i++ {
		chunkPos := i % runningchunkSize
		leds[i%ledCount] = leds[chunkPos%ledCount]
	}
}
