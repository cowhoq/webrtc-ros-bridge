package peerconnectionchannel

import (
	"testing"
	"github.com/3DRX/webrtc-ros-bridge/config"
)

func TestImgSpecfunc(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		expected [2]int
	}{
		{
			name: "valid config",
			cfg: &config.Config{
				Mode: "receiver",
				Addr: "10.3.9.3:8080",
				Topics: []config.TopicConfig{
					{
						NameIn:  "image_raw",
						NameOut: "image",
						Type:    "sensor_msgs/msg/Image",
						ImgSpec: config.ImageSpecifications{
							Width:     1920,
							Height:    1080,
							FrameRate: 30,
						},
					},
				},
			},
			expected: [2]int{1920, 1080},
		},
		{
			name: "valid config with hostname",
			cfg: &config.Config{
				Mode: "receiver",
				Addr: "localhost:8080",
				Topics: []config.TopicConfig{
					{
						NameIn:  "image_raw",
						NameOut: "image",
						Type:    "sensor_msgs/msg/Image",
						ImgSpec: config.ImageSpecifications{
							Width:         0,
							Height:        480,
							FrameRate:     29.97,
						},
					},
				},
			},
			expected: [2]int{640, 480},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := checkImgSpec(tt.cfg)
			if res != tt.expected {
				t.Errorf("TestCheckfunc failed: expected status %v, got %v", tt.expected, res)
			}
		})
	}
}
