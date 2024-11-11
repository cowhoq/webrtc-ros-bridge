package peerconnectionchannel

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
	"github.com/pion/webrtc/v4/pkg/media/ivfwriter"
)

type PeerConnectionChannel struct {
	sdpChan           <-chan webrtc.SessionDescription
	sdpReplyChan      chan<- webrtc.SessionDescription
	candidateChan     <-chan webrtc.ICECandidateInit
	pendingCandidates []*webrtc.ICECandidate
	candidatesMux     *sync.Mutex
	peerConnection    *webrtc.PeerConnection
	m                 *webrtc.MediaEngine
	signalCandidate   func(c webrtc.ICECandidateInit) error
}

func InitPeerConnectionChannel(
	sdpChan chan webrtc.SessionDescription,
	sdpReplyChan chan<- webrtc.SessionDescription,
	candidateChan <-chan webrtc.ICECandidateInit,
	pendingCandidates []*webrtc.ICECandidate,
	candidatesMux *sync.Mutex,
	signalCandidate func(c webrtc.ICECandidateInit) error,
) *PeerConnectionChannel {
	m := &webrtc.MediaEngine{}
	// Register VP8
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000, Channels: 0},
		PayloadType:        96,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}

	// Register VP9
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP9, ClockRate: 90000, Channels: 0},
		PayloadType:        98,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}

	// Register H264
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000, Channels: 0},
		PayloadType:        102,
	}, webrtc.RTPCodecTypeVideo); err != nil {
		panic(err)
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m))
	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}
	peerConnection, err := api.NewPeerConnection(config)
	if err != nil {
		panic(err)
	}
	return &PeerConnectionChannel{
		sdpChan:           sdpChan,
		sdpReplyChan:      sdpReplyChan,
		candidateChan:     candidateChan,
		pendingCandidates: pendingCandidates,
		candidatesMux:     candidatesMux,
		peerConnection:    peerConnection,
		m:                 m,
		signalCandidate:   signalCandidate,
	}
}

func handleSignalingMessage(pc *PeerConnectionChannel) {
	for {
		select {
		case sdp := <-pc.sdpChan:
			err := pc.peerConnection.SetRemoteDescription(sdp)
			if err != nil {
				panic(err)
			}
			answer, err := pc.peerConnection.CreateAnswer(nil)
			if err != nil {
				panic(err)
			}
			pc.sdpReplyChan <- answer
			err = pc.peerConnection.SetLocalDescription(answer)
			if err != nil {
				panic(err)
			}
			pc.candidatesMux.Lock()
			for _, c := range pc.pendingCandidates {
				onICECandidateErr := pc.signalCandidate(c.ToJSON())
				if onICECandidateErr != nil {
					panic(onICECandidateErr)
				}
			}
			pc.candidatesMux.Unlock()
		case candidate := <-pc.candidateChan:
			err := pc.peerConnection.AddICECandidate(candidate)
			if err != nil {
				panic(err)
			}
			slog.Info("received ICE candidate", "candidate", candidate)
		}
	}
}

func saveToDisk(i media.Writer, track *webrtc.TrackRemote) {
	defer func() {
		if err := i.Close(); err != nil {
			panic(err)
		}
	}()

	for {
		rtpPacket, _, err := track.ReadRTP()
		if err != nil {
			slog.Error(err.Error())
			return
		}
		if err := i.WriteRTP(rtpPacket); err != nil {
			slog.Error(err.Error())
			return
		}
	}
}

func (pc *PeerConnectionChannel) Spin() {
	_, err := pc.peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo)
	if err != nil {
		panic(err)
	}
	ivfFile, err := ivfwriter.New("output.ivf")
	if err != nil {
		panic(err)
	}
	pc.peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		pc.candidatesMux.Lock()
		defer pc.candidatesMux.Unlock()
		desc := pc.peerConnection.RemoteDescription()
		if desc == nil {
			pc.pendingCandidates = append(pc.pendingCandidates, c)
		} else if err := pc.signalCandidate(c.ToJSON()); err != nil {
			panic(err)
		}
	})
	pc.peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
		slog.Info("PeerConnectionChannel: connection state changed", "state", state)
		if state == webrtc.PeerConnectionStateFailed {
			// Wait until PeerConnection has had no network activity for 30 seconds or another failure. It may be reconnected using an ICE Restart.
			// Use webrtc.PeerConnectionStateDisconnected if you are interested in detecting faster timeout.
			// Note that the PeerConnection may come back from PeerConnectionStateDisconnected.
			slog.Info("Peer Connection has gone to failed exiting")
			os.Exit(0)
		}

		if state == webrtc.PeerConnectionStateClosed {
			// PeerConnection was explicitly closed. This usually happens from a DTLS CloseNotify
			slog.Info("Peer Connection has gone to closed exiting")
			os.Exit(0)
		}
	})
	pc.peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		slog.Info("PeerConnectionChannel: received track", "track", track)
		codec := track.Codec()
		if strings.EqualFold(codec.MimeType, webrtc.MimeTypeVP8) {
			fmt.Println("Got VP8 track, saving to disk as output.ivf")
			saveToDisk(ivfFile, track)
		}
	})
	pc.peerConnection.OnSignalingStateChange(func(state webrtc.SignalingState) {
		slog.Info("PeerConnectionChannel: signaling state changed", "state", state)
	})
	pc.peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		slog.Info("ICE Connection State changed",
			"state", connectionState,
			"signalingState", pc.peerConnection.SignalingState(),
			"connectionState", pc.peerConnection.ConnectionState(),
		)

		if connectionState == webrtc.ICEConnectionStateConnected {
			slog.Info("Ctrl+C the remote client to stop the demo")
		} else if connectionState == webrtc.ICEConnectionStateFailed || connectionState == webrtc.ICEConnectionStateClosed {
			if closeErr := ivfFile.Close(); closeErr != nil {
				panic(closeErr)
			}
			slog.Info("Done writing media files")
			// Gracefully shutdown the peer connection
			if closeErr := pc.peerConnection.Close(); closeErr != nil {
				panic(closeErr)
			}
			os.Exit(0)
		}
	})
	go handleSignalingMessage(pc)
}
