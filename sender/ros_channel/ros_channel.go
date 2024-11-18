package roschannel

import (
	"context"
	"log/slog"
	"strings"

	"github.com/3DRX/webrtc-ros-bridge/config"
	sensor_msgs_msg "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/sensor_msgs/msg"
	"github.com/tiiuae/rclgo/pkg/rclgo"
)

type ROSChannel struct {
	imgChan          chan<- *sensor_msgs_msg.Image
	cfg              *config.Config
	topicIdx         int
	haveTopicPromise chan struct{}
}

func InitROSChannel(
	cfg *config.Config,
	topicIdx int,
	imgChan chan<- *sensor_msgs_msg.Image,
) *ROSChannel {
	return &ROSChannel{
		cfg:              cfg,
		topicIdx:         topicIdx,
		imgChan:          imgChan,
		haveTopicPromise: make(chan struct{}),
	}
}

func (r *ROSChannel) Spin() <-chan struct{} {
	err := rclgo.Init(nil)
	if err != nil {
		panic(err)
	}
	nodeName := "webrtc_ros_bridge_" + r.cfg.Mode + "_" + r.cfg.Topics[r.topicIdx].Type + "_" + r.cfg.Topics[r.topicIdx].NameOut
	nodeName = strings.ReplaceAll(nodeName, "/", "_")
	slog.Info("creating node", "name", nodeName)
	go func() {
		node, err := rclgo.NewNode(nodeName, "")
		if err != nil {
			panic(err)
		}
		defer node.Close()
		receivedMsg := false
		sub, err := sensor_msgs_msg.NewImageSubscription(
			node,
			"/"+r.cfg.Topics[r.topicIdx].NameIn,
			nil,
			func(msg *sensor_msgs_msg.Image, info *rclgo.MessageInfo, err error) {
				r.imgChan <- msg
				if receivedMsg {
					r.haveTopicPromise <- struct{}{}
					receivedMsg = true
				}
			},
		)
		if err != nil {
			panic(err)
		}
		defer sub.Close()
		ws, err := rclgo.NewWaitSet()
		if err != nil {
			panic(err)
		}
		defer ws.Close()
		ws.AddSubscriptions(sub.Subscription)
		ws.Run(context.Background())
	}()
	return r.haveTopicPromise
}
