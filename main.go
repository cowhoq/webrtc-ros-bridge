package main

import (
	"flag"

	peerconnectionchannel "github.com/3DRX/webrtc-ros-bridge/peer_connection_channel"
	signalingchannel "github.com/3DRX/webrtc-ros-bridge/signaling_channel"
	"github.com/pion/webrtc/v4"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "http service address")
	sdpChan := make(chan webrtc.SessionDescription)
	sdpReplyChan := make(chan webrtc.SessionDescription)
	candidateChan := make(chan webrtc.ICECandidateInit)
	candidateToPeerChan := make(chan *webrtc.ICECandidate)
	flag.Parse()
	sc := signalingchannel.InitSignalingChannel(
		addr,
		sdpChan,
		sdpReplyChan,
		candidateChan,
		candidateToPeerChan,
	)
	pc := peerconnectionchannel.InitPeerConnectionChannel(
		sdpChan,
		sdpReplyChan,
		candidateChan,
		candidateToPeerChan,
		sc.SignalCandidate,
	)
	go sc.Spin()
	go pc.Spin()
	select {}
}
