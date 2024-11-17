package roschannel

import (
	"log/slog"

	sensor_msgs_msg "github.com/3DRX/webrtc-ros-bridge/ros_channel/msgs/sensor_msgs/msg"
	"github.com/tiiuae/rclgo/pkg/rclgo"
)

type ROSChannel struct {
	imgChan <-chan sensor_msgs_msg.Image
}

func InitROSChannel(imgChan <-chan sensor_msgs_msg.Image) *ROSChannel {
	return &ROSChannel{
		imgChan: imgChan,
	}
}

func (cc *ROSChannel) Spin() {
	err := rclgo.Init(nil)
	if err != nil {
		panic(err)
	}
	node, err := rclgo.NewNode("webrtc_bridge_client", "")
	if err != nil {
		panic(err)
	}
	defer node.Close()
	pub, err := sensor_msgs_msg.NewImagePublisher(node, "/image", nil)
	if err != nil {
		panic(err)
	}
	defer pub.Close()
	defer rclgo.Uninit()

	for {
		img := <-cc.imgChan
		err := pub.Publish(&img)
		if err != nil {
			slog.Error("Failed to publish image message", "error", err)
		}
	}
}
