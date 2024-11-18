package peerconnectionchannel

/*
#cgo LDFLAGS: -L. -lvp8encoder -lvpx -lm
#include "vp8_encoder.h"
#include <stdlib.h>
*/
import "C"

import (
	"log/slog"
	"os"
	"unsafe"

	sensor_msgs_msg "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/sensor_msgs/msg"
	"github.com/at-wat/ebml-go/webm"
)

type PeerConnectionChannel struct {
	imgChan <-chan *sensor_msgs_msg.Image
	codec   C.vpx_codec_ctx_t
	writer  webm.BlockWriteCloser
}

func InitPeerConnectionChannel(
	imgChan <-chan *sensor_msgs_msg.Image,
) *PeerConnectionChannel {
	pc := &PeerConnectionChannel{
		imgChan: imgChan,
	}
	if C.init_encoder(&pc.codec, 640, 480, 1000) != 0 {
		panic("Failed to initialize VP8 encoder")
	}
	file, err := os.Create("video.webm")
	if err != nil {
		panic(err)
	}

	writer, err := webm.NewSimpleBlockWriter(file, []webm.TrackEntry{
		{
			Name:            "Video",
			TrackNumber:     2,
			TrackUID:        67890,
			CodecID:         "V_VP8",
			TrackType:       1,
			DefaultDuration: 33333333,
			Video: &webm.Video{
				PixelWidth:  uint64(640),
				PixelHeight: uint64(480),
			},
		},
	})
	if err != nil {
		panic(err)
	}

	pc.writer = writer[0]
	return pc
}

func (pc *PeerConnectionChannel) Spin() {
	// defer pc.writer.Close()

	for {
		img := <-pc.imgChan
		slog.Info(
			"Received image message",
			"width", img.Width,
			"height", img.Height,
			"header", img.Header,
		)

		// Allocate memory for the C sensor_msgs__msg__Image structure
		cImg := (*C.sensor_msgs__msg__Image)(C.malloc(C.sizeof_sensor_msgs__msg__Image))
		defer C.free(unsafe.Pointer(cImg))

		// Copy data from Go structure to C structure
		cImg.width = C.uint32_t(img.Width)
		cImg.height = C.uint32_t(img.Height)
		cImg.step = C.uint32_t(img.Step)
		cImg.data.size = C.size_t(len(img.Data))
		cImg.data.capacity = C.size_t(cap(img.Data))
		cImg.data.data = (*C.uint8_t)(C.CBytes(img.Data))
		defer C.free(unsafe.Pointer(cImg.data.data))

		var vpxImg C.vpx_image_t
		C.ros_to_vpx_image(cImg, &vpxImg)

		var data *C.uint8_t
		var dataSize C.size_t
		if C.encode_frame(&pc.codec, &vpxImg, &data, &dataSize) != 0 {
			slog.Error("Failed to encode frame")
			continue
		}

		// Write encoded data to WebM file
		goData := C.GoBytes(unsafe.Pointer(data), C.int(dataSize))
		videoKeyframe := (goData[0] & 0x1) == 0 // Check if the frame is a keyframe

		// calculate timeStamp from img.Header
		timeStampMs := uint32(img.Header.Stamp.Sec)*1000 + img.Header.Stamp.Nanosec/1000000
		if _, err := pc.writer.Write(videoKeyframe, int64(timeStampMs), goData); err != nil {
			slog.Error("Failed to write to WebM file", "error", err)
		}

		C.cleanup_vpx_image(&vpxImg)
	}
}
