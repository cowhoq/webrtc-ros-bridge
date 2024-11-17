package main

import (
	"github.com/3DRX/webrtc-ros-bridge/config"
	sensor_msgs_msg "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/sensor_msgs/msg"
	peerconnectionchannel "github.com/3DRX/webrtc-ros-bridge/receiver/peer_connection_channel"
	roschannel "github.com/3DRX/webrtc-ros-bridge/receiver/ros_channel"
	signalingchannel "github.com/3DRX/webrtc-ros-bridge/receiver/signaling_channel"
	"github.com/pion/webrtc/v4"
)

func videoReceiver(cfg *config.Config, topicIdx int) {
	sdpChan := make(chan webrtc.SessionDescription)
	sdpReplyChan := make(chan webrtc.SessionDescription)
	candidateChan := make(chan webrtc.ICECandidateInit)
	imgChan := make(chan sensor_msgs_msg.Image)
	sc := signalingchannel.InitSignalingChannel(
		cfg,
		topicIdx,
		sdpChan,
		sdpReplyChan,
		candidateChan,
	)
	pc := peerconnectionchannel.InitPeerConnectionChannel(
		sdpChan,
		sdpReplyChan,
		candidateChan,
		sc.SignalCandidate,
		imgChan,
	)
	cc := roschannel.InitROSChannel(
		cfg,
		topicIdx,
		imgChan,
	)
	go sc.Spin()
	go pc.Spin()
	go cc.Spin()
	select {}
}

func main() {
	cfg := config.LoadCfg()
	if cfg.Mode == "receiver" {
		for i, t := range cfg.Topics {
			if t.Type == "sensor_msgs/msg/Image" {
				go videoReceiver(cfg, i)
			} else {
				panic("unsupported type")
			}
		}
	} else {
		panic("unsupported mode")
	}
	select {}
}
