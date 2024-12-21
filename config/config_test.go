package config

import (
	"testing"
	"github.com/tiiuae/rclgo/pkg/rclgo"

)

func TestCheckfunc(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *Config
		expected bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				Mode: "receiver",
				Addr: "localhost:8080",
				Topics: []TopicConfig{
					{
						NameIn:  "image_raw",
						NameOut: "image",
						Type:    "sensor_msgs/msg/Image",
						ImgSpec: ImageSpecifications{
							Width:     640,
							Height:    480,
							FrameRate: 29.97,
						},
						Qos: &rclgo.QosProfile{
							History:   rclgo.HistoryKeepLast,
							Reliability: rclgo.ReliabilityBestEffort,
							Durability: rclgo.DurabilityVolatile,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "invalid config without qos",
			cfg: &Config{
				Mode: "receiver",
				Addr: "localhost:8080",
				Topics: []TopicConfig{
					{
						NameIn:  "image_raw",
						NameOut: "image",
						Type:    "sensor_msgs/msg/Image",
						ImgSpec: ImageSpecifications{
							Width:     640,
							Height:    480,
							FrameRate: 29.97,
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "valid config",
			cfg: &Config{
				Mode: "sender",
				Addr: "10.3.9.3:8080",
				Topics: []TopicConfig{
					{
						NameIn:  "image_raw",
						NameOut: "image",
						Type:    "sensor_msgs/msg/Image",
						ImgSpec: ImageSpecifications{
							Width:     640,
							Height:    480,
							FrameRate: 30,
						},
					},
					{
						NameIn:  "image_raw",
						NameOut: "image",
						Type:    "sensor_msgs/msg/Image",
						ImgSpec: ImageSpecifications{
							Width:     640,
							Height:    480,
							FrameRate: 29.97,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "invalid config with wrong imgspec",
			cfg: &Config{
				Mode: "sender",
				Addr: "localhost:8080",
				Topics: []TopicConfig{
					{
						NameIn:  "image_raw",
						NameOut: "image",
						Type:    "sensor_msgs/msg/Image",
						ImgSpec: ImageSpecifications{
							Width:     0,
							Height:    480,
							FrameRate: 29.97,
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "valid config",
			cfg: &Config{
				Mode: "sender",
				Addr: "localhost:8080",
				Topics: []TopicConfig{
					{
						NameIn:  "image_raw",
						NameOut: "image",
						Type:    "sensor_msgs/msg/LaserScan",
					},
				},
			},
			expected: true,
		},
		{
			name: "invalid config",
			cfg: &Config{
				Mode: "",
				Addr: "",
				Topics: []TopicConfig{
					{
						NameIn:  "image_raw",
						NameOut: "image",
						Type:    "sensor_msgs/msg/Image",
					},
					{
						NameIn:  "image_raw",
						NameOut: "image",
						Type:    "sensor_msgs/msg/LaserScan",
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkCfg(tt.cfg)
			msg := true
			if err != nil {
				msg = false
			}
			if msg != tt.expected {
				t.Errorf("TestCheckfunc failed: expected status %v, got %v", tt.expected, err.Error())
			}
		})
	}
}
