package peerconnectionchannel

import (
	"fmt"
	"io"
	"time"

	"github.com/at-wat/ebml-go/webm"
	"github.com/pion/interceptor/pkg/jitterbuffer"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v4/pkg/media/samplebuilder"
)

const (
	naluTypeBitmask = 0b11111
	naluTypeSPS     = 7
)

type WebmSaver struct {
	videoWriter    webm.BlockWriteCloser
	vp8Builder     *samplebuilder.SampleBuilder
	videoTimestamp time.Duration

	h264JitterBuffer   *jitterbuffer.JitterBuffer
	lastVideoTimestamp uint32
	w                  io.WriteCloser
}

func newWebmSaver(w io.WriteCloser) *WebmSaver {
	return &WebmSaver{
		vp8Builder:       samplebuilder.New(10, &codecs.VP8Packet{}, 90000),
		h264JitterBuffer: jitterbuffer.New(),
		w:                w,
	}
}

func (s *WebmSaver) Close() {
	fmt.Printf("Finalizing webm...\n")
	if s.videoWriter != nil {
		if err := s.videoWriter.Close(); err != nil {
			panic(err)
		}
	}
}

func (s *WebmSaver) PushVP8(rtpPacket *rtp.Packet) {
	s.vp8Builder.Push(rtpPacket)

	for {
		sample := s.vp8Builder.Pop()
		if sample == nil {
			return
		}
		// Read VP8 header.
		videoKeyframe := (sample.Data[0]&0x1 == 0)
		if videoKeyframe {
			// Keyframe has frame information.
			raw := uint(sample.Data[6]) | uint(sample.Data[7])<<8 | uint(sample.Data[8])<<16 | uint(sample.Data[9])<<24
			width := int(raw & 0x3FFF)
			height := int((raw >> 16) & 0x3FFF)

			if s.videoWriter == nil {
				s.InitWriter(width, height)
			}
		}
		if s.videoWriter != nil {
			s.videoTimestamp += sample.Duration
			if _, err := s.videoWriter.Write(videoKeyframe, int64(s.videoTimestamp/time.Millisecond), sample.Data); err != nil {
				panic(err)
			}
		}
	}
}

func (s *WebmSaver) InitWriter(width, height int) {
	videoMimeType := "V_VP8"
	ws, err := webm.NewSimpleBlockWriter(s.w,
		[]webm.TrackEntry{
			{
				Name:            "Video",
				TrackNumber:     1,
				TrackUID:        67890,
				CodecID:         videoMimeType,
				TrackType:       1,
				DefaultDuration: 33333333,
				Video: &webm.Video{
					PixelWidth:  uint64(width),
					PixelHeight: uint64(height),
				},
			},
		})
	if err != nil {
		panic(err)
	}
	s.videoWriter = ws[0]
}
