package peerconnectionchannel

import (
	"encoding/json"
	"errors"
	"log/slog"

	sensor_msgs_msg "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/sensor_msgs/msg"
	send_signalingchannel "github.com/3DRX/webrtc-ros-bridge/sender/signaling_channel"
	"github.com/pion/interceptor"
	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/codec/vpx"
	_ "github.com/pion/mediadevices/pkg/driver/camera"
	// _ "github.com/pion/mediadevices/pkg/driver/videotest"
	"github.com/pion/mediadevices/pkg/prop"
	"github.com/pion/webrtc/v3"
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
	vp8Params, err := vpx.NewVP8Params()
	if err != nil {
		panic(err)
	}
	vp8Params.BitRate = 5_000_000
	codecselector := mediadevices.NewCodecSelector(
		mediadevices.WithVideoEncoders(&vp8Params),
	)
	m := &webrtc.MediaEngine{}
	codecselector.Populate(m)
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

	mediaStream, err := mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
		Video: func(constraint *mediadevices.MediaTrackConstraints) {
			// Query for ideal resolutions
			constraint.Width = prop.Int(640)
			constraint.Height = prop.Int(480)
		},
		Codec: codecselector,
	})
	// mediaStream, err := mediadevices.GetUserMedia(mediadevices.MediaStreamConstraints{
	// 	Video: func(c *mediadevices.MediaTrackConstraints) {
	// 		c.DeviceID = prop.String("ros_image_topic") // Must match the Label from Initialize()
	// 		c.Width = prop.Int(640)
	// 		c.Height = prop.Int(480)
	// 		c.FrameRate = prop.Float(30)
	// 	},
	// })
	if err != nil {
		panic(err)
	}
	for _, videoTrack := range mediaStream.GetVideoTracks() {
		videoTrack.OnEnded(func(err error) {
			slog.Error("Track ended", "error", err)
		})
		_, err := peerConnection.AddTransceiverFromTrack(
			videoTrack,
			webrtc.RtpTransceiverInit{
				Direction: webrtc.RTPTransceiverDirectionSendonly,
			},
		)
		if err != nil {
			panic(err)
		}
		slog.Info("add video track success")
	}

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
	return pc
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
