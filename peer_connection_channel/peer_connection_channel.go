package peerconnectionchannel

/*
#cgo LDFLAGS: -L. -lvp8decoder -lvpx -lm
#include "vp8_decoder.h"
*/
import "C"

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"unsafe"

	"github.com/pion/rtp/codecs"
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
	pc.peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		slog.Info("PeerConnectionChannel: received track", "track", track)

		if track.Codec().MimeType != webrtc.MimeTypeVP8 {
			slog.Info("Ignoring non-VP8 track", "mimeType", track.Codec().MimeType)
			return
		}

		width, height := C.uint(640), C.uint(480)
		var codec C.vpx_codec_ctx_t
		if C.init_decoder(&codec, width, height) != 0 {
			slog.Error("Failed to initialize decoder")
			return
		}
		// defer C.vpx_codec_destroy(&codec)
		slog.Info("Decoder initialized successfully", "width", width, "height", height)

		currentFrame := []byte{}
		seenKeyFrame := false
		frameCount := 0
		var lastSequenceNumber uint16

		for {
			packet, _, err := track.ReadRTP()
			if err != nil {
				slog.Error("Error reading RTP", "error", err)
				break
			}

			// Handle packet loss
			if lastSequenceNumber != 0 {
				diff := packet.SequenceNumber - lastSequenceNumber
				if diff > 1 {
					slog.Warn("Packet loss detected",
						"expected", lastSequenceNumber+1,
						"got", packet.SequenceNumber,
					)
					currentFrame = nil
					continue
				}
			}
			lastSequenceNumber = packet.SequenceNumber

			vp8Packet := codecs.VP8Packet{}
			if _, err := vp8Packet.Unmarshal(packet.Payload); err != nil {
				slog.Error("Failed to unmarshal VP8 packet", "error", err)
				continue
			}

			// Ensure we have at least one byte
			if len(vp8Packet.Payload) == 0 {
				continue
			}

			isKeyFrame := (vp8Packet.Payload[0] & 0x01) == 0

			// Wait for keyframe if we haven't seen one
			if !seenKeyFrame {
				if !isKeyFrame {
					continue
				}
				seenKeyFrame = true
				slog.Info("First keyframe received")
			}

			// Skip if we're waiting for start of frame
			if currentFrame == nil && vp8Packet.S != 1 {
				continue
			}

			// For debugging first few frames
			if frameCount < 5 && vp8Packet.S == 1 {
				slog.Debug("New frame starting",
					"frameCount", frameCount,
					"keyframe", isKeyFrame,
					"payloadSize", len(vp8Packet.Payload),
					"firstByte", fmt.Sprintf("%08b", vp8Packet.Payload[0]))
			}

			currentFrame = append(currentFrame, vp8Packet.Payload...)

			// Continue if not end of frame
			if !packet.Marker {
				continue
			}

			// Skip empty frames
			if len(currentFrame) == 0 {
				currentFrame = nil
				continue
			}

			// Debug logging for first few frames
			if frameCount < 5 {
				slog.Debug("Complete frame",
					"frameCount", frameCount,
					"size", len(currentFrame),
					"prefix", fmt.Sprintf("%x", currentFrame[:min(len(currentFrame), 16)]))
			}

			// Decode VP8 frame
			codecError := C.decode_frame(&codec, (*C.uint8_t)(&currentFrame[0]), C.size_t(len(currentFrame)))
			if codecError != 0 {
				slog.Error("Decode error", "errorCode", codecError)
				currentFrame = nil
				continue
			}

			// Get decoded frames
			var iter C.vpx_codec_iter_t
			img := C.vpx_codec_get_frame(&codec, &iter)
			if img == nil {
				slog.Error("Failed to get decoded frame")
				currentFrame = nil
				continue
			}

			actualWidth := int(img.d_w)
			actualHeight := int(img.d_h)

			// Create GoCV Mat
			goImg := gocv.NewMatWithSize(actualHeight, actualWidth, gocv.MatTypeCV8UC3)
			if goImg.Empty() {
				slog.Error("Failed to create Mat")
				currentFrame = nil
				continue
			}
			defer goImg.Close()

			// Get Mat data pointer
			goImgPtr, err := goImg.DataPtrUint8()
			if err != nil {
				slog.Error("Failed to get Mat data pointer", "error", err)
				currentFrame = nil
				continue
			}

			// Convert YUV to BGR
			C.copy_frame_to_mat(
				img,
				(*C.uchar)(unsafe.Pointer(&goImgPtr[0])),
				C.uint(actualWidth),
				C.uint(actualHeight),
			)

			// Save frame
			filename := fmt.Sprintf("frame_%d.jpg", frameCount)
			if ok := gocv.IMWrite(filename, goImg); !ok {
				slog.Error("Failed to write image", "filename", filename)
			} else {
				slog.Info("Saved frame",
					"filename", filename,
					"width", actualWidth,
					"height", actualHeight)
			}

			frameCount++
			currentFrame = nil
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
