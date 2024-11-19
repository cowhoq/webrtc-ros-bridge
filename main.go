package main

import (
	"github.com/3DRX/webrtc-ros-bridge/config"
	sensor_msgs_msg "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/sensor_msgs/msg"
	recv_peerconnectionchannel "github.com/3DRX/webrtc-ros-bridge/receiver/peer_connection_channel"
	recv_roschannel "github.com/3DRX/webrtc-ros-bridge/receiver/ros_channel"
	recv_signalingchannel "github.com/3DRX/webrtc-ros-bridge/receiver/signaling_channel"
	send_peerconnectionchannel "github.com/3DRX/webrtc-ros-bridge/sender/peer_connection_channel"
	send_roschannel "github.com/3DRX/webrtc-ros-bridge/sender/ros_channel"
	send_signalingchannel "github.com/3DRX/webrtc-ros-bridge/sender/signaling_channel"
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
	rc := recv_roschannel.InitROSChannel(
		cfg,
		topicIdx,
		imgChan,
	)
	go sc.Spin()
	go pc.Spin()
	go rc.Spin()
	select {}
}

func videoSender(cfg *config.Config, topicIdx int) {
	imgChan := make(chan *sensor_msgs_msg.Image)
	sendSDPChan := make(chan webrtc.SessionDescription)
	recvSDPChan := make(chan webrtc.SessionDescription)
	sendCandidateChan := make(chan webrtc.ICECandidateInit)
	recvCandidateChan := make(chan webrtc.ICECandidateInit)
	sc := send_signalingchannel.InitSignalingChannel(
		cfg,
		topicIdx,
		sendSDPChan,
		recvSDPChan,
		sendCandidateChan,
		recvCandidateChan,
	)
	haveReceiverPromise := sc.Spin()
	<-haveReceiverPromise
	actions := sc.GetActions()
	rc := send_roschannel.InitROSChannel(
		cfg,
		topicIdx,
		imgChan,
	)
	pc := send_peerconnectionchannel.InitPeerConnectionChannel(
		imgChan,
		sendSDPChan,
		recvSDPChan,
		sendCandidateChan,
		recvCandidateChan,
		actions,
	)
	go rc.Spin()
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
