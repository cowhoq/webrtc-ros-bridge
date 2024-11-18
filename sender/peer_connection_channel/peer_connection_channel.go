package peerconnectionchannel

/*
#cgo LDFLAGS: -L. -lvp8encoder -lvpx -lm
#include "vp8_encoder.h"
*/
import "C"

import (
	"log/slog"

	sensor_msgs_msg "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/sensor_msgs/msg"
)

type PeerConnectionChannel struct {
	imgChan <-chan *sensor_msgs_msg.Image
}

func InitPeerConnectionChannel(
	imgChan <-chan *sensor_msgs_msg.Image,
) *PeerConnectionChannel {
	return &PeerConnectionChannel{
		imgChan: imgChan,
	}
}

func (pc *PeerConnectionChannel) Spin() {
	for {
		img := <-pc.imgChan
		slog.Info(
			"Received image message",
			"width",
			img.Width,
			"height",
			img.Height,
			"header",
			img.Header,
		)
	}
}
