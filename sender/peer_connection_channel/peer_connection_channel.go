package peerconnectionchannel

/*
#cgo LDFLAGS: -L. -lvp8encoder -lvpx -lm
#include "vp8_encoder.h"
#include "../../receiver/peer_connection_channel/vp8_decoder.h"
#include <stdlib.h>
*/
import "C"

import (
	"log/slog"
	"os"
	"unsafe"

	sensor_msgs_msg "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/sensor_msgs/msg"
	"github.com/at-wat/ebml-go/webm"
	"github.com/pion/webrtc/v4"
)

type PeerConnectionChannel struct {
	imgChan           <-chan *sensor_msgs_msg.Image
	codec             C.vpx_codec_ctx_t
	writer            webm.BlockWriteCloser
	sendSDPChan       <-chan webrtc.SessionDescription
	recvSDPChan       chan<- webrtc.SessionDescription
	sendCandidateChan <-chan webrtc.ICECandidateInit
	recvCandidateChan chan<- webrtc.ICECandidateInit
}

func InitPeerConnectionChannel(
	imgChan <-chan *sensor_msgs_msg.Image,
	sendSDPChan <-chan webrtc.SessionDescription,
	recvSDPChan chan<- webrtc.SessionDescription,
	sendCandidateChan <-chan webrtc.ICECandidateInit,
	recvCandidateChan chan<- webrtc.ICECandidateInit,
) *PeerConnectionChannel {
	pc := &PeerConnectionChannel{
		imgChan:           imgChan,
		sendSDPChan:       sendSDPChan,
		recvSDPChan:       recvSDPChan,
		sendCandidateChan: sendCandidateChan,
		recvCandidateChan: recvCandidateChan,
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
	defer pc.writer.Close()

	for {
		img := <-pc.imgChan

		var ros_img_c C.sensor_msgs__msg__Image
		sensor_msgs_msg.ImageTypeSupport.AsCStruct(unsafe.Pointer(&ros_img_c), img)
		var data *C.uint8_t
		var dataSize C.size_t
		if C.convert_and_encode(&pc.codec, &ros_img_c, &data, &dataSize) != 0 {
			slog.Error("Failed to encode frame")
			continue
		}
		// Write encoded data to WebM file
		goData := C.GoBytes(unsafe.Pointer(data), C.int(dataSize))
		videoKeyframe := (goData[0] & 0x1) == 0 // Check if the frame is a keyframe
		// Calculate timeStamp from img.Header
		timeStampMs := uint32(img.Header.Stamp.Sec)*1000 + img.Header.Stamp.Nanosec/1000000
		if _, err := pc.writer.Write(videoKeyframe, int64(timeStampMs), goData); err != nil {
			slog.Error("Failed to write to WebM file", "error", err)
		}
		// Free the memory allocated for the encoded data
		C.cleanup_ros_image(&ros_img_c)
		C.free(unsafe.Pointer(data))

		slog.Info(
			"Received image message",
			"width", img.Width,
			"height", img.Height,
			"header", img.Header,
			"keyframe", videoKeyframe,
		)
	}
}
