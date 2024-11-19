package roschannel

import (
	"log/slog"
	"strings"
	"time"

	"github.com/3DRX/webrtc-ros-bridge/config"
	sensor_msgs_msg "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/sensor_msgs/msg"
	"github.com/tiiuae/rclgo/pkg/rclgo"
)

type ROSChannel struct {
	imgChan  <-chan *sensor_msgs_msg.Image
	cfg      *config.Config
	topicIdx int
}

func InitROSChannel(
	cfg *config.Config,
	topicIdx int,
	imgChan <-chan *sensor_msgs_msg.Image,
) *ROSChannel {
	return &ROSChannel{
		cfg:      cfg,
		topicIdx: topicIdx,
		imgChan:  imgChan,
	}
}

func (r *ROSChannel) Spin() {
	err := rclgo.Init(nil)
	if err != nil {
		panic(err)
	}
	nodeName := "webrtc_ros_bridge_" + r.cfg.Mode + "_" + r.cfg.Topics[r.topicIdx].Type + "_" + r.cfg.Topics[r.topicIdx].NameOut
	nodeName = strings.ReplaceAll(nodeName, "/", "_")
	node, err := rclgo.NewNode(nodeName, "")
	if err != nil {
		panic(err)
	}
	defer node.Close()
	pub, err := sensor_msgs_msg.NewImagePublisher(node, "/"+r.cfg.Topics[r.topicIdx].NameOut, nil)
	if err != nil {
		panic(err)
	}
	defer pub.Close()
	defer rclgo.Uninit()

	lastTimeStamp := time.Now().UnixMilli()

	for {
		img := <-r.imgChan
		now := time.Now().UnixMilli()
		err := pub.Publish(img)
		if err != nil {
			slog.Error("Failed to publish image message", "error", err)
		}
		slog.Info("Published image msg", "FPS", 1000/(now-lastTimeStamp))
		lastTimeStamp = now
	}
}
