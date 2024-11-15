package codecchannel

import (
	"fmt"
	"io"

	"gocv.io/x/gocv"
)

type CodecChannel struct {
	frameX    int
	frameY    int
	frameSize int
	ffmpegOut io.ReadCloser
}

func InitCodecChannel(
	frameX int,
	frameY int,
	frameSize int,
	ffmpegOut io.ReadCloser,
) *CodecChannel {
	return &CodecChannel{
		frameX:    frameX,
		frameY:    frameY,
		frameSize: frameSize,
		ffmpegOut: ffmpegOut,
	}
}

func (cc *CodecChannel) Spin() {
	window := gocv.NewWindow("Motion Window")
	defer window.Close()

	img := gocv.NewMat()
	defer img.Close()

	// count := 0

	for {
		buf := make([]byte, cc.frameSize)
		if _, err := io.ReadFull(cc.ffmpegOut, buf); err != nil {
			fmt.Println(err)
			continue
		}
		img, _ := gocv.NewMatFromBytes(
			cc.frameY,
			cc.frameX,
			gocv.MatTypeCV8UC3,
			buf,
		)
		if img.Empty() {
			continue
		}
		window.IMShow(img)
		if window.WaitKey(1) == 'q' {
			break
		}
		// // write to file
		// imgFile := fmt.Sprintf("frame_%d.png", count)
		// count++
		// gocv.IMWrite(imgFile, img)
	}
}
