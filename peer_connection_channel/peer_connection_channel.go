package peerconnectionchannel

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
	"gocv.io/x/gocv"
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
	imgChan           chan<- gocv.Mat
}

func registerHeaderExtensionURI(m *webrtc.MediaEngine, uris []string) {
	for _, uri := range uris {
		err := m.RegisterHeaderExtension(
			webrtc.RTPHeaderExtensionCapability{
				URI: uri,
			},
			webrtc.RTPCodecTypeVideo,
			webrtc.RTPTransceiverDirectionRecvonly,
		)
		if err != nil {
			panic(err)
		}
	}
}

func InitPeerConnectionChannel(
	sdpChan chan webrtc.SessionDescription,
	sdpReplyChan chan<- webrtc.SessionDescription,
	candidateChan <-chan webrtc.ICECandidateInit,
	pendingCandidates []*webrtc.ICECandidate,
	candidatesMux *sync.Mutex,
	signalCandidate func(c webrtc.ICECandidateInit) error,
	imgChan chan<- gocv.Mat,
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

	if err := m.RegisterDefaultCodecs(); err != nil {
		panic(err)
	}

	registerHeaderExtensionURI(m, []string{
		"urn:ietf:params:rtp-hdrext:toffset",
		"http://www.webrtc.org/experiments/rtp-hdrext/abs-send-time",
		"urn:3gpp:video-orientation",
		"http://www.ietf.org/id/draft-holmer-rmcat-transport-wide-cc-extensions-01",
		"http://www.webrtc.org/experiments/rtp-hdrext/playout-delay",
		"http://www.webrtc.org/experiments/rtp-hdrext/video-content-type",
		"http://www.webrtc.org/experiments/rtp-hdrext/video-timing",
		"http://www.webrtc.org/experiments/rtp-hdrext/color-space",
	})

	i := &interceptor.Registry{}
	// Use the default set of interceptors
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		panic(err)
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i))
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
		imgChan:           imgChan,
	}
}

func handleSignalingMessage(pc *PeerConnectionChannel) {
	for {
		select {
		case sdp := <-pc.sdpChan:
			slog.Info("received SDP", "sdp", sdp.SDP)
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

func (pc *PeerConnectionChannel) Spin() {
	webmSaver := newWebmSaver(pc.imgChan)
	_, err := pc.peerConnection.AddTransceiverFromKind(webrtc.RTPCodecTypeVideo,
		webrtc.RTPTransceiverInit{
			Direction: webrtc.RTPTransceiverDirectionRecvonly,
		},
	)
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
	pc.peerConnection.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		slog.Info("PeerConnectionChannel: received track", "track", track)
		fmt.Printf("Track has started, of type %d: %s \n", track.PayloadType(), track.Codec().RTPCodecCapability.MimeType)
		if track.Kind() == webrtc.RTPCodecTypeVideo {
			// Send a PLI on an interval so that the publisher is pushing a keyframe every rtcpPLIInterval
			go func() {
				ticker := time.NewTicker(time.Second * 3)
				for range ticker.C {
					errSend := pc.peerConnection.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(track.SSRC())}})
					if errSend != nil {
						fmt.Println(errSend)
					}
				}
			}()
		}
		for {
			rtp, _, readErr := track.ReadRTP()
			if readErr != nil {
				panic(readErr)
			}
			webmSaver.PushVP8(rtp)
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
			// Gracefully shutdown the peer connection
			if closeErr := pc.peerConnection.Close(); closeErr != nil {
				panic(closeErr)
			}
			os.Exit(0)
		}
	})
	go handleSignalingMessage(pc)
}
