package peerconnectionchannel

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/3DRX/webrtc-ros-bridge/consts"
	control_msgs "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/autoware_control_msgs/msg"
	planning_msgs "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/autoware_planning_msgs/msg"
	vehicle_msgs "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/autoware_vehicle_msgs/msg"
	geom_msgs "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/geometry_msgs/msg"
	nav_msgs "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/nav_msgs/msg"
	sensor_msgs_msg "github.com/3DRX/webrtc-ros-bridge/rclgo_gen/sensor_msgs/msg"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v4"
	"github.com/tiiuae/rclgo/pkg/rclgo"
	"github.com/tiiuae/rclgo/pkg/rclgo/types"
)

// TypeHeaderSize 固定32字节的消息类型头
const TypeHeaderSize = 32

// extractTypeHeader 从消息中提取类型标识
func extractTypeHeader(data []byte) (string, []byte, error) {
	if len(data) < TypeHeaderSize {
		return "", nil, fmt.Errorf("消息太短，无法提取类型标识")
	}

	// 提取类型标识字节
	typeBytes := data[:TypeHeaderSize]

	// 转换为字符串，去除填充的零字节
	var i int
	for i = 0; i < len(typeBytes); i++ {
		if typeBytes[i] == 0 {
			break
		}
	}

	typeStr := string(typeBytes[:i])

	// 返回类型标识和实际数据
	return typeStr, data[TypeHeaderSize:], nil
}

type PeerConnectionChannel struct {
	sdpChan         <-chan webrtc.SessionDescription
	sdpReplyChan    chan<- webrtc.SessionDescription
	candidateChan   <-chan webrtc.ICECandidateInit
	peerConnection  *webrtc.PeerConnection
	signalCandidate func(c webrtc.ICECandidateInit) error
	messageChan     chan<- types.Message
}

// TypeSupportMap 存储所有支持的消息类型及其TypeSupport
var TypeSupportMap = map[string]types.MessageTypeSupport{
	consts.MSG_LASER_SCAN:   sensor_msgs_msg.LaserScanTypeSupport,
	consts.MSG_KINEMATIC:    nav_msgs.OdometryTypeSupport,
	consts.MSG_POSE_COV:     geom_msgs.PoseWithCovarianceStampedTypeSupport,
	consts.MSG_CONTROL_CMD:  control_msgs.ControlTypeSupport,
	consts.MSG_TRAJECTORY:   planning_msgs.TrajectoryTypeSupport,
	consts.MSG_CONTROL_MODE: vehicle_msgs.ControlModeReportTypeSupport,
	consts.MSG_VELOCITY:     vehicle_msgs.VelocityReportTypeSupport,
	consts.MSG_STEERING:     vehicle_msgs.SteeringReportTypeSupport,
	consts.MSG_GEAR:         vehicle_msgs.GearReportTypeSupport,
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
	signalCandidate func(c webrtc.ICECandidateInit) error,
	messageChan chan<- types.Message,
) *PeerConnectionChannel {
	m := &webrtc.MediaEngine{}
	// Register VP8
	if err := m.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeVP8, ClockRate: 90000, Channels: 0},
		PayloadType:        96,
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
	return &PeerConnectionChannel{
		sdpChan:         sdpChan,
		sdpReplyChan:    sdpReplyChan,
		candidateChan:   candidateChan,
		peerConnection:  peerConnection,
		signalCandidate: signalCandidate,
		messageChan:     messageChan,
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
	webmSaver := newWebmSaver(pc.messageChan)
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
		if err := pc.signalCandidate(c.ToJSON()); err != nil {
			panic(err)
		}
	})
	pc.peerConnection.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		slog.Info("PeerConnectionChannel: received track", "track", track.ID())
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
	pc.peerConnection.OnDataChannel(func(d *webrtc.DataChannel) {
		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			serializedMsg := msg.Data

			// 首先尝试提取类型标识
			msgType, actualData, err := extractTypeHeader(serializedMsg)
			if err == nil && msgType != "" {
				// 找到类型标识，查找对应的TypeSupport
				if typeSupport, exists := TypeSupportMap[msgType]; exists {
					// 使用正确的TypeSupport反序列化
					sensorMsg, err := rclgo.Deserialize(actualData, typeSupport)
					if err == nil {
						// 解析成功，发送到消息通道
						slog.Info("接收到数据通道消息", "类型", msgType)
						pc.messageChan <- sensorMsg
						return
					}
					slog.Error("反序列化失败", "类型", msgType, "错误", err)
				} else {
					slog.Error("未知的消息类型", "类型", msgType)
				}
			} else {
				// 没有类型标识，尝试所有已知类型（兼容旧版本）
				slog.Warn("消息中没有类型标识，尝试所有可能的类型")
				for msgType, typeSupport := range TypeSupportMap {
					sensorMsg, err := rclgo.Deserialize(serializedMsg, typeSupport)
					if err != nil {
						continue
					}

					// 解析成功，发送到消息通道
					slog.Info("接收到数据通道消息（旧版本格式）", "类型", msgType)
					pc.messageChan <- sensorMsg
					return
				}

				slog.Error("无法解析接收到的消息，未知的消息类型")
			}
		})

		d.OnOpen(func() {
			slog.Info("datachannel open", "label", d.Label(), "ID", d.ID())
			d.SendText("Hello from receiver!")
		})
	})
	go handleSignalingMessage(pc)
}
