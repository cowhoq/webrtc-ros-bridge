package main

import (
	"github.com/3DRX/webrtc-ros-bridge/config"
	sensor_msgs_msg "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/sensor_msgs/msg"
	recv_peerconnectionchannel "github.com/3DRX/webrtc-ros-bridge/receiver/peer_connection_channel"
	recv_roschannel "github.com/3DRX/webrtc-ros-bridge/receiver/ros_channel"
	recv_signalingchannel "github.com/3DRX/webrtc-ros-bridge/receiver/signaling_channel"
	send_peerconnectionchannel "github.com/3DRX/webrtc-ros-bridge/sender/peer_connection_channel"
	send_roschannel "github.com/3DRX/webrtc-ros-bridge/sender/ros_channel"
	"github.com/pion/webrtc/v4"
)

func videoReceiver(cfg *config.Config, topicIdx int) {
	sdpChan := make(chan webrtc.SessionDescription)
	sdpReplyChan := make(chan webrtc.SessionDescription)
	candidateChan := make(chan webrtc.ICECandidateInit)
	imgChan := make(chan *sensor_msgs_msg.Image)
	sc := recv_signalingchannel.InitSignalingChannel(
		cfg,
		topicIdx,
		sdpChan,
		sdpReplyChan,
		candidateChan,
	)
	pc := recv_peerconnectionchannel.InitPeerConnectionChannel(
		sdpChan,
		sdpReplyChan,
		candidateChan,
		sc.SignalCandidate,
		imgChan,
	)
	cc := recv_roschannel.InitROSChannel(
		cfg,
		topicIdx,
		imgChan,
	)
	go sc.Spin()
	go pc.Spin()
	go cc.Spin()
	select {}
}

func videoSender(cfg *config.Config, topicIdx int) {
	imgChan := make(chan *sensor_msgs_msg.Image)
	cc := send_roschannel.InitROSChannel(
		cfg,
		topicIdx,
		imgChan,
	)
	pc := send_peerconnectionchannel.InitPeerConnectionChannel(
		imgChan,
	)
	go cc.Spin()
	go pc.Spin()
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
	} else if cfg.Mode == "sender" {
		for i, t := range cfg.Topics {
			if t.Type == "sensor_msgs/msg/Image" {
				go videoSender(cfg, i)
			} else {
				panic("unsupported type")
			}
		}
	} else {
		panic("unsupported mode")
	}
	select {}
}
