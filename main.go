package main

import (
	"bufio"
	"flag"
	"fmt"
	"os/exec"
	"strconv"
	"sync"

	codecchannel "github.com/3DRX/webrtc-ros-bridge/codec_channel"
	peerconnectionchannel "github.com/3DRX/webrtc-ros-bridge/peer_connection_channel"
	signalingchannel "github.com/3DRX/webrtc-ros-bridge/signaling_channel"
	"github.com/pion/webrtc/v4"
)

const (
	frameX    = 640
	frameY    = 480
	frameSize = frameX * frameY * 3
)

func main() {
	ffmpeg := exec.Command(
		"ffmpeg",
		"-i",
		"pipe:0",
		"-pix_fmt",
		"bgr24",
		"-s",
		strconv.Itoa(frameX)+"x"+strconv.Itoa(frameY),
		"-f",
		"rawvideo",
		"pipe:1",
	)
	ffmpegIn, err := ffmpeg.StdinPipe()
	if err != nil {
		panic(err)
	}
	ffmpegOut, err := ffmpeg.StdoutPipe()
	if err != nil {
		panic(err)
	}
	ffmpegErr, err := ffmpeg.StderrPipe()
	if err != nil {
		panic(err)
	}
	if err := ffmpeg.Start(); err != nil {
		panic(err)
	}
	go func() {
		scanner := bufio.NewScanner(ffmpegErr)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
	}()

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
		ffmpegIn,
	)
	cc := codecchannel.InitCodecChannel(
		frameX,
		frameY,
		frameSize,
		ffmpegOut,
	)
	go sc.Spin()
	go pc.Spin()
	go cc.Spin()
	select {}
}
