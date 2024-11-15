package main

import (
	"flag"
	"sync"

	peerconnectionchannel "github.com/3DRX/webrtc-ros-bridge/peer_connection_channel"
	roschannel "github.com/3DRX/webrtc-ros-bridge/ros_channel"
	signalingchannel "github.com/3DRX/webrtc-ros-bridge/signaling_channel"
	"github.com/pion/webrtc/v4"
	"gocv.io/x/gocv"
)

const (
	frameX    = 1280
	frameY    = 720
	frameSize = frameX * frameY * 3
)

func main() {
	addr := flag.String("addr", "localhost:8080", "http service address")
	sdpChan := make(chan webrtc.SessionDescription)
	sdpReplyChan := make(chan webrtc.SessionDescription)
	candidateChan := make(chan webrtc.ICECandidateInit)
	imgChan := make(chan gocv.Mat)
	var candidatesMux sync.Mutex
	pendingCandidates := make([]*webrtc.ICECandidate, 0)
	flag.Parse()
	sc := signalingchannel.InitSignalingChannel(
		addr,
		sdpChan,
		sdpReplyChan,
		candidateChan,
		pendingCandidates,
		&candidatesMux,
	)
	pc := peerconnectionchannel.InitPeerConnectionChannel(
		sdpChan,
		sdpReplyChan,
		candidateChan,
		pendingCandidates,
		&candidatesMux,
		sc.SignalCandidate,
		imgChan,
	)
	cc := roschannel.InitCodecChannel(imgChan)
	go sc.Spin()
	go pc.Spin()
	go cc.Spin()
	select {}
}
