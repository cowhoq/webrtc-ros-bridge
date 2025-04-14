package peerconnectionchannel

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/3DRX/webrtc-ros-bridge/config"
	sensor_msgs_msg "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/sensor_msgs/msg"
	rosmediadevicesadapter "github.com/3DRX/webrtc-ros-bridge/ros_mediadevices_adapter"
	bandwidthmanager "github.com/3DRX/webrtc-ros-bridge/sender/bandwidth_manager"
	send_signalingchannel "github.com/3DRX/webrtc-ros-bridge/sender/signaling_channel"
	"github.com/pion/interceptor"
	"github.com/pion/mediadevices"
	"github.com/pion/mediadevices/pkg/codec/vpx"
	"github.com/pion/mediadevices/pkg/prop"
	"github.com/pion/webrtc/v4"
	"github.com/tiiuae/rclgo/pkg/rclgo"
	"github.com/tiiuae/rclgo/pkg/rclgo/types"
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
	sensorChan        <-chan types.Message
	chanDispatcher    func()
	sendSDPChan       chan<- webrtc.SessionDescription
	recvSDPChan       <-chan webrtc.SessionDescription
	sendCandidateChan chan<- webrtc.ICECandidateInit
	recvCandidateChan <-chan webrtc.ICECandidateInit
	peerConnection    *webrtc.PeerConnection
	bandwidthManager  *bandwidthmanager.BandwidthManager
	vp8Params         vpx.VP8Params
}

func InitPeerConnectionChannel(
	messageChan <-chan types.Message,
	sendSDPChan chan<- webrtc.SessionDescription,
	recvSDPChan <-chan webrtc.SessionDescription,
	sendCandidateChan chan<- webrtc.ICECandidateInit,
	recvCandidateChan <-chan webrtc.ICECandidateInit,
	action *send_signalingchannel.Action,
	imgSpec *config.ImageSpecifications,
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
	// TODO: read data from action and use the action to select
	// ROS topic to send through bridge.
	// For now, we just send the ROS topic specified in the config.

	// create a dispatch goroutine to split image message from other sensor messages
	imgChan := make(chan *sensor_msgs_msg.Image, 10)
	sensorChan := make(chan types.Message, 10)
	var imgWidth, imgHeight int = 640, 480
	var frameRate float64 = 30.00
	if imgSpec.Width != 0 && imgSpec.Height != 0 && imgSpec.FrameRate != 0 {
		imgWidth = imgSpec.Width
		imgHeight = imgSpec.Height
		frameRate = imgSpec.FrameRate
	}

	rosmediadevicesadapter.Initialize(imgChan, imgWidth, imgHeight, frameRate)
	vp8Params, err := vpx.NewVP8Params()
	if err != nil {
		panic(err)
	}

	// 初始化带宽管理器
	bwManager := bandwidthmanager.NewBandwidthManager(bandwidthmanager.BandwidthManagerConfig{
		TotalBandwidth:          10_000_000, // 10 Mbps
		MinVideoBitrate:         500_000,    // 500 Kbps
		MaxVideoBitrate:         8_000_000,  // 8 Mbps
		TargetVideoBitrate:      5_000_000,  // 5 Mbps
		MinDataChannelBandwidth: 500_000,    // 500 Kbps
	})

	// 设置初始视频比特率
	vp8Params.BitRate = 5_000_000 // 使用固定值作为初始比特率

	// 更新带宽管理器初始比特率设置
	bwManager.SetInitialVideoBitrate(int(vp8Params.BitRate))

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
			constraint.Width = prop.Int(imgWidth)
			constraint.Height = prop.Int(imgHeight)
			constraint.FrameRate = prop.Float(frameRate)
		},
		Codec: codecselector,
	})
	if err != nil {
		panic(err)
	}
	for _, videoTrack := range mediaStream.GetVideoTracks() {
		videoTrack.OnEnded(func(err error) {
			slog.Error("Track ended", "error", err)
		})
		_, err := peerConnection.AddTransceiverFromTrack(
			videoTrack,
			webrtc.RTPTransceiverInit{
				Direction: webrtc.RTPTransceiverDirectionSendonly,
			},
		)
		if err != nil {
			panic(err)
		}
		slog.Info("add video track success")
	}

	pc := &PeerConnectionChannel{
		sendSDPChan:       sendSDPChan,
		recvSDPChan:       recvSDPChan,
		sendCandidateChan: sendCandidateChan,
		recvCandidateChan: recvCandidateChan,
		peerConnection:    peerConnection,
		imgChan:           imgChan,
		sensorChan:        sensorChan,
		bandwidthManager:  bwManager,
		vp8Params:         vp8Params,
		chanDispatcher: func() {
			for {
				msg := <-messageChan
				switch msg.(type) {
				case *sensor_msgs_msg.Image:
					imgChan <- msg.(*sensor_msgs_msg.Image)
				default:
					sensorChan <- msg
				}
			}
		},
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
	go pc.chanDispatcher()

	// 启动带宽管理器
	pc.bandwidthManager.Start()
	defer pc.bandwidthManager.Stop()

	datachannel, err := pc.peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		panic(err)
	}

	// 将数据通道注册到带宽管理器
	pc.bandwidthManager.SetDataChannel(datachannel)

	datachannel.OnOpen(func() {
		slog.Info("datachannel open", "label", datachannel.Label(), "ID", datachannel.ID())
		for {
			sensorMsg := <-pc.sensorChan
			serializedMsg, err := rclgo.Serialize(sensorMsg)
			if err != nil {
				slog.Error("failed to serialize sensor message", "error", err)
				continue
			}

			// 向带宽管理器报告数据使用情况
			msgType := getMsgType(sensorMsg)
			pc.bandwidthManager.RegisterMessageTraffic(msgType, len(serializedMsg))

			datachannel.Send(serializedMsg)
		}
	})
	datachannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		slog.Info("datachannel message", "data", string(msg.Data))
	})

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

func checkImgSpec(cfg *config.Config) [2]int {
	tmp := cfg.Topics[0].ImgSpec

	width, height := 640, 480
	if tmp.Width != 0 && tmp.Height != 0 {
		width = tmp.Width
		height = tmp.Height
	}
	return [2]int{
		width,
		height,
	}
}

// getMsgType 返回消息的类型名称
func getMsgType(msg types.Message) string {
	// 简单地使用类型名称作为标识
	return fmt.Sprintf("%T", msg)
}
