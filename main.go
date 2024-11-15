package main

import (
	"flag"

	peerconnectionchannel "github.com/3DRX/webrtc-ros-bridge/peer_connection_channel"
	roschannel "github.com/3DRX/webrtc-ros-bridge/ros_channel"
	signalingchannel "github.com/3DRX/webrtc-ros-bridge/signaling_channel"
	"github.com/pion/webrtc/v4"
	"gocv.io/x/gocv"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "http service address")
	sdpChan := make(chan webrtc.SessionDescription)
	sdpReplyChan := make(chan webrtc.SessionDescription)
	candidateChan := make(chan webrtc.ICECandidateInit)
	imgChan := make(chan gocv.Mat)
	flag.Parse()
	sc := signalingchannel.InitSignalingChannel(
		addr,
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
	cc := roschannel.InitCodecChannel(imgChan)
	go sc.Spin()
	go pc.Spin()
	go cc.Spin()
	select {}
}
