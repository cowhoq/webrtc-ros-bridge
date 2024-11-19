package peerconnectionchannel

/*
#cgo LDFLAGS: -L. -lvp8encoder -lvpx -lm
#include "vp8_encoder.h"
#include "../../receiver/peer_connection_channel/vp8_decoder.h"
#include <stdlib.h>
*/
import "C"

import (
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"time"
	"unsafe"

	sensor_msgs_msg "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/sensor_msgs/msg"
	send_signalingchannel "github.com/3DRX/webrtc-ros-bridge/sender/signaling_channel"
	"github.com/at-wat/ebml-go/webm"
	"github.com/pion/interceptor"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media"
)

type AddStreamAction struct {
	Type string `json:"type"`
	Id   string `json:"id"`
}

type AddVideoTrackAction struct {
	Type     string `json:"type"`
	Id       string `json:"id"`
	StreamId string `json:"stream_id"`
	SrcId    string `json:"src"`
}

type PeerConnectionChannel struct {
	imgChan           <-chan *sensor_msgs_msg.Image
	codec             C.vpx_codec_ctx_t
	writer            webm.BlockWriteCloser
	sendSDPChan       chan<- webrtc.SessionDescription
	recvSDPChan       <-chan webrtc.SessionDescription
	sendCandidateChan chan<- webrtc.ICECandidateInit
	recvCandidateChan <-chan webrtc.ICECandidateInit
	peerConnection    *webrtc.PeerConnection
	id                string
	streamId          string
}

func InitPeerConnectionChannel(
	imgChan <-chan *sensor_msgs_msg.Image,
	sendSDPChan chan<- webrtc.SessionDescription,
	recvSDPChan <-chan webrtc.SessionDescription,
	sendCandidateChan chan<- webrtc.ICECandidateInit,
	recvCandidateChan <-chan webrtc.ICECandidateInit,
	action *send_signalingchannel.Action,
) *PeerConnectionChannel {
	// parse action
	if action.Type != "configure" {
		panic("Invalid action type")
	}
	rawActions := action.Actions
	if len(rawActions) != 2 {
		panic("Invalid number of actions")
	}
	rawAddStream := rawActions[0]
	rawAddVideoTrack := rawActions[1]
	// bind raw actions to struct
	addStreamAction := AddStreamAction{}
	addVideoTrackAction := AddVideoTrackAction{}
	if err := unmarshalAction(rawAddStream, &addStreamAction); err != nil {
		panic(err)
	}
	if err := unmarshalAction(rawAddVideoTrack, &addVideoTrackAction); err != nil {
		panic(err)
	}
	id := addVideoTrackAction.Id
	streamId := addVideoTrackAction.StreamId
	// TODO: read data from action and use the action to select
	// ROS topic to send through bridge.
	// For now, we just send the ROS topic specified in the config.
	m := &webrtc.MediaEngine{}
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:  webrtc.MimeTypeVP8,
			ClockRate: 90000,
			Channels:  0,
		},
		PayloadType: 96,
	}, webrtc.RTPCodecTypeVideo); err != nil {
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
	slog.Info("Created peer connection")

	pc := &PeerConnectionChannel{
		imgChan:           imgChan,
		sendSDPChan:       sendSDPChan,
		recvSDPChan:       recvSDPChan,
		sendCandidateChan: sendCandidateChan,
		recvCandidateChan: recvCandidateChan,
		peerConnection:    peerConnection,
		id:                id,
		streamId:          streamId,
	}
	if C.init_encoder(&pc.codec, 640, 480, 1000) != 0 {
		panic("Failed to initialize VP8 encoder")
	}
	file, err := os.Create("video.webm")
	if err != nil {
		panic(err)
	}

	writer, err := webm.NewSimpleBlockWriter(file, []webm.TrackEntry{
		{
			Name:            "Video",
			TrackNumber:     2,
			TrackUID:        67890,
			CodecID:         "V_VP8",
			TrackType:       1,
			DefaultDuration: 33333333,
			Video: &webm.Video{
				PixelWidth:  uint64(640),
				PixelHeight: uint64(480),
			},
		},
	})
	if err != nil {
		panic(err)
	}

	pc.writer = writer[0]
	return pc
}

