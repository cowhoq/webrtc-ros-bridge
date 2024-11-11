package main

import (
	"flag"
	"sync"

	peerconnectionchannel "github.com/3DRX/webrtc-ros-bridge/peer_connection_channel"
	signalingchannel "github.com/3DRX/webrtc-ros-bridge/signaling_channel"
	"github.com/pion/webrtc/v4"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "http service address")
	sdpChan := make(chan webrtc.SessionDescription)
	sdpReplyChan := make(chan webrtc.SessionDescription)
	candidateChan := make(chan webrtc.ICECandidateInit)
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
	)
	go sc.Spin()
	go pc.Spin()
	select {}
}
