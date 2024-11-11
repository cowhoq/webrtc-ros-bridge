package signalingchannel

import (
	"encoding/json"
	"log/slog"
	"net/url"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
	"golang.org/x/exp/rand"
)

type SignalingChannel struct {
	addr                string
	recv                chan []byte
	c                   *websocket.Conn
	sdpChan             chan<- webrtc.SessionDescription
	sdpReplyChan        <-chan webrtc.SessionDescription
	candidateChan       chan<- webrtc.ICECandidateInit
	candidateToPeerChan <-chan *webrtc.ICECandidate
}

type signalingResponse struct {
	Sdp  string
	Type string
}

func InitSignalingChannel(
	addr *string,
	sdpChan chan webrtc.SessionDescription,
	sdpReplyChan <-chan webrtc.SessionDescription,
	candidateChan chan<- webrtc.ICECandidateInit,
	candidateToPeerChan <-chan *webrtc.ICECandidate,
) *SignalingChannel {
	return &SignalingChannel{
		addr:                *addr,
		recv:                make(chan []byte),
		c:                   nil,
		sdpChan:             sdpChan,
		sdpReplyChan:        sdpReplyChan,
		candidateChan:       candidateChan,
		candidateToPeerChan: candidateToPeerChan,
	}
}

func newStreamId() string {
	return "webrtc_ros-stream-" + strconv.Itoa(rand.Intn(1000000000))
}

func composeActions() map[string]interface{} {
	streamId := newStreamId()
	action := map[string]interface{}{
		"type": "configure",
		"actions": []map[string]interface{}{
			{
				"type": "add_stream",
				"id":   streamId,
			},
			{
				"type":      "add_video_track",
				"stream_id": streamId,
				"id":        streamId + "/subscribed_video",
				"src":       "ros_image:/image",
			},
		},
	}
	return action
}

func toTextMessage(data map[string]interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return jsonData, nil
}

func (s *SignalingChannel) SignalCandidate(candidate webrtc.ICECandidateInit) error {
	candidateMsg := map[string]interface{}{
		"type":            "ice_candidate",
		"candidate":       candidate.Candidate,
		"sdp_mid":         candidate.SDPMid,
		"sdp_mline_index": candidate.SDPMLineIndex,
	}
	payload, err := toTextMessage(candidateMsg)
	if err != nil {
		slog.Error("marshal error", "error", err)
	}
	s.c.WriteMessage(websocket.TextMessage, payload)
	slog.Info("send candidate", "candidate", string(payload))
	return nil
}

func (s *SignalingChannel) Spin() {
	u := url.URL{Scheme: "ws", Host: s.addr, Path: "/webrtc"}
	slog.Info("start spinning", "url", u.String())
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	s.c = c
	defer c.Close()
	if err != nil {
		slog.Error("dial error", "error", err)
		return
	}
	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				slog.Error("recv error", "err", err)
				return
			}
			s.recv <- message
		}
	}()
	slog.Info("dial success")

	cfgMessage, err := toTextMessage(composeActions())
	if err != nil {
		slog.Error("compose message error", "error", err)
		return
	}
	c.WriteMessage(websocket.TextMessage, cfgMessage)
	slog.Info("send configure message")
	recvRaw := <-s.recv
	sdp := webrtc.SessionDescription{}
	err = json.Unmarshal(recvRaw, &sdp)
	if err != nil {
		slog.Error("unmarshal error", "error", err)
		return
	}
	s.sdpChan <- sdp
	slog.Info("recv sdp")
	answer := <-s.sdpReplyChan // await answer from peer connection
	payload, err := json.Marshal(answer)
	if err != nil {
		slog.Error("marshal error", "error", err)
	}
	c.WriteMessage(websocket.TextMessage, payload)
	slog.Info("send answer")
	for {
		select {
		case candidateRaw := <-s.recv:
			iceCandidate := webrtc.ICECandidateInit{
				Candidate: string(candidateRaw),
			}
			s.candidateChan <- iceCandidate
		case candidate := <-s.candidateToPeerChan:
			if candidate == nil {
				slog.Info("nil candidate")
				continue
			}
			candidateJSON := candidate.ToJSON()
			s.SignalCandidate(candidateJSON)
		}
	}
}
