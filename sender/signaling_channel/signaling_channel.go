package signalingchannel

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/3DRX/webrtc-ros-bridge/config"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v4"
)

type Action struct {
	Type    string                   `json:"type"`
	Actions []map[string]interface{} `json:"actions"`
}

type SignalingChannel struct {
	cfg                 *config.Config
	topicIdx            int
	upgrader            *websocket.Upgrader
	conn                *websocket.Conn
	actions             *Action
	haveReceiverPromise chan struct{}
	sendSDPChan         chan<- webrtc.SessionDescription
	recvSDPChan         <-chan webrtc.SessionDescription
	sendCandidateChan   chan<- webrtc.ICECandidateInit
	recvCandidateChan   <-chan webrtc.ICECandidateInit
}

func InitSignalingChannel(
	cfg *config.Config,
	topicIdx int,
	sendSDPChan chan<- webrtc.SessionDescription,
	recvSDPChan <-chan webrtc.SessionDescription,
	sendCandidateChan chan<- webrtc.ICECandidateInit,
	recvCandidateChan <-chan webrtc.ICECandidateInit,
) *SignalingChannel {
	return &SignalingChannel{
		cfg:      cfg,
		topicIdx: topicIdx,
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		conn:                nil,
		actions:             nil,
		haveReceiverPromise: make(chan struct{}),
		sendSDPChan:         sendSDPChan,
		recvSDPChan:         recvSDPChan,
		sendCandidateChan:   sendCandidateChan,
		recvCandidateChan:   recvCandidateChan,
	}
}

func (s *SignalingChannel) Spin() <-chan struct{} {
	mux := http.NewServeMux()
	mux.Handle("GET /webrtc", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.conn != nil {
			slog.Warn("already have a receiver, rejecting new connection")
			w.WriteHeader(http.StatusConflict)
			return
		}
		w.Write([]byte("webrtc"))
		conn, err := s.upgrader.Upgrade(w, r, nil)
		if err != nil {
			panic(err)
		}
		s.conn = conn
		go s.handleMessages()
	}))

	httpServer := &http.Server{
		Addr:    s.cfg.Addr,
		Handler: mux,
	}
	go func() {
		err := httpServer.ListenAndServe()
		if err != nil {
			panic(err)
		}
	}()
	return s.haveReceiverPromise
}

func (s *SignalingChannel) handleMessages() {
	for {
		_, message, err := s.conn.ReadMessage()
		if err != nil {
			slog.Error("websocket read error", "error", err)
			return
		}
		slog.Info("received message", "message", string(message))
		if s.actions == nil {
			// try to parse the message as an action
			newAction := &Action{}
			err := json.Unmarshal(message, newAction)
			if err != nil {
				slog.Error("failed to parse message as action", "error", err)
				continue
			}
			s.actions = newAction
			s.haveReceiverPromise <- struct{}{}
		}
	}
}
