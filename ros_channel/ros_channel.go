package roschannel

import (
	"fmt"
	"time"

	"gocv.io/x/gocv"
)

type ROSChannel struct {
	imgChan <-chan gocv.Mat
}

func InitCodecChannel(
	imgChan <-chan gocv.Mat,
) *ROSChannel {
	return &ROSChannel{
		imgChan: imgChan,
	}
}

func (cc *ROSChannel) Spin() {
	window := gocv.NewWindow("Image Window")
	defer window.Close()

	count := 0

	for {
		img := <-cc.imgChan
		fmt.Printf("frame %d decoded %dms\n", count, time.Now().UnixMilli())
		count++
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