func registerHeaderExtensionURI(m *webrtc.MediaEngine, uris []string) {
	for _, uri := range uris {
		err := m.RegisterHeaderExtension(
			webrtc.RTPHeaderExtensionCapability{
				URI: uri,
			},
			webrtc.RTPCodecTypeVideo,
			webrtc.RTPTransceiverDirectionSendonly,
		)
		if err != nil {
			panic(err)
		}
	}
}

func (pc *PeerConnectionChannel) handleRemoteICECandidate() {
	for {
		candidate := <-pc.recvCandidateChan
		if err := pc.peerConnection.AddICECandidate(candidate); err != nil {
			panic(err)
		}
	}
}

func (pc *PeerConnectionChannel) Spin() {
	videoTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{
			MimeType: webrtc.MimeTypeVP8,
		},
		pc.id,
		pc.streamId,
	)
	if err != nil {
		panic(err)
	}
	rtpSender, err := pc.peerConnection.AddTrack(videoTrack)
	if err != nil {
		panic(err)
	}
	go func() {
		// Read incoming RTCP packets
		// Before these packets are returned they are processed by interceptors. For things
		// like NACK this needs to be called.
		rtcpBuf := make([]byte, 1500)
		for {
			if _, _, rtcpErr := rtpSender.Read(rtcpBuf); rtcpErr != nil {
				slog.Error("Failed to read RTCP packet", "error", rtcpErr)
				return
			}
		}
	}()

	defer pc.writer.Close()
	offer, err := pc.peerConnection.CreateOffer(nil)
	if err != nil {
		panic(err)
	}
	pc.peerConnection.SetLocalDescription(offer)
	pc.peerConnection.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		pc.sendCandidateChan <- c.ToJSON()
	})
	go pc.handleRemoteICECandidate()
	pc.sendSDPChan <- offer
	remoteSDP := <-pc.recvSDPChan
	pc.peerConnection.SetRemoteDescription(remoteSDP)

	for {
		img := <-pc.imgChan

		var ros_img_c C.sensor_msgs__msg__Image
		sensor_msgs_msg.ImageTypeSupport.AsCStruct(unsafe.Pointer(&ros_img_c), img)
		var data *C.uint8_t
		var dataSize C.size_t
		if C.convert_and_encode(&pc.codec, &ros_img_c, &data, &dataSize) != 0 {
			slog.Error("Failed to encode frame")
			continue
		}
		// Write encoded data to WebM file
		goData := C.GoBytes(unsafe.Pointer(data), C.int(dataSize))
		videoKeyframe := (goData[0] & 0x1) == 0 // Check if the frame is a keyframe
		// Calculate timeStamp from img.Header
		timeStampMs := uint32(img.Header.Stamp.Sec)*1000 + img.Header.Stamp.Nanosec/1000000

		// Write to file
		if _, err := pc.writer.Write(videoKeyframe, int64(timeStampMs), goData); err != nil {
			slog.Error("Failed to write to WebM file", "error", err)
		}
		// Write to WebRTC track
		if err = videoTrack.WriteSample(media.Sample{Data: goData, Duration: time.Second}); err != nil {
			slog.Error("Failed to write to video track", "error", err)
		}

		// Free the memory allocated for the encoded data
		C.cleanup_ros_image(&ros_img_c)
		C.free(unsafe.Pointer(data))

		slog.Info(
			"Received image message",
			"width", img.Width,
			"height", img.Height,
			"header", img.Header,
			"keyframe", videoKeyframe,
		)
	}
}

func unmarshalAction(rawAction interface{}, action interface{}) error {
	rawActionMap, ok := rawAction.(map[string]interface{})
	if !ok {
		return errors.New("Invalid action type")
	}
	rawActionBytes, err := json.Marshal(rawActionMap)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(rawActionBytes, action); err != nil {
		return err
	}
	return nil
}
