package roschannel

import (
	"gocv.io/x/gocv"
)

type ROSChannel struct {
	imgChan <-chan gocv.Mat
}

func InitCodecChannel(imgChan <-chan gocv.Mat) *ROSChannel {
	return &ROSChannel{
		imgChan: imgChan,
	}
}

func (cc *ROSChannel) Spin() {
	window := gocv.NewWindow("Image Window")
	defer window.Close()

	for {
		img := <-cc.imgChan
		if img.Empty() {
			continue
		}
		window.IMShow(img)
		if window.WaitKey(1) == 'q' {
			break
		}
		img.Close()
	}
}
